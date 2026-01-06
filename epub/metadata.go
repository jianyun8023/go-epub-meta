package epub

// GetTitle returns the first title found.
func (pkg *Package) GetTitle() string {
	if len(pkg.Metadata.Titles) > 0 {
		return pkg.Metadata.Titles[0].Value
	}
	return ""
}

// SetTitle updates the title. It overwrites existing titles.
func (pkg *Package) SetTitle(title string) {
	pkg.Metadata.Titles = []SimpleMeta{{Value: title}}
}

// GetAuthor returns the first creator.
func (pkg *Package) GetAuthor() string {
	if len(pkg.Metadata.Creators) > 0 {
		return pkg.Metadata.Creators[0].Value
	}
	return ""
}

// SetAuthor sets the author.
func (pkg *Package) SetAuthor(name string) {
	// Standard practice: role="aut"
	pkg.Metadata.Creators = []AuthorMeta{{
		SimpleMeta: SimpleMeta{Value: name},
		Role:       "aut",
	}}
}

// GetDescription returns the description.
func (pkg *Package) GetDescription() string {
	if len(pkg.Metadata.Descriptions) > 0 {
		return pkg.Metadata.Descriptions[0].Value
	}
	return ""
}

// SetDescription sets the description.
func (pkg *Package) SetDescription(desc string) {
	pkg.Metadata.Descriptions = []SimpleMeta{{Value: desc}}
}

// GetLanguage returns the language.
func (pkg *Package) GetLanguage() string {
	if len(pkg.Metadata.Languages) > 0 {
		return pkg.Metadata.Languages[0].Value
	}
	return ""
}

// SetLanguage sets the language.
func (pkg *Package) SetLanguage(lang string) {
	pkg.Metadata.Languages = []SimpleMeta{{Value: lang}}
}

// GetSeries attempts to find the series name.
// It looks for Calibre meta tags for EPUB 2.
func (pkg *Package) GetSeries() string {
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series" {
			return m.Content
		}
		// Todo: EPUB 3 meta with collection property?
	}
	return ""
}

// SetSeries sets the series using Calibre meta tag.
func (pkg *Package) SetSeries(series string) {
	// Remove existing series tag
	newMeta := []Meta{}
	found := false
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "calibre:series" {
			// Update existing
			m.Content = series
			newMeta = append(newMeta, m)
			found = true
		} else {
			newMeta = append(newMeta, m)
		}
	}

	if !found {
		newMeta = append(newMeta, Meta{
			Name:    "calibre:series",
			Content: series,
		})
	}
	pkg.Metadata.Meta = newMeta
}

// GetSubjects returns a list of tags.
func (pkg *Package) GetSubjects() []string {
	var subjects []string
	for _, s := range pkg.Metadata.Subjects {
		subjects = append(subjects, s.Value)
	}
	return subjects
}

// SetSubjects overwrites tags.
func (pkg *Package) SetSubjects(tags []string) {
	var newSubjects []SimpleMeta
	for _, t := range tags {
		newSubjects = append(newSubjects, SimpleMeta{Value: t})
	}
	pkg.Metadata.Subjects = newSubjects
}
