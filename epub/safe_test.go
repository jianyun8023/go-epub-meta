package epub

import (
	"archive/zip"
	"io"
	"os"
	"testing"
)

func TestSaveInPlace(t *testing.T) {
	// Create source EPUB
	srcF, err := os.CreateTemp("", "inplace.epub")
	if err != nil {
		t.Fatal(err)
	}
	srcPath := srcF.Name()
	defer os.Remove(srcPath)

	z := zip.NewWriter(srcF)
	m, _ := z.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	m.Write([]byte("application/epub+zip"))
	c, _ := z.Create("META-INF/container.xml")
	c.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))
	o, _ := z.Create("content.opf")
	o.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><package xmlns="http://www.idpf.org/2007/opf" version="2.0"><metadata><dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Original Title</dc:title></metadata></package>`))
	z.Close()
	srcF.Close()

	// Open
	ep, err := Open(srcPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Modify
	ep.Package.SetTitle("New Title")

	// Save to same path (In-Place)
	// Important: We must not hold open file handles that conflict on Windows,
	// but Rename usually handles atomic replacement if file isn't locked.
	// Our `Reader` holds `ep.file`.
	// `Save` creates a temp file, then closes `zipWriter`, then `Rename` over original.
	// On Linux/Mac this is fine even if `ep.file` is open (inode stays, path changes).
	// On Windows, you can't rename over an open file.
	// We need to ensure `Save` doesn't fail.

	// Wait, if we rename over `srcPath` while `ep.file` is open reading from `srcPath`...
	// On POSIX: `ep.file` continues to read from the old inode (deleted file). This is safe!
	// On Windows: Rename fails.

	// We should check if we can fix this for Windows support.
	// Ideally `Save` does not close `r.file`.
	// If `Save` fails on Windows, we'd need to Close `r.file` before Rename?
	// But `r` is still valid!
	// We can't close `r` inside `Save`.

	// For this task, assuming POSIX environment (Linux sandbox).
	// But let's verify logic.

	err = ep.Save(srcPath)
	if err != nil {
		t.Fatalf("Save in-place failed: %v", err)
	}

	// Verify content by re-opening
	// We can't reuse `ep` because its underlying file might refer to the OLD file (inode) which is now unlinked.
	// Or maybe it does? Yes, `ep` is stale.
	ep.Close()

	ep2, err := Open(srcPath)
	if err != nil {
		t.Fatalf("Re-open failed: %v", err)
	}
	defer ep2.Close()

	if ep2.Package.GetTitle() != "New Title" {
		t.Errorf("Title not updated in place")
	}
}

func TestLatin1Reader(t *testing.T) {
	// Test buffer boundary logic
	input := make([]byte, 10000)
	for i := range input {
		input[i] = 0xFF // ÿ (expands to 2 bytes)
	}
	// Add some ASCII mixed in
	input[5000] = 'A'

	r := &latin1Reader{r: &byteReader{data: input}}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	// Expected length: 9999 chars are 0xFF (2 bytes) + 1 char 'A' (1 byte) = 19998 + 1 = 19999
	expectedLen := 9999*2 + 1
	if len(out) != expectedLen {
		t.Errorf("Wrong length: got %d, expected %d", len(out), expectedLen)
	}
}

type byteReader struct {
	data []byte
	ptr  int
}

func (b *byteReader) Read(p []byte) (n int, err error) {
	if b.ptr >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.ptr:])
	b.ptr += n
	return n, nil
}
