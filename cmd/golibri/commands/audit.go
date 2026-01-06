package commands

import (
	"fmt"
	"golibri/epub"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(auditCmd)
}

var auditCmd = &cobra.Command{
	Use:   "audit [directory]",
	Short: "Scan EPUBs for errors (read-only)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		runAudit(path)
	},
}

func runAudit(root string) {
	files := []string{}
	// Recursive walk
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".epub") {
			files = append(files, p)
		}
		return nil
	})

	fmt.Printf("Scanning %d files...\n", len(files))

	success := 0
	failed := 0

	for _, f := range files {
		ep, err := epub.Open(f)
		if err != nil {
			fmt.Printf("[FAIL] %s: %v\n", f, err)
			failed++
		} else {
			// Basic checks
			if ep.Package.GetTitle() == "" {
				fmt.Printf("[WARN] %s: No Title\n", f)
			}
			ep.Close()
			success++
		}
	}

	fmt.Printf("Scan complete. Success: %d, Failed: %d\n", success, failed)
}
