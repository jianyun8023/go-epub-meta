package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// Save writes the modified EPUB to the specified output path.
// It ensures mimetype is written first and uncompressed.
func (r *Reader) Save(outputPath string) error {
	// 1. Create output file
	outF, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outF.Close()

	// 2. Create Zip Writer
	w := zip.NewWriter(outF)
	defer w.Close()

	// 3. Write mimetype (MUST be first, STORED, no extra fields)
	if err := writeMimetype(w); err != nil {
		return err
	}

	// 4. Serialize modified OPF
	// We need to write it to a buffer first to know its size, or just write it.
	// But we need to filter it out from the stream copy later.
	opfContent, err := xml.MarshalIndent(r.Package, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OPF: %w", err)
	}
	// Add XML header
	opfContent = append([]byte(xml.Header), opfContent...)

	// Write OPF to new zip
	// Note: We should preserve the original compression settings if possible, but for OPF text is fine.
	// Using Deflate (Default)
	fw, err := w.Create(r.OpfPath)
	if err != nil {
		return fmt.Errorf("failed to create OPF entry: %w", err)
	}
	if _, err := fw.Write(opfContent); err != nil {
		return fmt.Errorf("failed to write OPF content: %w", err)
	}

	// 5. Write Replacements (New/Modified files)
	// Track which files we've written to avoid duplicates if they exist in source
	writtenFiles := make(map[string]bool)
	writtenFiles["mimetype"] = true
	writtenFiles[r.OpfPath] = true

	if r.Replacements != nil {
		for path, content := range r.Replacements {
			// Create file in zip
			// Assume standard compression
			fw, err := w.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create replacement file %s: %w", path, err)
			}
			if _, err := fw.Write(content); err != nil {
				return fmt.Errorf("failed to write replacement content for %s: %w", path, err)
			}
			writtenFiles[path] = true
		}
	}

	// 6. Stream copy other files
	// We iterate over the original zipReader
	for _, f := range r.zipReader.File {
		// Skip if already written (mimetype, opf, or replaced files)
		if writtenFiles[f.Name] {
			continue
		}

		// Copy file
		if err := copyZipFile(f, w); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", f.Name, err)
		}
	}

	return nil
}

func writeMimetype(w *zip.Writer) error {
	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	}

	// Open writer for this header
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

func copyZipFile(f *zip.File, w *zip.Writer) error {
	// Create header from original
	// We reuse the name and method.
	// Note: We don't copy all extra fields to avoid corruption if offsets change?
	// archive/zip handles offsets.

	// We can't reuse FileHeader directly because it contains specific info about the old file.
	// We should construct a new one.
	header := &zip.FileHeader{
		Name:     f.Name,
		Method:   f.Method,
		Modified: f.Modified,
		// ExternalAttrs is important for permissions (though less critical for epub inner files)
	}

	// Open new entry
	fw, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	// Open original
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Copy
	_, err = io.Copy(fw, rc)
	return err
}
