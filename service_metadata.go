package main

import (
	"context"
	"time"
)

// RequestMetadata enqueues a single book for *priority* enrichment — used
// when the user has just opened that book's detail view. It jumps ahead of
// any background backfill in flight. Returns immediately; results land in
// the books table when (and if) the lookup succeeds.
func (s *LibraryService) RequestMetadata(bookID string) bool {
	if s.enricher == nil {
		return false
	}
	return s.enricher.RequestPriority(bookID)
}

// EnrichmentQueueDepth exposes the pending-lookup count for the UI.
func (s *LibraryService) EnrichmentQueueDepth() int {
	if s.enricher == nil {
		return 0
	}
	return s.enricher.QueueDepth()
}

// HydrateMetadata enqueues up to `limit` books that still need metadata into
// the throttled enricher and returns immediately with the number queued.
// Actual lookups happen in the background at a polite rate (~1.4 RPS) so
// large libraries don't hammer Open Library / Google Books.
func (s *LibraryService) HydrateMetadata(limit int) (int, error) {
	if !s.setBusy(true) {
		return 0, nil
	}
	defer s.setBusy(false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	books, err := s.store.BooksNeedingMetadata(ctx, limit)
	if err != nil {
		return 0, err
	}
	ids := make([]string, 0, len(books))
	for _, b := range books {
		ids = append(ids, b.ID)
	}
	return s.enricher.RequestMany(ids), nil
}
