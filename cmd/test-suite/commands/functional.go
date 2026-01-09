package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"golibri/epub"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/spf13/cobra"
)

var (
	funcConcurrency   int
	funcMode          string
	funcGolibriPath   string
	funcEbookMetaPath string
)

func init() {
	functionalCmd.Flags().IntVarP(&funcConcurrency, "concurrency", "c", 10, "Number of concurrent workers")
	functionalCmd.Flags().StringVarP(&funcMode, "mode", "m", "all", "Test mode: read, json, roundtrip, write, interop, all")
	functionalCmd.Flags().StringVar(&funcGolibriPath, "golibri", "./golibri", "Path to golibri binary")
	functionalCmd.Flags().StringVar(&funcEbookMetaPath, "ebook-meta", "ebook-meta", "Path to ebook-meta binary (Calibre)")

	rootCmd.AddCommand(functionalCmd)
}

var functionalCmd = &cobra.Command{
	Use:   "functional [directory]",
	Short: "Run functional tests on EPUB files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]
		runFunctional(dir)
	},
}

func runFunctional(dir string) {
	files, err := findEpubFiles(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d EPUB files. Mode: %s\n", len(files), funcMode)

	// Check if golibri binary exists
	if _, err := os.Stat(funcGolibriPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: golibri binary not found at %s\n", funcGolibriPath)
		fmt.Fprintf(os.Stderr, "Please build it first: go build -o golibri ./cmd/golibri/\n")
		os.Exit(1)
	}

	// For interop mode we also need ebook-meta
	if funcMode == "interop" {
		if _, err := exec.LookPath(funcEbookMetaPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: ebook-meta not found (%s). Please install Calibre or set --ebook-meta.\n", funcEbookMetaPath)
			os.Exit(1)
		}
	}

	var (
		passed  int64
		failed  int64
		corrupt int64
		wg      sync.WaitGroup
		sem     = make(chan struct{}, funcConcurrency)
	)

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := testFile(f, funcMode); err != nil {
				if isCorruptZipError(err) {
					fmt.Printf("[CORRUPT] %s: %v\n", filepath.Base(f), err)
					atomic.AddInt64(&corrupt, 1)
					return
				}
				fmt.Printf("[FAIL] %s: %v\n", filepath.Base(f), err)
				atomic.AddInt64(&failed, 1)
			} else {
				atomic.AddInt64(&passed, 1)
			}
		}(file)
	}

	wg.Wait()
	fmt.Printf("\nResult: %d passed, %d failed, %d corrupt\n", passed, failed, corrupt)
}

func isCorruptZipError(err error) bool {
	if err == nil {
		return false
	}
	// Go's archive/zip uses this exact error string in several failure paths.
	// Treat these as "dataset corrupt" rather than product regression failures.
	return strings.Contains(err.Error(), "zip: not a valid zip file")
}

