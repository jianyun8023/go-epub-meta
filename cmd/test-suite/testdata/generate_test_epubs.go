package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
)

func main() {
	// Generate valid test EPUBs
	generateValidEPUBs()
	// Generate invalid test EPUB
	generateInvalidEPUB()
	fmt.Println("Test EPUB files generated successfully!")
}

func generateValidEPUBs() {
	testCases := []struct {
		filename  string
		title     string
		author    string
		language  string
		isbn      string
		series    string
		publisher string
		published string
		titleSort string
		producer  string
	}{
		{
			filename:  "valid/simple.epub",
			title:     "Simple Test Book",
			author:    "Test Author",
			language:  "en",
			isbn:      "9781234567890",
			series:    "",
			publisher: "",
			published: "",
			titleSort: "",
			producer:  "",
		},
		{
			filename:  "valid/with_metadata.epub",
			title:     "Complete Test Book",
			author:    "John Doe",
			language:  "en",
			isbn:      "9780987654321",
			series:    "Test Series",
			publisher: "Test Publisher",
			published: "2024-01-01T00:00:00+00:00",
			titleSort: "Complete Test Book, The",
			producer:  "Test Producer",
		},
		{
			filename:  "valid/chinese.epub",
			title:     "测试书籍",
			author:    "张三",
			language:  "zh",
			isbn:      "9787544798501",
			series:    "测试系列",
			publisher: "测试出版社",
			published: "2023-12-31T16:00:00+00:00",
			titleSort: "",
			producer:  "",
		},
	}

	for _, tc := range testCases {
		if err := createEPUB(tc.filename, tc.title, tc.author, tc.language, tc.isbn, tc.series, tc.publisher, tc.published, tc.titleSort, tc.producer); err != nil {
			fmt.Printf("Error creating %s: %v\n", tc.filename, err)
		} else {
			fmt.Printf("Created %s\n", tc.filename)
		}
	}
}

func generateInvalidEPUB() {
	// Create a file that looks like EPUB but is invalid
	f, err := os.Create("invalid/broken.epub")
	if err != nil {
		fmt.Printf("Error creating invalid EPUB: %v\n", err)
		return
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// Write invalid container.xml
	cw, _ := w.Create("META-INF/container.xml")
	io.WriteString(cw, `<?xml version="1.0"?><invalid>broken</invalid>`)

	fmt.Println("Created invalid/broken.epub")
}

func createEPUB(filename, title, author, language, isbn, series, publisher, published, titleSort, producer string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// 1. mimetype (first file, uncompressed)
	mw, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		return err
	}
	io.WriteString(mw, "application/epub+zip")

	// 2. META-INF/container.xml
	cw, err := w.Create("META-INF/container.xml")
	if err != nil {
		return err
	}
	io.WriteString(cw, `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	// 3. content.opf
	opfContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="BookId">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>%s</dc:title>
    <dc:creator opf:role="aut">%s</dc:creator>
    <dc:language>%s</dc:language>
    <dc:identifier id="BookId" opf:scheme="ISBN">%s</dc:identifier>
    %s
    %s
    %s
    %s
    %s
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="chapter1"/>
  </spine>
</package>`, title, author, language, isbn,
		func() string {
			if series != "" {
				// Add both series and series_index (default to 1)
				return fmt.Sprintf(`<meta name="calibre:series" content="%s"/>
    <meta name="calibre:series_index" content="1.0"/>`, series)
			}
			return ""
		}(),
		func() string {
			if publisher != "" {
				return fmt.Sprintf(`<dc:publisher>%s</dc:publisher>`, publisher)
			}
			return ""
		}(),
		func() string {
			if published != "" {
				return fmt.Sprintf(`<dc:date opf:event="publication">%s</dc:date>`, published)
			}
			return ""
		}(),
		func() string {
			if titleSort != "" {
				return fmt.Sprintf(`<meta name="calibre:title_sort" content="%s"/>`, titleSort)
			}
			return ""
		}(),
		func() string {
			if producer != "" {
				// Use contributor with role 'bkp' for Producer, as ebook-meta likely prefers this
				return fmt.Sprintf(`<dc:contributor opf:role="bkp">%s</dc:contributor>`, producer)
			}
			return ""
		}(),
	)

	ow, err := w.Create("OEBPS/content.opf")
	if err != nil {
		return err
	}
	io.WriteString(ow, opfContent)

	// 4. toc.ncx
	tw, err := w.Create("OEBPS/toc.ncx")
	if err != nil {
		return err
	}
	io.WriteString(tw, fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="%s"/>
  </head>
  <docTitle>
    <text>%s</text>
  </docTitle>
  <navMap>
    <navPoint id="chapter1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`, isbn, title))

	// 5. chapter1.xhtml
	chw, err := w.Create("OEBPS/chapter1.xhtml")
	if err != nil {
		return err
	}
	io.WriteString(chw, fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>%s</title>
</head>
<body>
  <h1>Chapter 1</h1>
  <p>This is a test chapter.</p>
</body>
</html>`, title))

	return nil
}
