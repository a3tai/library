package library

import (
	"context"
	"database/sql"
	"strings"
)

type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// refreshBookFTS rebuilds the `book_search` row for one book. The FTS
// description column gets the prose description plus a tail of the book's
// subjects + categories, so search-by-subject works without a separate FTS
// column. Callers must already hold s.writeMu (or be inside a tx that does).
func (s *Store) refreshBookFTS(ctx context.Context, ex dbExec, bookID string) error {
	var title, authors, desc string
	if err := ex.QueryRowContext(ctx,
		`SELECT title, authors, description FROM books WHERE id = ?`, bookID).
		Scan(&title, &authors, &desc); err != nil {
		return err
	}
	parts := []string{desc}
	if subj, err := joinChildValues(ctx, ex,
		`SELECT subject FROM book_subjects WHERE book_id = ? ORDER BY subject`, bookID); err == nil && subj != "" {
		parts = append(parts, "Subjects: "+subj)
	}
	if cats, err := joinChildValues(ctx, ex,
		`SELECT category FROM book_categories WHERE book_id = ? ORDER BY category`, bookID); err == nil && cats != "" {
		parts = append(parts, "Categories: "+cats)
	}
	ftsDesc := strings.TrimSpace(strings.Join(parts, "\n"))
	if _, err := ex.ExecContext(ctx, `DELETE FROM book_search WHERE book_id = ?`, bookID); err != nil {
		return err
	}
	if _, err := ex.ExecContext(ctx,
		`INSERT INTO book_search(book_id, title, authors, description) VALUES (?, ?, ?, ?)`,
		bookID, title, authors, ftsDesc); err != nil {
		return err
	}
	return nil
}

// joinChildValues runs a single-column query that returns 0..N strings and
// joins them as ", "-separated. Used for FTS row composition.
func joinChildValues(ctx context.Context, ex dbExec, query, bookID string) (string, error) {
	type rowsLike interface {
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	}
	rl, ok := ex.(rowsLike)
	if !ok {
		return "", nil
	}
	rows, err := rl.QueryContext(ctx, query, bookID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return "", err
		}
		values = append(values, v)
	}
	return strings.Join(values, ", "), rows.Err()
}

// AddBookSubjects appends subjects to a book's subjects list. INSERT OR
// IGNORE means duplicates from a different source silently no-op; the first
// writer's source label sticks. Refreshes the FTS row so search-by-subject
// reflects the addition.
func (s *Store) AddBookSubjects(ctx context.Context, bookID string, subjects []string, source string) error {
	if bookID == "" || len(subjects) == 0 {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO book_subjects(book_id, subject, source) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, sub := range subjects {
		v := strings.TrimSpace(sub)
		if v == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, bookID, v, source); err != nil {
			return err
		}
	}
	if err := s.refreshBookFTS(ctx, tx, bookID); err != nil {
		return err
	}
	return tx.Commit()
}

// HasSubjects returns true when the book already has at least one entry in
// `book_subjects`. Used by the enricher to decide whether the AI step is
// still needed.
func (s *Store) HasSubjects(ctx context.Context, bookID string) (bool, error) {
	if bookID == "" {
		return false, nil
	}
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM book_subjects WHERE book_id = ? LIMIT 1)`, bookID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// AddBookCategories is the categories-table twin of AddBookSubjects.
func (s *Store) AddBookCategories(ctx context.Context, bookID string, categories []string, source string) error {
	if bookID == "" || len(categories) == 0 {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO book_categories(book_id, category, source) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, cat := range categories {
		v := strings.TrimSpace(cat)
		if v == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, bookID, v, source); err != nil {
			return err
		}
	}
	if err := s.refreshBookFTS(ctx, tx, bookID); err != nil {
		return err
	}
	return tx.Commit()
}

// IsIndexed returns true when a book row exists at the given file_path with
// the same file_size on disk AND its index_status is "indexed" — i.e. Pass 2
// has already extracted passages for this file. Used by the importer to skip
// re-inventorying files that are fully processed.
