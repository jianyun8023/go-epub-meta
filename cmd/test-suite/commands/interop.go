package commands

import (
	"encoding/json"
	"fmt"
	"golibri/epub"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

// =============================================================================
// Command Line Flags
// =============================================================================

var (
	interopFormat      string // oebps1, epub2, epub3-pure, epub3-hybrid, all
	interopMode        string // read_compare, golibri_write, ebook_meta_write, all
	interopField       string // all or specific field name
	interopOutput      string // output JSON file
	interopConcurrency int
	interopVerbose     bool
)

func init() {
	interopCmd.Flags().StringVarP(&interopFormat, "format", "f", "all", "EPUB format: oebps1, epub2, epub3-pure, epub3-hybrid, all")
	interopCmd.Flags().StringVarP(&interopMode, "mode", "m", "all", "Test mode: read_compare, golibri_write, ebook_meta_write, all")
	interopCmd.Flags().StringVar(&interopField, "field", "all", "Metadata field to test: title, author, language, publisher, date, series, series_index, identifiers, subjects, description, all")
	interopCmd.Flags().StringVarP(&interopOutput, "output", "o", "", "Output JSON file (optional)")
	interopCmd.Flags().IntVarP(&interopConcurrency, "concurrency", "c", 5, "Number of concurrent workers")
	interopCmd.Flags().BoolVarP(&interopVerbose, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(interopCmd)
}

var interopCmd = &cobra.Command{
	Use:   "interop [directory]",
	Short: "Run interoperability tests between golibri and ebook-meta",
	Long: `Run comprehensive interoperability tests comparing golibri and ebook-meta
for reading and writing EPUB metadata across different formats.

Test modes:
  read_compare      - Compare metadata read by both tools
  golibri_write     - golibri writes, ebook-meta reads to verify
  ebook_meta_write  - ebook-meta writes, golibri reads to verify
  all               - Run all test modes

EPUB formats (directories):
  oebps1            - OEBPS 1.x legacy format
  epub2             - EPUB 2.x format
  epub3-pure        - EPUB 3.x pure format (Nav only)
  epub3-hybrid      - EPUB 3.x hybrid format (Nav + NCX)
  all               - Test all formats`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]
		runInterop(dir)
	},
}

// =============================================================================
// Data Structures
// =============================================================================

// MetadataFields lists all testable metadata fields
var MetadataFields = []string{
	"title", "author", "language", "publisher", "date",
	"series", "series_index", "identifiers", "subjects", "description",
}

// EPUBFormats lists all supported EPUB format directories
var EPUBFormats = []string{"oebps1", "epub2", "epub3-pure", "epub3-hybrid"}

// InteropTestCase represents a single test case
type InteropTestCase struct {
	FilePath      string `json:"file_path"`
	EPUBFormat    string `json:"epub_format"`
	TestMode      string `json:"test_mode"`
	MetadataField string `json:"metadata_field"`
}

// FieldResult represents the result of testing a single field
type FieldResult struct {
	Field        string `json:"field"`
	GolibriValue string `json:"golibri_value"`
	EbookMeta    string `json:"ebook_meta_value"`
	TestValue    string `json:"test_value,omitempty"` // Value written during write tests
	Match        bool   `json:"match"`
	Error        string `json:"error,omitempty"`
}

// InteropResult represents the result of one test case
type InteropResult struct {
	TestCase     InteropTestCase        `json:"test_case"`
	FieldResults map[string]FieldResult `json:"field_results"`
	Passed       bool                   `json:"passed"`
	Duration     time.Duration          `json:"duration_ms"`
	Error        string                 `json:"error,omitempty"`
}

// InteropReport represents the complete test report
type InteropReport struct {
	Summary   ReportSummary   `json:"summary"`
	Results   []InteropResult `json:"results"`
	Failures  []FailureEntry  `json:"failures"`
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	TotalTime time.Duration   `json:"total_time_ms"`
}

// ReportSummary contains aggregated statistics
type ReportSummary struct {
	Total    int                      `json:"total"`
	Passed   int                      `json:"passed"`
	Failed   int                      `json:"failed"`
	ByFormat map[string]FormatSummary `json:"by_format"`
	ByMode   map[string]ModeSummary   `json:"by_mode"`
	ByField  map[string]FieldSummary  `json:"by_field"`
}

