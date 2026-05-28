package importer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html"

	"github.com/a3tai/library/internal/library"
)

type containerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	Manifest []opfManifestItem `xml:"manifest>item"`
	Spine    []opfSpineItem    `xml:"spine>itemref"`
}

type opfManifestItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
	Title     string `xml:"title,attr"`
}

type opfSpineItem struct {
	IDRef string `xml:"idref,attr"`
}

func ImportEPUB(filePath string) (library.ImportBook, error) {
	hash, err := fileHash(filePath)
	if err != nil {
		return library.ImportBook{}, err
	}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return library.ImportBook{}, err
	}
	defer r.Close()

	root, err := epubRootfile(&r.Reader)
	if err != nil {
		return library.ImportBook{}, err
	}
	opfData, err := readZipFile(&r.Reader, root)
	if err != nil {
		return library.ImportBook{}, err
	}
	metadata := parseOPFMetadata(opfData)
	var pkg opfPackage
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return library.ImportBook{}, err
	}

	bookID := hash
	book := library.Book{
		ID:          bookID,
		Title:       titleFromPath(filePath),
		FilePath:    filePath,
		Format:      "epub",
		FileHash:    hash,
		IndexStatus: "indexed",
		TextStatus:  "available",
	}
	book = mergeMetadata(book, metadata)

	baseDir := path.Dir(root)
	manifest := map[string]opfManifestItem{}
	for _, item := range pkg.Manifest {
		manifest[item.ID] = item
	}
	var ordered []opfManifestItem
	for _, spineItem := range pkg.Spine {
		item, ok := manifest[spineItem.IDRef]
		if ok && isEpubDocument(item.MediaType, item.Href) {
			ordered = append(ordered, item)
		}
	}
	if len(ordered) == 0 {
		for _, item := range pkg.Manifest {
			if isEpubDocument(item.MediaType, item.Href) {
				ordered = append(ordered, item)
			}
		}
		sort.Slice(ordered, func(a, b int) bool { return ordered[a].Href < ordered[b].Href })
	}

	var passages []library.Passage
	for _, item := range ordered {
		name := path.Clean(path.Join(baseDir, item.Href))
		data, err := readZipFile(&r.Reader, name)
		if err != nil {
			continue
		}
		text := htmlText(data)
		label := strings.TrimSpace(item.Title)
		if label == "" {
			label = strings.TrimSuffix(path.Base(item.Href), path.Ext(item.Href))
		}
		passages = append(passages, chunkPassages(bookID, label, text)...)
	}
	if len(passages) == 0 {
		book.TextStatus = "text_unavailable"
	}
	passages = normalizePassages(bookID, passages)
	return library.ImportBook{Book: book, Passages: passages}, nil
}

func epubRootfile(r *zip.Reader) (string, error) {
	data, err := readZipFile(r, "META-INF/container.xml")
	if err != nil {
		return "", err
	}
	var container containerXML
	if err := xml.Unmarshal(data, &container); err != nil {
		return "", err
	}
	if len(container.Rootfiles) == 0 || container.Rootfiles[0].FullPath == "" {
		return "", fmt.Errorf("EPUB container has no rootfile")
	}
	return container.Rootfiles[0].FullPath, nil
}

func readZipFile(r *zip.Reader, name string) ([]byte, error) {
	name = path.Clean(name)
	for _, file := range r.File {
		if path.Clean(file.Name) != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("%s not found in EPUB", name)
}

func isEpubDocument(mediaType, href string) bool {
	mediaType = strings.ToLower(mediaType)
	ext := strings.ToLower(filepath.Ext(href))
	return strings.Contains(mediaType, "html") || ext == ".xhtml" || ext == ".html" || ext == ".htm"
}

func htmlText(data []byte) string {
	node, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return cleanText(string(data))
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "svg", "math":
				return
			case "p", "br", "section", "article", "h1", "h2", "h3", "h4", "li":
				b.WriteByte('\n')
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return cleanText(b.String())
}

func parseOPFMetadata(data []byte) library.Book {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var book library.Book
	inMetadata := false
	var authors []string
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "metadata" {
				inMetadata = true
				continue
			}
			if !inMetadata {
				continue
			}
			var value string
			if err := decoder.DecodeElement(&value, &t); err != nil {
				continue
			}
			value = cleanText(value)
			switch strings.ToLower(t.Name.Local) {
			case "title":
				if book.Title == "" {
					book.Title = value
				}
			case "creator":
				if value != "" {
					authors = append(authors, value)
				}
			case "description":
				if book.Description == "" {
					book.Description = value
				}
			case "publisher":
				book.Publisher = value
			case "date":
				book.PublishedDate = value
			case "language":
				book.Language = value
			case "identifier":
				assignISBN(&book, value)
			}
		case xml.EndElement:
			if t.Name.Local == "metadata" {
				inMetadata = false
			}
		}
	}
	book.Authors = strings.Join(dedupe(authors), ", ")
	return book
}

var isbnDigits = regexp.MustCompile(`[0-9Xx]+`)

func assignISBN(book *library.Book, raw string) {
	parts := isbnDigits.FindAllString(raw, -1)
	joined := strings.ToUpper(strings.Join(parts, ""))
	if len(joined) == 10 && book.ISBN10 == "" {
		book.ISBN10 = joined
	}
	if len(joined) == 13 && book.ISBN13 == "" {
		book.ISBN13 = joined
	}
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, strings.TrimSpace(value))
	}
	return out
}
