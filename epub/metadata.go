package epub

import (
	"fmt"
	"regexp"
	"strings"
)

func (pkg *Package) isEPUB3() bool {
	// Declared version is the most stable signal.
	// Keep it conservative: only treat "3.*" as EPUB3.
	v := strings.TrimSpace(pkg.Version)
	return strings.HasPrefix(v, "3")
}

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
		// Only add legacy tag if not EPUB3 or if strictly required.
		// For EPUB3, we only maintain if present (which is handled above).
		if !pkg.isEPUB3() {
			newMeta = append(newMeta, Meta{
				Name:    "calibre:title_sort",
				Content: sort,
			})
		}
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
		if v := pkg.Metadata.Languages[0].Value; v != "" {
			return v
		}
	}
	// EPUB language is a BCP47 tag. If missing, use "und" (undetermined).
	// This keeps JSON output consistent and avoids omitting the field.
	return "und"
}

// SetLanguage sets the language.
func (pkg *Package) SetLanguage(lang string) {
	pkg.Metadata.Languages = []SimpleMeta{{Value: lang}}
}

// GetSeries attempts to find the series name.
// It looks for Calibre meta tags for EPUB 2.
func (pkg *Package) GetSeries() string {
	// EPUB 3: <meta property="belongs-to-collection" id="c01">Series</meta>
	//        <meta refines="#c01" property="collection-type">series</meta>
	// See: https://www.w3.org/publishing/epub32/epub-packages.html#sec-metadata-collection
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "belongs-to-collection" && m.Value != "" && m.ID != "" {
			id := "#" + m.ID
			for _, refine := range pkg.Metadata.Meta {
				if refine.Refines == id && refine.Property == "collection-type" && strings.TrimSpace(refine.Value) == "series" {
					return m.Value
				}
			}
		}
	}

	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series" {
			return m.Content
		}
	}
	return ""
}

// GetSeriesIndex returns the series index.
func (pkg *Package) GetSeriesIndex() string {
	// EPUB 3: <meta refines="#c01" property="group-position">3</meta>
	// Note: We don't require m.Value != "" here because series index can exist
	// even when series name is empty (edge case, but valid EPUB3 structure).
	for _, m := range pkg.Metadata.Meta {
		if m.Property == "belongs-to-collection" && m.ID != "" {
			id := "#" + m.ID
			for _, refine := range pkg.Metadata.Meta {
				if refine.Refines == id && refine.Property == "collection-type" && strings.TrimSpace(refine.Value) == "series" {
					// Now try group-position on same refines id
					for _, gp := range pkg.Metadata.Meta {
						if gp.Refines == id && gp.Property == "group-position" {
							return gp.Value
						}
					}
				}
			}
		}
	}

	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series_index" {
			return m.Content
		}
	}
	return ""
}