// FormatSummary contains per-format statistics
type FormatSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// ModeSummary contains per-mode statistics
type ModeSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// FieldSummary contains per-field statistics
type FieldSummary struct {
	Tested  int `json:"tested"`
	Matched int `json:"matched"`
}

// FailureEntry describes a single failure
type FailureEntry struct {
	File      string `json:"file"`
	Format    string `json:"format"`
	Mode      string `json:"mode"`
	Field     string `json:"field"`
	Golibri   string `json:"golibri"`
	EbookMeta string `json:"ebook_meta"`
	Reason    string `json:"reason"`
}

// =============================================================================
// Main Entry Point
// =============================================================================

func runInterop(dir string) {
	fmt.Println("=== Interoperability Test Suite ===")
	fmt.Printf("Directory: %s\n", dir)
	fmt.Printf("Format: %s, Mode: %s, Field: %s\n\n", interopFormat, interopMode, interopField)

	// Determine which formats to test
	formats := EPUBFormats
	if interopFormat != "all" {
		formats = []string{interopFormat}
	}

	// Determine which modes to test
	modes := []string{"read_compare", "golibri_write", "ebook_meta_write"}
	if interopMode != "all" {
		modes = []string{interopMode}
	}

	// Determine which fields to test
	fields := MetadataFields
	if interopField != "all" {
		fields = []string{interopField}
	}

	// Build test cases
	var testCases []InteropTestCase
	for _, format := range formats {
		formatDir := filepath.Join(dir, format)
		if _, err := os.Stat(formatDir); os.IsNotExist(err) {
			fmt.Printf("Warning: Format directory not found: %s\n", formatDir)
			continue
		}

		files, err := findEpubFiles(formatDir)
		if err != nil {
			fmt.Printf("Warning: Error scanning %s: %v\n", formatDir, err)
			continue
		}

		for _, file := range files {
			for _, mode := range modes {
				for _, field := range fields {
					testCases = append(testCases, InteropTestCase{
						FilePath:      file,
						EPUBFormat:    format,
						TestMode:      mode,
						MetadataField: field,
					})
				}
			}
		}
	}

	if len(testCases) == 0 {
		fmt.Println("No test cases to run. Check your directory structure.")
		fmt.Println("Expected: <dir>/epub2/, <dir>/epub3-pure/, etc.")
		return
	}

	fmt.Printf("Running %d test cases...\n\n", len(testCases))

	// Run tests
	report := runInteropTests(testCases)

	// Print summary
	printInteropSummary(report)

	// Write JSON output if requested
	if interopOutput != "" {
		if err := writeInteropJSON(report, interopOutput); err != nil {
			fmt.Printf("Error writing JSON: %v\n", err)
		} else {
			fmt.Printf("\nResults written to: %s\n", interopOutput)
		}
	}
}

// =============================================================================
// Test Execution
// =============================================================================

func runInteropTests(testCases []InteropTestCase) *InteropReport {
	report := &InteropReport{
		StartTime: time.Now(),
		Summary: ReportSummary{
			ByFormat: make(map[string]FormatSummary),
			ByMode:   make(map[string]ModeSummary),
			ByField:  make(map[string]FieldSummary),
		},
	}

	var (
		results []InteropResult
		mu      sync.Mutex
		counter int64
		total   = int64(len(testCases))
	)

	// Semaphore for concurrency control
	sem := make(chan struct{}, interopConcurrency)
	var wg sync.WaitGroup

	// Progress reporter
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				current := atomic.LoadInt64(&counter)
				pct := float64(current) / float64(total) * 100
				fmt.Printf("\rProgress: %d/%d (%.1f%%) ", current, total, pct)
			case <-done:
				return
			}
		}
	}()

	for _, tc := range testCases {
		wg.Add(1)
		go func(tc InteropTestCase) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result := runSingleInteropTest(tc)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()

			atomic.AddInt64(&counter, 1)
		}(tc)
	}

	wg.Wait()
	close(done)
	fmt.Printf("\rProgress: %d/%d (100.0%%) \n", total, total)

	report.Results = results
	report.EndTime = time.Now()
	report.TotalTime = report.EndTime.Sub(report.StartTime)

	// Calculate summary
	calculateSummary(report)

	return report
}

