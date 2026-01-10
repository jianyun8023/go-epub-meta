package epub

import (
	"testing"
)

// createTestPackage creates a Package with sample metadata for testing
func createTestPackage() *Package {
	return &Package{
		Metadata: Metadata{
			Titles: []SimpleMeta{
				{Value: "Test Title"},
			},
			Creators: []AuthorMeta{
				{SimpleMeta: SimpleMeta{Value: "Test Author"}, Role: "aut", FileAs: "Author, Test"},
			},
			Languages: []SimpleMeta{
				{Value: "en"},
			},
			Identifiers: []IDMeta{
				{Scheme: "uuid", Value: "test-uuid-1234"},
				{Scheme: "ISBN", Value: "978-0-123456-78-9"},
			},
			Publishers: []SimpleMeta{
				{Value: "Test Publisher"},
			},
			Descriptions: []SimpleMeta{
				{Value: "A test book description."},
			},
			Dates: []SimpleMeta{
				{Value: "2025-01-01"},
			},
			Subjects: []SimpleMeta{
				{Value: "Fiction"},
				{Value: "Test"},
			},
			Meta: []Meta{
				{Name: "calibre:series", Content: "Test Series"},
				{Name: "calibre:series_index", Content: "1"},
				{Name: "calibre:title_sort", Content: "Title, Test"},
			},
		},
	}
}

// createEmptyPackage creates a Package with no metadata
func createEmptyPackage() *Package {
	return &Package{
		Metadata: Metadata{},
	}
}

// =============================================================================
// Title Tests
// =============================================================================

func TestGetTitle(t *testing.T) {
	pkg := createTestPackage()
	title := pkg.GetTitle()
	if title != "Test Title" {
		t.Errorf("Expected 'Test Title', got '%s'", title)
	}
}

func TestGetTitleEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	title := pkg.GetTitle()
	if title != "" {
		t.Errorf("Expected empty string, got '%s'", title)
	}
}

func TestSetTitle(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetTitle("New Title")
	if pkg.GetTitle() != "New Title" {
		t.Errorf("Expected 'New Title', got '%s'", pkg.GetTitle())
	}
}

func TestSetTitleOverwrite(t *testing.T) {
	pkg := createTestPackage()
	pkg.SetTitle("Overwritten Title")
	if pkg.GetTitle() != "Overwritten Title" {
		t.Errorf("Expected 'Overwritten Title', got '%s'", pkg.GetTitle())
	}
	// SetTitle should replace all titles
	if len(pkg.Metadata.Titles) != 1 {
		t.Errorf("Expected 1 title, got %d", len(pkg.Metadata.Titles))
	}
}

// =============================================================================
// Author Tests
// =============================================================================

func TestGetAuthor(t *testing.T) {
	pkg := createTestPackage()
	author := pkg.GetAuthor()
	if author != "Test Author" {
		t.Errorf("Expected 'Test Author', got '%s'", author)
	}
}

func TestGetAuthorEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	author := pkg.GetAuthor()
	if author != "" {
		t.Errorf("Expected empty string, got '%s'", author)
	}
}

func TestGetAuthorSort(t *testing.T) {
	pkg := createTestPackage()
	authorSort := pkg.GetAuthorSort()
	if authorSort != "Author, Test" {
		t.Errorf("Expected 'Author, Test', got '%s'", authorSort)
	}
}

func TestGetAuthorSortEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	authorSort := pkg.GetAuthorSort()
	if authorSort != "" {
		t.Errorf("Expected empty string, got '%s'", authorSort)
	}
}

func TestSetAuthor(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetAuthor("New Author")
	if pkg.GetAuthor() != "New Author" {
		t.Errorf("Expected 'New Author', got '%s'", pkg.GetAuthor())
	}
	// Check role is set correctly
	if len(pkg.Metadata.Creators) != 1 || pkg.Metadata.Creators[0].Role != "aut" {
		t.Error("Expected author role to be 'aut'")
	}
}

// =============================================================================
// GetAuthors Tests (Multiple Authors)
// =============================================================================

func TestGetAuthors_SingleCreator(t *testing.T) {
	pkg := createTestPackage()
	authors := pkg.GetAuthors()
	if len(authors) != 1 || authors[0] != "Test Author" {
		t.Errorf("Expected ['Test Author'], got %v", authors)
	}
}

func TestGetAuthors_MultipleCreators(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Metadata.Creators = []AuthorMeta{
		{SimpleMeta: SimpleMeta{Value: "Author One"}},
		{SimpleMeta: SimpleMeta{Value: "Author Two"}},
		{SimpleMeta: SimpleMeta{Value: "Author Three"}},
	}
	authors := pkg.GetAuthors()
	expected := []string{"Author One", "Author Two", "Author Three"}
	if len(authors) != 3 {
		t.Errorf("Expected 3 authors, got %d", len(authors))
	}
	for i, exp := range expected {
		if authors[i] != exp {
			t.Errorf("Expected authors[%d]='%s', got '%s'", i, exp, authors[i])
		}
	}
}

func TestGetAuthors_Empty(t *testing.T) {
	pkg := createEmptyPackage()
	authors := pkg.GetAuthors()
	if len(authors) != 0 {
		t.Errorf("Expected empty authors, got %v", authors)
	}
}

