package commands

import (
	"fmt"
	"golibri/epub"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [directory]",
	Short: "Scan directory for EPUB version statistics",
	Long:  `Recursively scans a directory for .epub files and reports distribution of EPUB versions (2.0 vs 3.x).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runScan(args[0])
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(root string) {
	fmt.Printf("Scanning %s for EPUB files...\n", root)

	var (
		countV2         int64
		countV3         int64
		countV3Hybrid   int64 // EPUB 3 with NCX (backward compatible)
		countV1         int64 // OEBPS 1.x
		countUnknown    int64
		countError      int64
		countTotal      int64
		errors          []string
		unknownVersions []string
	)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Access error
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".epub") {
			return nil
		}

		atomic.AddInt64(&countTotal, 1)

		ver, hasNCX, err := analyzeEpub(path)
		if err != nil {
			atomic.AddInt64(&countError, 1)
			msg := fmt.Sprintf("%s: %v", filepath.Base(path), err)
			// Simple slice append is not thread safe if we were parallel, but we are sequential here?
			// The original code had a pseudo-wg but was effectively sequential inside the Walk.
			// Let's keep it sequential for safety with slice appends.
			errors = append(errors, msg)
			return nil
		}

		if strings.HasPrefix(ver, "2") {
			atomic.AddInt64(&countV2, 1)
		} else if strings.HasPrefix(ver, "3") {
			atomic.AddInt64(&countV3, 1)
			if hasNCX {
				atomic.AddInt64(&countV3Hybrid, 1)
			}
		} else if strings.HasPrefix(ver, "1") {
			// 1.0 or 1.2
			atomic.AddInt64(&countV1, 1)
		} else {
			atomic.AddInt64(&countUnknown, 1)
			unknownVersions = append(unknownVersions, fmt.Sprintf("%s (%s)", filepath.Base(path), ver))
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Walk error: %v\n", err)
	}

	fmt.Println("\nScan Results:")
	fmt.Printf("Total EPUBs:       %d\n", countTotal)
	fmt.Printf("EPUB 2.x:          %d\n", countV2)

	var pctPure, pctHybrid float64
	if countV3 > 0 {
		pctPure = float64(countV3-countV3Hybrid) / float64(countV3) * 100
		pctHybrid = float64(countV3Hybrid) / float64(countV3) * 100
	}

	fmt.Printf("EPUB 3.x Total:    %d\n", countV3)
	fmt.Printf("  - Pure EPUB 3:   %d (%.1f%% of V3)\n", countV3-countV3Hybrid, pctPure)
	fmt.Printf("  - Hybrid (w/NCX):%d (%.1f%% of V3)\n", countV3Hybrid, pctHybrid)
	fmt.Printf("OEBPS 1.x:         %d\n", countV1)
	fmt.Printf("Unknown Ver:       %d\n", countUnknown)
	fmt.Printf("Errors:            %d\n", countError)

	if len(unknownVersions) > 0 {
		fmt.Println("\nUnknown Version Samples:")
		for i, u := range unknownVersions {
			if i >= 10 {
				break
			}
			fmt.Printf("  - %s\n", u)
		}
	}

	if len(errors) > 0 {
		fmt.Println("\nError Details:")
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
	}
}

func analyzeEpub(path string) (version string, hasNCX bool, err error) {
	ep, err := epub.Open(path)
	if err != nil {
		return "", false, err
	}
	defer ep.Close()

	if ep.Package == nil {
		return "", false, fmt.Errorf("no package found")
	}

	// Check for NCX (toc attribute in spine or just standard toc.ncx existence)
	// EPUB 2 uses spine toc attribute pointing to manifest item.
	// EPUB 3 usually doesn't need it, but for backward compat it should have it.

	// Check spine toc ref
	if ep.Package.Spine.Toc != "" {
		// Verify the item actually exists in manifest
		for _, item := range ep.Package.Manifest.Items {
			if item.ID == ep.Package.Spine.Toc {
				hasNCX = true
				break
			}
		}
	}

	return ep.Package.Version, hasNCX, nil
}
