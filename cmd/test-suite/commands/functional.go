package commands

import (
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
	funcConcurrency int
	funcMode        string
	funcGolibriPath string
)

func init() {
	functionalCmd.Flags().IntVarP(&funcConcurrency, "concurrency", "c", 10, "Number of concurrent workers")
	functionalCmd.Flags().StringVarP(&funcMode, "mode", "m", "all", "Test mode: read, json, roundtrip, write, all")
	functionalCmd.Flags().StringVar(&funcGolibriPath, "golibri", "./golibri", "Path to golibri binary")

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
	files, err := FindEpubFiles(dir)
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