func TestGetAuthors_Deduplicate(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Metadata.Creators = []AuthorMeta{
		{SimpleMeta: SimpleMeta{Value: "Same Author"}},
		{SimpleMeta: SimpleMeta{Value: "Same Author"}},
	}
	authors := pkg.GetAuthors()
	if len(authors) != 1 || authors[0] != "Same Author" {
		t.Errorf("Expected deduplication, got %v", authors)
	}
}

// =============================================================================
// parseAuthorString Tests (Single creator with multiple authors)
// =============================================================================

func TestParseAuthorString_Ampersand(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"张三 & 李四", []string{"张三", "李四"}},
		{"Author A&Author B", []string{"Author A", "Author B"}},
		{"A & B & C", []string{"A", "B", "C"}},
	}
	for _, tc := range tests {
		result := parseAuthorString(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseAuthorString(%q): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i := range tc.expected {
			if result[i] != tc.expected[i] {
				t.Errorf("parseAuthorString(%q)[%d]: expected %q, got %q", tc.input, i, tc.expected[i], result[i])
			}
		}
	}
}

func TestParseAuthorString_ChineseComma(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"张三、李四", []string{"张三", "李四"}},
		{"作者A、作者B、作者C", []string{"作者A", "作者B", "作者C"}},
	}
	for _, tc := range tests {
		result := parseAuthorString(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseAuthorString(%q): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i := range tc.expected {
			if result[i] != tc.expected[i] {
				t.Errorf("parseAuthorString(%q)[%d]: expected %q, got %q", tc.input, i, tc.expected[i], result[i])
			}
		}
	}
}

func TestParseAuthorString_And(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"John and Jane", []string{"John", "Jane"}},
		{"A and B and C", []string{"A", "B", "C"}},
	}
	for _, tc := range tests {
		result := parseAuthorString(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseAuthorString(%q): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i := range tc.expected {
			if result[i] != tc.expected[i] {
				t.Errorf("parseAuthorString(%q)[%d]: expected %q, got %q", tc.input, i, tc.expected[i], result[i])
			}
		}
	}
}

func TestParseAuthorString_Semicolon(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Author A; Author B", []string{"Author A", "Author B"}},
		{"张三；李四", []string{"张三", "李四"}},
	}
	for _, tc := range tests {
		result := parseAuthorString(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseAuthorString(%q): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i := range tc.expected {
			if result[i] != tc.expected[i] {
				t.Errorf("parseAuthorString(%q)[%d]: expected %q, got %q", tc.input, i, tc.expected[i], result[i])
			}
		}
	}
}

func TestParseAuthorString_NoSplit(t *testing.T) {
	// These should NOT be split
	tests := []struct {
		input    string
		expected []string
	}{
		{"Single Author", []string{"Single Author"}},
		{"黄仁宇", []string{"黄仁宇"}},
		{"Doe, John", []string{"Doe, John"}}, // Last, First format - should not split
		{"张三", []string{"张三"}},
	}
	for _, tc := range tests {
		result := parseAuthorString(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseAuthorString(%q): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i := range tc.expected {
			if result[i] != tc.expected[i] {
				t.Errorf("parseAuthorString(%q)[%d]: expected %q, got %q", tc.input, i, tc.expected[i], result[i])
			}
		}
	}
}

func TestParseAuthorString_MultipleCommas(t *testing.T) {
	// Multiple commas should be split
	tests := []struct {
		input    string
		expected []string
	}{
		{"A, B, C", []string{"A", "B", "C"}},
		{"张三, 李四, 王五", []string{"张三", "李四", "王五"}},
	}
	for _, tc := range tests {
		result := parseAuthorString(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseAuthorString(%q): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i := range tc.expected {
			if result[i] != tc.expected[i] {
				t.Errorf("parseAuthorString(%q)[%d]: expected %q, got %q", tc.input, i, tc.expected[i], result[i])
			}
		}
	}
}

func TestParseAuthorString_Empty(t *testing.T) {
	result := parseAuthorString("")
	if result != nil {
		t.Errorf("Expected nil for empty string, got %v", result)
	}

	result = parseAuthorString("   ")
	if result != nil {
		t.Errorf("Expected nil for whitespace string, got %v", result)
	}
}

// =============================================================================
// Description Tests
// =============================================================================

func TestGetDescription(t *testing.T) {
	pkg := createTestPackage()
	desc := pkg.GetDescription()
	if desc != "A test book description." {
		t.Errorf("Expected 'A test book description.', got '%s'", desc)
	}
}

func TestGetDescriptionEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	desc := pkg.GetDescription()
	if desc != "" {
		t.Errorf("Expected empty string, got '%s'", desc)
	}
}

func TestSetDescription(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetDescription("New description")
	if pkg.GetDescription() != "New description" {
		t.Errorf("Expected 'New description', got '%s'", pkg.GetDescription())
	}
}

// =============================================================================
// Language Tests
// =============================================================================

func TestGetLanguage(t *testing.T) {
	pkg := createTestPackage()
	lang := pkg.GetLanguage()
	if lang != "en" {
		t.Errorf("Expected 'en', got '%s'", lang)
	}
}

func TestGetLanguageEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	lang := pkg.GetLanguage()
	if lang != "und" {
		t.Errorf("Expected 'und' (undetermined), got '%s'", lang)
	}
}

func TestSetLanguage(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetLanguage("zh")
	if pkg.GetLanguage() != "zh" {
		t.Errorf("Expected 'zh', got '%s'", pkg.GetLanguage())
	}
}

// =============================================================================
// Series Tests
// =============================================================================