func runSingleInteropTest(tc InteropTestCase) InteropResult {
	start := time.Now()
	result := InteropResult{
		TestCase:     tc,
		FieldResults: make(map[string]FieldResult),
		Passed:       true,
	}

	// Copy file to temp directory to avoid modifying source
	tempFile, err := copyToTemp(tc.FilePath)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to copy file: %v", err)
		result.Passed = false
		result.Duration = time.Since(start)
		return result
	}
	defer os.Remove(tempFile)

	switch tc.TestMode {
	case "read_compare":
		result = runReadCompareTest(tc, tempFile)
	case "golibri_write":
		result = runGolibriWriteTest(tc, tempFile)
	case "ebook_meta_write":
		result = runEbookMetaWriteTest(tc, tempFile)
	default:
		result.Error = fmt.Sprintf("Unknown test mode: %s", tc.TestMode)
		result.Passed = false
	}

	result.Duration = time.Since(start)
	return result
}

// copyToTemp copies a file to a temporary location
func copyToTemp(src string) (string, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	tempFile, err := os.CreateTemp("", "interop-*.epub")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, srcFile); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// =============================================================================
// Test Mode Implementations
// =============================================================================

// runReadCompareTest compares metadata read by golibri and ebook-meta
func runReadCompareTest(tc InteropTestCase, tempFile string) InteropResult {
	result := InteropResult{
		TestCase:     tc,
		FieldResults: make(map[string]FieldResult),
		Passed:       true,
	}

	// Read with golibri
	golibriMeta, err := readWithGolibri(tempFile)
	if err != nil {
		result.Error = fmt.Sprintf("golibri read failed: %v", err)
		result.Passed = false
		return result
	}

	// Read with ebook-meta
	ebookMeta, err := readWithEbookMeta(tempFile)
	if err != nil {
		result.Error = fmt.Sprintf("ebook-meta read failed: %v", err)
		result.Passed = false
		return result
	}

	// Compare the specified field(s)
	fieldsToTest := MetadataFields
	if tc.MetadataField != "all" {
		fieldsToTest = []string{tc.MetadataField}
	}

	for _, field := range fieldsToTest {
		fr := FieldResult{
			Field:        field,
			GolibriValue: golibriMeta[field],
			EbookMeta:    ebookMeta[field],
		}
		fr.Match = compareFieldValues(field, fr.GolibriValue, fr.EbookMeta)
		if !fr.Match {
			result.Passed = false
		}
		result.FieldResults[field] = fr
	}

	return result
}

// runGolibriWriteTest writes with golibri, reads with ebook-meta
func runGolibriWriteTest(tc InteropTestCase, tempFile string) InteropResult {
	result := InteropResult{
		TestCase:     tc,
		FieldResults: make(map[string]FieldResult),
		Passed:       true,
	}

	// Generate test value
	testValues := generateTestValues(tc.MetadataField)

	// Write with golibri
	if err := writeWithGolibri(tempFile, testValues); err != nil {
		result.Error = fmt.Sprintf("golibri write failed: %v", err)
		result.Passed = false
		return result
	}

	// Read with ebook-meta to verify
	ebookMeta, err := readWithEbookMeta(tempFile)
	if err != nil {
		result.Error = fmt.Sprintf("ebook-meta read failed: %v", err)
		result.Passed = false
		return result
	}

	// Compare written values with what ebook-meta read
	fieldsToTest := MetadataFields
	if tc.MetadataField != "all" {
		fieldsToTest = []string{tc.MetadataField}
	}

	for _, field := range fieldsToTest {
		testVal := testValues[field]
		if testVal == "" {
			continue // Skip fields we didn't write
		}
		fr := FieldResult{
			Field:        field,
			TestValue:    testVal,
			EbookMeta:    ebookMeta[field],
			GolibriValue: testVal, // What we wrote
		}
		fr.Match = compareFieldValues(field, testVal, fr.EbookMeta)
		if !fr.Match {
			result.Passed = false
		}
		result.FieldResults[field] = fr
	}

	return result
}

