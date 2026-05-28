package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/a3tai/library/internal/library"
)

// ImportText handles plain-text book dumps such as the *.epub.txt files in the
// books1/books3 corpora. Filenames may follow "Title - Author.epub.txt"; if so,
// the parts are extracted heuristically. Otherwise only the title is set and
// metadata enrichment is left to fill in the author.
func ImportText(path string) (library.ImportBook, error) {
	hash, err := fileHash(path)
	if err != nil {
		return library.ImportBook{}, err
	}
	info, err := importFileInfo(path)
	if err != nil {
		return library.ImportBook{}, err
	}
	if info.Size() > maxTextFileBytes {
		return library.ImportBook{}, fmt.Errorf("text file is too large: %d bytes (limit %d)", info.Size(), maxTextFileBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return library.ImportBook{}, err
	}

	title, authors := parseTextFilename(path)

	bookID := hash
	book := library.Book{
		ID:          bookID,
		Title:       title,
		Authors:     authors,
		FilePath:    path,
		Format:      textFormat(path),
		FileHash:    hash,
		IndexStatus: "indexed",
		TextStatus:  "available",
	}

	passages := chunkPassages(bookID, "", string(data))
	if len(passages) > maxPassagesPerBook {
		return library.ImportBook{}, fmt.Errorf("too many passages extracted from text file (limit %d)", maxPassagesPerBook)
	}
	if len(passages) == 0 {
		book.TextStatus = "text_unavailable"
	}
	passages = normalizePassages(bookID, passages)
	return library.ImportBook{Book: book, Passages: passages}, nil
}

// parseTextFilename derives a (title, authors) pair from a filename. It strips
// the .epub.txt or .txt extension, replaces underscores/multiple spaces, and
// when the remainder contains " - " splits left/right as title/authors. A
// trailing "_nodrm" tag (common in scraped dumps) is dropped.
func parseTextFilename(path string) (string, string) {
	name := filepath.Base(path)
	name = strings.TrimSuffix(strings.ToLower(name), ".txt")
	if strings.HasSuffix(name, ".epub") {
		name = strings.TrimSuffix(name, ".epub")
	}
	// Reload original casing now that the suffix is known.
	original := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.HasSuffix(strings.ToLower(original), ".epub") {
		original = original[:len(original)-len(".epub")]
	}
	original = strings.TrimSpace(strings.TrimSuffix(original, "_nodrm"))

	if i := strings.LastIndex(original, " - "); i > 0 {
		title := strings.TrimSpace(original[:i])
		author := strings.TrimSpace(original[i+3:])
		// Calibre-style "Last, First; Last2, First2"
		author = strings.ReplaceAll(author, ";", ",")
		author = strings.Join(splitAndTrim(author, ","), ", ")
		if title != "" && author != "" {
			return desugarSlug(title), desugarSlug(author)
		}
	}
	return desugarSlug(original), ""
}

// desugarSlug turns "100-incredible-happiness-hacks" into
// "100 Incredible Happiness Hacks". Leaves already-spaced strings alone.
func desugarSlug(s string) string {
	if s == "" {
		return "Untitled"
	}
	if !strings.ContainsAny(s, "-_") || strings.Contains(s, " ") {
		return s
	}
	r := strings.NewReplacer("_", " ", "-", " ")
	parts := strings.Fields(r.Replace(s))
	for i, p := range parts {
		parts[i] = titleCase(p)
	}
	return strings.Join(parts, " ")
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	if len(s) <= 3 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func textFormat(path string) string {
	lower := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(lower, ".epub.txt") {
		return "epubtxt"
	}
	return "txt"
}
