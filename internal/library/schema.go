package library

import (
	"context"
	"strings"
)

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS books (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			authors TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			publisher TEXT NOT NULL DEFAULT '',
			published_date TEXT NOT NULL DEFAULT '',
			isbn10 TEXT NOT NULL DEFAULT '',
			isbn13 TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			cover_url TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL,
			format TEXT NOT NULL,
			file_hash TEXT NOT NULL UNIQUE,
			metadata_source TEXT NOT NULL DEFAULT '',
			metadata_refreshed_at TIMESTAMP,
			index_status TEXT NOT NULL DEFAULT 'indexed',
			text_status TEXT NOT NULL DEFAULT 'available',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS passages (
			id TEXT PRIMARY KEY,
			book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
			label TEXT NOT NULL DEFAULT '',
			chunk_index INTEGER NOT NULL,
			text TEXT NOT NULL
		)`,
		// Add file_size for fast skip-already-imported checks. ALTER TABLE is
		// idempotent only if we ignore "duplicate column" errors, so handle
		// the migration after the CREATE TABLE.
		`CREATE INDEX IF NOT EXISTS idx_books_title ON books(title)`,
		`CREATE INDEX IF NOT EXISTS idx_books_file_path ON books(file_path)`,
		`CREATE INDEX IF NOT EXISTS idx_passages_book_id ON passages(book_id)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS book_search USING fts5(book_id UNINDEXED, title, authors, description)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS passage_search USING fts5(passage_id UNINDEXED, book_id UNINDEXED, label UNINDEXED, text)`,
		// Subjects and categories are first-class join tables. `source` records
		// where the entry came from (openlibrary | googlebooks | lmstudio |
		// backfill) so we can decide later whether to re-enrich. Composite PK
		// makes (book_id, value) unique without a surrogate.
		`CREATE TABLE IF NOT EXISTS book_subjects (
			book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
			subject TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (book_id, subject)
		)`,
		`CREATE TABLE IF NOT EXISTS book_categories (
			book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
			category TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (book_id, category)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_book_subjects_subject ON book_subjects(subject)`,
		`CREATE INDEX IF NOT EXISTS idx_book_categories_category ON book_categories(category)`,
		// Dense vector store for semantic passage search. modernc.org/sqlite
		// is pure-Go and can't load sqlite-vec, so vectors live in a regular
		// table and brute-force cosine runs in Go (see embeddings.go). The
		// `model` column tags each row so re-embedding under a new model
		// doesn't silently mix dimensionalities at query time.
		`CREATE TABLE IF NOT EXISTS passage_embeddings (
			passage_id TEXT PRIMARY KEY REFERENCES passages(id) ON DELETE CASCADE,
			book_id TEXT NOT NULL,
			model TEXT NOT NULL,
			dim INTEGER NOT NULL,
			vector BLOB NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_passage_embeddings_book_id ON passage_embeddings(book_id)`,
		`CREATE INDEX IF NOT EXISTS idx_passage_embeddings_model ON passage_embeddings(model)`,
		// User-editable settings (LM Studio endpoint, model ids, MCP port,
		// etc.). Plain k/v keeps the schema stable as we add settings
		// without further migrations. Empty value means "fall back to env
		// or built-in default" — see internal/library/settings.go.
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMP NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	// Idempotent ALTER TABLEs. SQLite returns "duplicate column" if the
	// column already exists; everything else is a real error.
	for _, alter := range []string{
		`ALTER TABLE books ADD COLUMN file_size INTEGER NOT NULL DEFAULT 0`,
		// Cached row count of `passages` for this book. Replaces the
		// correlated `(SELECT COUNT(*) ...)` subquery that previously ran
		// for every row in every list query.
		`ALTER TABLE books ADD COLUMN passage_count INTEGER NOT NULL DEFAULT 0`,
		// Cheap "still needs metadata or AI enrichment" flag, refreshed at
		// every write. Lets the scanner avoid a `instr(description, 'Subjects:')`
		// scan on every poll.
		`ALTER TABLE books ADD COLUMN needs_enrichment INTEGER NOT NULL DEFAULT 1`,
	} {
		if _, err := s.db.ExecContext(ctx, alter); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				return err
			}
		}
	}
	// Backfill passage_count once for existing rows where it's still 0 but
	// passages exist. Cheap on cold libraries, no-op on subsequent boots.
	if _, err := s.db.ExecContext(ctx, `UPDATE books SET passage_count = (
		SELECT COUNT(*) FROM passages p WHERE p.book_id = books.id
	) WHERE passage_count = 0`); err != nil {
		return err
	}
	// Backfill subjects/categories from the legacy description blob BEFORE
	// the needs_enrichment recompute — the recompute now keys off
	// book_subjects, so the join table needs to be populated first.
	if err := s.backfillSubjectCategoryFromDescription(ctx); err != nil {
		return err
	}
	// Backfill needs_enrichment off the same predicate the scanner uses.
	if _, err := s.db.ExecContext(ctx, `UPDATE books SET needs_enrichment = CASE
		WHEN metadata_source = '' OR cover_url = '' OR description = '' THEN 1
		WHEN index_status = 'indexed' AND passage_count > 0 AND NOT EXISTS(
			SELECT 1 FROM book_subjects WHERE book_id = books.id) THEN 1
		ELSE 0 END`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_books_needs_enrichment ON books(needs_enrichment) WHERE needs_enrichment = 1`); err != nil {
		return err
	}
	return nil
}

// backfillSubjectCategoryFromDescription is a one-shot migration: scan
// existing description blobs for the legacy "Subjects: a, b, c" and
// "Genre: x" lines and copy the values into the new join tables. INSERT OR
// IGNORE makes it idempotent — re-running on a populated DB is a no-op.
// The description blob is intentionally left alone so a re-import or a
// future readback path can still see it; new writes (after B2) won't add
// these lines anymore, so the blob's role narrows over time.
func (s *Store) backfillSubjectCategoryFromDescription(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, description FROM books WHERE description LIKE '%Subjects:%' OR description LIKE '%Genre:%'`)
	if err != nil {
		return err
	}
	type pending struct {
		id         string
		subjects   []string
		categories []string
	}
	var todo []pending
	for rows.Next() {
		var id, desc string
		if err := rows.Scan(&id, &desc); err != nil {
			rows.Close()
			return err
		}
		p := pending{id: id}
		for _, line := range strings.Split(desc, "\n") {
			lower := strings.ToLower(strings.TrimSpace(line))
			if strings.HasPrefix(lower, "subjects:") {
				val := strings.TrimSpace(line[len("subjects:"):])
				for _, part := range strings.Split(val, ",") {
					if v := strings.TrimSpace(part); v != "" {
						p.subjects = append(p.subjects, v)
					}
				}
			} else if strings.HasPrefix(lower, "genre:") {
				val := strings.TrimSpace(line[len("genre:"):])
				if val != "" {
					p.categories = append(p.categories, val)
				}
			}
		}
		if len(p.subjects) > 0 || len(p.categories) > 0 {
			todo = append(todo, p)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(todo) == 0 {
		return nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	subjStmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO book_subjects(book_id, subject, source) VALUES (?, ?, 'backfill')`)
	if err != nil {
		return err
	}
	defer subjStmt.Close()
	catStmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO book_categories(book_id, category, source) VALUES (?, ?, 'backfill')`)
	if err != nil {
		return err
	}
	defer catStmt.Close()
	for _, p := range todo {
		for _, s := range p.subjects {
			if _, err := subjStmt.ExecContext(ctx, p.id, s); err != nil {
				return err
			}
		}
		for _, c := range p.categories {
			if _, err := catStmt.ExecContext(ctx, p.id, c); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// dbExec is the subset of *sql.DB / *sql.Tx that the FTS-refresh helper
// needs. Lets the same code run inside a transaction or against the raw DB.