func TestGetSeries(t *testing.T) {
	pkg := createTestPackage()
	series := pkg.GetSeries()
	if series != "Test Series" {
		t.Errorf("Expected 'Test Series', got '%s'", series)
	}
}

func TestGetSeriesEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	series := pkg.GetSeries()
	if series != "" {
		t.Errorf("Expected empty string, got '%s'", series)
	}
}

func TestGetSeriesIndex(t *testing.T) {
	pkg := createTestPackage()
	index := pkg.GetSeriesIndex()
	if index != "1" {
		t.Errorf("Expected '1', got '%s'", index)
	}
}

func TestGetSeriesIndexEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	index := pkg.GetSeriesIndex()
	if index != "" {
		t.Errorf("Expected empty string, got '%s'", index)
	}
}

func TestSetSeries(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetSeries("New Series")
	if pkg.GetSeries() != "New Series" {
		t.Errorf("Expected 'New Series', got '%s'", pkg.GetSeries())
	}
}

func TestSetSeriesUpdate(t *testing.T) {
	pkg := createTestPackage()
	pkg.SetSeries("Updated Series")
	if pkg.GetSeries() != "Updated Series" {
		t.Errorf("Expected 'Updated Series', got '%s'", pkg.GetSeries())
	}
}

func TestSetSeriesIndex(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetSeriesIndex("5")
	if pkg.GetSeriesIndex() != "5" {
		t.Errorf("Expected '5', got '%s'", pkg.GetSeriesIndex())
	}
}

func TestSetSeriesIndexUpdate(t *testing.T) {
	pkg := createTestPackage()
	pkg.SetSeriesIndex("10")
	if pkg.GetSeriesIndex() != "10" {
		t.Errorf("Expected '10', got '%s'", pkg.GetSeriesIndex())
	}
}

// =============================================================================
// EPUB3 Series Tests (belongs-to-collection)
// =============================================================================

// createEPUB3Package creates a Package with EPUB 3.0 version
func createEPUB3Package() *Package {
	return &Package{
		Version: "3.0",
		Metadata: Metadata{
			Titles: []SimpleMeta{
				{Value: "EPUB3 Test Title"},
			},
			Creators: []AuthorMeta{
				{SimpleMeta: SimpleMeta{Value: "EPUB3 Author"}},
			},
			Languages: []SimpleMeta{
				{Value: "en"},
			},
		},
	}
}

// createEPUB3PackageWithSeries creates an EPUB3 Package with existing series metadata
func createEPUB3PackageWithSeries() *Package {
	return &Package{
		Version: "3.0",
		Metadata: Metadata{
			Titles: []SimpleMeta{
				{Value: "EPUB3 Series Book"},
			},
			Languages: []SimpleMeta{
				{Value: "en"},
			},
			Meta: []Meta{
				{ID: "c01", Property: "belongs-to-collection", Value: "Existing Series"},
				{Refines: "#c01", Property: "collection-type", Value: "series"},
				{Refines: "#c01", Property: "group-position", Value: "3"},
			},
		},
	}
}

func TestIsEPUB3(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		{"3.0", true},
		{"3.1", true},
		{"3.2", true},
		{"2.0", false},
		{"2.0.1", false},
		{"", false},
		{"  3.0  ", true}, // with whitespace
	}

	for _, tc := range tests {
		pkg := &Package{Version: tc.version}
		result := pkg.isEPUB3()
		if result != tc.expected {
			t.Errorf("isEPUB3(%q) = %v, expected %v", tc.version, result, tc.expected)
		}
	}
}

func TestSetSeriesEPUB3_NewSeries(t *testing.T) {
	pkg := createEPUB3Package()

	pkg.SetSeries("New EPUB3 Series")

	// Verify series is readable
	series := pkg.GetSeries()
	if series != "New EPUB3 Series" {
		t.Errorf("Expected 'New EPUB3 Series', got '%s'", series)
	}

	// Verify EPUB3 structure: belongs-to-collection + collection-type
	var foundCollection, foundType bool
	var collectionID string
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "belongs-to-collection" && m.Value == "New EPUB3 Series" {
			foundCollection = true
			collectionID = m.ID
		}
	}
	for _, m := range pkg.Metadata.Meta {
		if m.Refines == "#"+collectionID && m.Property == "collection-type" && m.Value == "series" {
			foundType = true
		}
	}

	if !foundCollection {
		t.Error("EPUB3 series should have belongs-to-collection meta")
	}
	if !foundType {
		t.Error("EPUB3 series should have collection-type=series meta")
	}
}

func TestSetSeriesEPUB3_UpdateExisting(t *testing.T) {
	pkg := createEPUB3PackageWithSeries()

	// Verify initial state
	if pkg.GetSeries() != "Existing Series" {
		t.Fatalf("Initial series should be 'Existing Series', got '%s'", pkg.GetSeries())
	}

	// Update series
	pkg.SetSeries("Updated EPUB3 Series")

	if pkg.GetSeries() != "Updated EPUB3 Series" {
		t.Errorf("Expected 'Updated EPUB3 Series', got '%s'", pkg.GetSeries())
	}

	// Verify we didn't create duplicate collection entries
	collectionCount := 0
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "belongs-to-collection" {
			collectionCount++
		}
	}
	if collectionCount != 1 {
		t.Errorf("Expected 1 belongs-to-collection, got %d", collectionCount)
	}
}

