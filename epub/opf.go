package epub

import "encoding/xml"

// XML Namespaces
const (
	NsDC      = "http://purl.org/dc/elements/1.1/"
	NsOPF     = "http://www.idpf.org/2007/opf"
	NsXML     = "http://www.w3.org/XML/1998/namespace"
	NsDCTerms = "http://purl.org/dc/terms/"
	NsCalibre = "http://calibre.kovidgoyal.net/2009/metadata"
)

// Package is the root element of the OPF file.
// It supports both EPUB 2.0 and EPUB 3.x attributes.
type Package struct {
	XMLName          xml.Name `xml:"http://www.idpf.org/2007/opf package"`
	Version          string   `xml:"version,attr"`
	UniqueIdentifier string   `xml:"unique-identifier,attr"`
	Prefix           string   `xml:"prefix,attr,omitempty"` // EPUB 3
	Dir              string   `xml:"dir,attr,omitempty"`    // text direction
	Id               string   `xml:"id,attr,omitempty"`

	Metadata Metadata `xml:"metadata"`
	Manifest Manifest `xml:"manifest"`
	Spine    Spine    `xml:"spine"`
	Guide    *Guide   `xml:"guide,omitempty"` // Deprecated in EPUB 3, but widely used
}

// PackageLoose is used for parsing OPF files without namespace constraints.
// This improves compatibility with malformed EPUB files.
type PackageLoose struct {
	XMLName          xml.Name `xml:"package"`
	Version          string   `xml:"version,attr"`
	UniqueIdentifier string   `xml:"unique-identifier,attr"`
	Prefix           string   `xml:"prefix,attr,omitempty"`
	Dir              string   `xml:"dir,attr,omitempty"`
	Id               string   `xml:"id,attr,omitempty"`

	Metadata Metadata `xml:"metadata"`
	Manifest Manifest `xml:"manifest"`
	Spine    Spine    `xml:"spine"`
	Guide    *Guide   `xml:"guide,omitempty"`
}

// ToPackage converts a PackageLoose to a standard Package.
func (pl *PackageLoose) ToPackage() Package {
	return Package{
		XMLName:          xml.Name{Space: NsOPF, Local: "package"},
		Version:          pl.Version,
		UniqueIdentifier: pl.UniqueIdentifier,
		Prefix:           pl.Prefix,
		Dir:              pl.Dir,
		Id:               pl.Id,
		Metadata:         pl.Metadata,
		Manifest:         pl.Manifest,
		Spine:            pl.Spine,
		Guide:            pl.Guide,
	}
}

// Metadata contains publication metadata.
// It handles the union of EPUB 2 (DC elements) and EPUB 3 (Meta properties).
type Metadata struct {
	XMLName xml.Name `xml:"metadata"`

	// Dublin Core Elements (Common in EPUB 2 & 3)
	// Note: We use specific field mappings for the most common elements.
	Titles       []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ title"`
	Creators     []AuthorMeta `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Subjects     []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ subject"`
	Descriptions []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ description"`
	Publishers   []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ publisher"`
	Contributors []AuthorMeta `xml:"http://purl.org/dc/elements/1.1/ contributor"`
	Dates        []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ date"`
	Types        []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ type"`
	Formats      []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ format"`
	Identifiers  []IDMeta     `xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Sources      []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ source"`
	Languages    []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ language"`
	Rights       []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ rights"`

	// Meta tags (Generic)
	// Covers:
	// - EPUB 2: <meta name="..." content="..." />
	// - EPUB 3: <meta property="...">Value</meta>
	Meta []Meta `xml:"meta"`
}

// SimpleMeta represents basic DC elements like <dc:title>Value</dc:title>
type SimpleMeta struct {
	Value string `xml:",chardata"`
	ID    string `xml:"id,attr,omitempty"`
	Dir   string `xml:"dir,attr,omitempty"`
	Lang  string `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
}

// AuthorMeta represents creator/contributor
type AuthorMeta struct {
	SimpleMeta
	FileAs string `xml:"http://www.idpf.org/2007/opf file-as,attr,omitempty"`
	Role   string `xml:"http://www.idpf.org/2007/opf role,attr,omitempty"`
	Scheme string `xml:"http://www.idpf.org/2007/opf scheme,attr,omitempty"`
}

// IDMeta represents <dc:identifier>
type IDMeta struct {
	Value  string `xml:",chardata"`
	ID     string `xml:"id,attr,omitempty"`
	Scheme string `xml:"http://www.idpf.org/2007/opf scheme,attr,omitempty"`
}

// Meta represents the generic <meta> tag.
type Meta struct {
	// Common ID
	ID string `xml:"id,attr,omitempty"`

	// EPUB 2 Attributes
	Name    string `xml:"name,attr,omitempty"`
	Content string `xml:"content,attr,omitempty"`

	// EPUB 3 Attributes
	Property string `xml:"property,attr,omitempty"`
	Refines  string `xml:"refines,attr,omitempty"`
	Scheme   string `xml:"scheme,attr,omitempty"`

	// Value is the inner text content (Used in EPUB 3 mainly)
	Value string `xml:",chardata"`
}

// Manifest lists all files in the EPUB.
type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	ID           string `xml:"id,attr"`
	Href         string `xml:"href,attr"`
	MediaType    string `xml:"media-type,attr"`
	Properties   string `xml:"properties,attr,omitempty"` // EPUB 3 (e.g., "cover-image")
	Fallback     string `xml:"fallback,attr,omitempty"`
	MediaOverlay string `xml:"media-overlay,attr,omitempty"`
}

// Spine defines the reading order.
type Spine struct {
	Toc      string    `xml:"toc,attr,omitempty"` // EPUB 2 NCX reference
	PageProg string    `xml:"page-progression-direction,attr,omitempty"`
	ItemRefs []ItemRef `xml:"itemref"`
}

type ItemRef struct {
	IDRef      string `xml:"idref,attr"`
	Linear     string `xml:"linear,attr,omitempty"` // "yes" or "no"
	Properties string `xml:"properties,attr,omitempty"`
}

// Guide is deprecated in EPUB 3 but maintained for EPUB 2 compatibility.
type Guide struct {
	References []Reference `xml:"reference"`
}

type Reference struct {
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr,omitempty"`
	Href  string `xml:"href,attr"`
}
