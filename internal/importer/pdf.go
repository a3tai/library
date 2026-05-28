package importer

import (
	"fmt"
	"strings"

	pdf "github.com/ledongthuc/pdf"

	"github.com/a3tai/library/internal/library"
)

func ImportPDF(filePath string) (library.ImportBook, error) {
	hash, err := fileHash(filePath)
	if err != nil {
		return library.ImportBook{}, err
	}
	bookID := hash
	book := library.Book{
		ID:          bookID,
		Title:       titleFromPath(filePath),
		FilePath:    filePath,
		Format:      "pdf",
		FileHash:    hash,
		IndexStatus: "indexed",
		TextStatus:  "available",
	}

	f, reader, err := pdf.Open(filePath)
	if err != nil {
		return library.ImportBook{}, err
	}
	defer f.Close()
	if reader.NumPage() > maxPDFPages {
		return library.ImportBook{}, fmt.Errorf("PDF has too many pages: %d (limit %d)", reader.NumPage(), maxPDFPages)
	}

	var passages []library.Passage
	for pageNo := 1; pageNo <= reader.NumPage(); pageNo++ {
		page := reader.Page(pageNo)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text = cleanText(text)
		if text == "" {
			continue
		}
		label := fmt.Sprintf("Page %d", pageNo)
		passages = append(passages, chunkPassages(bookID, label, strings.TrimSpace(text))...)
		if len(passages) > maxPassagesPerBook {
			return library.ImportBook{}, fmt.Errorf("too many passages extracted from PDF (limit %d)", maxPassagesPerBook)
		}
	}
	if len(passages) == 0 {
		book.TextStatus = "text_unavailable"
	}
	passages = normalizePassages(bookID, passages)
	return library.ImportBook{Book: book, Passages: passages}, nil
}
