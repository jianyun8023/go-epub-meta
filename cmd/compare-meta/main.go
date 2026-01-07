package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"golibri/epub"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ComparisonResult struct {
	FilePath string

	// golibri fields
	GolibriTitle       string
	GolibriAuthor      string
	GolibriPublisher   string
	GolibriPublished   string
	GolibriSeries      string
	GolibriLanguage    string
	GolibriIdentifiers string
	GolibriCover       string
	GolibriError       string
	GolibriTime        time.Duration

	// ebook-meta fields
	EbookTitle       string
	EbookTitleSort   string
	EbookAuthor      string
	EbookPublisher   string
	EbookProducer    string
	EbookLanguage    string
	EbookPublished   string
	EbookIdentifiers string
	EbookSeries      string
	EbookError       string
	EbookTime        time.Duration
}

var (
	concurrency = flag.Int("c", 10, "Number of concurrent workers")
	outputFile  = flag.String("o", "comparison_results.csv", "Output CSV file")
	skipErrors  = flag.Bool("skip-errors", true, "Skip files with errors")
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <directory>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	dir := flag.Arg(0)

	// Find all epub files
	fmt.Println("Scanning directory for EPUB files...")
	epubFiles, err := findEpubFiles(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d EPUB files\n", len(epubFiles))

	if len(epubFiles) == 0 {
		fmt.Println("No EPUB files found")
		return
	}

	// Process files concurrently
	results := processFiles(epubFiles, *concurrency)

	// Write results to CSV
	fmt.Printf("\nWriting results to %s...\n", *outputFile)
	if err := writeResults(results, *outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing results: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	printSummary(results)

	fmt.Println("\nDone!")
}

func findEpubFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".epub") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func processFiles(files []string, concurrency int) []ComparisonResult {
	var (
		wg      sync.WaitGroup
		results []ComparisonResult
		mu      sync.Mutex
		counter int64
		total   = int64(len(files))
	)

	// Create a channel to limit concurrency
	sem := make(chan struct{}, concurrency)

	// Progress reporter
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				current := atomic.LoadInt64(&counter)
				percentage := float64(current) / float64(total) * 100
				fmt.Printf("\rProgress: %d/%d (%.1f%%) ", current, total, percentage)
			case <-done:
				return
			}
		}
	}()

	for _, file := range files {
		wg.Add(1)
		go func(filepath string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			result := compareFile(filepath)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()

			atomic.AddInt64(&counter, 1)
		}(file)
	}

	wg.Wait()
	close(done)
	fmt.Printf("\rProgress: %d/%d (100.0%%) \n", total, total)

	return results
}

func compareFile(filepath string) ComparisonResult {
	result := ComparisonResult{
		FilePath: filepath,
	}

	// Test golibri
	start := time.Now()
	if ep, err := epub.Open(filepath); err != nil {
		result.GolibriError = err.Error()
	} else {
		defer ep.Close()
		result.GolibriTitle = ep.Package.GetTitle()
		result.GolibriAuthor = ep.Package.GetAuthor()
		result.GolibriSeries = ep.Package.GetSeries()
		result.GolibriLanguage = ep.Package.GetLanguage()
		result.GolibriPublisher = ep.Package.GetPublisher()
		result.GolibriPublished = ep.Package.GetPublishDate()

		// Get identifiers
		identifiers := ep.Package.GetIdentifiers()
		var idParts []string
		for scheme, value := range identifiers {
			idParts = append(idParts, fmt.Sprintf("%s:%s", scheme, value))
		}
		result.GolibriIdentifiers = strings.Join(idParts, ", ")

		// Check cover
		if _, _, err := ep.GetCoverImage(); err == nil {
			result.GolibriCover = "Found"
		} else {
			result.GolibriCover = "Not Found"
		}
	}
	result.GolibriTime = time.Since(start)

	// Test ebook-meta
	start = time.Now()
	output, err := exec.Command("ebook-meta", filepath).Output()
	result.EbookTime = time.Since(start)

	if err != nil {
		result.EbookError = err.Error()
	} else {
		parseEbookMeta(string(output), &result)
	}

	return result
}

