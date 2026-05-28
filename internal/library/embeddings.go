package library

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"strings"
)

// Vectors are stored as little-endian float32 BLOBs. Storing them
// pre-normalised lets cosine similarity collapse to a plain dot product
// during search, which matters because we score every row in Go.

// encodeVector packs a float32 vector into a little-endian byte buffer.
func encodeVector(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// decodeVector unpacks a little-endian float32 BLOB. The caller is expected
// to know the dim; we infer it from the byte length.
func decodeVector(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

// normalize returns a unit-length copy of v. A zero vector stays zero so
// downstream dot products score it 0 instead of producing NaNs.
func normalize(v []float32) []float32 {
	var sum float64
	for _, f := range v {
		sum += float64(f) * float64(f)
	}
	if sum == 0 {
		out := make([]float32, len(v))
		copy(out, v)
		return out
	}
	inv := float32(1.0 / math.Sqrt(sum))
	out := make([]float32, len(v))
	for i, f := range v {
		out[i] = f * inv
	}
	return out
}

func dot(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// PassageEmbedding pairs a passage id with the input text the embedder
// should see. Callers (importer, backfill worker) build a slice of these
// and hand it to UpsertPassageEmbeddings together with the matching
// vectors.
type PassageEmbedding struct {
	PassageID string
	BookID    string
	Vector    []float32
}

// UpsertPassageEmbeddings replaces all embeddings for a book under a given
// model in a single transaction. Vectors are normalised on the way in so
// VectorSearchPassages can use a plain dot product. Passing an empty
// `embeddings` slice deletes any existing rows for the book/model.
func (s *Store) UpsertPassageEmbeddings(ctx context.Context, bookID, model string, embeddings []PassageEmbedding) error {
	if bookID == "" {
		return errors.New("book id is required")
	}
	if model == "" {
		return errors.New("embedding model is required")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM passage_embeddings WHERE book_id = ? AND model = ?`, bookID, model); err != nil {
		return err
	}
	if len(embeddings) == 0 {
		return tx.Commit()
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO passage_embeddings(passage_id, book_id, model, dim, vector) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range embeddings {
		if e.PassageID == "" || len(e.Vector) == 0 {
			continue
		}
		v := normalize(e.Vector)
		if _, err := stmt.ExecContext(ctx, e.PassageID, bookID, model, len(v), encodeVector(v)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// BooksWithoutEmbeddings returns up to `limit` book IDs that have at least
// one passage but no embeddings under the given model. Ordered by recency
// so the user's latest imports get embedded first.
func (s *Store) BooksWithoutEmbeddings(ctx context.Context, model string, limit int) ([]string, error) {
	limit = saneLimit(limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id FROM books b
		WHERE b.passage_count > 0
		  AND b.text_status = 'available'
		  AND NOT EXISTS (
			SELECT 1 FROM passage_embeddings pe
			WHERE pe.book_id = b.id AND pe.model = ?
		)
		ORDER BY b.updated_at DESC
		LIMIT ?`, model, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CountBooksWithoutEmbeddings is the cheap "is there work?" check the
// backfill scanner uses before pulling a batch.
func (s *Store) CountBooksWithoutEmbeddings(ctx context.Context, model string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM books b
		WHERE b.passage_count > 0
		  AND b.text_status = 'available'
		  AND NOT EXISTS (
			SELECT 1 FROM passage_embeddings pe
			WHERE pe.book_id = b.id AND pe.model = ?
		)`, model).Scan(&n)
	return n, err
}

// HasAnyEmbeddings reports whether the embeddings table has at least one
// row for the given model. Lets the read path decide quickly whether to
// attempt vector search at all on a fresh DB.
func (s *Store) HasAnyEmbeddings(ctx context.Context, model string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM passage_embeddings WHERE model = ? LIMIT 1`, model).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// PassagesForEmbedding returns the (id, text) pairs the embedder needs for
// a given book, in chunk order. Skips empty texts so the embedder doesn't
// see them.
func (s *Store) PassagesForEmbedding(ctx context.Context, bookID string) ([]Passage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, book_id, label, chunk_index, text FROM passages
		 WHERE book_id = ? AND text != ''
		 ORDER BY chunk_index ASC`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Passage
	for rows.Next() {
		var p Passage
		if err := rows.Scan(&p.ID, &p.BookID, &p.Label, &p.ChunkIndex, &p.Text); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// VectorSearchPassages runs a brute-force cosine search over stored
// embeddings (for the given model) and returns the top-`limit` passages.
// queryVec does not need to be normalised — it's normalised here. If
// bookID is non-empty, the search is scoped to that book.
//
// Returned passages are populated with the same fields as SearchPassages,
// plus a synthesised Snippet (a short prefix of Text). The slice is
// ordered by descending similarity. dim mismatches between the query and
// stored vectors return zero hits rather than an error so a stale model
// transition surfaces as "no results" instead of a hard failure.
func (s *Store) VectorSearchPassages(ctx context.Context, queryVec []float32, model, bookID string, limit int) ([]Passage, error) {
	if len(queryVec) == 0 {
		return nil, errors.New("query vector is empty")
	}
	if model == "" {
		return nil, errors.New("model is required")
	}
	limit = saneLimit(limit)
	q := normalize(queryVec)

	args := []any{model}
	whereBook := ""
	if strings.TrimSpace(bookID) != "" {
		whereBook = " AND book_id = ?"
		args = append(args, bookID)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT passage_id, book_id, dim, vector FROM passage_embeddings WHERE model = ?`+whereBook, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type hit struct {
		passageID string
		bookID    string
		score     float32
	}
	var hits []hit
	for rows.Next() {
		var (
			pid, bid string
			dim      int
			blob     []byte
		)
		if err := rows.Scan(&pid, &bid, &dim, &blob); err != nil {
			return nil, err
		}
		if dim != len(q) {
			continue
		}
		v := decodeVector(blob)
		if len(v) != dim {
			continue
		}
		hits = append(hits, hit{passageID: pid, bookID: bid, score: dot(q, v)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return []Passage{}, nil
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > limit {
		hits = hits[:limit]
	}

	// Resolve top hits to full Passage rows in one round-trip per hit.
	// limit is small (defaults to 80, capped at 500) so the N+1 here is
	// cheaper than building an IN(...) clause and re-sorting in Go.
	out := make([]Passage, 0, len(hits))
	for _, h := range hits {
		p, err := s.GetPassage(ctx, h.passageID)
		if err != nil {
			continue
		}
		p.Snippet = snippetFromText(p.Text, 220)
		out = append(out, p)
	}
	return out, nil
}

// snippetFromText returns a short, single-line preview of text suitable for
// the search results UI. FTS-based SearchPassages produces snippets via the
// FTS5 snippet() function; for vector search we don't have token offsets,
// so we just take a leading slice with whitespace collapsed.
func snippetFromText(text string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 220
	}
	t := strings.Join(strings.Fields(text), " ")
	if len(t) <= maxLen {
		return t
	}
	return strings.TrimSpace(t[:maxLen]) + "…"
}