func TestSetSeriesIndexEPUB3(t *testing.T) {
	pkg := createEPUB3PackageWithSeries()

	pkg.SetSeriesIndex("7")

	if pkg.GetSeriesIndex() != "7" {
		t.Errorf("Expected '7', got '%s'", pkg.GetSeriesIndex())
	}

	// Verify EPUB3 structure: group-position
	var foundPosition bool
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "group-position" && m.Value == "7" {
			foundPosition = true
		}
	}
	if !foundPosition {
		t.Error("EPUB3 series index should have group-position meta")
	}
}

func TestSetSeriesIndexEPUB3_NewSeriesCreated(t *testing.T) {
	pkg := createEPUB3Package()

	// Set index first (should trigger series creation)
	pkg.SetSeriesIndex("5")

	// Verify structure was created
	idx := pkg.GetSeriesIndex()
	if idx != "5" {
		t.Errorf("Expected '5', got '%s'", idx)
	}
}

// TestSetSeriesIndexEPUB3_LegacyOnlyFallback tests the fallback behavior when
// an EPUB3 file has only legacy calibre:series metadata (no belongs-to-collection).
// This is the scenario fixed by the SetSeriesIndex fallback implementation.
func TestSetSeriesIndexEPUB3_LegacyOnlyFallback(t *testing.T) {
	// Create EPUB3 package with only legacy series (no belongs-to-collection)
	pkg := &Package{
		Version: "3.0",
		Metadata: Metadata{
			Titles: []SimpleMeta{
				{Value: "EPUB3 Legacy Series Book"},
			},
			Languages: []SimpleMeta{
				{Value: "en"},
			},
			Meta: []Meta{
				// Legacy calibre:series ONLY - no belongs-to-collection
				{Name: "calibre:series", Content: "Legacy Series Name"},
			},
		},
	}

	// Verify GetSeries returns the legacy series
	if series := pkg.GetSeries(); series != "Legacy Series Name" {
		t.Fatalf("Setup error: GetSeries() should return 'Legacy Series Name', got '%s'", series)
	}

	// Set series index - this should use the fallback (calibre:series_index property)
	pkg.SetSeriesIndex("7")

	// Verify the index is readable via GetSeriesIndex
	if idx := pkg.GetSeriesIndex(); idx != "7" {
		t.Errorf("Expected series index '7', got '%s'", idx)
	}

	// Verify the fallback was used: should have calibre:series_index property
	var foundFallbackProperty bool
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "calibre:series_index" && m.Value == "7" {
			foundFallbackProperty = true
			break
		}
	}
	if !foundFallbackProperty {
		t.Error("Expected calibre:series_index property to be set as fallback")
	}

	// Verify NO belongs-to-collection was created (we don't want to pollute legacy-only files)
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "belongs-to-collection" {
			t.Error("Should NOT create belongs-to-collection for legacy-only series files")
		}
	}
}

func TestSetSeriesEPUB2_UsesCalibresStyle(t *testing.T) {
	pkg := createEmptyPackage() // Version is empty, treated as EPUB2

	pkg.SetSeries("EPUB2 Series")

	// Verify calibre:series is used
	var foundCalibreSeries bool
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series" && m.Content == "EPUB2 Series" {
			foundCalibreSeries = true
		}
	}
	if !foundCalibreSeries {
		t.Error("EPUB2 should use calibre:series meta tag")
	}

	// Verify EPUB3 style is NOT used
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "belongs-to-collection" {
			t.Error("EPUB2 should not use belongs-to-collection")
		}
	}
}

func TestGetSeriesEPUB3_ReadsBelongsToCollection(t *testing.T) {
	pkg := createEPUB3PackageWithSeries()

	series := pkg.GetSeries()
	if series != "Existing Series" {
		t.Errorf("Expected 'Existing Series', got '%s'", series)
	}
}

func TestGetSeriesIndexEPUB3_ReadsGroupPosition(t *testing.T) {
	pkg := createEPUB3PackageWithSeries()

	idx := pkg.GetSeriesIndex()
	if idx != "3" {
		t.Errorf("Expected '3', got '%s'", idx)
	}
}

// =============================================================================
// Subject Tests
// =============================================================================

func TestGetSubjects(t *testing.T) {
	pkg := createTestPackage()
	subjects := pkg.GetSubjects()
	if len(subjects) != 2 {
		t.Errorf("Expected 2 subjects, got %d", len(subjects))
	}
	if subjects[0] != "Fiction" || subjects[1] != "Test" {
		t.Errorf("Unexpected subjects: %v", subjects)
	}
}

func TestGetSubjectsEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	subjects := pkg.GetSubjects()
	if len(subjects) != 0 {
		t.Errorf("Expected 0 subjects, got %d", len(subjects))
	}
}

func TestSetSubjects(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetSubjects([]string{"Science", "Technology", "History"})
	subjects := pkg.GetSubjects()
	if len(subjects) != 3 {
		t.Errorf("Expected 3 subjects, got %d", len(subjects))
	}
}

// =============================================================================
// Identifier Tests
// =============================================================================

func TestGetIdentifiers(t *testing.T) {
	pkg := createTestPackage()
	ids := pkg.GetIdentifiers()
	// UUID should be filtered out
	if _, ok := ids["uuid"]; ok {
		t.Error("UUID should be filtered out from GetIdentifiers")
	}
	if ids["isbn"] != "978-0-123456-78-9" {
		t.Errorf("Expected ISBN '978-0-123456-78-9', got '%s'", ids["isbn"])
	}
}

func TestGetIdentifiersEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	ids := pkg.GetIdentifiers()
	if len(ids) != 0 {
		t.Errorf("Expected empty map, got %d identifiers", len(ids))
	}
}

func TestGetISBN(t *testing.T) {
	pkg := createTestPackage()
	isbn := pkg.GetISBN()
	if isbn != "978-0-123456-78-9" {
		t.Errorf("Expected '978-0-123456-78-9', got '%s'", isbn)
	}
}

func TestGetISBNEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	isbn := pkg.GetISBN()
	if isbn != "" {
		t.Errorf("Expected empty string, got '%s'", isbn)
	}
}

func TestGetASIN(t *testing.T) {
	pkg := &Package{
		Metadata: Metadata{
			Identifiers: []IDMeta{
				{Scheme: "ASIN", Value: "B08XYZABC1"},
			},
		},
	}
	asin := pkg.GetASIN()
	if asin != "B08XYZABC1" {
		t.Errorf("Expected 'B08XYZABC1', got '%s'", asin)
	}
}

func TestGetASINEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	asin := pkg.GetASIN()
	if asin != "" {
		t.Errorf("Expected empty string, got '%s'", asin)
	}
}

func TestSetIdentifier(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetIdentifier("douban", "12345678")
	ids := pkg.GetIdentifiers()
	if ids["douban"] != "12345678" {
		t.Errorf("Expected douban ID '12345678', got '%s'", ids["douban"])
	}
}

func TestSetIdentifierUpdate(t *testing.T) {
	pkg := createTestPackage()
	pkg.SetIdentifier("ISBN", "978-1-111111-11-1")
	ids := pkg.GetIdentifiers()
	if ids["isbn"] != "978-1-111111-11-1" {
		t.Errorf("Expected ISBN '978-1-111111-11-1', got '%s'", ids["isbn"])
	}
}

func TestSetISBN(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetISBN("978-3-16-148410-0")
	isbn := pkg.GetISBN()
	if isbn != "978-3-16-148410-0" {
		t.Errorf("Expected '978-3-16-148410-0', got '%s'", isbn)
	}
}

func TestSetASIN(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetASIN("B08TEST123")
	asin := pkg.GetASIN()
	if asin != "B08TEST123" {
		t.Errorf("Expected 'B08TEST123', got '%s'", asin)
	}
}

// =============================================================================
// Publisher Tests
// =============================================================================

func TestGetPublisher(t *testing.T) {
	pkg := createTestPackage()
	publisher := pkg.GetPublisher()
	if publisher != "Test Publisher" {
		t.Errorf("Expected 'Test Publisher', got '%s'", publisher)
	}
}

func TestGetPublisherEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	publisher := pkg.GetPublisher()
	if publisher != "" {
		t.Errorf("Expected empty string, got '%s'", publisher)
	}
}

func TestSetPublisher(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetPublisher("New Publisher")
	if pkg.GetPublisher() != "New Publisher" {
		t.Errorf("Expected 'New Publisher', got '%s'", pkg.GetPublisher())
	}
}

// =============================================================================
// Publisher Date Tests
// =============================================================================

func TestGetPublishDate(t *testing.T) {
	pkg := createTestPackage()
	date := pkg.GetPublishDate()
	if date != "2025-01-01" {
		t.Errorf("Expected '2025-01-01', got '%s'", date)
	}
}

func TestGetPublishDateEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	date := pkg.GetPublishDate()
	if date != "" {
		t.Errorf("Expected empty string, got '%s'", date)
	}
}

func TestSetPublishDate(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetPublishDate("2024-12-25")
	if pkg.GetPublishDate() != "2024-12-25" {
		t.Errorf("Expected '2024-12-25', got '%s'", pkg.GetPublishDate())
	}
}

// =============================================================================
// Producer Tests
// =============================================================================

func TestGetProducerFromContributor(t *testing.T) {
	pkg := &Package{
		Metadata: Metadata{
			Contributors: []AuthorMeta{
				{SimpleMeta: SimpleMeta{Value: "calibre 7.0"}, Role: "bkp"},
			},
		},
	}
	producer := pkg.GetProducer()
	if producer != "calibre 7.0" {
		t.Errorf("Expected 'calibre 7.0', got '%s'", producer)
	}
}

func TestGetProducerFromMeta(t *testing.T) {
	pkg := &Package{
		Metadata: Metadata{
			Meta: []Meta{
				{Name: "generator", Content: "EPUB Generator 1.0"},
			},
		},
	}
	producer := pkg.GetProducer()
	if producer != "EPUB Generator 1.0" {
		t.Errorf("Expected 'EPUB Generator 1.0', got '%s'", producer)
	}
}

func TestGetProducerEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	producer := pkg.GetProducer()
	if producer != "" {
		t.Errorf("Expected empty string, got '%s'", producer)
	}
}

// =============================================================================
// Identifier Parsing Tests
// =============================================================================

func TestParseIdentifierURN(t *testing.T) {
	scheme, value := parseIdentifier("", "urn:isbn:9780123456789")
	if scheme != "isbn" || value != "9780123456789" {
		t.Errorf("Expected (isbn, 9780123456789), got (%s, %s)", scheme, value)
	}
}

func TestParseIdentifierSchemeValue(t *testing.T) {
	scheme, value := parseIdentifier("", "calibre:12345")
	if scheme != "calibre" || value != "12345" {
		t.Errorf("Expected (calibre, 12345), got (%s, %s)", scheme, value)
	}
}