func parseEbookMeta(output string, result *ComparisonResult) {
	lines := strings.Split(output, "\n")

	// Regular expressions to parse ebook-meta output
	patterns := map[string]*regexp.Regexp{
		"title":       regexp.MustCompile(`^Title\s*:\s*(.+)$`),
		"titleSort":   regexp.MustCompile(`^Title sort\s*:\s*(.+)$`),
		"author":      regexp.MustCompile(`^Author\(s\)\s*:\s*(.+)$`),
		"publisher":   regexp.MustCompile(`^Publisher\s*:\s*(.+)$`),
		"producer":    regexp.MustCompile(`^Book Producer\s*:\s*(.+)$`),
		"language":    regexp.MustCompile(`^Languages?\s*:\s*(.+)$`),
		"published":   regexp.MustCompile(`^Published\s*:\s*(.+)$`),
		"identifiers": regexp.MustCompile(`^Identifiers?\s*:\s*(.+)$`),
		"series":      regexp.MustCompile(`^Series\s*:\s*(.+)$`),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if m := patterns["title"].FindStringSubmatch(line); m != nil {
			result.EbookTitle = strings.TrimSpace(m[1])
		} else if m := patterns["titleSort"].FindStringSubmatch(line); m != nil {
			result.EbookTitleSort = strings.TrimSpace(m[1])
		} else if m := patterns["author"].FindStringSubmatch(line); m != nil {
			result.EbookAuthor = strings.TrimSpace(m[1])
		} else if m := patterns["publisher"].FindStringSubmatch(line); m != nil {
			result.EbookPublisher = strings.TrimSpace(m[1])
		} else if m := patterns["producer"].FindStringSubmatch(line); m != nil {
			result.EbookProducer = strings.TrimSpace(m[1])
		} else if m := patterns["language"].FindStringSubmatch(line); m != nil {
			result.EbookLanguage = strings.TrimSpace(m[1])
		} else if m := patterns["published"].FindStringSubmatch(line); m != nil {
			result.EbookPublished = strings.TrimSpace(m[1])
		} else if m := patterns["identifiers"].FindStringSubmatch(line); m != nil {
			result.EbookIdentifiers = strings.TrimSpace(m[1])
		} else if m := patterns["series"].FindStringSubmatch(line); m != nil {
			result.EbookSeries = strings.TrimSpace(m[1])
		}
	}
}

