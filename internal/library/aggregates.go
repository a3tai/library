package library

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ────────────────── Aggregation helpers for sidebar views ──────────────────

// AggregateGroup is a name + count pair used by the "By Author" /
// "By Subject" / "Categories" sidebar views.
type AggregateGroup struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ListAuthors returns the distinct author entries with their book counts,
// ordered by count desc. Multi-author rows ("A, B, C") are split into
// individual entries so each contributor is visible.
func (s *Store) ListAuthors(ctx context.Context, limit int) ([]AggregateGroup, error) {
	limit = saneLimit(limit)
	rows, err := s.db.QueryContext(ctx, `SELECT authors FROM books WHERE authors != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tally := map[string]int{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		for _, name := range splitAuthors(raw) {
			tally[name]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rankedGroups(tally, limit), nil
}

func splitAuthors(s string) []string {
	out := []string{}
	for _, raw := range strings.Split(s, ",") {
		// Also split on " and " / " & " for "John & Jane" style.
		for _, alt := range splitMultiSep(raw, []string{" and ", " & ", ";"}) {
			n := strings.TrimSpace(alt)
			if n != "" {
				out = append(out, n)
			}
		}
	}
	return out
}

func splitMultiSep(s string, seps []string) []string {
	parts := []string{s}
	for _, sep := range seps {
		next := []string{}
		for _, p := range parts {
			next = append(next, strings.Split(p, sep)...)
		}
		parts = next
	}
	return parts
}

// ListSubjects returns the most-used subjects across the library.
// Sourced from the book_subjects join table — much cheaper than the old
// description-blob scan it replaced.
func (s *Store) ListSubjects(ctx context.Context, limit int) ([]AggregateGroup, error) {
	return s.aggregateChild(ctx, "subject", "book_subjects", saneLimit(limit))
}

// ListCategories returns the most-used categories across the library.
// Sourced from the book_categories join table.
func (s *Store) ListCategories(ctx context.Context, limit int) ([]AggregateGroup, error) {
	return s.aggregateChild(ctx, "category", "book_categories", saneLimit(limit))
}

// aggregateChild groups + counts a single-value column from a join table,
// ordered by count desc, then value asc, capped to limit.
func (s *Store) aggregateChild(ctx context.Context, col, table string, limit int) ([]AggregateGroup, error) {
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT %s, COUNT(*) FROM %s GROUP BY %s ORDER BY 2 DESC, 1 ASC LIMIT ?`, col, table, col),
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AggregateGroup{}
	for rows.Next() {
		var g AggregateGroup
		if err := rows.Scan(&g.Name, &g.Count); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// rankedGroups converts a tally map into a deterministic slice sorted by
// count desc, then name asc, capped to limit.
func rankedGroups(tally map[string]int, limit int) []AggregateGroup {
	out := make([]AggregateGroup, 0, len(tally))
	for name, count := range tally {
		out = append(out, AggregateGroup{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// RecentlyAdded returns the most recently created books, ordered by
// created_at DESC. Used by the "Recently Added" sidebar view; pagination
// follows the same limit/offset contract as ListBooks.
func (s *Store) RecentlyAdded(ctx context.Context, limit, offset int) ([]Book, error) {
	limit = saneLimit(limit)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx,
		bookSelect()+` ORDER BY b.created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}

// ListUnprocessed returns books that still need work — either no metadata
// source, no cover, no description, or not fully indexed.
func (s *Store) ListUnprocessed(ctx context.Context, limit, offset int) ([]Book, error) {
	limit = saneLimit(limit)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, bookSelect()+`
		WHERE
			b.metadata_source = ''
			OR b.cover_url = ''
			OR b.description = ''
			OR b.index_status != 'indexed'
		ORDER BY b.updated_at DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}
