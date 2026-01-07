package commands

import (
	"encoding/json"
	"fmt"
	"golibri/epub"
	"io"
	"os"
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
)

func init() {
	metaCmd.Flags().StringVarP(&metaTitle, "title", "t", "", "Set title")
	metaCmd.Flags().StringVarP(&metaAuthor, "author", "a", "", "Set author")
	metaCmd.Flags().StringVarP(&metaSeries, "series", "s", "", "Set series")
	metaCmd.Flags().StringVarP(&metaCover, "cover", "c", "", "Set cover image path")
	metaCmd.Flags().StringVar(&metaISBN, "isbn", "", "Set ISBN identifier")
	metaCmd.Flags().StringVar(&metaASIN, "asin", "", "Set ASIN identifier")
	metaCmd.Flags().StringArrayVarP(&metaIdentifiers, "identifier", "i", []string{}, "Set identifier (format: scheme:value, e.g., douban:12345678)")
	metaCmd.Flags().StringVarP(&metaOutput, "output", "o", "", "Output file path (required for modification)")
	metaCmd.Flags().BoolVar(&metaJSON, "json", false, "Output metadata in JSON format (compatible with ebook-meta)")

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

		// Read Mode
		if metaTitle == "" && metaAuthor == "" && metaSeries == "" && metaCover == "" && metaISBN == "" && metaASIN == "" && len(metaIdentifiers) == 0 {
			if metaJSON {
				printMetadataJSON(ep)
			} else {
				printMetadata(ep)
			}
			return
		}

		// Write Mode
		if metaOutput == "" {
			fmt.Println("Error: -o (output) is required when modifying metadata.")
			os.Exit(1)
		}

		if err := applyChanges(ep); err != nil {
			fmt.Printf("Error applying changes: %v\n", err)
			os.Exit(1)
		}

		if err := ep.Save(metaOutput); err != nil {
			fmt.Printf("Error saving EPUB: %v\n", err)
			os.Exit(1)
		}

		if metaJSON {
			// If JSON requested after write, we should probably output the NEW metadata
			// Re-opening might be expensive, so we just use the current state since applyChanges updated it.
			printMetadataJSON(ep)
		} else {
			fmt.Printf("Saved to %s\n", metaOutput)
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
	Identifiers map[string]string `json:"identifiers"`
	Cover       bool              `json:"cover"`
}

func printMetadataJSON(ep *epub.Reader) {
	meta := MetadataJSON{
		Title:       ep.Package.GetTitle(),
		Authors:     []string{ep.Package.GetAuthor()}, // Simple conversion for now, ideally GetAuthors()
		Publisher:   ep.Package.GetPublisher(),
		Published:   ep.Package.GetPublishDate(),
		Language:    ep.Package.GetLanguage(),
		Series:      ep.Package.GetSeries(),
		Identifiers: ep.Package.GetIdentifiers(),
		Cover:       false,
	}

	// Fix Authors: if it contains commas, maybe split?
	// For now, consistent with existing GetAuthor() which returns a single string.
	// We might want to improve this in the future if multiple creators exist.

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
	fmt.Printf("Author:      %s\n", ep.Package.GetAuthor())

	// Display publisher if available
	if publisher := ep.Package.GetPublisher(); publisher != "" {
		fmt.Printf("Publisher:   %s\n", publisher)
	}

	// Display publication date if available
	if published := ep.Package.GetPublishDate(); published != "" {
		fmt.Printf("Published:   %s\n", published)
	}

	fmt.Printf("Language:    %s\n", ep.Package.GetLanguage())

	// Display series if available
	if series := ep.Package.GetSeries(); series != "" {
		fmt.Printf("Series:      %s\n", series)
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

	_, _, err := ep.GetCoverImage()
	if err == nil {
		fmt.Println("Cover:       Found")
	} else {
		fmt.Println("Cover:       Not Found")
	}
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
	return nil
}
