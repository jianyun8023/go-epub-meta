package epub

import (
	"fmt"
	"regexp"
	"strings"
)

// GetTitle returns the first title found.
func (pkg *Package) GetTitle() string {
	if len(pkg.Metadata.Titles) > 0 {
		return pkg.Metadata.Titles[0].Value
	}
	return ""
}

// SetTitle updates the title. It overwrites existing titles.
func (pkg *Package) SetTitle(title string) {
	pkg.Metadata.Titles = []SimpleMeta{{Value: title}}
}

// GetTitleSort returns the sortable title.
func (pkg *Package) GetTitleSort() string {
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:title_sort" {
			return m.Content
		}
	}
	return ""
}

// SetTitleSort sets the sortable title.
func (pkg *Package) SetTitleSort(sort string) {
	newMeta := []Meta{}
	found := false
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:title_sort" {
			m.Content = sort
			newMeta = append(newMeta, m)
			found = true
		} else {
			newMeta = append(newMeta, m)
		}
	}
	if !found {
		newMeta = append(newMeta, Meta{
			Name:    "calibre:title_sort",
			Content: sort,
		})
	}
	pkg.Metadata.Meta = newMeta
}

// GetAuthor returns the first creator.
func (pkg *Package) GetAuthor() string {
	if len(pkg.Metadata.Creators) > 0 {
		return pkg.Metadata.Creators[0].Value
	}
	return ""
}

// GetAuthors returns all creators as a list of author names.
// It handles multiple dc:creator elements and also attempts to split
// single creator values that contain multiple authors separated by
// common delimiters (&, 、, and, etc.).
func (pkg *Package) GetAuthors() []string {
	var authors []string
	for _, creator := range pkg.Metadata.Creators {
		parsed := parseAuthorString(creator.Value)
		authors = append(authors, parsed...)
	}
	// Deduplicate while preserving order
	return deduplicateStrings(authors)
}

// parseAuthorString attempts to split a single author string that may contain
// multiple authors separated by common delimiters.
// Safe delimiters (always split): &, 、, ;, " and "
// Unsafe delimiter (comma): only split if not "Last, First" format
func parseAuthorString(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Try safe delimiters first (these rarely appear in single author names)
	// Order matters: try longer patterns first
	safeDelimiters := []string{
		" & ",   // English ampersand with spaces
		"&",     // Ampersand without spaces
		"、",     // Chinese enumeration comma (顿号)
		" and ", // English "and" (case insensitive handled below)
		" AND ",
		"；", // Chinese semicolon
		";", // Semicolon
	}

	for _, delim := range safeDelimiters {
		if strings.Contains(s, delim) {
			parts := strings.Split(s, delim)
			var result []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					result = append(result, p)
				}
			}
			if len(result) > 1 {
				return result
			}
		}
	}

	// Handle comma carefully - avoid splitting "Last, First" format
	// Heuristic: if comma exists and there are more than 2 parts,
	// or if the parts don't look like "Last, First" (first part has space),
	// then it's likely multiple authors
	if strings.Contains(s, ",") || strings.Contains(s, "，") {
		// Use both English and Chinese comma
		normalized := strings.ReplaceAll(s, "，", ",")
		parts := strings.Split(normalized, ",")

		if len(parts) >= 2 {
			// Check if it looks like "Last, First" format (exactly 2 parts, no space in first part)
			firstPart := strings.TrimSpace(parts[0])
			secondPart := strings.TrimSpace(parts[1])

			// If first part contains space OR there are more than 2 parts,
			// treat as multiple authors
			isMultipleAuthors := len(parts) > 2 ||
				strings.Contains(firstPart, " ") ||
				(len(parts) == 2 && strings.Contains(secondPart, " ") && !strings.Contains(firstPart, " "))

			if isMultipleAuthors && len(parts) > 2 {
				var result []string
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						result = append(result, p)
					}
				}
				return result
			}
		}
	}

	// No splitting needed
	return []string{s}
}

// deduplicateStrings removes duplicates while preserving order.
func deduplicateStrings(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// GetAuthorSort returns the sortable author name from the first creator.
func (pkg *Package) GetAuthorSort() string {
	if len(pkg.Metadata.Creators) > 0 {
		return pkg.Metadata.Creators[0].FileAs
	}
	return ""
}

// SetAuthor sets the author.
func (pkg *Package) SetAuthor(name string) {
	// Standard practice: role="aut"
	pkg.Metadata.Creators = []AuthorMeta{{
		SimpleMeta: SimpleMeta{Value: name},
		Role:       "aut",
	}}
}

// GetDescription returns the description.
func (pkg *Package) GetDescription() string {
	if len(pkg.Metadata.Descriptions) > 0 {
		return pkg.Metadata.Descriptions[0].Value
	}
	return ""
}

// SetDescription sets the description.
func (pkg *Package) SetDescription(desc string) {
	pkg.Metadata.Descriptions = []SimpleMeta{{Value: desc}}
}

// GetLanguage returns the language.
func (pkg *Package) GetLanguage() string {
	if len(pkg.Metadata.Languages) > 0 {
		return pkg.Metadata.Languages[0].Value
	}
	return ""
}

// SetLanguage sets the language.
func (pkg *Package) SetLanguage(lang string) {
	pkg.Metadata.Languages = []SimpleMeta{{Value: lang}}
}

// GetSeries attempts to find the series name.
// It looks for Calibre meta tags for EPUB 2.
func (pkg *Package) GetSeries() string {
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series" {
			return m.Content
		}
		// Todo: EPUB 3 meta with collection property?
	}
	return ""
}

