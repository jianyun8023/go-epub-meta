package commands

import (
	"encoding/json"
	"fmt"
	"golibri/epub"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	metaTitle       string
	metaAuthor      string
	metaSeries      string
	metaCover       string
	metaOutput      string
	metaISBN        string
	metaASIN        string
	metaIdentifiers []string
	metaJSON        bool
	metaGetCover    string // Export cover to file
	// New write flags
	metaPublisher   string
	metaDate        string
	metaLanguage    string
	metaTags        string
	metaComments    string
	metaSeriesIndex string
	metaRating      int
)

func init() {
	metaCmd.Flags().StringVarP(&metaTitle, "title", "t", "", "Set title")
	metaCmd.Flags().StringVarP(&metaAuthor, "author", "a", "", "Set author")
	metaCmd.Flags().StringVarP(&metaSeries, "series", "s", "", "Set series")
	metaCmd.Flags().StringVarP(&metaCover, "cover", "c", "", "Set cover image path")
	metaCmd.Flags().StringVar(&metaISBN, "isbn", "", "Set ISBN identifier")
	metaCmd.Flags().StringVar(&metaASIN, "asin", "", "Set ASIN identifier")
	metaCmd.Flags().StringArrayVarP(&metaIdentifiers, "identifier", "i", []string{}, "Set identifier (format: scheme:value, e.g., douban:12345678)")
	metaCmd.Flags().StringVarP(&metaOutput, "output", "o", "", "Output file path (default: modify in-place)")
	metaCmd.Flags().BoolVar(&metaJSON, "json", false, "Output metadata in JSON format (compatible with ebook-meta)")
	metaCmd.Flags().StringVar(&metaGetCover, "get-cover", "", "Export cover image to specified file")
	// New write flags
	metaCmd.Flags().StringVar(&metaPublisher, "publisher", "", "Set publisher")
	metaCmd.Flags().StringVar(&metaDate, "date", "", "Set publication date")
	metaCmd.Flags().StringVar(&metaLanguage, "language", "", "Set language (e.g., zh, en, zh-CN)")
	metaCmd.Flags().StringVar(&metaTags, "tags", "", "Set tags/subjects (comma-separated)")
	metaCmd.Flags().StringVar(&metaComments, "comments", "", "Set description/comments")
	metaCmd.Flags().StringVar(&metaSeriesIndex, "series-index", "", "Set series index")
	metaCmd.Flags().IntVar(&metaRating, "rating", -1, "Set rating (0-5, Calibre extension)")

	rootCmd.AddCommand(metaCmd)
}

var metaCmd = &cobra.Command{
	Use:   "meta [flags] input.epub",
	Short: "Read or modify EPUB metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		ep, err := epub.Open(inputFile)
		if err != nil {
			fmt.Printf("Error opening %s: %v\n", inputFile, err)
			os.Exit(1)
		}
		defer ep.Close()

		// Extract cover mode
		if metaGetCover != "" {
			if err := extractCover(ep, metaGetCover); err != nil {
				fmt.Printf("Error extracting cover: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Cover exported to %s\n", metaGetCover)
			return
		}

		// Read Mode - check if any write flag is set
		isWriteMode := metaTitle != "" || metaAuthor != "" || metaSeries != "" || metaCover != "" ||
			metaISBN != "" || metaASIN != "" || len(metaIdentifiers) > 0 ||
			metaPublisher != "" || metaDate != "" || metaLanguage != "" ||
			metaTags != "" || metaComments != "" || metaSeriesIndex != "" || metaRating >= 0
		if !isWriteMode {
			if metaJSON {
				printMetadataJSON(ep)
			} else {
				printMetadata(ep)
			}
			return
		}

		// Write Mode
		// If no output specified, modify in-place
		outputPath := metaOutput
		if outputPath == "" {
			outputPath = inputFile
		}

		if err := applyChanges(ep); err != nil {
			fmt.Printf("Error applying changes: %v\n", err)
			os.Exit(1)
		}

		if err := ep.Save(outputPath); err != nil {
			fmt.Printf("Error saving EPUB: %v\n", err)
			os.Exit(1)
		}

		if metaJSON {
			// If JSON requested after write, we should probably output the NEW metadata
			// Re-opening might be expensive, so we just use the current state since applyChanges updated it.
			printMetadataJSON(ep)
		} else {
			fmt.Printf("Saved to %s\n", outputPath)
		}
	},
}

