package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Store) IsIndexed(ctx context.Context, path string, size int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM books WHERE file_path = ? AND file_size = ? AND index_status = 'indexed' LIMIT 1`,
		path, size).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// IndexedPaths returns the set of (path, size) pairs from `paths` that are
// already fully indexed. Lets the importer skip those files in a single
// query instead of issuing IsIndexed per file. Map key is "<path>\x00<size>".
func (s *Store) IndexedPaths(ctx context.Context, paths []string, sizes []int64) (map[string]bool, error) {
	if len(paths) == 0 {
		return map[string]bool{}, nil
	}
	if len(paths) != len(sizes) {
		return nil, fmt.Errorf("paths/sizes length mismatch")
	}
	// Chunk to stay well under SQLite's compile-time SQLITE_MAX_VARIABLE_NUMBER.
	out := map[string]bool{}
	const chunk = 400
	for i := 0; i < len(paths); i += chunk {
		end := i + chunk
		if end > len(paths) {
			end = len(paths)
		}
		placeholders := make([]string, 0, end-i)
		args := make([]any, 0, end-i)
		for j := i; j < end; j++ {
			placeholders = append(placeholders, "?")
			args = append(args, paths[j])
		}
		query := `SELECT file_path, file_size FROM books
			WHERE index_status = 'indexed' AND file_path IN (` +
			strings.Join(placeholders, ",") + `)`
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		want := map[string]int64{}
		for j := i; j < end; j++ {
			want[paths[j]] = sizes[j]
		}
		for rows.Next() {
			var p string
			var sz int64
			if err := rows.Scan(&p, &sz); err != nil {
				rows.Close()
				return nil, err
			}
			if w, ok := want[p]; ok && w == sz {
				out[indexedKey(p, sz)] = true
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// indexedKey is the lookup string used by IndexedPaths' return map.
func indexedKey(path string, size int64) string {
	return path + "\x00" + strconv.FormatInt(size, 10)
}

// UpsertInventoryBook writes a book row WITHOUT touching passages or FTS.
// Used by Pass 1 of the importer: just establish the file's existence in the
// library so the user sees it immediately. The row is marked
// index_status="queued" so the indexer worker will pick it up for Pass 2.
// Returns (existed bool, err) — existed indicates we updated rather than inserted.
func (s *Store) UpsertInventoryBook(ctx context.Context, book Book) (bool, error) {
	if book.ID == "" {
		return false, errors.New("book id is required")
	}
	now := time.Now().UTC()
	book.UpdatedAt = now
	if book.CreatedAt.IsZero() {
		book.CreatedAt = now
	}
	if book.IndexStatus == "" {
		book.IndexStatus = "queued"
	}
	if book.TextStatus == "" {
		book.TextStatus = "available"
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var existed int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM books WHERE id = ?`, book.ID).Scan(&existed); err != nil {
		return false, err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO books (
		id, title, authors, description, publisher, published_date, isbn10, isbn13, language, cover_url,
		file_path, format, file_hash, file_size, metadata_source, metadata_refreshed_at, index_status, text_status, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title=CASE WHEN books.title != '' AND excluded.title = '' THEN books.title ELSE excluded.title END,
		authors=CASE WHEN books.authors != '' AND excluded.authors = '' THEN books.authors ELSE excluded.authors END,
		file_path=excluded.file_path,
		format=excluded.format,
		file_hash=excluded.file_hash,
		file_size=excluded.file_size,
		text_status=excluded.text_status,
		updated_at=excluded.updated_at`,
		book.ID, book.Title, book.Authors, book.Description, book.Publisher, book.PublishedDate,
		book.ISBN10, book.ISBN13, book.Language, book.CoverURL, book.FilePath, book.Format,
		book.FileHash, book.FileSize, book.MetadataSource, nullTime(book.MetadataRefreshedAt),
		book.IndexStatus, book.TextStatus, book.CreatedAt, book.UpdatedAt)
	if err != nil {
		return false, err
	}

	// Keep the FTS row in sync so the title is searchable even before Pass 2.
	if _, err := tx.ExecContext(ctx, `DELETE FROM book_search WHERE book_id = ?`, book.ID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO book_search(book_id, title, authors, description) VALUES (?, ?, ?, ?)`,
		book.ID, book.Title, book.Authors, book.Description); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return existed > 0, nil
}

