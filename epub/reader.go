package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
)

// Container is the structure for META-INF/container.xml
type Container struct {
	XMLName   xml.Name    `xml:"urn:oasis:names:tc:opendocument:xmlns:container container"`
	Version   string      `xml:"version,attr"`
	RootFiles []RootFile  `xml:"rootfiles>rootfile"`
}

type RootFile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// Reader is the main entry point for reading an EPUB file.
type Reader struct {
	zipReader *zip.Reader
	closer    io.Closer

	// OpfPath is the location of the OPF file relative to root
	OpfPath string

	// Package is the parsed OPF structure
	Package *Package

	// Replacements maps full paths to new content (for added/modified files).
	// Used by Save() to inject content.
	Replacements map[string][]byte
}

// Open opens an EPUB file for reading.
func Open(filepath string) (*Reader, error) {
	z, err := zip.OpenReader(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	r := &Reader{
		zipReader: &z.Reader,
		closer:    z,
	}

	if err := r.parseContainer(); err != nil {
		r.Close()
		return nil, fmt.Errorf("failed to parse container: %w", err)
	}

	if err := r.parseOPF(); err != nil {
		r.Close()
		return nil, fmt.Errorf("failed to parse OPF: %w", err)
	}

	return r, nil
}

// Close closes the underlying zip file.
func (r *Reader) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// parseContainer reads META-INF/container.xml to find the OPF file.
func (r *Reader) parseContainer() error {
	f, err := r.openFile("META-INF/container.xml")
	if err != nil {
		return fmt.Errorf("container.xml missing: %w", err)
	}
	defer f.Close()

	var c Container
	if err := xml.NewDecoder(f).Decode(&c); err != nil {
		return fmt.Errorf("malformed container.xml: %w", err)
	}

	if len(c.RootFiles) == 0 {
		return fmt.Errorf("no rootfile found in container.xml")
	}

	// Normally we take the first one with correct mimetype, or just the first one.
	// Standard says application/oebps-package+xml
	for _, rf := range c.RootFiles {
		if rf.MediaType == "application/oebps-package+xml" {
			r.OpfPath = rf.FullPath
			return nil
		}
	}

	// Fallback: take the first one
	r.OpfPath = c.RootFiles[0].FullPath
	return nil
}

// parseOPF reads and parses the OPF file using the path found in container.xml.
func (r *Reader) parseOPF() error {
	f, err := r.openFile(r.OpfPath)
	if err != nil {
		return fmt.Errorf("OPF file %s missing: %w", r.OpfPath, err)
	}
	defer f.Close()

	// Use a tolerant decoder
	d := xml.NewDecoder(f)
	d.Strict = false
	d.CharsetReader = charsetReader

	var pkg Package
	if err := d.Decode(&pkg); err != nil {
		return fmt.Errorf("malformed OPF: %w", err)
	}

	r.Package = &pkg
	return nil
}

// openFile helps find a file in the zip by name.
func (r *Reader) openFile(name string) (io.ReadCloser, error) {
	// Standard zip names are forward slash, case sensitive usually.
	// However, Windows paths might be mixed. We stick to forward slash.
	// "container.xml" vs "META-INF/container.xml"

	for _, f := range r.zipReader.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// charsetReader implements a simple fallback for non-UTF-8 encodings.
// Used for "Zero-dependency" requirements.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	// Standard library only supports UTF-8.
	// We can implement a simple pass-through for "us-ascii", "iso-8859-1" (latin1)
	// which is strictly not correct (latin1 maps 1:1 to unicode points 0-255),
	// but for now we might just support basic ones.

	switch charset {
	case "UTF-8", "utf-8", "UTF8", "utf8":
		return input, nil
	case "ISO-8859-1", "iso-8859-1", "Latin1", "latin1", "Windows-1252", "windows-1252":
		// This creates a reader that converts Latin1/Windows-1252 to UTF-8.
		// Windows-1252 is a superset of ISO-8859-1.
		return &latin1Reader{r: input}, nil
	default:
		return input, nil
	}
}

type latin1Reader struct {
	r   io.Reader
	buf []byte // Internal buffer to hold source bytes
}

func (l *latin1Reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// We need an internal buffer to read source bytes before expansion.
	// We read up to len(p)/2 source bytes to guarantee they fit into p after expansion (max 2x or 3x).
	// Actually, Latin1 to UTF8 is max 2 bytes per char (for 0x80-0xFF).

	// Ensure we have a buffer
	if l.buf == nil {
		l.buf = make([]byte, 4096)
	}

	// Read from source into l.buf
	// We want to read enough to fill p partially.
	// We read at most len(p) / 2 bytes from source, or full buffer.
	readSize := len(p) / 2
	if readSize == 0 {
		readSize = 1
	}
	if readSize > len(l.buf) {
		readSize = len(l.buf)
	}

	rn, rErr := l.r.Read(l.buf[:readSize])

	// Convert l.buf[:rn] to p
	// Write to p
	n = 0
	for i := 0; i < rn; i++ {
		b := l.buf[i]
		if b < 0x80 {
			p[n] = b
			n++
		} else {
			// Convert to UTF-8 (2 bytes)
			// 0x80-0xFF -> 110xxxxx 10xxxxxx
			// 0xC2 (11000010) + (b >> 6) ? No.
			// Latin1 codepoint matches Unicode code point.
			// For U+0080 to U+07FF:
			// Byte 1: 110aaaaa -> 0xC0 | (code >> 6)
			// Byte 2: 10bbbbbb -> 0x80 | (code & 0x3F)

			// Since b is 0x80..0xFF, code is b.
			// 0x80 >> 6 = 2. 0xC0 | 2 = 0xC2.
			// 0xFF >> 6 = 3. 0xC0 | 3 = 0xC3.

			p[n] = 0xC0 | (b >> 6)
			n++
			p[n] = 0x80 | (b & 0x3F)
			n++
		}
	}

	return n, rErr
}
