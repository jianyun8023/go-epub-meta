package main

import (
	"flag"
	"fmt"
	"golibri/epub"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "meta":
		handleMeta(args)
	case "audit":
		handleAudit(args)
	case "stress-test":
		handleStressTest(args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: golibri <command> [args]")
	fmt.Println("Commands:")
	fmt.Println("  meta         Read or modify metadata")
	fmt.Println("  audit        Scan EPUBs for errors (read-only)")
	fmt.Println("  stress-test  Run stress test harness")
}

func handleMeta(args []string) {
	cmd := flag.NewFlagSet("meta", flag.ExitOnError)
	title := cmd.String("t", "", "Set title")
	author := cmd.String("a", "", "Set author")
	series := cmd.String("s", "", "Set series")
	cover := cmd.String("c", "", "Set cover image path")
	output := cmd.String("o", "", "Output file path (required for modification)")

	cmd.Parse(args)

	if cmd.NArg() == 0 {
		fmt.Println("Please provide an input EPUB file.")
		os.Exit(1)
	}

	inputFile := cmd.Arg(0)
	ep, err := epub.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening %s: %v\n", inputFile, err)
		os.Exit(1)
	}
	defer ep.Close()

	// Read Mode
	if *title == "" && *author == "" && *series == "" && *cover == "" {
		fmt.Println("--- Metadata ---")
		fmt.Printf("Title:    %s\n", ep.Package.GetTitle())
		fmt.Printf("Author:   %s\n", ep.Package.GetAuthor())
		fmt.Printf("Series:   %s\n", ep.Package.GetSeries())
		fmt.Printf("Language: %s\n", ep.Package.GetLanguage())
		_, _, err := ep.GetCoverImage()
		if err == nil {
			fmt.Println("Cover:    Found")
		} else {
			fmt.Println("Cover:    Not Found")
		}
		return
	}

	// Write Mode
	if *output == "" {
		fmt.Println("Error: -o (output) is required when modifying metadata.")
		os.Exit(1)
	}

	if *title != "" {
		ep.Package.SetTitle(*title)
	}
	if *author != "" {
		ep.Package.SetAuthor(*author)
	}
	if *series != "" {
		ep.Package.SetSeries(*series)
	}
	if *cover != "" {
		f, err := os.Open(*cover)
		if err != nil {
			fmt.Printf("Error opening cover %s: %v\n", *cover, err)
			os.Exit(1)
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			fmt.Printf("Error reading cover: %v\n", err)
			os.Exit(1)
		}

		// Detect mime
		mime := "image/jpeg"
		if strings.HasSuffix(strings.ToLower(*cover), ".png") {
			mime = "image/png"
		}

		ep.SetCover(data, mime)
	}

	if err := ep.Save(*output); err != nil {
		fmt.Printf("Error saving EPUB: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Saved to %s\n", *output)
}

func handleAudit(args []string) {
	// args[0] should be directory or file
	if len(args) == 0 {
		fmt.Println("Usage: golibri audit <path>")
		return
	}
	path := args[0]

	files := []string{}
	// Recursive walk
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".epub") {
			files = append(files, p)
		}
		return nil
	})

	fmt.Printf("Scanning %d files...\n", len(files))

	success := 0
	failed := 0

	for _, f := range files {
		ep, err := epub.Open(f)
		if err != nil {
			fmt.Printf("[FAIL] %s: %v\n", f, err)
			failed++
		} else {
			// Basic checks
			if ep.Package.GetTitle() == "" {
				fmt.Printf("[WARN] %s: No Title\n", f)
			}
			ep.Close()
			success++
		}
	}

	fmt.Printf("Scan complete. Success: %d, Failed: %d\n", success, failed)
}

func handleStressTest(args []string) {
	// Concurrency test harness
	if len(args) == 0 {
		fmt.Println("Usage: golibri stress-test <path>")
		return
	}
	root := args[0]

	var files []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".epub") {
			files = append(files, p)
		}
		return nil
	})

	workers := 16
	jobs := make(chan string, len(files))
	results := make(chan error, len(files))

	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				// Round trip test
				err := runRoundTrip(f)
				results <- err
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	wg.Wait()
	close(results)

	passed := 0
	failed := 0
	for err := range results {
		if err == nil {
			passed++
		} else {
			failed++
			// fmt.Println(err) // Too noisy for 130k
		}
	}

	duration := time.Since(start)
	fmt.Printf("Processed %d files in %s. Passed: %d, Failed: %d\n", len(files), duration, passed, failed)
}

func runRoundTrip(path string) error {
	// 1. Open
	ep, err := epub.Open(path)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}
	defer ep.Close()

	// 2. Modify (Mock)
	ep.Package.SetTitle(ep.Package.GetTitle() + " [MOD]")

	// 3. Save to memory/temp
	// Use os.CreateTemp to avoid polluting source directory
	f, err := os.CreateTemp("", "golibri_stress_*.epub")
	if err != nil {
		return fmt.Errorf("create temp failed: %w", err)
	}
	tmp := f.Name()
	f.Close()

	if err := ep.Save(tmp); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("save failed: %w", err)
	}
	defer os.Remove(tmp)

	// 4. Re-open and Verify
	ep2, err := epub.Open(tmp)
	if err != nil {
		return fmt.Errorf("re-open failed: %w", err)
	}
	defer ep2.Close()

	if !strings.Contains(ep2.Package.GetTitle(), "[MOD]") {
		return fmt.Errorf("verification failed: title mismatch")
	}

	return nil
}
