package library

import "time"

type Book struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	Authors             string    `json:"authors"`
	Description         string    `json:"description"`
	Publisher           string    `json:"publisher"`
	PublishedDate       string    `json:"publishedDate"`
	ISBN10              string    `json:"isbn10"`
	ISBN13              string    `json:"isbn13"`
	Language            string    `json:"language"`
	CoverURL            string    `json:"coverUrl"`
	FilePath            string    `json:"filePath"`
	Format              string    `json:"format"`
	FileHash            string    `json:"fileHash"`
	FileSize            int64     `json:"fileSize"`
	MetadataSource      string    `json:"metadataSource"`
	MetadataRefreshedAt time.Time `json:"metadataRefreshedAt"`
	IndexStatus         string    `json:"indexStatus"`
	TextStatus          string    `json:"textStatus"`
	PassageCount        int       `json:"passageCount"`
	// Subjects and Categories are populated from the book_subjects /
	// book_categories join tables on read. They're omitempty so callers who
	// don't load them (e.g. inventory writers) don't carry empty arrays.
	Subjects   []string  `json:"subjects,omitempty"`
	Categories []string  `json:"categories,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Passage struct {
	ID         string `json:"id"`
	BookID     string `json:"bookId"`
	BookTitle  string `json:"bookTitle,omitempty"`
	Authors    string `json:"authors,omitempty"`
	Label      string `json:"label"`
	ChunkIndex int    `json:"chunkIndex"`
	Text       string `json:"text"`
	Snippet    string `json:"snippet,omitempty"`
}

type ImportBook struct {
	Book     Book
	Passages []Passage
}

type ImportSummary struct {
	Imported int      `json:"imported"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Messages []string `json:"messages"`
}

type Stats struct {
	Books         int `json:"books"`
	Passages      int `json:"passages"`
	NeedsMetadata int `json:"needsMetadata"`
}

// TOCEntry represents a single contiguous span of passages sharing the same
// label. For EPUBs that's typically a chapter or spine document; for PDFs it
// usually maps to a page section. ChunkIndex is the index of the first
// passage in the span, suitable for jumping into the reader.
type TOCEntry struct {
	Label      string `json:"label"`
	ChunkIndex int    `json:"chunkIndex"`
	Pages      int    `json:"pages"`
}
