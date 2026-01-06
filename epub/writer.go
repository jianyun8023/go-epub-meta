package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Save writes the modified EPUB to the specified output path.
// It ensures mimetype is written first and uncompressed.
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
		if err := copyZipFile(r, f, w); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", f.Name, err)
		}
	}

	// Close Writer explicitly to flush
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %w", err)
	}

	// We are done writing to temp
	// Close file handled by defer, but we need to rename now.
	// Windows rename needs close first.
	tmpF.Close()

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("failed to move temp file to output: %w", err)
	}

	success = true
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

func copyZipFile(r *Reader, f *zip.File, w *zip.Writer) error {
	// Use CreateRaw to avoid re-compression.
	// This requires reading raw bytes from the underlying file.

	// Construct header
	// We can mostly copy the header from f, but we need to be careful.
	// f.FileHeader contains compressed size, uncompressed size, crc32.
	// CreateRaw expects these to be set correctly if we are writing raw.

	header := f.FileHeader
	// Clear flags that might confuse Writer if we are not careful?
	// Actually, CreateRaw documentation says:
	// "The provided FileHeader is treated as immutable"
	// "The caller must write exactly UncompressedSize bytes" -> WAIT.
	// Docs for CreateRaw: "The caller must write exactly CompressedSize bytes to the returned writer".
	// Yes.

	fw, err := w.CreateRaw(&header)
	if err != nil {
		return err
	}

	// Read raw bytes from Reader's underlying file
	offset, err := f.DataOffset()
	if err != nil {
		return fmt.Errorf("failed to get data offset: %w", err)
	}

	// Seek to offset
	// We can't share the file pointer easily if concurrent, but we are sequential here.
	// r.file is *os.File

	section := io.NewSectionReader(r.file, offset, int64(f.CompressedSize64))

	_, err = io.Copy(fw, section)
	return err
}