func TestParseIdentifierWithScheme(t *testing.T) {
	scheme, value := parseIdentifier("ISBN", "978-0-123456-78-9")
	if scheme != "isbn" || value != "978-0-123456-78-9" {
		t.Errorf("Expected (isbn, 978-0-123456-78-9), got (%s, %s)", scheme, value)
	}
}

func TestSetIdentifier_CaseInsensitiveUpdate(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Metadata.Identifiers = []IDMeta{
		{Scheme: "isbn", Value: "OLD"},
	}

	pkg.SetIdentifier("ISBN", "NEW")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	if got := pkg.Metadata.Identifiers[0].Value; got != "NEW" {
		t.Fatalf("Expected value NEW, got %q", got)
	}
	if gotScheme := pkg.Metadata.Identifiers[0].Scheme; gotScheme != "isbn" {
		t.Fatalf("Expected scheme preserved as 'isbn', got %q", gotScheme)
	}
}

func TestSetISBN_UpdatesExistingSchemeLowercase(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Metadata.Identifiers = []IDMeta{
		{Scheme: "isbn", Value: "9780000000000"},
	}

	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	if got := pkg.Metadata.Identifiers[0].Value; got != "9780306406157" {
		t.Fatalf("Expected updated ISBN value, got %q", got)
	}
	if gotScheme := pkg.Metadata.Identifiers[0].Scheme; gotScheme != "ISBN" {
		// EPUB2 write normalization prefers opf:scheme="ISBN"
		t.Fatalf("Expected scheme normalized to 'ISBN', got %q", gotScheme)
	}
}

func TestSetISBN_UpdatesExistingURN(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Metadata.Identifiers = []IDMeta{
		{Scheme: "", Value: "urn:isbn:9780000000000"},
	}

	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	// EPUB2 (empty version) uses plain value with scheme
	if got := pkg.Metadata.Identifiers[0].Value; got != "9780306406157" {
		t.Fatalf("Expected updated ISBN value, got %q", got)
	}
	if gotScheme := pkg.Metadata.Identifiers[0].Scheme; gotScheme != "ISBN" {
		t.Fatalf("Expected scheme ISBN for EPUB2, got %q", gotScheme)
	}
}

func TestSetISBN_AddsURNIfMissing(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	if got := pkg.Metadata.Identifiers[0].Value; got != "9780306406157" {
		t.Fatalf("Expected plain ISBN value, got %q", got)
	}
	if gotScheme := pkg.Metadata.Identifiers[0].Scheme; gotScheme != "ISBN" {
		t.Fatalf("Expected scheme ISBN for new identifier, got %q", gotScheme)
	}
}

func TestSetISBN_AddsURNIfMissing_EPUB3(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Version = "3.0"
	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	// Hybrid ISBN format: uses "isbn:" prefix for Calibre compatibility
	if got := pkg.Metadata.Identifiers[0].Value; got != "isbn:9780306406157" {
		t.Fatalf("Expected isbn: prefixed value for EPUB3, got %q", got)
	}
	if gotScheme := pkg.Metadata.Identifiers[0].Scheme; gotScheme != "" {
		t.Fatalf("Expected empty scheme for EPUB3 isbn: prefix, got %q", gotScheme)
	}
}

func TestSetISBN_UpdatesAllISBNEntries(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Metadata.Identifiers = []IDMeta{
		{ID: "ISBN", Value: "978-7-5001-6815-7"},
		{Scheme: "ISBN", Value: "9787500168157"},
	}

	pkg.SetISBN("9780306406157")

	// SetISBN only updates the FIRST ISBN entry found
	if len(pkg.Metadata.Identifiers) != 2 {
		t.Fatalf("Expected 2 identifiers, got %d", len(pkg.Metadata.Identifiers))
	}
	// First entry is updated
	if got := pkg.Metadata.Identifiers[0].Value; got != "9780306406157" {
		t.Fatalf("Expected first entry updated, got %q", got)
	}
	// Second entry remains unchanged
	if got := pkg.Metadata.Identifiers[1].Value; got != "9787500168157" {
		t.Fatalf("Expected second entry unchanged, got %q", got)
	}
}

// TestSetISBN_NormalizesFullwidthSeparators tests that fullwidth separators
// (U+FF0D FULLWIDTH HYPHEN-MINUS) are normalized to ASCII hyphen (U+002D).
func TestSetISBN_NormalizesFullwidthSeparators(t *testing.T) {
	pkg := createEmptyPackage()
	// Original ISBN with fullwidth hyphen-minus (U+FF0D): 978－7－300－23400-7
	pkg.Metadata.Identifiers = []IDMeta{
		{Scheme: "ISBN", Value: "978－7－300－23400-7"},
	}

	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	// SetISBN replaces the value entirely with the new clean ISBN
	if got := pkg.Metadata.Identifiers[0].Value; got != "9780306406157" {
		t.Fatalf("Expected clean ISBN '9780306406157', got %q", got)
	}
}

