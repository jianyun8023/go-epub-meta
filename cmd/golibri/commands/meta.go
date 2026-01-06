package commands

import (
	"fmt"
	"golibri/epub"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	metaTitle  string
	metaAuthor string
	metaSeries string
	metaCover  string
	metaOutput string
)

func init() {
	metaCmd.Flags().StringVarP(&metaTitle, "title", "t", "", "Set title")
	metaCmd.Flags().StringVarP(&metaAuthor, "author", "a", "", "Set author")
	metaCmd.Flags().StringVarP(&metaSeries, "series", "s", "", "Set series")
	metaCmd.Flags().StringVarP(&metaCover, "cover", "c", "", "Set cover image path")
	metaCmd.Flags().StringVarP(&metaOutput, "output", "o", "", "Output file path (required for modification)")

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
		if metaTitle == "" && metaAuthor == "" && metaSeries == "" && metaCover == "" {
			printMetadata(ep)
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
		fmt.Printf("Saved to %s\n", metaOutput)
	},
}

func printMetadata(ep *epub.Reader) {
	fmt.Println("--- Metadata ---")
	fmt.Printf("Title:    %s\n", ep.Package.GetTitle())
	fmt.Printf("Author:   %s\n", ep.Package.GetAuthor())
	fmt.Printf("Series:   %s\n", ep.Package.GetSeries())
	fmt.Printf("Language: %s\n", ep.Package.GetLanguage())
	_, _, err := ep.GetCoverImage()
	if err == nil {
		fmt.Println("Cover:    Found")
	} else {
		fmt.Println("Cover:    Not Found")
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
