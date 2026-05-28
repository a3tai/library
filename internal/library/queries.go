package library

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func (s *Store) ListBooks(ctx context.Context, query string, limit, offset int) ([]Book, error) {
	limit = saneLimit(limit)
	if offset < 0 {
		offset = 0
	}
	if strings.TrimSpace(query) != "" {
		return s.SearchBooks(ctx, query, limit, offset)
	}
	rows, err := s.db.QueryContext(ctx, bookSelect()+` ORDER BY b.updated_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}

func (s *Store) SearchBooks(ctx context.Context, query string, limit, offset int) ([]Book, error) {
	match := ftsQuery(query)
	if match == "" {
		return s.ListBooks(ctx, "", limit, offset)
	}
	limit = saneLimit(limit)
	rows, err := s.db.QueryContext(ctx, bookSelect()+` JOIN book_search bs ON bs.book_id = b.id WHERE book_search MATCH ? ORDER BY rank LIMIT ? OFFSET ?`, match, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}

func (s *Store) SearchPassages(ctx context.Context, query, bookID string, limit int) ([]Passage, error) {
	match := ftsQuery(query)
	if match == "" {
		return nil, nil
	}
	limit = saneLimit(limit)
	whereBook := ""
	args := []any{match}
	if strings.TrimSpace(bookID) != "" {
		whereBook = " AND ps.book_id = ?"
		args = append(args, bookID)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT p.id, p.book_id, b.title, b.authors, p.label, p.chunk_index, p.text,
		snippet(passage_search, 3, '', '', '...', 22) AS snippet_text
		FROM passage_search ps
		JOIN passages p ON p.id = ps.passage_id
		JOIN books b ON b.id = p.book_id
		WHERE passage_search MATCH ?`+whereBook+` ORDER BY rank LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	passages := []Passage{}
	for rows.Next() {
		var p Passage
		if err := rows.Scan(&p.ID, &p.BookID, &p.BookTitle, &p.Authors, &p.Label, &p.ChunkIndex, &p.Text, &p.Snippet); err != nil {
			return nil, err
		}
		passages = append(passages, p)
	}
	return passages, rows.Err()
}

func (s *Store) GetBook(ctx context.Context, id string) (Book, error) {
	row := s.db.QueryRowContext(ctx, bookSelect()+` WHERE b.id = ?`, id)
	book, err := scanBook(row)
	if err != nil {
		return Book{}, err
	}
	// Populate the join-table fields. List queries skip this — the BookCard
	// view doesn't render them, and a per-row N+1 would be wasteful.
	if subjects, err := s.bookChildren(ctx, "subject", "book_subjects", id); err == nil {
		book.Subjects = subjects
	}
	if cats, err := s.bookChildren(ctx, "category", "book_categories", id); err == nil {
		book.Categories = cats
	}
	return book, nil
}

// bookChildren reads ordered values from a (book_id, value) join table.
func (s *Store) bookChildren(ctx context.Context, col, table, bookID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT %s FROM %s WHERE book_id = ? ORDER BY %s`, col, table, col),
		bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetPassage(ctx context.Context, id string) (Passage, error) {
	row := s.db.QueryRowContext(ctx, `SELECT p.id, p.book_id, b.title, b.authors, p.label, p.chunk_index, p.text
		FROM passages p JOIN books b ON b.id = p.book_id WHERE p.id = ?`, id)
	var p Passage
	if err := row.Scan(&p.ID, &p.BookID, &p.BookTitle, &p.Authors, &p.Label, &p.ChunkIndex, &p.Text); err != nil {
		return Passage{}, err
	}
	return p, nil
}

// BookPassages returns passages belonging to a book in stored order.
// Pagination uses chunk_index ASC so callers can scroll a reader without
// surprises. limit defaults to 1000 when ≤ 0; offset is clamped to 0.
func (s *Store) BookPassages(ctx context.Context, bookID string, offset, limit int) ([]Passage, error) {
	if limit <= 0 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id, p.book_id, b.title, b.authors, p.label, p.chunk_index, p.text
		FROM passages p JOIN books b ON b.id = p.book_id
		WHERE p.book_id = ?
		ORDER BY p.chunk_index ASC
		LIMIT ? OFFSET ?`, bookID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	passages := []Passage{}
	for rows.Next() {
		var p Passage
		if err := rows.Scan(&p.ID, &p.BookID, &p.BookTitle, &p.Authors, &p.Label, &p.ChunkIndex, &p.Text); err != nil {
			return nil, err
		}
		passages = append(passages, p)
	}
	return passages, rows.Err()
}

// BookTOC groups consecutive passages by label and returns the first
// chunk_index of each span plus the count. Spans with empty labels are
// dropped — callers can fall back to the raw passage list when no entries
// are returned.
func (s *Store) BookTOC(ctx context.Context, bookID string) ([]TOCEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT label, chunk_index
		FROM passages WHERE book_id = ? ORDER BY chunk_index ASC`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TOCEntry
	var current *TOCEntry
	for rows.Next() {
		var label string
		var idx int
		if err := rows.Scan(&label, &idx); err != nil {
			return nil, err
		}
		label = strings.TrimSpace(label)
		if label == "" {
			current = nil
			continue
		}
		if current != nil && current.Label == label {
			current.Pages++
			continue
		}
		entries = append(entries, TOCEntry{Label: label, ChunkIndex: idx, Pages: 1})
		current = &entries[len(entries)-1]
	}
	if entries == nil {
		entries = []TOCEntry{}
	}
	return entries, rows.Err()
}

func (s *Store) BooksNeedingMetadata(ctx context.Context, limit int) ([]Book, error) {
	limit = saneLimit(limit)
	rows, err := s.db.QueryContext(ctx, bookSelect()+` WHERE (b.metadata_source = '' OR b.cover_url = '' OR b.description = '') ORDER BY b.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}

// BooksNeedingEnrichment returns up to `limit` books that the background
// scanner should hand to the enricher. A book qualifies if it is missing
// basic metadata (no source / cover / description) OR it is fully indexed
// but has not yet been through AI enrichment (description has no
// "Subjects:" marker emitted by the metadata client and the AI step).
//
// The result is ordered to favour books the user just touched (recent
// updated_at) so visible items get fixed first.
func (s *Store) BooksNeedingEnrichment(ctx context.Context, limit int) ([]Book, error) {
	limit = saneLimit(limit)
	rows, err := s.db.QueryContext(ctx, bookSelect()+`
		WHERE b.needs_enrichment = 1
		ORDER BY b.updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}

// CountBooksNeedingEnrichment returns the total number of books still
// pending enrichment. The scanner uses it to decide between work and idle
// sleep cadences without pulling rows it isn't going to enqueue.
func (s *Store) CountBooksNeedingEnrichment(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM books WHERE needs_enrichment = 1`).Scan(&n)
	return n, err
}

func (s *Store) UpdateMetadata(ctx context.Context, book Book) error {
	book.MetadataRefreshedAt = time.Now().UTC()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `UPDATE books SET
		title = CASE WHEN ? != '' THEN ? ELSE title END,
		authors = CASE WHEN ? != '' THEN ? ELSE authors END,
		description = CASE WHEN ? != '' THEN ? ELSE description END,
		publisher = CASE WHEN ? != '' THEN ? ELSE publisher END,
		published_date = CASE WHEN ? != '' THEN ? ELSE published_date END,
		isbn10 = CASE WHEN ? != '' THEN ? ELSE isbn10 END,
		isbn13 = CASE WHEN ? != '' THEN ? ELSE isbn13 END,
		language = CASE WHEN ? != '' THEN ? ELSE language END,
		cover_url = CASE WHEN ? != '' THEN ? ELSE cover_url END,
		metadata_source = ?, metadata_refreshed_at = ?, updated_at = ?
		WHERE id = ?`, book.Title, book.Title, book.Authors, book.Authors, book.Description, book.Description,
		book.Publisher, book.Publisher, book.PublishedDate, book.PublishedDate, book.ISBN10, book.ISBN10,
		book.ISBN13, book.ISBN13, book.Language, book.Language, book.CoverURL, book.CoverURL,
		book.MetadataSource, book.MetadataRefreshedAt, book.MetadataRefreshedAt, book.ID)
	if err != nil {
		return err
	}
	// Refresh FTS through the helper so subjects/categories aren't clobbered.
	if err := s.refreshBookFTS(ctx, s.db, book.ID); err != nil {
		return err
	}
	// Refresh the cached needs_enrichment flag using the post-update row.
	// "Has subjects" is now sourced from book_subjects rather than a substring
	// search of the description blob.
	_, err = s.db.ExecContext(ctx, `UPDATE books SET needs_enrichment = CASE
		WHEN metadata_source = '' OR cover_url = '' OR description = '' THEN 1
		WHEN index_status = 'indexed' AND passage_count > 0 AND NOT EXISTS(
			SELECT 1 FROM book_subjects WHERE book_id = books.id) THEN 1
		ELSE 0 END WHERE id = ?`, book.ID)
	return err
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&stats.Books); err != nil {
		return stats, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM passages`).Scan(&stats.Passages); err != nil {
		return stats, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books WHERE metadata_source = '' OR cover_url = '' OR description = ''`).Scan(&stats.NeedsMetadata); err != nil {
		return stats, err
	}
	return stats, nil
}

func bookSelect() string {
	return `SELECT b.id, b.title, b.authors, b.description, b.publisher, b.published_date, b.isbn10, b.isbn13,
		b.language, b.cover_url, b.file_path, b.format, b.file_hash, b.file_size, b.metadata_source, b.metadata_refreshed_at,
		b.index_status, b.text_status, b.passage_count,
		b.created_at, b.updated_at FROM books b`
}

type scanner interface {
	Scan(dest ...any) error
}

func scanBook(row scanner) (Book, error) {
	var book Book
	var refreshed sql.NullTime
	if err := row.Scan(&book.ID, &book.Title, &book.Authors, &book.Description, &book.Publisher, &book.PublishedDate,
		&book.ISBN10, &book.ISBN13, &book.Language, &book.CoverURL, &book.FilePath, &book.Format, &book.FileHash,
		&book.FileSize, &book.MetadataSource, &refreshed, &book.IndexStatus, &book.TextStatus, &book.PassageCount, &book.CreatedAt, &book.UpdatedAt); err != nil {
		return Book{}, err
	}
	if refreshed.Valid {
		book.MetadataRefreshedAt = refreshed.Time
	}
	return book, nil
}

func scanBooks(rows *sql.Rows) ([]Book, error) {
	books := []Book{}
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	return books, rows.Err()
}

func saneLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

var ftsToken = regexp.MustCompile(`[\pL\pN]+`)

// ftsFieldAlias maps the user-facing field name to the FTS column name.
// "subject" routes into description because that's where the metadata
// pipeline currently writes "Subjects: …" lines (AI + Open Library).
// When the schema gets a dedicated subjects column, just point this at it.
var ftsFieldAlias = map[string]string{
	"title":   "title",
	"author":  "authors",
	"authors": "authors",
	"by":      "authors",
	"subject": "description",
	"topic":   "description",
	"genre":   "description",
	"about":   "description",
	"desc":    "description",
}

// fieldQualifier matches "key:value" prefixes, where value is a single token
// or a quoted phrase. Quoted phrases are passed through verbatim (FTS5
// supports phrase prefix only on bare terms, so we drop the wildcard for
// phrases). Field qualifiers from outside ftsFieldAlias degrade to
// untargeted free-text search of that token.
var fieldQualifier = regexp.MustCompile(`(?i)([a-z]+):("[^"]+"|\S+)`)

// ftsQuery turns a user query into a valid FTS5 MATCH expression. It
// supports field-qualified terms (`author:turing`, `subject:halting`) by
// rewriting them as column-qualified FTS5 queries; any unqualified tokens
// remain untargeted prefix matches against all columns, joined with the
// implicit AND.
func ftsQuery(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	var clauses []string
	// First: peel off any field-qualified terms.
	rest := fieldQualifier.ReplaceAllStringFunc(q, func(match string) string {
		m := fieldQualifier.FindStringSubmatch(match)
		if len(m) != 3 {
			return ""
		}
		key := strings.ToLower(m[1])
		raw := strings.TrimSpace(m[2])
		col, ok := ftsFieldAlias[key]
		if !ok {
			// Not a recognized qualifier — leave as plain text in `rest`.
			return raw
		}
		// Quoted phrase: pass through (no prefix wildcard, FTS5 limitation).
		if strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`) && len(raw) >= 2 {
			phrase := strings.ToLower(strings.Trim(raw, `"`))
			if strings.TrimSpace(phrase) == "" {
				return ""
			}
			clauses = append(clauses, fmt.Sprintf(`%s:"%s"`, col, escapeFTSPhrase(phrase)))
			return ""
		}
		// Bare token: lowercase, alphanumeric, prefix match.
		token := strings.ToLower(raw)
		toks := ftsToken.FindAllString(token, -1)
		if len(toks) == 0 {
			return ""
		}
		for _, t := range toks {
			if len(t) < 2 {
				continue
			}
			clauses = append(clauses, fmt.Sprintf("%s:%s*", col, t))
		}
		return ""
	})

	// Second: any leftover free-text tokens.
	for _, part := range ftsToken.FindAllString(strings.ToLower(rest), -1) {
		if len(part) < 2 {
			continue
		}
		clauses = append(clauses, part+"*")
	}
	return strings.Join(clauses, " ")
}

func escapeFTSPhrase(s string) string {
	return strings.ReplaceAll(s, `"`, `""`)
}
