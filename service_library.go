package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/a3tai/library/internal/library"
)

func (s *LibraryService) Snapshot(query string) (LibrarySnapshot, error) {
	ctx := context.Background()
	books, err := s.store.ListBooks(ctx, query, 120, 0)
	if err != nil {
		return LibrarySnapshot{}, err
	}
	stats, err := s.store.Stats(ctx)
	if err != nil {
		return LibrarySnapshot{}, err
	}
	dbPath, _ := library.DefaultDBPath()
	return LibrarySnapshot{Books: books, Stats: stats, DBPath: dbPath, Hydrating: s.isBusy(), MCP: s.MCPStatus()}, nil
}

func (s *LibraryService) ListBooks(query string, limit int, offset int) ([]library.Book, error) {
	return s.store.ListBooks(context.Background(), query, limit, offset)
}

// SearchPassages prefers semantic (vector) search when an embedding model
// is reachable and the DB has at least one vector under that model. Falls
// back to the FTS5 path otherwise so the feature degrades gracefully on
// fresh libraries or when LM Studio isn't running.
func (s *LibraryService) SearchPassages(query string, bookID string, limit int) ([]library.Passage, error) {
	ctx := context.Background()
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if passages, ok := s.vectorPassageSearch(ctx, q, bookID, limit); ok {
		return passages, nil
	}
	return s.store.SearchPassages(ctx, q, bookID, limit)
}

// vectorPassageSearch returns (results, true) when vector search ran
// successfully and (nil, false) when the caller should fall back to FTS.
// "Successfully" here means LM Studio answered AND the DB had at least one
// vector to score against — a vector run that matches zero rows is still
// "successful" and overrides the FTS fallback (the user's library just
// hasn't been embedded for this query topic).
func (s *LibraryService) vectorPassageSearch(ctx context.Context, query, bookID string, limit int) ([]library.Passage, bool) {
	client := s.embed.Get()
	if client == nil {
		return nil, false
	}
	model := client.Model
	has, err := s.store.HasAnyEmbeddings(ctx, model)
	if err != nil || !has {
		return nil, false
	}
	embedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	vec, err := client.EmbedOne(embedCtx, query)
	cancel()
	if err != nil {
		log.Printf("embed query: %v", err)
		return nil, false
	}
	results, err := s.store.VectorSearchPassages(ctx, vec, model, bookID, limit)
	if err != nil {
		log.Printf("vector search: %v", err)
		return nil, false
	}
	return results, true
}

// ListAuthors returns up to `limit` authors, sorted by count desc.
// Used by the "By Author" sidebar view.
func (s *LibraryService) ListAuthors(limit int) ([]library.AggregateGroup, error) {
	return s.cachedAggregation(&s.aggAuthors, limit, s.store.ListAuthors)
}

// ListSubjects returns up to `limit` subjects parsed from book metadata.
// Used by the "By Subject" sidebar view.
func (s *LibraryService) ListSubjects(limit int) ([]library.AggregateGroup, error) {
	return s.cachedAggregation(&s.aggSubject, limit, s.store.ListSubjects)
}

// ListCategories returns the most-used categories for the "Categories" sidebar
// view. Sourced from the book_categories join table.
func (s *LibraryService) ListCategories(limit int) ([]library.AggregateGroup, error) {
	return s.cachedAggregation(&s.aggCategory, limit, s.store.ListCategories)
}

// cachedAggregation memoises the result of a slow aggregation scan
// (ListAuthors / ListSubjects / ListCategories). The cache is keyed by limit
// because callers ask for different sizes (sidebar wants 200, search wants
// 400). Entries expire after aggCacheTTL or when invalidateAggregations
// is called (after a write).
func (s *LibraryService) cachedAggregation(slot *aggCacheEntry, limit int,
	fetch func(context.Context, int) ([]library.AggregateGroup, error),
) ([]library.AggregateGroup, error) {
	s.aggMu.Lock()
	if slot.limit >= limit && time.Now().Before(slot.expires) && slot.data != nil {
		out := slot.data
		if len(out) > limit {
			out = out[:limit]
		}
		s.aggMu.Unlock()
		return out, nil
	}
	s.aggMu.Unlock()

	data, err := fetch(context.Background(), limit)
	if err != nil {
		return nil, err
	}
	s.aggMu.Lock()
	slot.limit = limit
	slot.data = data
	slot.expires = time.Now().Add(aggCacheTTL)
	s.aggMu.Unlock()
	return data, nil
}