// TestSetISBN_ReplacesInvalidFormat tests that ISBN with invalid characters
// (e.g., "/I·246" suffix) is replaced with clean ISBN value.
func TestSetISBN_ReplacesInvalidFormat(t *testing.T) {
	pkg := createEmptyPackage()
	// Original ISBN with invalid suffix: 7-80639-918-6/I·246
	pkg.Metadata.Identifiers = []IDMeta{
		{Scheme: "ISBN", Value: "7-80639-918-6/I·246"},
	}

	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	// Invalid format should be replaced with clean ISBN
	if got := pkg.Metadata.Identifiers[0].Value; got != "9780306406157" {
		t.Fatalf("Expected clean ISBN '9780306406157', got %q", got)
	}
}

func TestSetISBN_ConvertsValuePrefixISBN_ToSchemeForEPUB2(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Version = "2.0"
	pkg.Metadata.Identifiers = []IDMeta{
		{Scheme: "", Value: "isbn:9780000000000"},
	}

	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	if gotScheme := pkg.Metadata.Identifiers[0].Scheme; gotScheme != "ISBN" {
		t.Fatalf("Expected scheme ISBN for EPUB2, got %q", gotScheme)
	}
	if got := pkg.Metadata.Identifiers[0].Value; got != "9780306406157" {
		t.Fatalf("Expected plain ISBN value for EPUB2, got %q", got)
	}
}

func TestSetISBN_ConvertsValuePrefixISBN_ToURNForEPUB3(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Version = "3.0"
	pkg.Metadata.Identifiers = []IDMeta{
		{Scheme: "", Value: "isbn:9780000000000"},
	}

	pkg.SetISBN("9780306406157")

	if len(pkg.Metadata.Identifiers) != 1 {
		t.Fatalf("Expected 1 identifier, got %d", len(pkg.Metadata.Identifiers))
	}
	if gotScheme := pkg.Metadata.Identifiers[0].Scheme; gotScheme != "" {
		t.Fatalf("Expected empty scheme for EPUB3 isbn: prefix, got %q", gotScheme)
	}
	// Hybrid ISBN: uses "isbn:" prefix for Calibre compatibility
	if got := pkg.Metadata.Identifiers[0].Value; got != "isbn:9780306406157" {
		t.Fatalf("Expected isbn: prefixed value for EPUB3, got %q", got)
	}
}

func TestParseIdentifierUUID(t *testing.T) {
	scheme, value := parseIdentifier("", "urn:uuid:12345678-1234-1234-1234-123456789012")
	if scheme != "uuid" {
		t.Errorf("Expected scheme 'uuid', got '%s'", scheme)
	}
	if value != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("Expected UUID value, got '%s'", value)
	}
}

func TestParseIdentifierISBNHeuristic(t *testing.T) {
	// ISBN-13 format
	scheme, value := parseIdentifier("", "9780123456789")
	if scheme != "isbn" || value != "9780123456789" {
		t.Errorf("Expected (isbn, 9780123456789), got (%s, %s)", scheme, value)
	}
}

func TestParseIdentifierUnknown(t *testing.T) {
	scheme, value := parseIdentifier("", "random-value")
	if scheme != "unknown" {
		t.Errorf("Expected scheme 'unknown', got '%s'", scheme)
	}
	if value != "random-value" {
		t.Errorf("Expected value 'random-value', got '%s'", value)
	}
}

// =============================================================================
// ISBN Detection Tests
// =============================================================================

