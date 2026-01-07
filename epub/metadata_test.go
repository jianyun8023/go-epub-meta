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

func TestGetTitleSort(t *testing.T) {
	pkg := createTestPackage()
	titleSort := pkg.GetTitleSort()
	if titleSort != "Title, Test" {
		t.Errorf("Expected 'Title, Test', got '%s'", titleSort)
	}
}

func TestGetTitleSortEmpty(t *testing.T) {
	pkg := createEmptyPackage()
	titleSort := pkg.GetTitleSort()
	if titleSort != "" {
		t.Errorf("Expected empty string, got '%s'", titleSort)
	}
}

func TestSetTitleSort(t *testing.T) {
	pkg := createEmptyPackage()
	pkg.SetTitleSort("Sort, Title")
	if pkg.GetTitleSort() != "Sort, Title" {
		t.Errorf("Expected 'Sort, Title', got '%s'", pkg.GetTitleSort())
	}
}

func TestSetTitleSortUpdate(t *testing.T) {
	pkg := createTestPackage()
	pkg.SetTitleSort("New Sort Title")
	if pkg.GetTitleSort() != "New Sort Title" {
		t.Errorf("Expected 'New Sort Title', got '%s'", pkg.GetTitleSort())
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
	if lang != "" {
		t.Errorf("Expected empty string, got '%s'", lang)
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
		{"123456789", false},  // Too short
		{"12345678901", false}, // Too long
		{"012345678A", false}, // Invalid char
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
		{"1234567890123", false}, // Doesn't start with 978/979
		{"978012345678", false},  // Too short
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
