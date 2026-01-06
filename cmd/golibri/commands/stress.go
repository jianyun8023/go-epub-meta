package commands

import (
	"fmt"
	"golibri/epub"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stressCmd)
}

var stressCmd = &cobra.Command{
	Use:   "stress-test [directory]",
	Short: "Run stress test harness (Round-trip verification)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		runStressTest(path)
	},
}

func runStressTest(root string) {
	var files []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".epub") {
			files = append(files, p)
		}
		return nil
	})

	workers := 16
	jobs := make(chan string, len(files))
	results := make(chan error, len(files))

	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				// Round trip test
				err := runRoundTrip(f)
				results <- err
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	wg.Wait()
	close(results)

	passed := 0
	failed := 0
	for err := range results {
		if err == nil {
			passed++
		} else {
			failed++
		}
	}

	duration := time.Since(start)
	fmt.Printf("Processed %d files in %s. Passed: %d, Failed: %d\n", len(files), duration, passed, failed)
}

func runRoundTrip(path string) error {
	// 1. Open
	ep, err := epub.Open(path)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}
	defer ep.Close()

	// 2. Modify (Mock)
	ep.Package.SetTitle(ep.Package.GetTitle() + " [MOD]")

	// 3. Save to memory/temp
	// Use os.CreateTemp to avoid polluting source directory
	f, err := os.CreateTemp("", "golibri_stress_*.epub")
	if err != nil {
		return fmt.Errorf("create temp failed: %w", err)
	}
	tmp := f.Name()
	f.Close()

	if err := ep.Save(tmp); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("save failed: %w", err)
	}
	defer os.Remove(tmp)

	// 4. Re-open and Verify
	ep2, err := epub.Open(tmp)
	if err != nil {
		return fmt.Errorf("re-open failed: %w", err)
	}
	defer ep2.Close()

	if !strings.Contains(ep2.Package.GetTitle(), "[MOD]") {
		return fmt.Errorf("verification failed: title mismatch")
	}

	return nil
}
