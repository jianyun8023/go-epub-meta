package commands

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// EbookMetadata represents the metadata fields parsed from ebook-meta output
type EbookMetadata struct {
	Title       string
	Authors     string
	Publisher   string
	Published   string
	Language    string
	Tags        string
	Comments    string
	Identifiers string
	Series      string
	SeriesIndex string
}

// StripSortSuffix removes [sort] suffix from a value
// e.g., "余秋雨 [余秋雨]" -> "余秋雨"
func StripSortSuffix(value string) string {
	re := regexp.MustCompile(`\s*\[.*?\]`)
	return strings.TrimSpace(re.ReplaceAllString(value, ""))
}

// ParseEbookMetaOutput parses ebook-meta text output into EbookMetadata
func ParseEbookMetaOutput(output string) EbookMetadata {
	meta := EbookMetadata{}
	sc := bufio.NewScanner(strings.NewReader(output))
	var currentKey string

	fieldMap := map[string]string{
		"Title":            "title",
		"Author(s)":        "authors",
		"Authors":          "authors",
		"Author":           "authors",
		"Languages":        "language",
		"Language":         "language",
		"Publisher":        "publisher",
		"Published":        "published",
		"Publication Date": "published",
		"Series":           "series",
		"Identifiers":      "identifiers",
		"Identifier":       "identifiers",
		"Tags":             "tags",
		"Subjects":         "tags",
		"Comments":         "comments",
		"Comment":          "comments",
		"Description":      "comments",
	}

	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			// Multi-line continuation (commonly Comments)
			if currentKey == "comments" && (strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t")) {
				meta.Comments = strings.TrimSpace(meta.Comments + "\n" + line)
			}
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if fieldName, ok := fieldMap[key]; ok {
			currentKey = fieldName
			switch fieldName {
			case "title":
				meta.Title = value
			case "authors":
				meta.Authors = StripSortSuffix(value)
			case "publisher":
				meta.Publisher = value
			case "published":
				meta.Published = value
			case "language":
				meta.Language = value
			case "tags":
				meta.Tags = value
			case "comments":
				meta.Comments = value
			case "identifiers":
				meta.Identifiers = value
			case "series":
				meta.Series = value
				if strings.Contains(value, "#") {
					parts := strings.SplitN(value, "#", 2)
					meta.Series = strings.TrimSpace(parts[0])
					if len(parts) > 1 {
						meta.SeriesIndex = strings.TrimSpace(parts[1])
					}
				}
			}
		} else {
			currentKey = ""
		}
	}
	return meta
}

// FindEpubFiles recursively finds all .epub files in a directory
func FindEpubFiles(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // Gracefully handle non-existent root
	}
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".epub") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// NormalizeLanguage maps common language codes to 3-letter codes
func NormalizeLanguage(lang string) string {
	// Simple mapping for common codes to match ebook-meta 3-letter codes
	// ebook-meta usually outputs ISO 639-2
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "zh", "zh-cn", "zh-sg", "zh-hans":
		return "zho"
	case "zh-tw", "zh-hk", "zh-hant":
		return "zho"
	case "en", "en-us", "en-gb":
		return "eng"
	case "ja", "jp":
		return "jpn"
	case "fr":
		return "fra"
	case "de":
		return "deu"
	case "es":
		return "spa"
	case "it":
		return "ita"
	case "ru":
		return "rus"
	}
	// Return as is if no match or already 3 letters
	return lang
}

// NormalizeDate extracts YYYY-MM-DD from various date formats
func NormalizeDate(d string) string {
	// Try parsing as RFC3339 (e.g., 2023-12-31T16:00:00+00:00)
	if t, err := time.Parse(time.RFC3339, d); err == nil {
		// Just extract the date part in the original timezone
		return t.Format("2006-01-02")
	}

	// Fallback: Extract YYYY-MM-DD
	if len(d) >= 10 {
		return d[:10]
	}
	return d
}

// NormalizeIdentifiers normalizes identifier strings for comparison
func NormalizeIdentifiers(ids string) string {
	// Split by comma, trim, filter calibre UUID, sort, rejoin
	parts := strings.Split(ids, ",")
	var filtered []string
	for i := range parts {
		part := strings.ToLower(strings.TrimSpace(parts[i]))
		// Skip calibre UUID (e.g., "calibre:e341dd8c-...") and generic UUID ("uuid:...")
		if strings.HasPrefix(part, "calibre:") || strings.HasPrefix(part, "uuid:") {
			continue
		}
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	sort.Strings(filtered)
	return strings.Join(filtered, ",")
}
