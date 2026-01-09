package epub

import (
	"github.com/beevik/etree"
)

// marshalOPFWithEtree serializes the Package to XML using etree.
// This produces cleaner namespace prefixes (e.g., dc:identifier instead of identifier xmlns="...").
func (pkg *Package) marshalOPFWithEtree() ([]byte, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	// Create root package element
	root := doc.CreateElement("package")
	root.CreateAttr("xmlns", NsOPF)
	root.CreateAttr("version", pkg.Version)
	root.CreateAttr("unique-identifier", pkg.UniqueIdentifier)
	if pkg.Prefix != "" {
		root.CreateAttr("prefix", pkg.Prefix)
	}
	if pkg.Dir != "" {
		root.CreateAttr("dir", pkg.Dir)
	}
	if pkg.Id != "" {
		root.CreateAttr("id", pkg.Id)
	}

	// Create metadata element with namespace declarations
	metadata := root.CreateElement("metadata")
	metadata.CreateAttr("xmlns:dc", NsDC)
	metadata.CreateAttr("xmlns:opf", NsOPF)

	// Add DC elements
	for _, title := range pkg.Metadata.Titles {
		el := metadata.CreateElement("dc:title")
		el.SetText(title.Value)
		if title.ID != "" {
			el.CreateAttr("id", title.ID)
		}
		if title.Lang != "" {
			el.CreateAttr("xml:lang", title.Lang)
		}
		if title.Dir != "" {
			el.CreateAttr("dir", title.Dir)
		}
	}

	for _, creator := range pkg.Metadata.Creators {
		el := metadata.CreateElement("dc:creator")
		el.SetText(creator.Value)
		if creator.ID != "" {
			el.CreateAttr("id", creator.ID)
		}
		if creator.FileAs != "" {
			el.CreateAttr("opf:file-as", creator.FileAs)
		}
		if creator.Role != "" {
			el.CreateAttr("opf:role", creator.Role)
		}
	}

	for _, id := range pkg.Metadata.Identifiers {
		el := metadata.CreateElement("dc:identifier")
		el.SetText(id.Value)
		if id.ID != "" {
			el.CreateAttr("id", id.ID)
		}
		if id.Scheme != "" {
			el.CreateAttr("opf:scheme", id.Scheme)
		}
	}

	for _, lang := range pkg.Metadata.Languages {
		el := metadata.CreateElement("dc:language")
		el.SetText(lang.Value)
		if lang.ID != "" {
			el.CreateAttr("id", lang.ID)
		}
	}

	for _, pub := range pkg.Metadata.Publishers {
		el := metadata.CreateElement("dc:publisher")
		el.SetText(pub.Value)
	}

	for _, date := range pkg.Metadata.Dates {
		el := metadata.CreateElement("dc:date")
		el.SetText(date.Value)
	}

	for _, desc := range pkg.Metadata.Descriptions {
		el := metadata.CreateElement("dc:description")
		el.SetText(desc.Value)
	}

	for _, subj := range pkg.Metadata.Subjects {
		el := metadata.CreateElement("dc:subject")
		el.SetText(subj.Value)
	}

	for _, contrib := range pkg.Metadata.Contributors {
		el := metadata.CreateElement("dc:contributor")
		el.SetText(contrib.Value)
		if contrib.ID != "" {
			el.CreateAttr("id", contrib.ID)
		}
		if contrib.FileAs != "" {
			el.CreateAttr("opf:file-as", contrib.FileAs)
		}
		if contrib.Role != "" {
			el.CreateAttr("opf:role", contrib.Role)
		}
	}

	for _, typ := range pkg.Metadata.Types {
		el := metadata.CreateElement("dc:type")
		el.SetText(typ.Value)
	}

	for _, format := range pkg.Metadata.Formats {
		el := metadata.CreateElement("dc:format")
		el.SetText(format.Value)
	}

	for _, src := range pkg.Metadata.Sources {
		el := metadata.CreateElement("dc:source")
		el.SetText(src.Value)
	}

	for _, rights := range pkg.Metadata.Rights {
		el := metadata.CreateElement("dc:rights")
		el.SetText(rights.Value)
	}

	// Add meta elements
	for _, m := range pkg.Metadata.Meta {
		el := metadata.CreateElement("meta")
		if m.Property != "" {
			// EPUB 3 style
			el.CreateAttr("property", m.Property)
			if m.Refines != "" {
				el.CreateAttr("refines", m.Refines)
			}
			if m.Scheme != "" {
				el.CreateAttr("scheme", m.Scheme)
			}
			if m.ID != "" {
				el.CreateAttr("id", m.ID)
			}
			el.SetText(m.Value)
		} else if m.Name != "" {
			// EPUB 2 style
			el.CreateAttr("name", m.Name)
			el.CreateAttr("content", m.Content)
		}
	}

	// Create manifest
	manifest := root.CreateElement("manifest")
	for _, item := range pkg.Manifest.Items {
		el := manifest.CreateElement("item")
		el.CreateAttr("id", item.ID)
		el.CreateAttr("href", item.Href)
		el.CreateAttr("media-type", item.MediaType)
		if item.Properties != "" {
			el.CreateAttr("properties", item.Properties)
		}
		if item.Fallback != "" {
			el.CreateAttr("fallback", item.Fallback)
		}
		if item.MediaOverlay != "" {
			el.CreateAttr("media-overlay", item.MediaOverlay)
		}
	}

	// Create spine
	spine := root.CreateElement("spine")
	if pkg.Spine.Toc != "" {
		spine.CreateAttr("toc", pkg.Spine.Toc)
	}
	if pkg.Spine.PageProg != "" {
		spine.CreateAttr("page-progression-direction", pkg.Spine.PageProg)
	}
	for _, itemref := range pkg.Spine.ItemRefs {
		el := spine.CreateElement("itemref")
		el.CreateAttr("idref", itemref.IDRef)
		if itemref.Linear != "" {
			el.CreateAttr("linear", itemref.Linear)
		}
		if itemref.Properties != "" {
			el.CreateAttr("properties", itemref.Properties)
		}
	}

	// Create guide (if present)
	if pkg.Guide != nil && len(pkg.Guide.References) > 0 {
		guide := root.CreateElement("guide")
		for _, ref := range pkg.Guide.References {
			el := guide.CreateElement("reference")
			el.CreateAttr("type", ref.Type)
			if ref.Title != "" {
				el.CreateAttr("title", ref.Title)
			}
			el.CreateAttr("href", ref.Href)
		}
	}

	doc.Indent(2)
	return doc.WriteToBytes()
}
