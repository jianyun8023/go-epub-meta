package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Normalize Function Tests
// =============================================================================

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"  SPACES  ", "spaces"},
		{"MixedCase", "mixedcase"},
		{"", ""},
		{"  ", ""},
		{"Already Lowercase", "already lowercase"},
	}

	for _, tc := range tests {
		result := normalize(tc.input)
		if result != tc.expected {
			t.Errorf("normalize(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// =============================================================================
// Language Normalization Tests
// =============================================================================

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Chinese variants
		{"zh", "zho"},
		{"zh-cn", "zho"},
		{"zh-sg", "zho"},
		{"zh-hans", "zho"},
		{"zh-tw", "zho"},
		{"zh-hk", "zho"},
		{"zh-hant", "zho"},
		{"ZH-CN", "zho"},
		{"  zh  ", "zho"},

		// English variants
		{"en", "eng"},
		{"en-us", "eng"},
		{"en-gb", "eng"},
		{"EN-US", "eng"},

		// Japanese
		{"ja", "jpn"},
		{"jp", "jpn"},

		// European languages
		{"fr", "fra"},
		{"de", "deu"},
		{"es", "spa"},
		{"it", "ita"},
		{"ru", "rus"},

		// Already 3-letter codes (pass through)
		{"eng", "eng"},
		{"zho", "zho"},
		{"fra", "fra"},

		// Unknown (pass through)
		{"unknown", "unknown"},
		{"xyz", "xyz"},
	}

	for _, tc := range tests {
		result := normalizeLanguage(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeLanguage(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// =============================================================================
// Date Normalization Tests
// =============================================================================

func TestNormalizeDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Full ISO 8601 with timezone
		{"2023-12-31T16:00:00+00:00", "2023-12-31"},
		{"2025-01-07T00:00:00Z", "2025-01-07"},

		// Date only
		{"2023-12-31", "2023-12-31"},
		{"2025-01-01", "2025-01-01"},

		// With time but no timezone
		{"2023-12-31T12:00:00", "2023-12-31"},

		// Short formats (less than 10 chars, return as-is)
		{"2023", "2023"},
		{"2023-12", "2023-12"},
		{"", ""},
	}

	for _, tc := range tests {
		result := normalizeDate(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeDate(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// =============================================================================
// Identifier Normalization Tests
// =============================================================================

func TestNormalizeIdentifiers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Simple case
		{"isbn:123, asin:456", "asin:456,isbn:123"},
		{"ASIN:ABC, ISBN:123", "asin:abc,isbn:123"},

		// With extra spaces
		{"  isbn:123  ,  asin:456  ", "asin:456,isbn:123"},

		// Single identifier
		{"isbn:9780123456789", "isbn:9780123456789"},

		// Already sorted
		{"asin:abc,isbn:123", "asin:abc,isbn:123"},

		// Empty
		{"", ""},

		// Multiple same prefix
		{"isbn:111, isbn:222, asin:333", "asin:333,isbn:111,isbn:222"},
	}

	for _, tc := range tests {
		result := normalizeIdentifiers(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeIdentifiers(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// =============================================================================
// File Discovery Tests
// =============================================================================

func TestFindEpubFiles(t *testing.T) {
	// Create temp directory with some test files
	tmpDir, err := os.MkdirTemp("", "epub-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	testFiles := []string{
		"book1.epub",
		"book2.EPUB",
		"book3.txt",
		"readme.md",
		"subdir/book4.epub",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("test"), 0644)
	}

	// Find EPUB files
	files, err := findEpubFiles(tmpDir)
	if err != nil {
		t.Fatalf("findEpubFiles failed: %v", err)
	}

	// Should find 3 EPUB files (case insensitive)
	if len(files) != 3 {
		t.Errorf("Expected 3 EPUB files, found %d: %v", len(files), files)
	}

	// Verify all found files end with .epub (case insensitive)
	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != ".epub" && ext != ".EPUB" {
			t.Errorf("Found non-EPUB file: %s", f)
		}
	}
}

func TestFindEpubFilesEmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "epub-empty-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files, err := findEpubFiles(tmpDir)
	if err != nil {
		t.Fatalf("findEpubFiles failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files in empty directory, found %d", len(files))
	}
}

func TestFindEpubFilesNonExistentDir(t *testing.T) {
	files, err := findEpubFiles("/non/existent/directory")
	// Should not return an error but an empty list
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files for non-existent directory, found %d", len(files))
	}
}

// =============================================================================
// ComparisonResult Tests
// =============================================================================

func TestComparisonResultFields(t *testing.T) {
	// Test that ComparisonResult can hold all expected fields
	result := ComparisonResult{
		FilePath:           "/path/to/book.epub",
		GolibriTitle:       "Test Title",
		GolibriAuthor:      "Test Author",
		GolibriPublisher:   "Test Publisher",
		GolibriSeries:      "Test Series",
		GolibriLanguage:    "en",
		GolibriIdentifiers: "isbn:123",
		GolibriCover:       "Found",
		EbookTitle:         "Test Title",
		EbookAuthor:        "Test Author",
		EbookPublisher:     "Test Publisher",
		EbookSeries:        "Test Series",
		EbookLanguage:      "eng",
		EbookIdentifiers:   "isbn:123",
	}

	// Verify fields are set correctly
	if result.FilePath != "/path/to/book.epub" {
		t.Errorf("FilePath mismatch")
	}
	if result.GolibriTitle != "Test Title" {
		t.Errorf("GolibriTitle mismatch")
	}
	if result.EbookLanguage != "eng" {
		t.Errorf("EbookLanguage mismatch")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestNormalizeEmptyAndWhitespace(t *testing.T) {
	// Empty string
	if normalize("") != "" {
		t.Error("Empty string should normalize to empty")
	}

	// Only whitespace
	if normalize("   ") != "" {
		t.Error("Whitespace-only string should normalize to empty")
	}

	// Tabs and newlines
	if normalize("\t\n ") != "" {
		t.Error("Tab/newline should normalize to empty")
	}
}

func TestNormalizeDateEdgeCases(t *testing.T) {
	// Exactly 10 characters
	result := normalizeDate("2023-12-31")
	if result != "2023-12-31" {
		t.Errorf("Expected '2023-12-31', got '%s'", result)
	}

	// 9 characters (return as-is)
	result = normalizeDate("2023-12-3")
	if result != "2023-12-3" {
		t.Errorf("Expected '2023-12-3', got '%s'", result)
	}
}

func TestNormalizeIdentifiersEdgeCases(t *testing.T) {
	// Single space
	result := normalizeIdentifiers(" ")
	if result != "" {
		t.Errorf("Expected empty, got '%s'", result)
	}

	// Only commas
	result = normalizeIdentifiers(",,,")
	if result != ",,," {
		t.Errorf("Expected ',,,', got '%s'", result)
	}
}