// invalidateAggregations resets the cached author/subject/genre lists.
// Called after any write that may have changed the aggregations: import
// commits, index passes, metadata enrichments.
func (s *LibraryService) invalidateAggregations() {
	s.aggMu.Lock()
	s.aggAuthors = aggCacheEntry{}
	s.aggSubject = aggCacheEntry{}
	s.aggCategory = aggCacheEntry{}
	s.aggMu.Unlock()
}

// ListUnprocessed returns books that still need metadata or indexing —
// the "Unprocessed" sidebar view.
func (s *LibraryService) ListUnprocessed(limit, offset int) ([]library.Book, error) {
	return s.store.ListUnprocessed(context.Background(), limit, offset)
}

// ListRecentlyAdded returns the most recently created books — the
// "Recently Added" sidebar view. Ordered by created_at DESC.
func (s *LibraryService) ListRecentlyAdded(limit, offset int) ([]library.Book, error) {
	return s.store.RecentlyAdded(context.Background(), limit, offset)
}

// SearchResults bundles the kinds of hits we surface on the global
// search page: matching books, matching passages across the whole
// library, and the top author/subject groupings whose names contain
// the query string.
type SearchResults struct {
	Query    string                   `json:"query"`
	Books    []library.Book           `json:"books"`
	Passages []library.Passage        `json:"passages"`
	Authors  []library.AggregateGroup `json:"authors"`
	Subjects []library.AggregateGroup `json:"subjects"`
}

// Search runs a unified search across books and passages, plus a quick
// substring scan over cached author/subject aggregations. Each list is
// independently capped — callers can use the counts directly without
// pagination for the initial preview, then drill in via per-type calls.
func (s *LibraryService) Search(query string) (SearchResults, error) {
	q := strings.TrimSpace(query)
	out := SearchResults{Query: q}
	if q == "" {
		return out, nil
	}
	ctx := context.Background()
	books, err := s.store.SearchBooks(ctx, q, 30, 0)
	if err == nil {
		out.Books = books
	}
	passages, err := s.store.SearchPassages(ctx, q, "", 40)
	if err == nil {
		out.Passages = passages
	}
	if authors, err := s.cachedAggregation(&s.aggAuthors, 200, s.store.ListAuthors); err == nil {
		needle := strings.ToLower(q)
		for _, a := range authors {
			if strings.Contains(strings.ToLower(a.Name), needle) {
				out.Authors = append(out.Authors, a)
				if len(out.Authors) >= 12 {
					break
				}
			}
		}
	}
	if subs, err := s.cachedAggregation(&s.aggSubject, 400, s.store.ListSubjects); err == nil {
		needle := strings.ToLower(q)
		for _, sub := range subs {
			if strings.Contains(strings.ToLower(sub.Name), needle) {
				out.Subjects = append(out.Subjects, sub)
				if len(out.Subjects) >= 12 {
					break
				}
			}
		}
	}
	return out, nil
}

func (s *LibraryService) GetBook(id string) (library.Book, error) {
	return s.store.GetBook(context.Background(), id)
}

func (s *LibraryService) GetPassage(id string) (library.Passage, error) {
	return s.store.GetPassage(context.Background(), id)
}

// BookPassages returns passages for the reader, in chunk_index order.
// limit is clamped server-side to 1000.
func (s *LibraryService) BookPassages(bookID string, offset int, limit int) ([]library.Passage, error) {
	return s.store.BookPassages(context.Background(), bookID, offset, limit)
}

// BookTOC returns the table of contents derived from the stored passages.
// May be empty for plain-text imports that have no chapter labels. Cached
// per book; the cache is invalidated on re-index.
func (s *LibraryService) BookTOC(bookID string) ([]library.TOCEntry, error) {
	s.tocMu.Lock()
	if cached, ok := s.tocCache[bookID]; ok {
		s.tocMu.Unlock()
		return cached, nil
	}
	s.tocMu.Unlock()
	toc, err := s.store.BookTOC(context.Background(), bookID)
	if err != nil {
		return nil, err
	}
	s.tocMu.Lock()
	if s.tocCache == nil {
		s.tocCache = make(map[string][]library.TOCEntry)
	}
	s.tocCache[bookID] = toc
	s.tocMu.Unlock()
	return toc, nil
}

// invalidateTOC drops the cached TOC for one book. Called after re-indexing,
// since label / chunk_index data has changed.
func (s *LibraryService) invalidateTOC(bookID string) {
	s.tocMu.Lock()
	delete(s.tocCache, bookID)
	s.tocMu.Unlock()
}
