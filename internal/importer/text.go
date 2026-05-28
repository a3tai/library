package importer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/a3tai/library/internal/library"
)

const chunkTarget = 1800

var whitespace = regexp.MustCompile(`\s+`)

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = whitespace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func chunkPassages(bookID, label, text string) []library.Passage {
	text = cleanText(text)
	if text == "" {
		return nil
	}
	var passages []library.Passage
	chunk := 0
	for len(text) > 0 {
		end := len(text)
		if end > chunkTarget {
			end = chunkTarget
			if boundary := strings.LastIndexAny(text[:end], ".!?\n;:"); boundary > chunkTarget/2 {
				end = boundary + 1
			} else if boundary := strings.LastIndex(text[:end], " "); boundary > chunkTarget/2 {
				end = boundary
			}
		}
		part := strings.TrimSpace(text[:end])
		if part != "" {
			passages = append(passages, library.Passage{
				ID:         fmt.Sprintf("%s-%06d", bookID, len(passages)+1),
				BookID:     bookID,
				Label:      label,
				ChunkIndex: chunk,
				Text:       part,
			})
			chunk++
		}
		text = strings.TrimSpace(text[end:])
	}
	return passages
}

func normalizePassages(bookID string, passages []library.Passage) []library.Passage {
	for index := range passages {
		passages[index].ID = fmt.Sprintf("%s-%06d", bookID, index+1)
		passages[index].BookID = bookID
		passages[index].ChunkIndex = index
	}
	return passages
}

func mergeMetadata(base, update library.Book) library.Book {
	if update.Title != "" && update.Title != "Untitled" {
		base.Title = update.Title
	}
	if update.Authors != "" {
		base.Authors = update.Authors
	}
	if update.Description != "" {
		base.Description = update.Description
	}
	if update.Publisher != "" {
		base.Publisher = update.Publisher
	}
	if update.PublishedDate != "" {
		base.PublishedDate = update.PublishedDate
	}
	if update.ISBN10 != "" {
		base.ISBN10 = update.ISBN10
	}
	if update.ISBN13 != "" {
		base.ISBN13 = update.ISBN13
	}
	if update.Language != "" {
		base.Language = update.Language
	}
	if update.CoverURL != "" {
		base.CoverURL = update.CoverURL
	}
	return base
}
