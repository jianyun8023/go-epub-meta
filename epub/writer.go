package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Save writes the modified EPUB to the specified output path.
// It preserves the original ZIP entry order (except mimetype which must be first).
// It preserves the original compression method for each entry.
// It writes to a temporary file first to support in-place rewriting.
func (r *Reader) Save(outputPath string) error {
	// 1. Create temp file
	tempDir := filepath.Dir(outputPath)
	if tempDir == "." {
		tempDir = ""
	}
	tmpF, err := os.CreateTemp(tempDir, "golibri-save-*.epub")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpF.Name()

	// Ensure cleanup if something goes wrong
	success := false
	defer func() {
		tmpF.Close()
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// 2. Create Zip Writer
	w := zip.NewWriter(tmpF)

	// 3. Write mimetype (MUST be first, STORED, no extra fields)
	if err := writeMimetype(w); err != nil {
		return err
	}

	// 4. Prepare modified content
	// Serialize OPF
	opfContent, err := xml.MarshalIndent(r.Package, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OPF: %w", err)
	}
	// Add XML header
	opfContent = append([]byte(xml.Header), opfContent...)

	// Track which files we've written
	writtenFiles := make(map[string]bool)
	writtenFiles["mimetype"] = true

	// Build a map of original files for looking up compression method
	originalFiles := make(map[string]*zip.File)
	for _, f := range r.zipReader.File {
		originalFiles[f.Name] = f
	}

	// 5. Stream copy files in ORIGINAL order, replacing OPF and Replacements
	for _, f := range r.zipReader.File {
		name := f.Name

		// Skip mimetype (already written first)
		if name == "mimetype" {
			continue
		}

		// Skip directory entries
		if f.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
			continue
		}

		// Mark as written
		writtenFiles[name] = true

		// Determine what content to write
		if name == r.OpfPath {
			// Write modified OPF, preserving original compression method
			if err := writeContentWithMethod(w, name, opfContent, f.Method); err != nil {
				return fmt.Errorf("failed to write OPF: %w", err)
			}
		} else if r.Replacements != nil {
			if content, ok := r.Replacements[name]; ok {
				// Write replacement content, preserving original compression method
				if err := writeContentWithMethod(w, name, content, f.Method); err != nil {
					return fmt.Errorf("failed to write replacement %s: %w", name, err)
				}
			} else {
				// Copy original file unchanged (raw copy, no re-compression)
				if err := copyZipFile(r, f, w); err != nil {
					return fmt.Errorf("failed to copy file %s: %w", name, err)
				}
			}
		} else {
			// Copy original file unchanged
			if err := copyZipFile(r, f, w); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", name, err)
			}
		}
	}

	// 6. Write any NEW Replacement files (not in original ZIP)
	if r.Replacements != nil {
		for path, content := range r.Replacements {
			if writtenFiles[path] {
				continue
			}
			// New file: use Deflate by default, but if there's an original with same path, inherit its method
			method := zip.Deflate
			if orig, ok := originalFiles[path]; ok {
				method = orig.Method
			}
			if err := writeContentWithMethod(w, path, content, method); err != nil {
				return fmt.Errorf("failed to write new file %s: %w", path, err)
			}
			writtenFiles[path] = true
		}
	}

	// 7. Close Writer explicitly to flush
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %w", err)
	}

	// Close temp file before rename (required on Windows)
	tmpF.Close()

	// 8. Atomic rename
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("failed to move temp file to output: %w", err)
	}

	success = true
	return nil
}

// writeContentWithMethod writes content to the zip with specified compression method.
func writeContentWithMethod(w *zip.Writer, name string, content []byte, method uint16) error {
	header := &zip.FileHeader{
		Name:   name,
		Method: method,
	}

	fw, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = fw.Write(content)
	return err
}

func writeMimetype(w *zip.Writer) error {
	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	}

	fw, err := w.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create mimetype header: %w", err)
	}

	// Write content "application/epub+zip"
	// MUST NOT have newline or extra chars
	if _, err := fw.Write([]byte("application/epub+zip")); err != nil {
		return fmt.Errorf("failed to write mimetype content: %w", err)
	}

	return nil
}

// copyZipFile copies a file entry from source to destination zip using raw copy.
// This preserves the original compression without re-encoding.
func copyZipFile(r *Reader, f *zip.File, w *zip.Writer) error {
	// Directory entries are optional; skip them to avoid "zip: write to directory".
	if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
		return nil
	}

	// Use CreateRaw to avoid re-compression.
	// This requires reading raw bytes from the underlying file.

	// Copy the header (CreateRaw treats it as immutable)
	header := f.FileHeader

	fw, err := w.CreateRaw(&header)
	if err != nil {
		return err
	}

	// Read raw bytes from Reader's underlying file
	offset, err := f.DataOffset()
	if err != nil {
		return fmt.Errorf("failed to get data offset: %w", err)
	}

	// Read exactly CompressedSize64 bytes from the raw offset
	section := io.NewSectionReader(r.file, offset, int64(f.CompressedSize64))

	_, err = io.Copy(fw, section)
	return err
}