// runEbookMetaWriteTest writes with ebook-meta, reads with golibri
func runEbookMetaWriteTest(tc InteropTestCase, tempFile string) InteropResult {
	result := InteropResult{
		TestCase:     tc,
		FieldResults: make(map[string]FieldResult),
		Passed:       true,
	}

	// Generate test value
	testValues := generateTestValues(tc.MetadataField)

	// Write with ebook-meta
	if err := writeWithEbookMeta(tempFile, testValues); err != nil {
		result.Error = fmt.Sprintf("ebook-meta write failed: %v", err)
		result.Passed = false
		return result
	}

	// Read with golibri to verify
	golibriMeta, err := readWithGolibri(tempFile)
	if err != nil {
		result.Error = fmt.Sprintf("golibri read failed: %v", err)
		result.Passed = false
		return result
	}

	// Compare written values with what golibri read
	fieldsToTest := MetadataFields
	if tc.MetadataField != "all" {
		fieldsToTest = []string{tc.MetadataField}
	}

	for _, field := range fieldsToTest {
		testVal := testValues[field]
		if testVal == "" {
			continue
		}
		fr := FieldResult{
			Field:        field,
			TestValue:    testVal,
			GolibriValue: golibriMeta[field],
			EbookMeta:    testVal, // What we wrote
		}
		fr.Match = compareFieldValues(field, fr.GolibriValue, testVal)
		if !fr.Match {
			result.Passed = false
		}
		result.FieldResults[field] = fr
	}

	return result
}

// =============================================================================
// Read Functions
// =============================================================================

// readWithGolibri reads metadata using golibri library
func readWithGolibri(filePath string) (map[string]string, error) {
	ep, err := epub.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer ep.Close()

	meta := make(map[string]string)
	meta["title"] = ep.Package.GetTitle()
	meta["author"] = ep.Package.GetAuthor()
	meta["language"] = ep.Package.GetLanguage()
	meta["publisher"] = ep.Package.GetPublisher()
	meta["date"] = ep.Package.GetPublishDate()
	meta["series"] = ep.Package.GetSeries()
	meta["series_index"] = ep.Package.GetSeriesIndex()
	meta["description"] = ep.Package.GetDescription()

	// Format identifiers (skip calibre UUID as it's internal)
	ids := ep.Package.GetIdentifiers()
	var idParts []string
	for scheme, value := range ids {
		// Skip calibre UUID as ebook-meta doesn't output it
		if strings.ToLower(scheme) == "calibre" {
			continue
		}
		idParts = append(idParts, fmt.Sprintf("%s:%s", scheme, value))
	}
	meta["identifiers"] = strings.Join(idParts, ", ")

	// Format subjects
	subjects := ep.Package.GetSubjects()
	meta["subjects"] = strings.Join(subjects, ", ")

	return meta, nil
}

// readWithEbookMeta reads metadata using ebook-meta command
func readWithEbookMeta(filePath string) (map[string]string, error) {
	output, err := exec.Command("ebook-meta", filePath).Output()
	if err != nil {
		return nil, err
	}

	return parseEbookMetaOutput(string(output)), nil
}

// parseEbookMetaOutput parses ebook-meta text output into a map
func parseEbookMetaOutput(output string) map[string]string {
	meta := make(map[string]string)
	lines := strings.Split(output, "\n")

	fieldMap := map[string]string{
		"Title":       "title",
		"Author(s)":   "author",
		"Languages":   "language",
		"Language":    "language",
		"Publisher":   "publisher",
		"Published":   "date",
		"Series":      "series",
		"Identifiers": "identifiers",
		"Identifier":  "identifiers",
		"Tags":        "subjects",
		"Comments":    "description",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if fieldName, ok := fieldMap[key]; ok {
			// Remove [sort] suffix for title and author
			if fieldName == "title" || fieldName == "author" {
				value = stripSortSuffix(value)
			}
			meta[fieldName] = value
		}

		// Extract series index from "Series" field (format: "Name #Index")
		if key == "Series" && strings.Contains(value, "#") {
			parts := strings.SplitN(value, "#", 2)
			meta["series"] = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				meta["series_index"] = strings.TrimSpace(parts[1])
			}
		}
	}

	return meta
}

// stripSortSuffix removes [sort] suffix from a value
// e.g., "余秋雨 [余秋雨]" -> "余秋雨"
// e.g., "The Great Gatsby [Great Gatsby, The]" -> "The Great Gatsby"
func stripSortSuffix(value string) string {
	if idx := strings.LastIndex(value, " ["); idx != -1 {
		if strings.HasSuffix(value, "]") {
			return strings.TrimSpace(value[:idx])
		}
	}
	return value
}

