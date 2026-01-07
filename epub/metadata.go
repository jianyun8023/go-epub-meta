package epub

import (
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

// GetAuthor returns the first creator.
func (pkg *Package) GetAuthor() string {
	if len(pkg.Metadata.Creators) > 0 {
		return pkg.Metadata.Creators[0].Value
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
