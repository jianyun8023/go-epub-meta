package epub

import (
	"archive/zip"
	"os"
	"testing"
)

func TestNonDestructiveCopy(t *testing.T) {
	// Create a source EPUB with a known compressed size
	srcF, _ := os.CreateTemp("", "src_nd.epub")
	defer os.Remove(srcF.Name())

	z := zip.NewWriter(srcF)

	// Create a file that is compressible
	content := []byte("AAAAAAAAAABBBBBBBBBBCCCCCCCCCC")
	// Use Deflate
	f, _ := z.CreateHeader(&zip.FileHeader{
		Name:   "content.txt",
		Method: zip.Deflate,
	})
	f.Write(content)

	// Create mimetype (Store)
	m, _ := z.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	m.Write([]byte("application/epub+zip"))

	// Create OPF
	o, _ := z.Create("OEBPS/content.opf")
	// Proper namespaces
	o.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Title</dc:title>
  </metadata>
  <manifest>
    <item id="content" href="../content.txt" media-type="text/plain"/>
  </manifest>
</package>`))
	// Need container
	c, _ := z.Create("META-INF/container.xml")
	c.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	z.Close()
	srcF.Close()

	// Open and capture original CompressedSize
	zr, _ := zip.OpenReader(srcF.Name())
	var origSize uint64
	for _, f := range zr.File {
		if f.Name == "content.txt" {
			origSize = f.CompressedSize64
		}
	}
	zr.Close()

	// Modify (metadata only)
	ep, err := Open(srcF.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	ep.Package.SetTitle("New Title")

	// Save
	outF, _ := os.CreateTemp("", "out_nd.epub")
	outPath := outF.Name()
	outF.Close()
	defer os.Remove(outPath)

	if err := ep.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	ep.Close()

	// Check output
	zr2, _ := zip.OpenReader(outPath)
	defer zr2.Close()

	var newSize uint64
	for _, f := range zr2.File {
		if f.Name == "content.txt" {
			newSize = f.CompressedSize64
		}
	}

	if newSize != origSize {
		t.Errorf("Compressed size changed! Original: %d, New: %d. Re-compression occurred.", origSize, newSize)
	}
}
