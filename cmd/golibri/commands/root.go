package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "golibri",
	Short: "Golibri is a high-performance EPUB metadata editor",
	Long: `Golibri is a zero-dependency Go library and CLI tool for reading
and modifying EPUB 2/3 metadata, designed for performance and correctness.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
