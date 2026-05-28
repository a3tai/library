package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/a3tai/library/internal/importer"
)

// ImportPath kicks off (or queues) a non-blocking import of the file or
// directory at path. If no import is currently running, it starts one
// immediately. If another import is already in flight, path is appended
// to a FIFO queue and processed once the current run finishes — that's
// the contract behind the Add Books button never being disabled. Poll
// ImporterStatus() for live progress; QueuedPaths lists the pending tail.
func (s *LibraryService) ImportPath(path string) (ImporterStatus, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return s.ImporterStatus(), fmt.Errorf("path is required")
	}
	s.impMu.Lock()
	if s.impState.Running {
		s.impQueue = append(s.impQueue, path)
		status := s.statusLocked()
		s.impMu.Unlock()
		return status, nil
	}
	s.startImportLocked(path)
	status := s.statusLocked()
	s.impMu.Unlock()
	return status, nil
}

// startImportLocked transitions impState into a fresh running run for
// `path` and spawns the runImport goroutine. Caller must hold impMu.
func (s *LibraryService) startImportLocked(path string) {
	ctx, cancel := context.WithCancel(context.Background())
	s.impCancel = cancel
	s.impState = importerState{
		Running:   true,
		Path:      path,
		StartedAt: time.Now(),
	}
	go s.runImport(ctx, path)
}

// CancelImport requests the running import to stop. Returns true if a
// cancel signal was sent.
func (s *LibraryService) CancelImport() bool {
	s.impMu.Lock()
	cancel := s.impCancel
	s.impMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// ImporterStatus returns a snapshot of the current importer state.
// Safe to call any time; cheap enough to poll at ~2 Hz.
func (s *LibraryService) ImporterStatus() ImporterStatus {
	s.impMu.Lock()
	defer s.impMu.Unlock()
	return s.statusLocked()
}

func (s *LibraryService) statusLocked() ImporterStatus {
	out := ImporterStatus{importerState: s.impState}
	if !s.impState.StartedAt.IsZero() {
		end := s.impState.FinishedAt
		if end.IsZero() {
			end = time.Now()
		}
		out.DurationMs = end.Sub(s.impState.StartedAt).Milliseconds()
	}
	if s.enricher != nil {
		out.EnricherQueueDepth = s.enricher.QueueDepth()
	}
	if len(s.impQueue) > 0 {
		out.QueuedPaths = append([]string(nil), s.impQueue...)
	}
	s.idxMu.Lock()
	out.Indexer = s.idxState
	s.idxMu.Unlock()
	return out
}

func (s *LibraryService) runImport(ctx context.Context, path string) {
	// Pass 1: fast inventory (book rows only, no passages).
	summary, err := s.importer.InventoryPath(ctx, path, s.onImportProgress)

	s.impMu.Lock()
	s.impState.Running = false
	s.impState.Done = true
	s.impState.FinishedAt = time.Now()
	s.impState.Current = ""
	s.impState.Cancelled = ctx.Err() != nil
	if err != nil && ctx.Err() == nil {
		s.impState.Error = err.Error()
	}
	sum := summary
	s.impState.Summary = &sum
	s.impState.Imported = summary.Imported
	s.impState.Updated = summary.Updated
	s.impState.Skipped = summary.Skipped
	s.impState.Failed = summary.Failed
	s.impCancel = nil

	// Drain the next queued path, if any. Cancellation skips the queue —
	// if the user cancelled the active run, they probably don't want the
	// pending ones to keep going either.
	var nextPath string
	if ctx.Err() == nil && len(s.impQueue) > 0 {
		nextPath = s.impQueue[0]
		s.impQueue = s.impQueue[1:]
	} else if ctx.Err() != nil && len(s.impQueue) > 0 {
		s.impQueue = nil
	}
	if nextPath != "" {
		s.startImportLocked(nextPath)
	}
	s.impMu.Unlock()

	// Wake Pass 2 (indexer) and the enrichment scanner. The scanner will
	// pull eligible books in batches and feed them to the enricher. The
	// embedding scanner is also kicked so any books that finished Pass 2
	// during this import get vectorised promptly.
	s.kickIndexer()
	s.kickScanner()
	s.kickEmbedder()
	s.invalidateAggregations()
}

// kickIndexer is a non-blocking nudge for the indexer worker. Multiple
// kicks coalesce into a single wakeup since idxWake has capacity 1.
func (s *LibraryService) kickIndexer() {
	select {
	case s.idxWake <- struct{}{}:
	default:
	}
}

// runIndexer is the Pass 2 worker. It serialises all writes (matches our
// SetMaxOpenConns(1) + WAL setup), processes one queued book per turn, and
// goes back to sleep when the queue is empty. Any kickIndexer() call wakes
// it back up.
func (s *LibraryService) runIndexer() {
	for range s.idxWake {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			pending, err := s.store.QueuedCount(ctx)
			cancel()
			if err != nil || pending == 0 {
				s.idxMu.Lock()
				s.idxState.Running = false
				s.idxState.Current = ""
				s.idxState.Pending = 0
				s.idxMu.Unlock()
				break
			}

			s.idxMu.Lock()
			s.idxState.Running = true
			s.idxState.Pending = pending
			s.idxMu.Unlock()

			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			books, err := s.store.QueuedBooks(ctx, 1)
			cancel()
			if err != nil || len(books) == 0 {
				break
			}
			b := books[0]

			s.idxMu.Lock()
			s.idxState.Current = b.FilePath
			s.idxMu.Unlock()

			ctx, cancel = context.WithTimeout(context.Background(), 90*time.Second)
			passages, perr := importer.IndexBook(b)
			if perr != nil {
				_ = s.store.MarkIndexFailed(ctx, b.ID)
				s.idxMu.Lock()
				s.idxState.Failed++
				s.idxMu.Unlock()
			} else if err := s.store.IndexBookPassages(ctx, b.ID, passages); err != nil {
				_ = s.store.MarkIndexFailed(ctx, b.ID)
				s.idxMu.Lock()
				s.idxState.Failed++
				s.idxMu.Unlock()
			} else {
				s.idxMu.Lock()
				s.idxState.Indexed++
				s.idxMu.Unlock()
				// Queue this freshly-indexed book for metadata enrichment
				// and embedding backfill.
				s.enricher.Request(b.ID)
				s.kickEmbedder()
				s.invalidateAggregations()
				s.invalidateTOC(b.ID)
			}
			cancel()
		}
	}
}

func (s *LibraryService) onImportProgress(p importer.Progress) {
	s.impMu.Lock()
	defer s.impMu.Unlock()
	s.impState.Discovering = p.Discovering
	s.impState.Total = p.Total
	s.impState.Processed = p.Processed
	s.impState.Imported = p.Imported
	s.impState.Updated = p.Updated
	s.impState.Skipped = p.Skipped
	s.impState.Failed = p.Failed
	s.impState.Current = p.Current
	if p.LastError != "" {
		s.impState.RecentErrors = appendCapped(s.impState.RecentErrors, p.LastError, 20)
	}
}

func appendCapped(slice []string, value string, max int) []string {
	slice = append(slice, value)
	if len(slice) > max {
		slice = slice[len(slice)-max:]
	}
	return slice
}