func TestIsISBN10(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0123456789", true},
		{"012345678X", true},
		{"012345678x", true},
		{"0-12-345678-9", true},
		{"0 12 345678 9", true},
		{"123456789", false},   // Too short
		{"12345678901", false}, // Too long
		{"012345678A", false},  // Invalid char
	}

	for _, tc := range tests {
		result := isISBN(tc.input)
		if result != tc.expected {
			t.Errorf("isISBN(%s) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestIsISBN13(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"9780123456789", true},
		{"9790123456789", true},
		{"978-0-12-345678-9", true},
		{"979 0 12 345678 9", true},
		{"1234567890123", false},  // Doesn't start with 978/979
		{"978012345678", false},   // Too short
		{"97801234567890", false}, // Too long
	}

	for _, tc := range tests {
		result := isISBN(tc.input)
		if result != tc.expected {
			t.Errorf("isISBN(%s) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

// =============================================================================
// Scheme Normalization Tests
// =============================================================================

func TestNormalizeScheme(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ISBN", "isbn"},
		{"isbn", "isbn"},
		{"ASIN", "asin"},
		{"mobi-asin", "mobi-asin"},
		{"UUID", "uuid"},
		{"urn:isbn", "isbn"},
		{"urn:uuid", "uuid"},
		{"calibre", "calibre"},
		{"CALIBRE", "calibre"},
		{"douban", "douban"},
		{"new_douban", "new_douban"},
		{"amazon_cn", "amazon_cn"},
		{"google", "google"},
		{"doi", "doi"},
		{"custom", "custom"},
		{"  SPACES  ", "spaces"},
	}

	for _, tc := range tests {
		result := normalizeScheme(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeScheme(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

// =============================================================================
// Multiple Identifiers Tests
// =============================================================================

func TestMultipleIdentifiers(t *testing.T) {
	pkg := &Package{
		Metadata: Metadata{
			Identifiers: []IDMeta{
				{Scheme: "ISBN", Value: "978-0-123456-78-9"},
				{Scheme: "ASIN", Value: "B08XYZABC1"},
				{Scheme: "douban", Value: "12345678"},
				{Scheme: "uuid", Value: "test-uuid"},
			},
		},
	}

	ids := pkg.GetIdentifiers()

	// UUID should be filtered
	if _, ok := ids["uuid"]; ok {
		t.Error("UUID should be filtered")
	}

	if ids["isbn"] != "978-0-123456-78-9" {
		t.Errorf("Expected ISBN, got '%s'", ids["isbn"])
	}
	if ids["asin"] != "B08XYZABC1" {
		t.Errorf("Expected ASIN, got '%s'", ids["asin"])
	}
	if ids["douban"] != "12345678" {
		t.Errorf("Expected douban, got '%s'", ids["douban"])
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestSpecialCharactersInTitle(t *testing.T) {
	pkg := createEmptyPackage()
	specialTitle := `书名 & "特殊字符" <测试> 📚`
	pkg.SetTitle(specialTitle)
	if pkg.GetTitle() != specialTitle {
		t.Errorf("Special characters not preserved: got '%s'", pkg.GetTitle())
	}
}

func TestSpecialCharactersInAuthor(t *testing.T) {
	pkg := createEmptyPackage()
	specialAuthor := "张三 & 李四 <合著>"
	pkg.SetAuthor(specialAuthor)
	if pkg.GetAuthor() != specialAuthor {
		t.Errorf("Special characters not preserved: got '%s'", pkg.GetAuthor())
	}
}

func TestEmptyStringValues(t *testing.T) {
	pkg := createEmptyPackage()

	// Setting empty strings should work
	pkg.SetTitle("")
	pkg.SetAuthor("")
	pkg.SetSeries("")

	if pkg.GetTitle() != "" {
		t.Errorf("Expected empty title, got '%s'", pkg.GetTitle())
	}
	if pkg.GetAuthor() != "" {
		t.Errorf("Expected empty author, got '%s'", pkg.GetAuthor())
	}
	if pkg.GetSeries() != "" {
		t.Errorf("Expected empty series, got '%s'", pkg.GetSeries())
	}
}

func TestWhitespaceHandling(t *testing.T) {
	pkg := createEmptyPackage()

	// Whitespace should be preserved (not trimmed) in values
	pkg.SetTitle("  Title with Spaces  ")
	if pkg.GetTitle() != "  Title with Spaces  " {
		t.Errorf("Whitespace not preserved: got '%s'", pkg.GetTitle())
	}
}

// =============================================================================
// SetRating Tests (Calibre Extension)
// =============================================================================

func TestSetRating_BasicValues(t *testing.T) {
	tests := []struct {
		name        string
		input       int
		expectedGet int
		expectedRaw string
	}{
		{"Rating 0", 0, 0, "0"},
		{"Rating 1", 1, 1, "2"},
		{"Rating 2", 2, 2, "4"},
		{"Rating 3", 3, 3, "6"},
		{"Rating 4", 4, 4, "8"},
		{"Rating 5", 5, 5, "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := createEmptyPackage()
			pkg.SetRating(tt.input)

			if got := pkg.GetRating(); got != tt.expectedGet {
				t.Errorf("GetRating() = %d, want %d", got, tt.expectedGet)
			}
			if got := pkg.GetRatingRaw(); got != tt.expectedRaw {
				t.Errorf("GetRatingRaw() = %s, want %s", got, tt.expectedRaw)
			}
		})
	}
}

func TestSetRating_UpdateExisting(t *testing.T) {
	pkg := createEmptyPackage()

	// Set initial rating
	pkg.SetRating(3)
	if pkg.GetRating() != 3 {
		t.Errorf("Initial rating: got %d, want 3", pkg.GetRating())
	}

	// Update rating
	pkg.SetRating(5)
	if pkg.GetRating() != 5 {
		t.Errorf("Updated rating: got %d, want 5", pkg.GetRating())
	}

	// Verify only one calibre:rating meta exists
	count := 0
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:rating" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected 1 calibre:rating meta, got %d", count)
	}
}

func TestSetRating_UpdateExisting_EPUB3Property(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Version = "3.0"
	pkg.Metadata.Meta = []Meta{
		{Property: "calibre:rating", Value: "10"},
	}

	pkg.SetRating(4) // 4 -> 8

	if got := pkg.GetRatingRaw(); got != "8" {
		t.Errorf("GetRatingRaw() = %s, want %s", got, "8")
	}
	if got := pkg.GetRating(); got != 4 {
		t.Errorf("GetRating() = %d, want %d", got, 4)
	}
}

func TestSetRating_UpdateBothForms_WhenBothExist(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.Version = "3.0"
	pkg.Metadata.Meta = []Meta{
		{Property: "calibre:rating", Value: "10"},
		{Name: "calibre:rating", Content: "10"},
	}

	pkg.SetRating(2) // 2 -> 4

	var prop, name string
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "calibre:rating" {
			prop = m.Value
		}
		if m.Name == "calibre:rating" {
			name = m.Content
		}
	}
	if prop != "4" {
		t.Errorf("Expected property rating '4', got '%s'", prop)
	}
	if name != "4" {
		t.Errorf("Expected name rating '4', got '%s'", name)
	}
}

func TestSetRating_ClampValues(t *testing.T) {
	pkg := createEmptyPackage()

	// Values outside 0-5 should be clamped
	pkg.SetRating(-1)
	if pkg.GetRating() != 0 {
		t.Errorf("Negative rating should clamp to 0, got %d", pkg.GetRating())
	}

	pkg.SetRating(10)
	if pkg.GetRating() != 5 {
		t.Errorf("Rating > 5 should clamp to 5, got %d", pkg.GetRating())
	}
}