// =============================================================================
// Write Functions
// =============================================================================

// writeWithGolibri writes metadata using golibri library
func writeWithGolibri(filePath string, values map[string]string) error {
	ep, err := epub.Open(filePath)
	if err != nil {
		return err
	}

	if v, ok := values["title"]; ok && v != "" {
		ep.Package.SetTitle(v)
	}
	if v, ok := values["author"]; ok && v != "" {
		ep.Package.SetAuthor(v)
	}
	if v, ok := values["language"]; ok && v != "" {
		ep.Package.SetLanguage(v)
	}
	if v, ok := values["publisher"]; ok && v != "" {
		ep.Package.SetPublisher(v)
	}
	if v, ok := values["date"]; ok && v != "" {
		ep.Package.SetPublishDate(v)
	}
	if v, ok := values["series"]; ok && v != "" {
		ep.Package.SetSeries(v)
	}
	if v, ok := values["series_index"]; ok && v != "" {
		ep.Package.SetSeriesIndex(v)
	}
	if v, ok := values["description"]; ok && v != "" {
		ep.Package.SetDescription(v)
	}
	if v, ok := values["subjects"]; ok && v != "" {
		subjects := strings.Split(v, ", ")
		ep.Package.SetSubjects(subjects)
	}
	// Note: identifiers have special handling - using ISBN for now
	if v, ok := values["identifiers"]; ok && v != "" && strings.HasPrefix(v, "isbn:") {
		isbn := strings.TrimPrefix(v, "isbn:")
		ep.Package.SetISBN(isbn)
	}

	err = ep.Save(filePath)
	ep.Close()
	return err
}

// writeWithEbookMeta writes metadata using ebook-meta command
func writeWithEbookMeta(filePath string, values map[string]string) error {
	args := []string{filePath}

	if v, ok := values["title"]; ok && v != "" {
		args = append(args, "--title", v)
	}
	if v, ok := values["author"]; ok && v != "" {
		args = append(args, "--authors", v)
	}
	if v, ok := values["language"]; ok && v != "" {
		args = append(args, "--language", v)
	}
	if v, ok := values["publisher"]; ok && v != "" {
		args = append(args, "--publisher", v)
	}
	if v, ok := values["date"]; ok && v != "" {
		args = append(args, "--date", v)
	}
	if v, ok := values["series"]; ok && v != "" {
		args = append(args, "--series", v)
	}
	if v, ok := values["series_index"]; ok && v != "" {
		args = append(args, "--index", v)
	}
	if v, ok := values["description"]; ok && v != "" {
		args = append(args, "--comments", v)
	}
	if v, ok := values["subjects"]; ok && v != "" {
		args = append(args, "--tags", v)
	}
	if v, ok := values["identifiers"]; ok && v != "" && strings.HasPrefix(v, "isbn:") {
		isbn := strings.TrimPrefix(v, "isbn:")
		args = append(args, "--isbn", isbn)
	}

	cmd := exec.Command("ebook-meta", args...)
	return cmd.Run()
}

// =============================================================================
// Utility Functions
// =============================================================================

// generateTestValues generates test values for writing
func generateTestValues(field string) map[string]string {
	timestamp := time.Now().Format("20060102150405")
	testValues := map[string]string{
		"title":        fmt.Sprintf("Test Title %s", timestamp),
		"author":       fmt.Sprintf("Test Author %s", timestamp),
		"language":     "en",
		"publisher":    fmt.Sprintf("Test Publisher %s", timestamp),
		"date":         "2025-01-09",
		"series":       fmt.Sprintf("Test Series %s", timestamp),
		"series_index": "1",
		"description":  fmt.Sprintf("Test Description %s", timestamp),
		"subjects":     "test, interop, golibri",
		"identifiers":  "isbn:9781234567890",
	}

	if field == "all" {
		return testValues
	}

	// Return only the specified field
	if val, ok := testValues[field]; ok {
		return map[string]string{field: val}
	}
	return map[string]string{}
}

