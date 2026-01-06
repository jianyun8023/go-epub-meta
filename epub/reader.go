package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
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

	// file is the underlying file, needed for raw access
	file *os.File

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
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	z, err := zip.NewReader(f, info.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to open zip reader: %w", err)
	}

	r := &Reader{
		zipReader: z,
		closer:    f,
		file:      f,
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
	buf []byte // Reused buffer
	pending []byte // Bytes read from r that couldn't fit into p yet
}

func (l *latin1Reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// If we have pending expanded bytes, satisfy from there first
	if len(l.pending) > 0 {
		n = copy(p, l.pending)
		l.pending = l.pending[n:]
		return n, nil
	}

	// Initialize buffer if needed
	if l.buf == nil {
		l.buf = make([]byte, 4096)
	}

	// We want to read enough to potentially fill p, but not overflow too much.
	// Since 1 byte can become 2, let's read min(len(p), len(l.buf)).
	toRead := len(p)
	if toRead > len(l.buf) {
		toRead = len(l.buf)
	}

	rn, rErr := l.r.Read(l.buf[:toRead])

	// Expand
	// Max expansion is 2x. We need a place to store it.
	// We could alloc here, or use another member buffer.
	// Since we immediately copy to p, let's alloc a small slice or reuse if we want to be super optimized.
	// For now, let's just make expanded slice.

	expanded := make([]byte, 0, rn*2)
	for i := 0; i < rn; i++ {
		b := l.buf[i]
		if b < 0x80 {
			expanded = append(expanded, b)
		} else {
			expanded = append(expanded, 0xC0|(b>>6))
			expanded = append(expanded, 0x80|(b&0x3F))
		}
	}

	// Copy to p
	n = copy(p, expanded)

	// Store rest in pending
	if n < len(expanded) {
		l.pending = expanded[n:]
	}

	return n, rErr
}
