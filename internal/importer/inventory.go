package importer

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pdf "github.com/ledongthuc/pdf"

	"github.com/a3tai/library/internal/library"
)

// Inventory builds a Book row from the file's metadata WITHOUT extracting
// passages. This is Pass 1 of the multi-pass import: cheap enough that the
// user sees their entire library populate in seconds, with text + FTS
// happening later in Pass 2.
//
// The file is hashed (book id), stat'd (file size), and where possible the
// embedded metadata is peeked at — EPUB OPF for title/author/ISBN, filename
// heuristic everywhere else. PDFs intentionally skip the info dictionary
// here because reading it requires opening the document and we'd rather
// defer that work to Pass 2.
func Inventory(filePath string) (library.Book, error) {
	// Inventory uses a partial-content hash (size + first 64 KiB + last
	// 64 KiB) instead of full SHA-256 — enough to dedup a real library at
	// a fraction of the I/O cost. Pass 2 (IndexBook) keeps the same id
	// because it just re-uses the row's existing book.ID.
	hash, err := partialFileHash(filePath)
	if err != nil {
		return library.Book{}, err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return library.Book{}, err
	}

	book := library.Book{
		ID:          hash,
		FilePath:    filePath,
		FileHash:    hash,
		FileSize:    info.Size(),
		IndexStatus: "queued",
		TextStatus:  "available",
		Title:       titleFromPath(filePath),
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".epub":
		book.Format = "epub"
		fillEPUBInventory(filePath, &book)
	case ".pdf":
		book.Format = "pdf"
		fillPDFInventory(filePath, &book)
	case ".txt":
		book.Format = textFormat(filePath)
		t, a := parseTextFilename(filePath)
		if t != "" {
			book.Title = t
		}
		if a != "" {
			book.Authors = a
		}
	default:
		return library.Book{}, fmt.Errorf("unsupported file type: %s", filepath.Ext(filePath))
	}

	if strings.TrimSpace(book.Title) == "" {
		book.Title = titleFromPath(filePath)
	}
	return book, nil
}

// fillEPUBInventory peeks at META-INF/container.xml + the OPF rootfile and
// populates whatever metadata is present. Errors are swallowed because
// "couldn't read OPF" should not fail Pass 1 — we still want the file in the
// library, just with the filename-derived title.
func fillEPUBInventory(filePath string, book *library.Book) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return
	}
	defer r.Close()

	root, err := epubRootfile(&r.Reader)
	if err != nil {
		return
	}
	opfData, err := readZipFile(&r.Reader, root)
	if err != nil {
		return
	}
	meta := parseOPFMetadata(opfData)
	if meta.Title != "" {
		book.Title = meta.Title
	}
	if meta.Authors != "" {
		book.Authors = meta.Authors
	}
	if meta.Description != "" {
		book.Description = meta.Description
	}
	if meta.Publisher != "" {
		book.Publisher = meta.Publisher
	}
	if meta.PublishedDate != "" {
		book.PublishedDate = meta.PublishedDate
	}
	if meta.Language != "" {
		book.Language = meta.Language
	}
	if meta.ISBN10 != "" {
		book.ISBN10 = meta.ISBN10
	}
	if meta.ISBN13 != "" {
		book.ISBN13 = meta.ISBN13
	}
}

// fillPDFInventory tries the PDF Info dictionary (Title/Author/Subject/etc.)
// using the existing PDF library. Cheap relative to text extraction since we
// only touch the trailer; per-page text extraction happens in Pass 2.
func fillPDFInventory(filePath string, book *library.Book) {
	f, reader, err := pdf.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	trailer := reader.Trailer()
	info := trailer.Key("Info")
	if info.IsNull() {
		return
	}
	if t := strings.TrimSpace(info.Key("Title").Text()); t != "" {
		book.Title = t
	}
	if a := strings.TrimSpace(info.Key("Author").Text()); a != "" {
		book.Authors = a
	}
	if s := strings.TrimSpace(info.Key("Subject").Text()); s != "" {
		book.Description = s
	}
	if d := strings.TrimSpace(info.Key("CreationDate").Text()); d != "" {
		book.PublishedDate = pdfDate(d)
	}
}

// pdfDate trims PDF's "D:YYYYMMDDHHMMSS..." prefix down to YYYY-MM-DD or YYYY.
func pdfDate(raw string) string {
	s := strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(raw), "D:"))
	if len(s) >= 8 {
		return fmt.Sprintf("%s-%s-%s", s[:4], s[4:6], s[6:8])
	}
	if len(s) >= 4 {
		return s[:4]
	}
	return ""
}

// IndexBook reparses the on-disk file and returns its passages. Pass 2 of
// the multi-pass import flow: the heavy work (HTML/PDF text extraction +
// chunking) lives here so the inventory pass can stay snappy.
func IndexBook(book library.Book) ([]library.Passage, error) {
	imported, err := ImportFile(book.FilePath)
	if err != nil {
		return nil, err
	}
	// Re-key passages to the existing book id (in case the file's hash
	// changed mid-flight, which would mint a new id under our normal upsert).
	for i := range imported.Passages {
		imported.Passages[i].BookID = book.ID
		imported.Passages[i].ID = sprintfPassageID(book.ID, i+1)
	}
	return imported.Passages, nil
}

func sprintfPassageID(bookID string, n int) string {
	return fmt.Sprintf("%s-%06d", bookID, n)
}

// (kept here for symmetry with ImportFile / Inventory) — XML decoding helpers
// reused by fillEPUBInventory live in epub.go via parseOPFMetadata.
var _ = xml.Unmarshal