// UpsertInventoryBookResult is one entry in the result of a batched
// inventory upsert. Existed=true means the row was already present (we
// updated); Existed=false means we inserted. Err is set when this
// individual book failed validation; the rest of the batch still commits.
type UpsertInventoryBookResult struct {
	ID      string
	Existed bool
	Err     error
}

// UpsertInventoryBooks writes a batch of inventory rows in a single
// transaction. Replaces N separate BeginTx → Insert → Commit cycles with
// one transaction + one fsync for the whole batch — the dominant cost
// during large imports.
func (s *Store) UpsertInventoryBooks(ctx context.Context, books []Book) ([]UpsertInventoryBookResult, error) {
	if len(books) == 0 {
		return nil, nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Prepare statements once and reuse for every book in the batch.
	existsStmt, err := tx.PrepareContext(ctx, `SELECT 1 FROM books WHERE id = ?`)
	if err != nil {
		return nil, err
	}
	defer existsStmt.Close()
	insertStmt, err := tx.PrepareContext(ctx, `INSERT INTO books (
		id, title, authors, description, publisher, published_date, isbn10, isbn13, language, cover_url,
		file_path, format, file_hash, file_size, metadata_source, metadata_refreshed_at, index_status, text_status, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title=CASE WHEN books.title != '' AND excluded.title = '' THEN books.title ELSE excluded.title END,
		authors=CASE WHEN books.authors != '' AND excluded.authors = '' THEN books.authors ELSE excluded.authors END,
		file_path=excluded.file_path,
		format=excluded.format,
		file_hash=excluded.file_hash,
		file_size=excluded.file_size,
		text_status=excluded.text_status,
		updated_at=excluded.updated_at`)
	if err != nil {
		return nil, err
	}
	defer insertStmt.Close()
	deleteFTSStmt, err := tx.PrepareContext(ctx, `DELETE FROM book_search WHERE book_id = ?`)
	if err != nil {
		return nil, err
	}
	defer deleteFTSStmt.Close()
	insertFTSStmt, err := tx.PrepareContext(ctx, `INSERT INTO book_search(book_id, title, authors, description) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	defer insertFTSStmt.Close()

	results := make([]UpsertInventoryBookResult, 0, len(books))
	now := time.Now().UTC()
	for _, book := range books {
		if book.ID == "" {
			results = append(results, UpsertInventoryBookResult{Err: errors.New("book id is required")})
			continue
		}
		book.UpdatedAt = now
		if book.CreatedAt.IsZero() {
			book.CreatedAt = now
		}
		if book.IndexStatus == "" {
			book.IndexStatus = "queued"
		}
		if book.TextStatus == "" {
			book.TextStatus = "available"
		}

		var existedRow int
		switch err := existsStmt.QueryRowContext(ctx, book.ID).Scan(&existedRow); {
		case err == sql.ErrNoRows:
			existedRow = 0
		case err != nil:
			return nil, err
		}

		if _, err := insertStmt.ExecContext(ctx,
			book.ID, book.Title, book.Authors, book.Description, book.Publisher, book.PublishedDate,
			book.ISBN10, book.ISBN13, book.Language, book.CoverURL, book.FilePath, book.Format,
			book.FileHash, book.FileSize, book.MetadataSource, nullTime(book.MetadataRefreshedAt),
			book.IndexStatus, book.TextStatus, book.CreatedAt, book.UpdatedAt); err != nil {
			return nil, err
		}
		if _, err := deleteFTSStmt.ExecContext(ctx, book.ID); err != nil {
			return nil, err
		}
		if _, err := insertFTSStmt.ExecContext(ctx, book.ID, book.Title, book.Authors, book.Description); err != nil {
			return nil, err
		}
		results = append(results, UpsertInventoryBookResult{ID: book.ID, Existed: existedRow > 0})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return results, nil
}

// IndexBookPassages writes passages + FTS for a book that already has its
// row from Pass 1, and marks index_status="indexed". Does not touch the
// book's metadata fields.
func (s *Store) IndexBookPassages(ctx context.Context, bookID string, passages []Passage) error {
	if bookID == "" {
		return errors.New("book id is required")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM passages WHERE book_id = ?`, bookID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM passage_search WHERE book_id = ?`, bookID); err != nil {
		return err
	}

	textStatus := "available"
	if len(passages) == 0 {
		textStatus = "text_unavailable"
	}

	// Prepare the per-row insert statements once and reuse — far cheaper
	// than re-parsing the same SQL for every passage in a long book.
	pStmt, err := tx.PrepareContext(ctx, `INSERT INTO passages(id, book_id, label, chunk_index, text) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer pStmt.Close()
	psStmt, err := tx.PrepareContext(ctx, `INSERT INTO passage_search(passage_id, book_id, label, text) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer psStmt.Close()

	written := 0
	for _, p := range passages {
		if p.ID == "" || p.Text == "" {
			continue
		}
		if _, err := pStmt.ExecContext(ctx, p.ID, bookID, p.Label, p.ChunkIndex, p.Text); err != nil {
			return err
		}
		if _, err := psStmt.ExecContext(ctx, p.ID, bookID, p.Label, p.Text); err != nil {
			return err
		}
		written++
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE books SET index_status = 'indexed', text_status = ?, passage_count = ?, updated_at = ? WHERE id = ?`,
		textStatus, written, time.Now().UTC(), bookID); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkIndexFailed flags a book whose Pass 2 indexing errored so the worker
// doesn't loop on it. Stored as index_status="failed".
func (s *Store) MarkIndexFailed(ctx context.Context, bookID string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx,
		`UPDATE books SET index_status = 'failed', updated_at = ? WHERE id = ?`,
		time.Now().UTC(), bookID)
	return err
}

// QueuedBooks returns books awaiting Pass 2 indexing.
func (s *Store) QueuedBooks(ctx context.Context, limit int) ([]Book, error) {
	limit = saneLimit(limit)
	rows, err := s.db.QueryContext(ctx,
		bookSelect()+` WHERE b.index_status = 'queued' ORDER BY b.updated_at ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}

// QueuedCount returns the number of books currently waiting for Pass 2.
func (s *Store) QueuedCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books WHERE index_status = 'queued'`).Scan(&n)
	return n, err
}

// IsImported is the legacy single-pass check — kept for callers that don't
// distinguish between Pass 1 and Pass 2. Most code should prefer IsIndexed.
func (s *Store) IsImported(ctx context.Context, path string, size int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM books WHERE file_path = ? AND file_size = ? LIMIT 1`,
		path, size).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) UpsertImportedBook(ctx context.Context, imported ImportBook) (bool, error) {
	if imported.Book.ID == "" {
		return false, errors.New("book id is required")
	}
	now := time.Now().UTC()
	imported.Book.UpdatedAt = now
	if imported.Book.CreatedAt.IsZero() {
		imported.Book.CreatedAt = now
	}
	if imported.Book.IndexStatus == "" {
		imported.Book.IndexStatus = "indexed"
	}
	if imported.Book.TextStatus == "" {
		imported.Book.TextStatus = "available"
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var existed int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM books WHERE id = ?`, imported.Book.ID).Scan(&existed); err != nil {
		return false, err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO books (
		id, title, authors, description, publisher, published_date, isbn10, isbn13, language, cover_url,
		file_path, format, file_hash, file_size, metadata_source, metadata_refreshed_at, index_status, text_status, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title=excluded.title,
		authors=excluded.authors,
		description=CASE WHEN books.description != '' AND excluded.description = '' THEN books.description ELSE excluded.description END,
		publisher=CASE WHEN books.publisher != '' AND excluded.publisher = '' THEN books.publisher ELSE excluded.publisher END,
		published_date=CASE WHEN books.published_date != '' AND excluded.published_date = '' THEN books.published_date ELSE excluded.published_date END,
		isbn10=CASE WHEN books.isbn10 != '' AND excluded.isbn10 = '' THEN books.isbn10 ELSE excluded.isbn10 END,
		isbn13=CASE WHEN books.isbn13 != '' AND excluded.isbn13 = '' THEN books.isbn13 ELSE excluded.isbn13 END,
		language=CASE WHEN books.language != '' AND excluded.language = '' THEN books.language ELSE excluded.language END,
		cover_url=CASE WHEN books.cover_url != '' AND excluded.cover_url = '' THEN books.cover_url ELSE excluded.cover_url END,
		file_path=excluded.file_path,
		format=excluded.format,
		file_hash=excluded.file_hash,
		file_size=excluded.file_size,
		metadata_source=CASE WHEN books.metadata_source != '' AND excluded.metadata_source = '' THEN books.metadata_source ELSE excluded.metadata_source END,
		metadata_refreshed_at=CASE WHEN books.metadata_refreshed_at IS NOT NULL AND excluded.metadata_refreshed_at IS NULL THEN books.metadata_refreshed_at ELSE excluded.metadata_refreshed_at END,
		index_status=excluded.index_status,
		text_status=excluded.text_status,
		updated_at=excluded.updated_at`, imported.Book.ID, imported.Book.Title, imported.Book.Authors, imported.Book.Description,
		imported.Book.Publisher, imported.Book.PublishedDate, imported.Book.ISBN10, imported.Book.ISBN13, imported.Book.Language,
		imported.Book.CoverURL, imported.Book.FilePath, imported.Book.Format, imported.Book.FileHash, imported.Book.FileSize, imported.Book.MetadataSource,
		nullTime(imported.Book.MetadataRefreshedAt), imported.Book.IndexStatus, imported.Book.TextStatus, imported.Book.CreatedAt, imported.Book.UpdatedAt)
	if err != nil {
		return false, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM passages WHERE book_id = ?`, imported.Book.ID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM book_search WHERE book_id = ?`, imported.Book.ID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM passage_search WHERE book_id = ?`, imported.Book.ID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO book_search(book_id, title, authors, description) VALUES (?, ?, ?, ?)`, imported.Book.ID, imported.Book.Title, imported.Book.Authors, imported.Book.Description); err != nil {
		return false, err
	}
	pStmt, err := tx.PrepareContext(ctx, `INSERT INTO passages(id, book_id, label, chunk_index, text) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return false, err
	}
	defer pStmt.Close()
	psStmt, err := tx.PrepareContext(ctx, `INSERT INTO passage_search(passage_id, book_id, label, text) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return false, err
	}
	defer psStmt.Close()
	written := 0
	for _, passage := range imported.Passages {
		if passage.ID == "" || passage.Text == "" {
			continue
		}
		if _, err := pStmt.ExecContext(ctx, passage.ID, imported.Book.ID, passage.Label, passage.ChunkIndex, passage.Text); err != nil {
			return false, err
		}
		if _, err := psStmt.ExecContext(ctx, passage.ID, imported.Book.ID, passage.Label, passage.Text); err != nil {
			return false, err
		}
		written++
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE books SET passage_count = ?, needs_enrichment = ? WHERE id = ?`,
		written, computeNeedsEnrichment(imported.Book, written), imported.Book.ID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return existed > 0, nil
}

// computeNeedsEnrichment returns 1 if the book still needs the enricher
// to do work (missing metadata fields, or text-indexed but no AI subjects
// extracted yet). Mirrors the predicate in BooksNeedingEnrichment so the
// flag stays consistent with what the scanner would re-derive.
func computeNeedsEnrichment(b Book, passageCount int) int {
	if b.MetadataSource == "" || b.CoverURL == "" || b.Description == "" {
		return 1
	}
	if b.IndexStatus == "indexed" && passageCount > 0 && !strings.Contains(b.Description, "Subjects:") {
		return 1
	}
	return 0
}
