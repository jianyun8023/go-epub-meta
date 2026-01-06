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