// compareFieldValues compares two field values with normalization
func compareFieldValues(field, val1, val2 string) bool {
	// Normalize both values
	v1 := normalizeForComparison(field, val1)
	v2 := normalizeForComparison(field, val2)
	return v1 == v2
}

// normalizeForComparison normalizes a value for comparison
func normalizeForComparison(field, value string) string {
	value = strings.TrimSpace(value)

	switch field {
	case "title", "author":
		// Remove [sort] suffix first, then lowercase
		value = stripSortSuffix(value)
		return strings.ToLower(value)
	case "language":
		return normalizeLanguage(strings.ToLower(value))
	case "date":
		return normalizeDate(value)
	case "identifiers":
		return normalizeIdentifiers(value)
	case "series_index":
		// Normalize "1.0" to "1"
		return strings.TrimSuffix(strings.ToLower(value), ".0")
	}
	return strings.ToLower(value)
}

// =============================================================================
// Summary and Reporting
// =============================================================================

func calculateSummary(report *InteropReport) {
	for _, r := range report.Results {
		report.Summary.Total++
		if r.Passed {
			report.Summary.Passed++
		} else {
			report.Summary.Failed++
		}

		// By format
		fs := report.Summary.ByFormat[r.TestCase.EPUBFormat]
		if r.Passed {
			fs.Passed++
		} else {
			fs.Failed++
		}
		report.Summary.ByFormat[r.TestCase.EPUBFormat] = fs

		// By mode
		ms := report.Summary.ByMode[r.TestCase.TestMode]
		if r.Passed {
			ms.Passed++
		} else {
			ms.Failed++
		}
		report.Summary.ByMode[r.TestCase.TestMode] = ms

		// By field
		for field, fr := range r.FieldResults {
			fs := report.Summary.ByField[field]
			fs.Tested++
			if fr.Match {
				fs.Matched++
			}
			report.Summary.ByField[field] = fs

			// Add to failures
			if !fr.Match {
				report.Failures = append(report.Failures, FailureEntry{
					File:      filepath.Base(r.TestCase.FilePath),
					Format:    r.TestCase.EPUBFormat,
					Mode:      r.TestCase.TestMode,
					Field:     field,
					Golibri:   fr.GolibriValue,
					EbookMeta: fr.EbookMeta,
					Reason:    fr.Error,
				})
			}
		}
	}
}

func printInteropSummary(report *InteropReport) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("INTEROPERABILITY TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("\nTotal: %d | Passed: %d | Failed: %d\n",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)

	fmt.Printf("Duration: %v\n", report.TotalTime.Round(time.Millisecond))

	// By Format
	fmt.Println("\n--- By Format ---")
	for format, stats := range report.Summary.ByFormat {
		total := stats.Passed + stats.Failed
		pct := float64(stats.Passed) / float64(total) * 100
		fmt.Printf("  %-15s: %d/%d (%.1f%%)\n", format, stats.Passed, total, pct)
	}

	// By Mode
	fmt.Println("\n--- By Mode ---")
	for mode, stats := range report.Summary.ByMode {
		total := stats.Passed + stats.Failed
		pct := float64(stats.Passed) / float64(total) * 100
		fmt.Printf("  %-20s: %d/%d (%.1f%%)\n", mode, stats.Passed, total, pct)
	}

	// By Field
	fmt.Println("\n--- By Field ---")
	for field, stats := range report.Summary.ByField {
		pct := 0.0
		if stats.Tested > 0 {
			pct = float64(stats.Matched) / float64(stats.Tested) * 100
		}
		fmt.Printf("  %-15s: %d/%d matched (%.1f%%)\n", field, stats.Matched, stats.Tested, pct)
	}

	// Show first few failures
	if len(report.Failures) > 0 {
		fmt.Println("\n--- Sample Failures ---")
		limit := 10
		if len(report.Failures) < limit {
			limit = len(report.Failures)
		}
		for i := 0; i < limit; i++ {
			f := report.Failures[i]
			fmt.Printf("  [%s/%s] %s.%s: golibri=%q vs ebook-meta=%q\n",
				f.Format, f.Mode, f.File, f.Field, f.Golibri, f.EbookMeta)
		}
		if len(report.Failures) > limit {
			fmt.Printf("  ... and %d more failures\n", len(report.Failures)-limit)
		}
	}
}

func writeInteropJSON(report *InteropReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