// SetSeries sets the series using Calibre meta tag.
func (pkg *Package) SetSeries(series string) {
	// EPUB 3: write standards-based collection/series metadata.
	// If declared EPUB3, prefer EPUB3-style meta instead of calibre:* tags.
	if pkg.isEPUB3() {
		// Try to find existing series collection
		var collectionID string
		for _, m := range pkg.Metadata.Meta {
			if m.Property == "belongs-to-collection" && m.ID != "" {
				id := "#" + m.ID
				for _, refine := range pkg.Metadata.Meta {
					if refine.Refines == id && refine.Property == "collection-type" && strings.TrimSpace(refine.Value) == "series" {
						collectionID = m.ID
						break
					}
				}
			}
			if collectionID != "" {
				break
			}
		}

		if collectionID == "" {
			// Generate deterministic id: c01, c02, ...
			used := make(map[string]bool)
			for _, m := range pkg.Metadata.Meta {
				if m.ID != "" {
					used[m.ID] = true
				}
			}
			for i := 1; ; i++ {
				candidate := fmt.Sprintf("c%02d", i)
				if !used[candidate] {
					collectionID = candidate
					break
				}
			}
			// Add belongs-to-collection
			pkg.Metadata.Meta = append(pkg.Metadata.Meta, Meta{
				ID:       collectionID,
				Property: "belongs-to-collection",
				Value:    series,
			})
			// Add collection-type=series
			pkg.Metadata.Meta = append(pkg.Metadata.Meta, Meta{
				Refines:  "#" + collectionID,
				Property: "collection-type",
				Value:    "series",
			})
		} else {
			// Update existing belongs-to-collection value
			for i := range pkg.Metadata.Meta {
				if pkg.Metadata.Meta[i].Property == "belongs-to-collection" && pkg.Metadata.Meta[i].ID == collectionID {
					pkg.Metadata.Meta[i].Value = series
					break
				}
			}
			// Ensure collection-type=series exists
			foundType := false
			for i := range pkg.Metadata.Meta {
				if pkg.Metadata.Meta[i].Refines == "#"+collectionID && pkg.Metadata.Meta[i].Property == "collection-type" {
					pkg.Metadata.Meta[i].Value = "series"
					foundType = true
					break
				}
			}
			if !foundType {
				pkg.Metadata.Meta = append(pkg.Metadata.Meta, Meta{
					Refines:  "#" + collectionID,
					Property: "collection-type",
					Value:    "series",
				})
			}
		}

		// Compatibility: Also update legacy calibre:series IF IT EXISTS
		for i := range pkg.Metadata.Meta {
			if pkg.Metadata.Meta[i].Name == "calibre:series" {
				pkg.Metadata.Meta[i].Content = series
			}
		}
		return
	}

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
	// EPUB 3: write group-position on the series collection.
	if pkg.isEPUB3() {
		// Ensure series exists first (creates collection-type too)
		if pkg.GetSeries() == "" {
			pkg.SetSeries("") // creates a series collection with empty value
		}

		// Find series collection id
		var collectionID string
		for _, m := range pkg.Metadata.Meta {
			if m.Property == "belongs-to-collection" && m.ID != "" {
				id := "#" + m.ID
				for _, refine := range pkg.Metadata.Meta {
					if refine.Refines == id && refine.Property == "collection-type" && strings.TrimSpace(refine.Value) == "series" {
						collectionID = m.ID
						break
					}
				}
			}
			if collectionID != "" {
				break
			}
		}
		if collectionID == "" {
			// Fallback: behave like EPUB2 if we couldn't establish a collection.
		} else {
			// Update or add group-position
			foundPos := false
			for i := range pkg.Metadata.Meta {
				if pkg.Metadata.Meta[i].Refines == "#"+collectionID && pkg.Metadata.Meta[i].Property == "group-position" {
					pkg.Metadata.Meta[i].Value = index
					foundPos = true
					break
				}
			}
			if !foundPos {
				pkg.Metadata.Meta = append(pkg.Metadata.Meta, Meta{
					Refines:  "#" + collectionID,
					Property: "group-position",
					Value:    index,
				})
			}
		}

		// Compatibility: Also update legacy calibre:series_index IF IT EXISTS
		for i := range pkg.Metadata.Meta {
			if pkg.Metadata.Meta[i].Name == "calibre:series_index" {
				pkg.Metadata.Meta[i].Content = index
			}
		}
		return
	}

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
		if m.Property == "calibre:rating" && strings.TrimSpace(m.Value) != "" {
			// Parse as integer, Calibre uses 0-10 scale
			var rating int
			if _, err := fmt.Sscanf(m.Value, "%d", &rating); err == nil {
				return rating / 2
			}
			var ratingFloat float64
			if _, err := fmt.Sscanf(m.Value, "%f", &ratingFloat); err == nil {
				return int(ratingFloat) / 2
			}
		}
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
		if m.Property == "calibre:rating" && strings.TrimSpace(m.Value) != "" {
			return m.Value
		}
		if m.Name == "calibre:rating" {
			return m.Content
		}
	}
	return ""
}

// SetRating sets the rating (0-5 scale, converted to Calibre's 0-10 scale).
// Values outside 0-5 are clamped.
func (pkg *Package) SetRating(rating int) {
	// Clamp to 0-5 range
	if rating < 0 {
		rating = 0
	}
	if rating > 5 {
		rating = 5
	}

	// Convert to Calibre's 0-10 scale
	calibreRating := fmt.Sprintf("%d", rating*2)

	// Update or add calibre:rating meta
	newMeta := []Meta{}
	found := false
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:rating" {
			m.Content = calibreRating
			found = true
		}
		if m.Property == "calibre:rating" {
			m.Value = calibreRating
			found = true
		}
		newMeta = append(newMeta, m)
	}
	if !found {
		if pkg.isEPUB3() {
			newMeta = append(newMeta, Meta{Property: "calibre:rating", Value: calibreRating})
		} else {
			newMeta = append(newMeta, Meta{Name: "calibre:rating", Content: calibreRating})
		}
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
		scheme, value := parseIdentifier(id.Scheme, id.Value)
		if scheme == "isbn" {
			return value
		}
	}
	return ""
}