// MetadataJSON represents the JSON output format compatible with ebook-meta
type MetadataJSON struct {
	Title       string            `json:"title"`
	Authors     []string          `json:"authors"`
	Publisher   string            `json:"publisher,omitempty"`
	Published   string            `json:"published,omitempty"`
	Language    string            `json:"language,omitempty"`
	Series      string            `json:"series,omitempty"`
	SeriesIndex string            `json:"series_index,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Rating      int               `json:"rating,omitempty"`
	Identifiers map[string]string `json:"identifiers"`
	Producer    string            `json:"producer,omitempty"`
	Comments    string            `json:"comments,omitempty"`
	Cover       bool              `json:"cover"`
}

func printMetadataJSON(ep *epub.Reader) {
	meta := MetadataJSON{
		Title:       ep.Package.GetTitle(),
		Authors:     ep.Package.GetAuthors(),
		Publisher:   ep.Package.GetPublisher(),
		Published:   ep.Package.GetPublishDate(),
		Language:    ep.Package.GetLanguage(),
		Series:      ep.Package.GetSeries(),
		SeriesIndex: ep.Package.GetSeriesIndex(),
		Tags:        ep.Package.GetSubjects(),
		Rating:      ep.Package.GetRating(),
		Identifiers: ep.Package.GetIdentifiers(),
		Producer:    ep.Package.GetProducer(),
		Comments:    ep.Package.GetDescription(),
		Cover:       false,
	}

	// Ensure Authors is not nil for JSON output
	if meta.Authors == nil {
		meta.Authors = []string{}
	}

	// Check cover
	_, _, err := ep.GetCoverImage()
	if err == nil {
		meta.Cover = true
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(meta); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

func printMetadata(ep *epub.Reader) {
	fmt.Println("--- Metadata ---")
	fmt.Printf("Title:       %s\n", ep.Package.GetTitle())

	// Display authors (joined with ", " on a single line)
	authors := ep.Package.GetAuthors()
	if len(authors) > 0 {
		fmt.Printf("Author:      %s\n", strings.Join(authors, ", "))
	} else {
		fmt.Printf("Author:      \n")
	}

	// Display publisher if available
	if publisher := ep.Package.GetPublisher(); publisher != "" {
		fmt.Printf("Publisher:   %s\n", publisher)
	}

	// Display publication date if available
	if published := ep.Package.GetPublishDate(); published != "" {
		fmt.Printf("Published:   %s\n", published)
	}

	fmt.Printf("Language:    %s\n", ep.Package.GetLanguage())

	// Display series with index if available (format: "Series Name #1")
	if series := ep.Package.GetSeries(); series != "" {
		if idx := ep.Package.GetSeriesIndex(); idx != "" {
			fmt.Printf("Series:      %s #%s\n", series, idx)
		} else {
			fmt.Printf("Series:      %s\n", series)
		}
	}

	// Display tags/subjects if available
	if tags := ep.Package.GetSubjects(); len(tags) > 0 {
		fmt.Printf("Tags:        %s\n", strings.Join(tags, ", "))
	}

	// Display rating if available (0-5 scale)
	if rating := ep.Package.GetRating(); rating > 0 {
		fmt.Printf("Rating:      %d\n", rating)
	}

	// Display identifiers (ISBN, ASIN, etc.)
	identifiers := ep.Package.GetIdentifiers()
	if len(identifiers) > 0 {
		fmt.Print("Identifiers: ")
		first := true
		// Display in a consistent order: isbn, asin, then others
		displayOrder := []string{"isbn", "asin", "mobi-asin"}
		for _, scheme := range displayOrder {
			if value, ok := identifiers[scheme]; ok {
				if !first {
					fmt.Print(", ")
				}
				fmt.Printf("%s:%s", scheme, value)
				first = false
				delete(identifiers, scheme) // Remove to avoid duplicate display
			}
		}
		// Display remaining identifiers
		for scheme, value := range identifiers {
			if !first {
				fmt.Print(", ")
			}
			fmt.Printf("%s:%s", scheme, value)
			first = false
		}
		fmt.Println()
	}

	// Display producer/generator if available
	if producer := ep.Package.GetProducer(); producer != "" {
		fmt.Printf("Producer:    %s\n", producer)
	}

	// Display description/comments if available (strip HTML, truncate for readability)
	if desc := ep.Package.GetDescription(); desc != "" {
		plainDesc := stripHTML(desc)
		if plainDesc != "" {
			fmt.Printf("Comments:    %s\n", truncate(plainDesc, 200))
		}
	}

	_, _, err := ep.GetCoverImage()
	if err == nil {
		fmt.Println("Cover:       Found")
	} else {
		fmt.Println("Cover:       Not Found")
	}
}

// stripHTML removes HTML tags from a string for plain text display.
func stripHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, " ")
	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	// Normalize whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// truncate truncates a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// extractCover extracts the cover image from EPUB and saves it to the specified file.
func extractCover(ep *epub.Reader, outputPath string) error {
	rc, _, err := ep.GetCoverImage()
	if err != nil {
		return fmt.Errorf("no cover found in EPUB: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("failed to read cover data: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cover to %s: %w", outputPath, err)
	}

	return nil
}

func applyChanges(ep *epub.Reader) error {
	if metaTitle != "" {
		ep.Package.SetTitle(metaTitle)
	}
	if metaAuthor != "" {
		ep.Package.SetAuthor(metaAuthor)
	}
	if metaSeries != "" {
		ep.Package.SetSeries(metaSeries)
	}
	if metaISBN != "" {
		ep.Package.SetISBN(metaISBN)
	}
	if metaASIN != "" {
		ep.Package.SetASIN(metaASIN)
	}

	// Handle custom identifiers (format: scheme:value)
	for _, id := range metaIdentifiers {
		parts := strings.SplitN(id, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid identifier format '%s', expected 'scheme:value' (e.g., douban:12345678)", id)
		}
		scheme := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if scheme == "" || value == "" {
			return fmt.Errorf("invalid identifier '%s', both scheme and value must be non-empty", id)
		}
		ep.Package.SetIdentifier(scheme, value)
	}

	if metaCover != "" {
		f, err := os.Open(metaCover)
		if err != nil {
			return fmt.Errorf("error opening cover %s: %w", metaCover, err)
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("error reading cover: %w", err)
		}

		// Detect mime
		mime := "image/jpeg"
		if strings.HasSuffix(strings.ToLower(metaCover), ".png") {
			mime = "image/png"
		}

		ep.SetCover(data, mime)
	}

	// New write fields
	if metaPublisher != "" {
		ep.Package.SetPublisher(metaPublisher)
	}
	if metaDate != "" {
		ep.Package.SetPublishDate(metaDate)
	}
	if metaLanguage != "" {
		ep.Package.SetLanguage(metaLanguage)
	}
	if metaTags != "" {
		// Parse comma-separated tags
		tags := strings.Split(metaTags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		ep.Package.SetSubjects(tags)
	}
	if metaComments != "" {
		ep.Package.SetDescription(metaComments)
	}
	if metaSeriesIndex != "" {
		ep.Package.SetSeriesIndex(metaSeriesIndex)
	}
	if metaRating >= 0 {
		ep.Package.SetRating(metaRating)
	}

	return nil
}
