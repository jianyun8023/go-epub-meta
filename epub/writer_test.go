package epub

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSave(t *testing.T) {
	// 1. Create source EPUB
	srcF, _ := os.CreateTemp("", "src.epub")
	defer os.Remove(srcF.Name())

	z := zip.NewWriter(srcF)

	// mimetype
	m, _ := z.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	m.Write([]byte("application/epub+zip"))

	// container
	c, _ := z.Create("META-INF/container.xml")
	c.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	// content.opf
	o, _ := z.Create("content.opf")
	o.Write([]byte(`<package xmlns="http://www.idpf.org/2007/opf" version="2.0"><metadata><dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Original Title</dc:title></metadata></package>`))

	// extra file
	e, _ := z.Create("chapter1.html")
	e.Write([]byte("<h1>Chapter 1</h1>"))

	z.Close()
	srcF.Close()

	// 2. Open and Modify
	r, err := Open(srcF.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Modify Title (In-Memory)
	if len(r.Package.Metadata.Titles) > 0 {
		r.Package.Metadata.Titles[0].Value = "New Title"
	}

	// 3. Save to new file
	outF, _ := os.CreateTemp("", "out.epub")
	outPath := outF.Name()
	outF.Close()
	defer os.Remove(outPath)

	if err := r.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 4. Verify Output
	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("Failed to open output: %v", err)
	}
	defer zr.Close()

	// Check Mimetype is first
	if zr.File[0].Name != "mimetype" {
		t.Errorf("First file is not mimetype: %s", zr.File[0].Name)
	}
	if zr.File[0].Method != zip.Store {
		t.Errorf("Mimetype is not stored: %d", zr.File[0].Method)
	}

	// Check content.opf is modified
	foundOpf := false
	for _, f := range zr.File {
		if f.Name == "content.opf" {
			foundOpf = true
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			content := string(b)
			if !strings.Contains(content, "New Title") {
				t.Errorf("OPF not updated. Content: %s", content)
			}
		}
		if f.Name == "chapter1.html" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			if string(b) != "<h1>Chapter 1</h1>" {
				t.Errorf("Chapter 1 corrupted")
			}
		}
	}

	if !foundOpf {
		t.Error("content.opf not found in output")
	}
}

// TestSavePreservesEntryOrder verifies that Save() preserves the original ZIP entry order
// (except mimetype which must always be first per EPUB spec).
func TestSavePreservesEntryOrder(t *testing.T) {
	// 1. Create source EPUB with specific order
	srcF, _ := os.CreateTemp("", "order-src.epub")
	defer os.Remove(srcF.Name())

	z := zip.NewWriter(srcF)

	// Write files in a specific order (mimetype first as required)
	entries := []string{
		"mimetype",
		"META-INF/container.xml",
		"OEBPS/content.opf",
		"OEBPS/chapter1.xhtml",
		"OEBPS/chapter2.xhtml",
		"OEBPS/styles/main.css",
		"OEBPS/images/cover.jpg",
	}

	// mimetype (Store)
	m, _ := z.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	m.Write([]byte("application/epub+zip"))

	// container
	c, _ := z.Create("META-INF/container.xml")
	c.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	// OPF
	o, _ := z.Create("OEBPS/content.opf")
	o.Write([]byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="2.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Order Test</dc:title></metadata><manifest/><spine/></package>`))

	// Chapters
	ch1, _ := z.Create("OEBPS/chapter1.xhtml")
	ch1.Write([]byte("<html><body>Chapter 1</body></html>"))

	ch2, _ := z.Create("OEBPS/chapter2.xhtml")
	ch2.Write([]byte("<html><body>Chapter 2</body></html>"))

	// CSS
	css, _ := z.Create("OEBPS/styles/main.css")
	css.Write([]byte("body { font-family: serif; }"))

	// Image (simulate binary)
	img, _ := z.Create("OEBPS/images/cover.jpg")
	img.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0}) // JPEG header

	z.Close()
	srcF.Close()

	// 2. Open and modify
	r, err := Open(srcF.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Modify title
	r.Package.SetTitle("Modified Title")

	// 3. Save
	outF, _ := os.CreateTemp("", "order-out.epub")
	outPath := outF.Name()
	outF.Close()
	defer os.Remove(outPath)

	if err := r.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 4. Verify order
	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("Failed to open output: %v", err)
	}
	defer zr.Close()

	// Check mimetype is first
	if zr.File[0].Name != "mimetype" {
		t.Errorf("First entry should be mimetype, got: %s", zr.File[0].Name)
	}

	// Check remaining entries preserve original order
	var outputEntries []string
	for _, f := range zr.File {
		outputEntries = append(outputEntries, f.Name)
	}

	// Expected: mimetype first, then original order (excluding mimetype from original)
	expectedOrder := entries // mimetype already first in our source

	if len(outputEntries) != len(expectedOrder) {
		t.Errorf("Entry count mismatch: got %d, expected %d", len(outputEntries), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(outputEntries) {
			break
		}
		if outputEntries[i] != expected {
			t.Errorf("Entry %d: got %s, expected %s", i, outputEntries[i], expected)
		}
	}
}

// TestSavePreservesCompressionMethod verifies that Save() preserves the original compression method.
func TestSavePreservesCompressionMethod(t *testing.T) {
	// 1. Create source EPUB with mixed compression methods
	srcF, _ := os.CreateTemp("", "compress-src.epub")
	defer os.Remove(srcF.Name())

	z := zip.NewWriter(srcF)

	// mimetype: Store (required)
	m, _ := z.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	m.Write([]byte("application/epub+zip"))

	// container: Deflate
	c, _ := z.CreateHeader(&zip.FileHeader{Name: "META-INF/container.xml", Method: zip.Deflate})
	c.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	// OPF: Store (some tools use Store for small text files)
	o, _ := z.CreateHeader(&zip.FileHeader{Name: "content.opf", Method: zip.Store})
	o.Write([]byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="2.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Compress Test</dc:title></metadata><manifest/><spine/></package>`))

	// Chapter: Deflate
	ch, _ := z.CreateHeader(&zip.FileHeader{Name: "chapter.xhtml", Method: zip.Deflate})
	ch.Write([]byte("<html><body>Chapter content with enough text to be compressible...</body></html>"))

	// Image: Store (already compressed binary)
	img, _ := z.CreateHeader(&zip.FileHeader{Name: "cover.jpg", Method: zip.Store})
	img.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46})

	z.Close()
	srcF.Close()

	// Record original methods
	origMethods := make(map[string]uint16)
	zrOrig, _ := zip.OpenReader(srcF.Name())
	for _, f := range zrOrig.File {
		origMethods[f.Name] = f.Method
	}
	zrOrig.Close()

	// 2. Open and modify
	r, err := Open(srcF.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Modify title (triggers OPF rewrite)
	r.Package.SetTitle("Modified Title")

	// 3. Save
	outF, _ := os.CreateTemp("", "compress-out.epub")
	outPath := outF.Name()
	outF.Close()
	defer os.Remove(outPath)

	if err := r.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 4. Verify compression methods
	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("Failed to open output: %v", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		origMethod, ok := origMethods[f.Name]
		if !ok {
			continue // New file
		}

		// OPF is rewritten but should preserve method
		if f.Method != origMethod {
			t.Errorf("Compression method changed for %s: original=%d, new=%d", f.Name, origMethod, f.Method)
		}
	}

	// Specifically check mimetype is Store
	if zr.File[0].Name != "mimetype" || zr.File[0].Method != zip.Store {
		t.Errorf("mimetype should be Store, got method=%d", zr.File[0].Method)
	}
}