// GetASIN returns the ASIN identifier if it exists.
func (pkg *Package) GetASIN() string {
	for _, id := range pkg.Metadata.Identifiers {
		scheme, value := parseIdentifier(id.Scheme, id.Value)
		if scheme == "asin" || scheme == "mobi-asin" {
			return value
		}
	}
	return ""
}

// SetIdentifier sets or updates an identifier with the given scheme.
// Common schemes: "ISBN", "ASIN", "DOI", "UUID", etc.
func (pkg *Package) SetIdentifier(scheme, value string) {
	// Find and update existing identifier with the same scheme
	for i, id := range pkg.Metadata.Identifiers {
		if normalizeScheme(id.Scheme) == normalizeScheme(scheme) {
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
	trimmed := strings.TrimSpace(isbn)
	clean := strings.TrimSpace(isbn)
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, " ", "")
	inputHasSep := strings.Contains(trimmed, "-") || strings.Contains(trimmed, " ")

	// Respect original format: update existing ISBN-like identifiers in-place.
	found := false
	for i := range pkg.Metadata.Identifiers {
		parsedScheme, _ := parseIdentifier(pkg.Metadata.Identifiers[i].Scheme, pkg.Metadata.Identifiers[i].Value)
		if parsedScheme != "isbn" {
			continue
		}

		raw := strings.TrimSpace(pkg.Metadata.Identifiers[i].Value)
		lower := strings.ToLower(raw)
		switch {
		case strings.HasPrefix(lower, "urn:isbn:"):
			// Standards-friendly (EPUB3-preferred) URN form.
			pkg.Metadata.Identifiers[i].Scheme = ""
			pkg.Metadata.Identifiers[i].Value = "urn:isbn:" + clean
		case strings.HasPrefix(lower, "isbn:"):
			// "isbn:" prefix inside value is a common but non-standard EPUB2 pattern.
			// Normalize to EPUB2/3 standard representation based on declared version.
			if pkg.isEPUB3() {
				pkg.Metadata.Identifiers[i].Scheme = ""
				pkg.Metadata.Identifiers[i].Value = "urn:isbn:" + clean
			} else {
				pkg.Metadata.Identifiers[i].Scheme = "ISBN"
				if inputHasSep {
					pkg.Metadata.Identifiers[i].Value = trimmed
				} else {
					pkg.Metadata.Identifiers[i].Value = clean
				}
			}
		default:
			if pkg.isEPUB3() {
				// EPUB3: prefer URN form.
				pkg.Metadata.Identifiers[i].Scheme = ""
				pkg.Metadata.Identifiers[i].Value = "urn:isbn:" + clean
			} else {
				// EPUB2: prefer opf:scheme="ISBN" + value (preserve separators when possible).
				pkg.Metadata.Identifiers[i].Scheme = "ISBN"
				if inputHasSep {
					pkg.Metadata.Identifiers[i].Value = trimmed
				} else if strings.Contains(raw, "-") || strings.Contains(raw, " ") {
					if replaced, ok := replaceDigitsPreserveSeparators(raw, clean); ok {
						pkg.Metadata.Identifiers[i].Value = replaced
					} else {
						pkg.Metadata.Identifiers[i].Value = clean
					}
				} else {
					pkg.Metadata.Identifiers[i].Value = clean
				}
			}
		}
		found = true
	}

	if found {
		return
	}

	// If missing, add in a version-appropriate and minimally-invasive form:
	// - EPUB2 commonly uses opf:scheme="ISBN" + plain value
	// - EPUB3 often prefers URN form (urn:isbn:...) without opf:scheme
	if pkg.isEPUB3() {
		pkg.Metadata.Identifiers = append(pkg.Metadata.Identifiers, IDMeta{Value: "urn:isbn:" + clean})
		return
	}
	if inputHasSep {
		pkg.Metadata.Identifiers = append(pkg.Metadata.Identifiers, IDMeta{Scheme: "ISBN", Value: trimmed})
		return
	}
	pkg.Metadata.Identifiers = append(pkg.Metadata.Identifiers, IDMeta{Scheme: "ISBN", Value: clean})
}

func replaceDigitsPreserveSeparators(template, digits string) (string, bool) {
	// Count digit-like chars (0-9, X/x) in template and ensure it matches.
	count := 0
	for _, r := range template {
		if (r >= '0' && r <= '9') || r == 'X' || r == 'x' {
			count++
		}
	}
	if count != len(digits) {
		return "", false
	}

	out := make([]rune, 0, len(template))
	j := 0
	for _, r := range template {
		if (r >= '0' && r <= '9') || r == 'X' || r == 'x' {
			out = append(out, rune(digits[j]))
			j++
			continue
		}
		out = append(out, r)
	}
	return string(out), true
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