func writeResults(results []ComparisonResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"FilePath",
		"GolibriTitle", "GolibriAuthor", "GolibriPublisher", "GolibriPublished", "GolibriSeries", "GolibriLanguage", "GolibriIdentifiers", "GolibriCover",
		"GolibriError", "GolibriTime(ms)",
		"EbookTitle", "EbookTitleSort", "EbookAuthor", "EbookPublisher", "EbookProducer",
		"EbookLanguage", "EbookPublished", "EbookIdentifiers", "EbookSeries",
		"EbookError", "EbookTime(ms)",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, r := range results {
		row := []string{
			r.FilePath,
			r.GolibriTitle, r.GolibriAuthor, r.GolibriPublisher, r.GolibriPublished, r.GolibriSeries, r.GolibriLanguage, r.GolibriIdentifiers, r.GolibriCover,
			r.GolibriError, fmt.Sprintf("%.2f", r.GolibriTime.Seconds()*1000),
			r.EbookTitle, r.EbookTitleSort, r.EbookAuthor, r.EbookPublisher, r.EbookProducer,
			r.EbookLanguage, r.EbookPublished, r.EbookIdentifiers, r.EbookSeries,
			r.EbookError, fmt.Sprintf("%.2f", r.EbookTime.Seconds()*1000),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func printSummary(results []ComparisonResult) {
	total := len(results)
	golibriSuccess := 0
	ebookSuccess := 0

	// Field statistics
	fieldStats := map[string]struct{ ebook, golibri, match int }{
		"Title":       {0, 0, 0},
		"Author":      {0, 0, 0},
		"Series":      {0, 0, 0},
		"Language":    {0, 0, 0},
		"Identifiers": {0, 0, 0},
		"Publisher":   {0, 0, 0},
		"Published":   {0, 0, 0},
		"Producer":    {0, 0, 0},
		"TitleSort":   {0, 0, 0},
	}

	for _, r := range results {
		if r.GolibriError == "" {
			golibriSuccess++
		}
		if r.EbookError == "" {
			ebookSuccess++
		}

		// Count non-empty fields
		if r.EbookTitle != "" {
			s := fieldStats["Title"]
			s.ebook++
			fieldStats["Title"] = s
		}
		if r.GolibriTitle != "" {
			s := fieldStats["Title"]
			s.golibri++
			if normalize(r.GolibriTitle) == normalize(r.EbookTitle) {
				s.match++
			}
			fieldStats["Title"] = s
		}

		if r.EbookAuthor != "" {
			s := fieldStats["Author"]
			s.ebook++
			fieldStats["Author"] = s
		}
		if r.GolibriAuthor != "" {
			s := fieldStats["Author"]
			s.golibri++
			if normalize(r.GolibriAuthor) == normalize(r.EbookAuthor) {
				s.match++
			}
			fieldStats["Author"] = s
		}

		if r.EbookSeries != "" {
			s := fieldStats["Series"]
			s.ebook++
			fieldStats["Series"] = s
		}
		if r.GolibriSeries != "" {
			s := fieldStats["Series"]
			s.golibri++
			if normalize(r.GolibriSeries) == normalize(r.EbookSeries) {
				s.match++
			}
			fieldStats["Series"] = s
		}

		if r.EbookLanguage != "" {
			s := fieldStats["Language"]
			s.ebook++
			fieldStats["Language"] = s
		}
		if r.GolibriLanguage != "" {
			s := fieldStats["Language"]
			s.golibri++
			fieldStats["Language"] = s
		}

		if r.EbookIdentifiers != "" {
			s := fieldStats["Identifiers"]
			s.ebook++
			fieldStats["Identifiers"] = s
		}
		if r.GolibriIdentifiers != "" {
			s := fieldStats["Identifiers"]
			s.golibri++
			fieldStats["Identifiers"] = s
		}

		if r.EbookPublisher != "" {
			s := fieldStats["Publisher"]
			s.ebook++
			fieldStats["Publisher"] = s
		}
		if r.GolibriPublisher != "" {
			s := fieldStats["Publisher"]
			s.golibri++
			if normalize(r.GolibriPublisher) == normalize(r.EbookPublisher) {
				s.match++
			}
			fieldStats["Publisher"] = s
		}

		if r.EbookPublished != "" {
			s := fieldStats["Published"]
			s.ebook++
			fieldStats["Published"] = s
		}
		if r.GolibriPublished != "" {
			s := fieldStats["Published"]
			s.golibri++
			if normalize(r.GolibriPublished) == normalize(r.EbookPublished) {
				s.match++
			}
			fieldStats["Published"] = s
		}

		if r.EbookProducer != "" {
			s := fieldStats["Producer"]
			s.ebook++
			fieldStats["Producer"] = s
		}

		if r.EbookTitleSort != "" {
			s := fieldStats["TitleSort"]
			s.ebook++
			fieldStats["TitleSort"] = s
		}
	}

	fmt.Println("\n=== SUMMARY ===")
	fmt.Printf("Total files: %d\n", total)
	fmt.Printf("Golibri success: %d (%.1f%%)\n", golibriSuccess, float64(golibriSuccess)/float64(total)*100)
	fmt.Printf("Ebook-meta success: %d (%.1f%%)\n", ebookSuccess, float64(ebookSuccess)/float64(total)*100)

	fmt.Println("\n=== FIELD COVERAGE ===")
	fmt.Printf("%-15s  %-20s  %-20s  %-15s\n", "Field", "Ebook-meta", "Golibri", "Match Rate")
	fmt.Println(strings.Repeat("-", 80))

	for field, stats := range fieldStats {
		ebookPct := 0.0
		if total > 0 {
			ebookPct = float64(stats.ebook) / float64(total) * 100
		}
		golibriPct := 0.0
		if total > 0 {
			golibriPct = float64(stats.golibri) / float64(total) * 100
		}
		matchPct := 0.0
		if stats.golibri > 0 {
			matchPct = float64(stats.match) / float64(stats.golibri) * 100
		}

		fmt.Printf("%-15s  %5d (%.1f%%)        %5d (%.1f%%)       %.1f%%\n",
			field,
			stats.ebook, ebookPct,
			stats.golibri, golibriPct,
			matchPct,
		)
	}
}

func normalize(s string) string {
	// Simple normalization for comparison
	return strings.ToLower(strings.TrimSpace(s))
}
