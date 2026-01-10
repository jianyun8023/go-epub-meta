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
	case "cover":
		return testCover(path)
	case "all":
		// Run all tests (read + write + cover coverage)
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
		if err := testCover(path); err != nil {
			return fmt.Errorf("cover test: %w", err)
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

// testCover tests cover extraction and writing
func testCover(path string) error {
	// First, check if the EPUB has a cover
	ep, err := epub.Open(path)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}

	_, _, coverErr := ep.GetCoverImage()
	ep.Close()

	// Test 1: If EPUB has cover, test extraction via CLI
	if coverErr == nil {
		// Create temp file for extracted cover
		coverTmp, err := os.CreateTemp("", "cover-extract-*.jpg")
		if err != nil {
			return err
		}
		coverPath := coverTmp.Name()
		coverTmp.Close()
		defer os.Remove(coverPath)

		// Extract cover using CLI
		cmd := exec.Command(funcGolibriPath, "meta", path, "--get-cover", coverPath)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cover extraction failed: %w, stderr: %s", err, stderr.String())
		}

		// Verify extracted file exists and has content
		info, err := os.Stat(coverPath)
		if err != nil {
			return fmt.Errorf("extracted cover not found: %w", err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("extracted cover is empty")
		}
	}

	// Test 2: Test cover writing (write → extract → verify)
	// Create temp cover image (minimal PNG)
	coverData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
		0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59,
		0xE7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	coverFile, err := os.CreateTemp("", "test-cover-*.png")
	if err != nil {
		return err
	}
	coverFile.Write(coverData)
	coverInputPath := coverFile.Name()
	coverFile.Close()
	defer os.Remove(coverInputPath)

	// Create output EPUB with new cover
	outputTmp, err := os.CreateTemp("", "golibri-cover-write-*.epub")
	if err != nil {
		return err
	}
	outputPath := outputTmp.Name()
	outputTmp.Close()
	defer os.Remove(outputPath)

	cmd := exec.Command(funcGolibriPath, "meta", path, "-c", coverInputPath, "-o", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cover write failed: %w, stderr: %s", err, stderr.String())
	}

	// Verify the cover was written
	ep2, err := epub.Open(outputPath)
	if err != nil {
		return fmt.Errorf("failed to open modified EPUB: %w", err)
	}
	defer ep2.Close()

	_, mime, err := ep2.GetCoverImage()
	if err != nil {
		return fmt.Errorf("cover not found after write: %w", err)
	}
	if mime != "image/png" {
		return fmt.Errorf("unexpected cover mime type: %s", mime)
	}

	return nil
}