func testFile(path string, mode string) error {
	switch mode {
	case "read":
		return testRead(path)
	case "json":
		return testJSON(path)
	case "roundtrip":
		return testRoundtrip(path)
	case "write":
		return testWrite(path)
	case "interop":
		return testWriteEbookMetaRead(path)
	case "all":
		// Run all tests (read + write coverage)
		if err := testRead(path); err != nil {
			return fmt.Errorf("read test: %w", err)
		}
		if err := testJSON(path); err != nil {
			return fmt.Errorf("json test: %w", err)
		}
		if err := testRoundtrip(path); err != nil {
			return fmt.Errorf("roundtrip test: %w", err)
		}
		if err := testWrite(path); err != nil {
			return fmt.Errorf("write test: %w", err)
		}
		// Interop (optional) - only if ebook-meta exists
		if _, err := exec.LookPath(funcEbookMetaPath); err == nil {
			if err := testWriteEbookMetaRead(path); err != nil {
				return fmt.Errorf("interop test: %w", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

// testRead tests basic EPUB reading
func testRead(path string) error {
	ep, err := epub.Open(path)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}
	defer ep.Close()

	// Verify we can read basic metadata
	title := ep.Package.GetTitle()
	if title == "" {
		return fmt.Errorf("title is empty")
	}

	return nil
}

// testJSON tests JSON output format
func testJSON(path string) error {
	// Execute golibri with --json flag
	cmd := exec.Command(funcGolibriPath, "meta", "--json", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("golibri command failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse JSON output
	var metadata map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &metadata); err != nil {
		return fmt.Errorf("invalid JSON output: %w, output: %s", err, stdout.String())
	}

	// Verify required fields exist
	requiredFields := []string{"title", "authors", "language", "identifiers", "cover"}
	for _, field := range requiredFields {
		if _, ok := metadata[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Verify field types
	if _, ok := metadata["authors"].([]interface{}); !ok {
		return fmt.Errorf("authors field should be an array")
	}
	if _, ok := metadata["identifiers"].(map[string]interface{}); !ok {
		return fmt.Errorf("identifiers field should be an object")
	}
	if _, ok := metadata["cover"].(bool); !ok {
		return fmt.Errorf("cover field should be a boolean")
	}

	return nil
}

// testRoundtrip tests save and re-read functionality
func testRoundtrip(path string) error {
	ep, err := epub.Open(path)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}
	defer ep.Close()

	// Get original title
	originalTitle := ep.Package.GetTitle()

	// Save to temp file
	tmp, err := os.CreateTemp("", "golibri-test-*.epub")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	if err := ep.Save(tmpPath); err != nil {
		return fmt.Errorf("save failed: %w", err)
	}

	// Verify re-open and data integrity
	ep2, err := epub.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("re-open failed: %w", err)
	}
	defer ep2.Close()

	newTitle := ep2.Package.GetTitle()
	if newTitle != originalTitle {
		return fmt.Errorf("title mismatch: original=%s, after-roundtrip=%s", originalTitle, newTitle)
	}

	return nil
}

// testWrite tests metadata modification
func testWrite(path string) error {
	// Create a temp output file
	tmp, err := os.CreateTemp("", "golibri-write-*.epub")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	// Use golibri CLI to modify metadata
	cmd := exec.Command(funcGolibriPath, "meta", path,
		"-t", "Test Title Modified",
		"-a", "Test Author",
		"-o", tmpPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write command failed: %w, stderr: %s", err, stderr.String())
	}

	// Verify the modification
	ep, err := epub.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open modified file: %w", err)
	}
	defer ep.Close()

	if title := ep.Package.GetTitle(); title != "Test Title Modified" {
		return fmt.Errorf("title not modified correctly: got %s", title)
	}
	if author := ep.Package.GetAuthor(); !strings.Contains(author, "Test Author") {
		return fmt.Errorf("author not modified correctly: got %s", author)
	}

	return nil
}

// testWriteEbookMetaRead verifies interoperability: golibri writes -> ebook-meta reads.
// ebook-meta typically outputs text (no stable JSON output across versions), so we parse text output.
func testWriteEbookMetaRead(path string) error {
	// Create a temp output file
	tmp, err := os.CreateTemp("", "golibri-interop-*.epub")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	// Use golibri CLI to modify metadata (write)
	// Use unique marker values to avoid false positives from existing metadata.
	const (
		interopTitle     = "INTEROP_TITLE_MODIFIED"
		interopAuthor    = "INTEROP_AUTHOR"
		interopPublisher = "INTEROP_PUBLISHER"
		interopDate      = "2025-01-07"
		interopLanguage  = "eo" // Esperanto -> ebook-meta often displays as "epo"
		interopTags      = "interop_tag_a,interop_tag_b"
		interopComments  = "INTEROP_COMMENT_MARKER"
		interopSeries    = "INTEROP_SERIES"
		interopSeriesIdx = "7"
		interopRating    = "4"
		// Use a known-valid ISBN-13 (Calibre may ignore invalid ISBNs).
		interopISBN     = "9780306406157"
		interopASIN     = "B08XYZABC1"
		interopCustomID = "interop:12345"
	)

	cmd := exec.Command(funcGolibriPath, "meta", path,
		"-t", interopTitle,
		"-a", interopAuthor,
		"-s", interopSeries,
		"--series-index", interopSeriesIdx,
		"--publisher", interopPublisher,
		"--date", interopDate,
		"--language", interopLanguage,
		"--tags", interopTags,
		"--comments", interopComments,
		"--rating", interopRating,
		"--isbn", interopISBN,
		"--asin", interopASIN,
		"-i", interopCustomID,
		"-o", tmpPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Treat corrupt zips as dataset issues (not product regression) for interop mode.
		// This commonly happens when DataOffset cannot be read for some entries.
		combined := stdout.String() + "\n" + stderr.String()
		if strings.Contains(combined, "zip: not a valid zip file") || strings.Contains(err.Error(), "zip: not a valid zip file") {
			// Important: keep the exact substring so isCorruptZipError can categorize it as CORRUPT.
			return fmt.Errorf("zip: not a valid zip file")
		}
		return fmt.Errorf("write command failed: %w, output: %s", err, strings.TrimSpace(combined))
	}

	// Use ebook-meta to read (text) and verify the write is visible cross-tool
	ebookCmd := exec.Command(funcEbookMetaPath, tmpPath)
	var out, ebookErr bytes.Buffer
	ebookCmd.Stdout = &out
	ebookCmd.Stderr = &ebookErr
	if err := ebookCmd.Run(); err != nil {
		return fmt.Errorf("ebook-meta failed: %w, stderr: %s", err, ebookErr.String())
	}

	fields := parseEbookMetaText(out.String())

	// Title
	if fields["title"] != interopTitle {
		return fmt.Errorf("ebook-meta title mismatch: got=%q", fields["title"])
	}

	// Author(s)
	if !strings.Contains(fields["authors"], interopAuthor) {
		return fmt.Errorf("ebook-meta authors mismatch: got=%q", fields["authors"])
	}

	// Publisher
	if fields["publisher"] != interopPublisher {
		return fmt.Errorf("ebook-meta publisher mismatch: got=%q", fields["publisher"])
	}

	// Published date (ebook-meta formatting can vary; ensure marker date is present)
	if fields["published"] == "" || !strings.Contains(fields["published"], "2025") {
		return fmt.Errorf("ebook-meta published mismatch: got=%q", fields["published"])
	}

	// Language: ebook-meta may convert BCP47 (eo) to ISO639-2 (epo)
	if fields["language"] == "" || !(strings.Contains(fields["language"], "epo") || strings.Contains(fields["language"], "eo")) {
		return fmt.Errorf("ebook-meta language mismatch: got=%q", fields["language"])
	}

	// Tags
	if fields["tags"] == "" || !(strings.Contains(fields["tags"], "interop_tag_a") && strings.Contains(fields["tags"], "interop_tag_b")) {
		return fmt.Errorf("ebook-meta tags mismatch: got=%q", fields["tags"])
	}

	// Comments (may include extra formatting; ensure marker is present)
	if fields["comments"] == "" || !strings.Contains(fields["comments"], interopComments) {
		return fmt.Errorf("ebook-meta comments mismatch: got=%q", fields["comments"])
	}

	// Series (ebook-meta usually prints combined series string; ensure both series name and index exist)
	if fields["series"] == "" || !strings.Contains(fields["series"], interopSeries) || !strings.Contains(fields["series"], interopSeriesIdx) {
		return fmt.Errorf("ebook-meta series mismatch: got=%q", fields["series"])
	}

	// NOTE: Rating verification skipped.
	// calibre:rating is written to the OPF by golibri, but ebook-meta does NOT read it back.
	// Rating in Calibre is database-level metadata, not file-level metadata that ebook-meta parses.

	// Identifiers (scheme names can vary; at least ensure ISBN and custom scheme appear)
	identNorm := strings.ReplaceAll(fields["identifiers"], "-", "")
	identNorm = strings.ReplaceAll(identNorm, " ", "")
	wantISBN := strings.ReplaceAll(interopISBN, "-", "")
	wantISBN = strings.ReplaceAll(wantISBN, " ", "")
	if fields["identifiers"] == "" || !strings.Contains(strings.ToLower(fields["identifiers"]), "isbn") || !strings.Contains(identNorm, wantISBN) {
		return fmt.Errorf("ebook-meta identifiers missing ISBN: got=%q", fields["identifiers"])
	}
	if !strings.Contains(fields["identifiers"], "interop") || !strings.Contains(fields["identifiers"], "12345") {
		return fmt.Errorf("ebook-meta identifiers missing custom scheme: got=%q", fields["identifiers"])
	}

	return nil
}

func parseEbookMetaText(s string) map[string]string {
	// ebook-meta output is typically like:
	// Title               : Foo
	// Author(s)           : Bar
	// Publisher           : ...
	out := make(map[string]string)
	sc := bufio.NewScanner(strings.NewReader(s))
	var currentKey string
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			// Multi-line continuation (commonly Comments). Only treat indented lines as continuation.
			if currentKey != "" && (strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t")) {
				out[currentKey] = strings.TrimSpace(out[currentKey] + "\n" + line)
			}
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		normalizedKey := strings.ToLower(strings.ReplaceAll(key, " ", ""))
		switch normalizedKey {
		case "title":
			out["title"] = val
			currentKey = "title"
		case "author(s)", "authors", "author":
			out["authors"] = val
			currentKey = "authors"
		case "publisher":
			out["publisher"] = val
			currentKey = "publisher"
		case "published", "publicationdate", "pubdate", "date":
			out["published"] = val
			currentKey = "published"
		case "language", "languages":
			out["language"] = val
			currentKey = "language"
		case "tags", "subjects":
			out["tags"] = val
			currentKey = "tags"
		case "comments", "comment", "description":
			out["comments"] = val
			currentKey = "comments"
		case "series":
			out["series"] = val
			currentKey = "series"
		case "rating":
			out["rating"] = val
			currentKey = "rating"
		case "identifiers", "identifier":
			out["identifiers"] = val
			currentKey = "identifiers"
		default:
			currentKey = ""
		}
	}
	return out
}
