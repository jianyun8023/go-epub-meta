package epub

import (
	"archive/zip"
	"os"
	"testing"
)

func TestOpen(t *testing.T) {
	// Create temp file
	f, err := os.CreateTemp("", "test.epub")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Create zip
	w := zip.NewWriter(f)

	// 1. container.xml
	container := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
    <rootfiles>
        <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
    </rootfiles>
</container>`
	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(container))

	// 2. OPF (Standard EPUB 2)
	opf := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="uuid_id" version="2.0">
    <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
        <dc:title>Test Book</dc:title>
        <dc:creator opf:role="aut">John Doe</dc:creator>
        <meta name="cover" content="cover-image" />
    </metadata>
    <manifest>
         <item id="cover-image" href="cover.jpg" media-type="image/jpeg"/>
    </manifest>
    <spine/>
</package>`
	ow, _ := w.Create("OEBPS/content.opf")
	ow.Write([]byte(opf))

	w.Close()
	f.Close()

	// Test Open
	r, err := Open(f.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	if r.OpfPath != "OEBPS/content.opf" {
		t.Errorf("Wrong OPF path: %s", r.OpfPath)
	}

	if len(r.Package.Metadata.Titles) == 0 {
		t.Fatalf("Title not found")
	}
	if r.Package.Metadata.Titles[0].Value != "Test Book" {
		t.Errorf("Wrong title: got %s", r.Package.Metadata.Titles[0].Value)
	}

	if len(r.Package.Metadata.Creators) == 0 {
		t.Fatalf("Creator not found")
	}
	if r.Package.Metadata.Creators[0].Value != "John Doe" {
		t.Errorf("Wrong creator: got %s", r.Package.Metadata.Creators[0].Value)
	}
    // Check role attribute (namespace handling check)
	if r.Package.Metadata.Creators[0].Role != "aut" {
		t.Errorf("Wrong creator role: got %s", r.Package.Metadata.Creators[0].Role)
	}

    // Check meta
    if len(r.Package.Metadata.Meta) == 0 {
        t.Errorf("Meta tag not found")
    }
    if r.Package.Metadata.Meta[0].Name != "cover" {
         t.Errorf("Meta name wrong: %s", r.Package.Metadata.Meta[0].Name)
    }
}