// GetSeriesIndex returns the series index.
func (pkg *Package) GetSeriesIndex() string {
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series_index" {
			return m.Content
		}
	}
	return ""
}

// SetSeries sets the series using Calibre meta tag.
func (pkg *Package) SetSeries(series string) {
	// Remove existing series tag
	newMeta := []Meta{}
	found := false
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series" {
			// Update existing
			m.Content = series
			newMeta = append(newMeta, m)
			found = true
		} else {
			newMeta = append(newMeta, m)
		}
	}

	if !found {
		newMeta = append(newMeta, Meta{
			Name:    "calibre:series",
			Content: series,
		})
	}
	pkg.Metadata.Meta = newMeta
}

// SetSeriesIndex sets the series index using Calibre meta tag.
func (pkg *Package) SetSeriesIndex(index string) {
	newMeta := []Meta{}
	found := false
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series_index" {
			m.Content = index
			newMeta = append(newMeta, m)
			found = true
		} else {
			newMeta = append(newMeta, m)
		}
	}
	if !found {
		newMeta = append(newMeta, Meta{
			Name:    "calibre:series_index",
			Content: index,
		})
	}
	pkg.Metadata.Meta = newMeta
}

// GetRating returns the book rating (0-5 scale, compatible with ebook-meta).
// Calibre stores rating as 0-10, this converts to 0-5.
func (pkg *Package) GetRating() int {
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:rating" {
			// Parse as integer, Calibre uses 0-10 scale
			var rating int
			if _, err := fmt.Sscanf(m.Content, "%d", &rating); err == nil {
				// Convert 0-10 to 0-5 (divide by 2)
				return rating / 2
			}
			// Try parsing as float (e.g., "8.0")
			var ratingFloat float64
			if _, err := fmt.Sscanf(m.Content, "%f", &ratingFloat); err == nil {
				return int(ratingFloat) / 2
			}
		}
	}
	return 0
}

// GetRatingRaw returns the raw rating value as stored (0-10 for Calibre).
func (pkg *Package) GetRatingRaw() string {
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:rating" {
			return m.Content
		}
	}
	return ""
}

// GetSubjects returns a list of tags.
func (pkg *Package) GetSubjects() []string {
	var subjects []string
	for _, s := range pkg.Metadata.Subjects {
		subjects = append(subjects, s.Value)
	}
	return subjects
}

// SetSubjects overwrites tags.
func (pkg *Package) SetSubjects(tags []string) {
	var newSubjects []SimpleMeta
	for _, t := range tags {
		newSubjects = append(newSubjects, SimpleMeta{Value: t})
	}
	pkg.Metadata.Subjects = newSubjects
}

// GetIdentifiers returns all identifiers (ISBN, ASIN, etc.) as a map.
// The key is the scheme (e.g., "isbn", "asin"), and the value is the identifier value.
// UUID identifiers are filtered out as they are typically not user-relevant.
func (pkg *Package) GetIdentifiers() map[string]string {
	result := make(map[string]string)
	for _, id := range pkg.Metadata.Identifiers {
		scheme, value := parseIdentifier(id.Scheme, id.Value)
		// Skip UUID identifiers (not useful for users)
		if scheme != "" && scheme != "uuid" && scheme != "unknown" {
			result[scheme] = value
		}
	}
	return result
}

// parseIdentifier extracts the scheme and value from an identifier.
// It handles various formats:
// - URN notation: "urn:isbn:9780123456789"
// - Direct scheme:value notation: "isbn:9780123456789"
// - Calibre format: "calibre:12345"
func parseIdentifier(scheme, value string) (string, string) {
	// If scheme is already set and valid, use it
	if scheme != "" && scheme != "unknown" {
		return normalizeScheme(scheme), value
	}

	// Try to parse from URN format (e.g., "urn:isbn:9780123456789")
	if strings.HasPrefix(strings.ToLower(value), "urn:") {
		parts := strings.SplitN(value, ":", 3)
		if len(parts) >= 3 {
			extractedScheme := normalizeScheme(parts[1])
			extractedValue := parts[2]
			// Skip UUID
			if extractedScheme == "uuid" {
				return "uuid", extractedValue
			}
			return extractedScheme, extractedValue
		}
		// Handle urn:uuid format specifically (only 2 parts)
		if len(parts) == 2 && normalizeScheme(parts[1]) == "uuid" {
			return "uuid", value
		}
	}

	// Try to parse from "scheme:value" format (e.g., "isbn:9780123456789", "calibre:12345")
	if strings.Contains(value, ":") && !strings.HasPrefix(strings.ToLower(value), "uuid:") {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) == 2 {
			extractedScheme := normalizeScheme(parts[0])
			extractedValue := parts[1]
			// Skip UUID
			if extractedScheme == "uuid" {
				return "uuid", extractedValue
			}
			// Valid scheme found
			if extractedScheme != "" && extractedScheme != "unknown" {
				return extractedScheme, extractedValue
			}
		}
	}

	// Handle uuid: prefix specifically
	if strings.HasPrefix(strings.ToLower(value), "uuid:") {
		return "uuid", strings.TrimPrefix(value, "uuid:")
	}

	// Heuristic: Check if value looks like an ISBN
	if isISBN(value) {
		return "isbn", value
	}

	// If we can't determine the scheme, return unknown
	return "unknown", value
}

