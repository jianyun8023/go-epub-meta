package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "test-suite",
	Short: "Golibri Test Suite",
	Long:  `A comprehensive test suite for Golibri, including functional tests, comparison with ebook-meta, and reporting.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
