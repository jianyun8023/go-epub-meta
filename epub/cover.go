package epub

import (
	"fmt"
	"io"
	"path"
)

// GetCoverImage returns the content of the cover image and its media type.
func (r *Reader) GetCoverImage() (io.ReadCloser, string, error) {
	// Strategy 1: Look for "cover" item in manifest (EPUB 2/3)
	// Sometimes item id="cover" or id="cover-image"
	// Or item properties="cover-image" (EPUB 3)

	// Strategy 2: Look for <meta name="cover" content="item-id"> (EPUB 2)

	var coverItemID string

	// 1. Check EPUB 2 Meta
	for _, m := range r.Package.Metadata.Meta {
		if m.Name == "cover" {
			coverItemID = m.Content
			break
		}
	}

	// 2. If not found, check Manifest properties (EPUB 3)
	if coverItemID == "" {
		for _, item := range r.Package.Manifest.Items {
			if item.Properties == "cover-image" {
				coverItemID = item.ID
				break
			}
		}
	}

	// 3. Fallback: Check for item with id="cover" or "cover-image"
	if coverItemID == "" {
		for _, item := range r.Package.Manifest.Items {
			if item.ID == "cover" || item.ID == "cover-image" {
				coverItemID = item.ID
				break
			}
		}
	}

	if coverItemID == "" {
		return nil, "", fmt.Errorf("no cover found")
	}

	// Find item in manifest
	var coverItem *Item
	for i := range r.Package.Manifest.Items {
		if r.Package.Manifest.Items[i].ID == coverItemID {
			coverItem = &r.Package.Manifest.Items[i]
			break
		}
	}

	if coverItem == nil {
		return nil, "", fmt.Errorf("cover item %s not found in manifest", coverItemID)
	}

	// Resolve path relative to OPF
	// OpfPath is e.g. "OEBPS/content.opf".
	// coverItem.Href is relative to OPF folder.
	opfDir := path.Dir(r.OpfPath)
	fullPath := path.Join(opfDir, coverItem.Href)

	rc, err := r.openFile(fullPath)
	if err != nil {
		return nil, "", err
	}

	return rc, coverItem.MediaType, nil
}

// Note: SetCover logic is complex because it involves writing a NEW file into the zip
// which is handled during Save/Repack, or we need to buffer the new image.
// Since our Writer assumes "Read old -> Write new", modifying the *content* of an existing file
// means we need to intercept it during `copyZipFile` or add it to a "Pending Writes" list.
//
// Current `Writer` implementation:
// 1. Writes Mimetype
// 2. Writes OPF
// 3. Copies everything else from ZipReader (except mimetype and OPF).
//
// To support changing Cover (or adding one), we need:
// A mechanism to inject new files or replace existing ones by path.

// We need to upgrade `Reader` (which acts as the Editor state) to hold "New/Modified Files".

// SetCover replaces or adds a cover image.
// It updates the Manifest, Metadata, and adds the file to Replacements.
func (r *Reader) SetCover(data []byte, mediaType string) {
	// 1. Determine path for new cover
	// If cover exists, overwrite it.
	// If not, create "cover.jpg" (or matching ext) in OEBPS folder.

	opfDir := path.Dir(r.OpfPath)
	// Simple strategy: use "cover.jpg" or "cover.png" based on mediatype
	ext := ".jpg"
	if mediaType == "image/png" {
		ext = ".png"
	}
	// ... add other types if needed

	fileName := "cover" + ext
	fullPath := path.Join(opfDir, fileName)

	// 2. Add to Replacements
	if r.Replacements == nil {
		r.Replacements = make(map[string][]byte)
	}
	r.Replacements[fullPath] = data

	// 3. Update Manifest
	// Check if item exists
	var itemID string
	found := false
	for i, item := range r.Package.Manifest.Items {
		if item.Properties == "cover-image" || item.ID == "cover" {
			// Update existing item
			r.Package.Manifest.Items[i].Href = fileName
			r.Package.Manifest.Items[i].MediaType = mediaType
			itemID = item.ID
			found = true
			break
		}
	}

	if !found {
		// Create new item
		itemID = "cover-image"
		newItem := Item{
			ID:         itemID,
			Href:       fileName,
			MediaType:  mediaType,
			Properties: "cover-image", // EPUB 3
		}
		r.Package.Manifest.Items = append(r.Package.Manifest.Items, newItem)
	}

	// 4. Update Metadata (EPUB 2 compatibility)
	// Ensure <meta name="cover" content="item-id" /> exists
	metaFound := false
	for i, m := range r.Package.Metadata.Meta {
		if m.Name == "cover" {
			r.Package.Metadata.Meta[i].Content = itemID
			metaFound = true
			break
		}
	}

	if !metaFound {
		r.Package.setLegacyMeta("cover", itemID)
	}
}