// normalizeScheme converts scheme to lowercase and handles common variations.
func normalizeScheme(scheme string) string {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	// Map common variations to standard forms
	switch scheme {
	case "isbn", "urn:isbn":
		return "isbn"
	case "asin":
		return "asin"
	case "mobi-asin":
		return "mobi-asin"
	case "uuid", "urn:uuid":
		return "uuid"
	case "calibre":
		return "calibre"
	case "douban", "new_douban":
		return scheme
	case "amazon_cn", "google", "doi":
		return scheme
	}
	return scheme
}

// isISBN checks if a string looks like an ISBN (ISBN-10 or ISBN-13).
func isISBN(s string) bool {
	// Remove common separators
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")

	// ISBN-10: 10 digits (last may be X)
	// ISBN-13: 13 digits starting with 978 or 979
	if len(s) == 10 {
		matched, _ := regexp.MatchString(`^\d{9}[\dXx]$`, s)
		return matched
	}
	if len(s) == 13 {
		matched, _ := regexp.MatchString(`^(978|979)\d{10}$`, s)
		return matched
	}
	return false
}

// GetISBN returns the ISBN identifier if it exists.
func (pkg *Package) GetISBN() string {
	for _, id := range pkg.Metadata.Identifiers {
		if id.Scheme == "ISBN" || id.Scheme == "isbn" {
			return id.Value
		}
	}
	return ""
}

// GetASIN returns the ASIN identifier if it exists.
func (pkg *Package) GetASIN() string {
	for _, id := range pkg.Metadata.Identifiers {
		if id.Scheme == "ASIN" || id.Scheme == "asin" {
			return id.Value
		}
	}
	return ""
}

// SetIdentifier sets or updates an identifier with the given scheme.
// Common schemes: "ISBN", "ASIN", "DOI", "UUID", etc.
func (pkg *Package) SetIdentifier(scheme, value string) {
	// Find and update existing identifier with the same scheme
	for i, id := range pkg.Metadata.Identifiers {
		if id.Scheme == scheme {
			pkg.Metadata.Identifiers[i].Value = value
			return
		}
	}

	// Add new identifier if not found
	pkg.Metadata.Identifiers = append(pkg.Metadata.Identifiers, IDMeta{
		Scheme: scheme,
		Value:  value,
	})
}

// SetISBN sets the ISBN identifier.
func (pkg *Package) SetISBN(isbn string) {
	pkg.SetIdentifier("ISBN", isbn)
}

// SetASIN sets the ASIN identifier.
func (pkg *Package) SetASIN(asin string) {
	pkg.SetIdentifier("ASIN", asin)
}

// GetPublisher returns the publisher name.
func (pkg *Package) GetPublisher() string {
	if len(pkg.Metadata.Publishers) > 0 {
		return pkg.Metadata.Publishers[0].Value
	}
	return ""
}

// GetProducer returns the book producer (generator).
func (pkg *Package) GetProducer() string {
	// Check contributors with role 'bkp'
	for _, c := range pkg.Metadata.Contributors {
		if c.Role == "bkp" {
			return c.Value
		}
	}
	// Check meta generator
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "generator" {
			return m.Content
		}
	}
	return ""
}

// SetPublisher sets the publisher name.
func (pkg *Package) SetPublisher(publisher string) {
	pkg.Metadata.Publishers = []SimpleMeta{{Value: publisher}}
}

// GetPublishDate returns the publication date.
// It looks for dates in order of preference and returns the first one found.
func (pkg *Package) GetPublishDate() string {
	if len(pkg.Metadata.Dates) > 0 {
		// Return the first date found
		// In EPUB 2/3, dates can have opf:event attributes to distinguish types
		// but for simplicity, we return the first date
		return pkg.Metadata.Dates[0].Value
	}
	return ""
}

// SetPublishDate sets the publication date.
func (pkg *Package) SetPublishDate(date string) {
	pkg.Metadata.Dates = []SimpleMeta{{Value: date}}
}
