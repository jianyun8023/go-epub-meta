package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Test Value Generation
// =============================================================================

func TestGenerateTestValues_All(t *testing.T) {
	values := generateTestValues("all")

	expectedFields := []string{
		"title", "author", "language", "publisher", "date",
		"series", "series_index", "description", "subjects", "identifiers",
	}

	for _, field := range expectedFields {
		if _, ok := values[field]; !ok {
			t.Errorf("Expected field %q in test values", field)
		}
	}
}

func TestGenerateTestValues_SingleField(t *testing.T) {
	values := generateTestValues("title")

	if len(values) != 1 {
		t.Errorf("Expected 1 field, got %d", len(values))
	}

	if _, ok := values["title"]; !ok {
		t.Error("Expected 'title' field in test values")
	}
}

func TestGenerateTestValues_InvalidField(t *testing.T) {
	values := generateTestValues("invalid_field")

	if len(values) != 0 {
		t.Errorf("Expected 0 fields for invalid field, got %d", len(values))
	}
}

// =============================================================================
// Field Comparison Tests
// =============================================================================

func TestCompareFieldValues_Exact(t *testing.T) {
	tests := []struct {
		field string
		val1  string
		val2  string
		match bool
	}{
		{"title", "Test Book", "Test Book", true},
		{"title", "Test Book", "test book", true}, // Case insensitive
		{"title", "Test Book", "Other Book", false},
		{"author", "John Doe", "john doe", true},
		{"author", "John Doe", "Jane Doe", false},
	}

	for _, tc := range tests {
		result := compareFieldValues(tc.field, tc.val1, tc.val2)
		if result != tc.match {
			t.Errorf("compareFieldValues(%q, %q, %q) = %v, want %v",
				tc.field, tc.val1, tc.val2, result, tc.match)
		}
	}
}

func TestStripSortSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"余秋雨 [余秋雨]", "余秋雨"},
		{"John Doe [Doe, John]", "John Doe"},
		{"The Great Gatsby [Great Gatsby, The]", "The Great Gatsby"},
		{"涂睿明 & 山本耀司 [涂睿明 & 山本耀司]", "涂睿明 & 山本耀司"},
		{"Simple Name", "Simple Name"},
		{"", ""},
	}

	for _, tc := range tests {
		result := StripSortSuffix(tc.input)
		if result != tc.expected {
			t.Errorf("StripSortSuffix(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestCompareFieldValues_AuthorWithSort(t *testing.T) {
	tests := []struct {
		golibri   string
		ebookMeta string
		match     bool
	}{
		{"余秋雨", "余秋雨 [余秋雨]", true},
		{"John Doe", "John Doe [Doe, John]", true},
		{"张三", "张三 [张三]", true},
	}

	for _, tc := range tests {
		result := compareFieldValues("author", tc.golibri, tc.ebookMeta)
		if result != tc.match {
			t.Errorf("compareFieldValues(author, %q, %q) = %v, want %v",
				tc.golibri, tc.ebookMeta, result, tc.match)
		}
	}
}

func TestCompareFieldValues_Language(t *testing.T) {
	tests := []struct {
		val1  string
		val2  string
		match bool
	}{
		{"en", "eng", true},
		{"en-us", "eng", true},
		{"zh", "zho", true},
		{"zh-cn", "zho", true},
		{"ja", "jpn", true},
		{"en", "zho", false},
	}

	for _, tc := range tests {
		result := compareFieldValues("language", tc.val1, tc.val2)
		if result != tc.match {
			t.Errorf("compareFieldValues(language, %q, %q) = %v, want %v",
				tc.val1, tc.val2, result, tc.match)
		}
	}
}

func TestCompareFieldValues_Date(t *testing.T) {
	tests := []struct {
		val1  string
		val2  string
		match bool
	}{
		{"2025-01-09", "2025-01-09", true},
		{"2025-01-09T00:00:00Z", "2025-01-09", true},
		{"2025-01-09T12:30:00+08:00", "2025-01-09", true},
		{"2025-01-09", "2025-01-10", false},
	}

	for _, tc := range tests {
		result := compareFieldValues("date", tc.val1, tc.val2)
		if result != tc.match {
			t.Errorf("compareFieldValues(date, %q, %q) = %v, want %v",
				tc.val1, tc.val2, result, tc.match)
		}
	}
}

func TestCompareFieldValues_SeriesIndex(t *testing.T) {
	tests := []struct {
		val1  string
		val2  string
		match bool
	}{
		{"1", "1", true},
		{"1.0", "1", true},
		{"2.0", "2", true},
		{"1", "2", false},
	}

	for _, tc := range tests {
		result := compareFieldValues("series_index", tc.val1, tc.val2)
		if result != tc.match {
			t.Errorf("compareFieldValues(series_index, %q, %q) = %v, want %v",
				tc.val1, tc.val2, result, tc.match)
		}
	}
}

// =============================================================================
// Ebook-meta Output Parsing Tests
// =============================================================================

func TestParseEbookMetaOutput(t *testing.T) {
	output := `Title               : Test Book
Author(s)           : John Doe
Publisher           : Test Publisher
Languages           : eng
Published           : 2025-01-09
Identifiers         : isbn:9781234567890
Series              : Test Series #1
Tags                : fiction, test
Comments            : A test book description`

	meta := ParseEbookMetaOutput(output)

	tests := []struct {
		field    string
		value    string
		expected string
	}{
		{"title", meta.Title, "Test Book"},
		{"author", meta.Authors, "John Doe"},
		{"publisher", meta.Publisher, "Test Publisher"},
		{"language", meta.Language, "eng"},
		{"date", meta.Published, "2025-01-09"},
		{"identifiers", meta.Identifiers, "isbn:9781234567890"},
		{"series", meta.Series, "Test Series"},
		{"series_index", meta.SeriesIndex, "1"},
		{"subjects", meta.Tags, "fiction, test"},
		{"description", meta.Comments, "A test book description"},
	}

	for _, tc := range tests {
		if tc.value != tc.expected {
			t.Errorf("ParseEbookMetaOutput field %q = %q, want %q",
				tc.field, tc.value, tc.expected)
		}
	}
}

func TestParseEbookMetaOutput_Empty(t *testing.T) {
	meta := ParseEbookMetaOutput("")

	if meta.Title != "" {
		t.Errorf("Expected empty title for empty output, got %q", meta.Title)
	}
}

// =============================================================================
// Copy to Temp Tests
// =============================================================================

func TestCopyToTemp(t *testing.T) {
	// Create a temp source file
	srcFile, err := os.CreateTemp("", "test-src-*.epub")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(srcFile.Name())

	testContent := []byte("test epub content")
	if _, err := srcFile.Write(testContent); err != nil {
		t.Fatal(err)
	}
	srcFile.Close()

	// Copy to temp
	tempFile, err := copyToTemp(srcFile.Name())
	if err != nil {
		t.Fatalf("copyToTemp failed: %v", err)
	}
	defer os.Remove(tempFile)

	// Verify content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Content mismatch: got %q, want %q", content, testContent)
	}

	// Verify different path (not modifying source)
	if tempFile == srcFile.Name() {
		t.Error("Temp file should have different path from source")
	}
}

func TestCopyToTemp_NonExistent(t *testing.T) {
	_, err := copyToTemp("/non/existent/file.epub")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// =============================================================================
// Test Case Structure Tests
// =============================================================================

func TestInteropTestCase_Fields(t *testing.T) {
	tc := InteropTestCase{
		FilePath:      "/path/to/book.epub",
		EPUBFormat:    "epub2",
		TestMode:      "read_compare",
		MetadataField: "title",
	}

	if tc.FilePath != "/path/to/book.epub" {
		t.Error("FilePath mismatch")
	}
	if tc.EPUBFormat != "epub2" {
		t.Error("EPUBFormat mismatch")
	}
	if tc.TestMode != "read_compare" {
		t.Error("TestMode mismatch")
	}
	if tc.MetadataField != "title" {
		t.Error("MetadataField mismatch")
	}
}

// =============================================================================
// Format Discovery Tests
// =============================================================================

func TestEPUBFormats(t *testing.T) {
	expected := []string{"oebps1", "epub2", "epub3-pure", "epub3-hybrid"}

	if len(EPUBFormats) != len(expected) {
		t.Errorf("Expected %d formats, got %d", len(expected), len(EPUBFormats))
	}

	for i, format := range expected {
		if EPUBFormats[i] != format {
			t.Errorf("Format[%d] = %q, want %q", i, EPUBFormats[i], format)
		}
	}
}

func TestMetadataFields(t *testing.T) {
	expected := []string{
		"title", "author", "language", "publisher", "date",
		"series", "series_index", "identifiers", "subjects", "description",
	}

	if len(MetadataFields) != len(expected) {
		t.Errorf("Expected %d fields, got %d", len(expected), len(MetadataFields))
	}

	for i, field := range expected {
		if MetadataFields[i] != field {
			t.Errorf("Field[%d] = %q, want %q", i, MetadataFields[i], field)
		}
	}
}

// =============================================================================
// Summary Calculation Tests
// =============================================================================

func TestCalculateSummary(t *testing.T) {
	report := &InteropReport{
		Summary: ReportSummary{
			ByFormat: make(map[string]FormatSummary),
			ByMode:   make(map[string]ModeSummary),
			ByField:  make(map[string]FieldSummary),
		},
		Results: []InteropResult{
			{
				TestCase: InteropTestCase{
					EPUBFormat: "epub2",
					TestMode:   "read_compare",
				},
				Passed: true,
				FieldResults: map[string]FieldResult{
					"title": {Field: "title", Match: true},
				},
			},
			{
				TestCase: InteropTestCase{
					EPUBFormat: "epub2",
					TestMode:   "read_compare",
				},
				Passed: false,
				FieldResults: map[string]FieldResult{
					"title": {Field: "title", Match: false},
				},
			},
		},
	}

	calculateSummary(report)

	if report.Summary.Total != 2 {
		t.Errorf("Total = %d, want 2", report.Summary.Total)
	}
	if report.Summary.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Summary.Passed)
	}
	if report.Summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", report.Summary.Failed)
	}

	// Check format summary
	fs := report.Summary.ByFormat["epub2"]
	if fs.Passed != 1 || fs.Failed != 1 {
		t.Errorf("ByFormat[epub2] = {%d, %d}, want {1, 1}", fs.Passed, fs.Failed)
	}

	// Check field summary
	flds := report.Summary.ByField["title"]
	if flds.Tested != 2 || flds.Matched != 1 {
		t.Errorf("ByField[title] = {%d, %d}, want {2, 1}", flds.Tested, flds.Matched)
	}

	// Check failures
	if len(report.Failures) != 1 {
		t.Errorf("Failures = %d, want 1", len(report.Failures))
	}
}

// =============================================================================
// Integration Test Helper - Requires testdata
// =============================================================================

func TestFindTestDataFiles(t *testing.T) {
	// Find testdata directory
	testdataDir := filepath.Join("..", "..", "..", "testdata", "samples")

	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("Skipping: testdata/samples directory not found")
	}

	for _, format := range EPUBFormats {
		formatDir := filepath.Join(testdataDir, format)
		if _, err := os.Stat(formatDir); os.IsNotExist(err) {
			t.Logf("Format directory not found: %s", formatDir)
			continue
		}

		files, err := FindEpubFiles(formatDir)
		if err != nil {
			t.Errorf("FindEpubFiles(%s) error: %v", formatDir, err)
			continue
		}

		t.Logf("Found %d EPUB files in %s", len(files), format)

		if len(files) != 5 {
			t.Logf("Warning: Expected 5 files in %s, found %d", format, len(files))
		}
	}
}
