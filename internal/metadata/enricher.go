package metadata

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/a3tai/library/internal/aimeta"
	"github.com/a3tai/library/internal/library"
)

// Enricher runs metadata lookups against external sources at a polite rate.
// A small pool of worker goroutines pops book IDs from two channels — a
// high-priority queue (for books the user just viewed) and a larger
// background queue (for bulk catch-up) — looks up metadata, and persists
// the result. Workers drain the high queue first. Repeated requests for
// the same book are deduplicated, and recent failures are skipped for a
// TTL so the queue doesn't loop on books with no online record.
//
// Politeness toward Open Library / Google Books is enforced by the rate
// limiter inside metadata.Client, not by the per-worker cooldown — the
// limiter is shared across all workers, so the global request rate is
// independent of enricherWorkers.
type Enricher struct {
	store     *library.Store
	client    *Client
	ai        *aimeta.Client
	queueHigh chan string
	queueBg   chan string
	cooldown  time.Duration
	failTTL   time.Duration

	mu       sync.Mutex
	failed   map[string]time.Time
	inflight map[string]bool
	enqueued int64
	updated  int64
}

// NewEnricher constructs an enricher and starts its worker pool.
// Workers exit when the queues close, but this codebase has no shutdown
// hook today so the enricher lives for the process lifetime.
//
// Concurrency: enricherWorkers worker goroutines pull from the same two
// queues (high-priority drained first). Politeness is enforced inside
// metadata.Client by a shared rate limiter, so the global request rate
// to Open Library / Google Books stays the same regardless of worker
// count. The cooldown field is kept as a tiny per-worker yield so a
// single hot book ID doesn't starve the limiter for the others.
const enricherWorkers = 4

func NewEnricher(store *library.Store, client *Client, ai *aimeta.Client) *Enricher {
	if client == nil {
		client = New()
	}
	e := &Enricher{
		store:     store,
		client:    client,
		ai:        ai,
		queueHigh: make(chan string, 256),
		queueBg:   make(chan string, 4096),
		cooldown:  50 * time.Millisecond,
		failTTL:   30 * time.Minute,
		failed:    make(map[string]time.Time),
		inflight:  make(map[string]bool),
	}
	for i := 0; i < enricherWorkers; i++ {
		go e.run()
	}
	return e
}

// Request enqueues bookID for background enrichment. Returns true if the book
// was accepted; false if it was already in flight, recently failed, or the
// background queue is full. Use RequestPriority for books the user just
// viewed so they jump ahead of the bulk backlog.
func (e *Enricher) Request(bookID string) bool {
	return e.enqueue(bookID, e.queueBg)
}

// RequestPriority enqueues bookID into the high-priority queue. Bookkeeping
// is identical to Request; only the channel choice differs. The worker
// always drains queueHigh before pulling from queueBg.
func (e *Enricher) RequestPriority(bookID string) bool {
	return e.enqueue(bookID, e.queueHigh)
}

func (e *Enricher) enqueue(bookID string, ch chan<- string) bool {
	if strings.TrimSpace(bookID) == "" {
		return false
	}
	e.mu.Lock()
	if e.inflight[bookID] {
		e.mu.Unlock()
		return false
	}
	if t, ok := e.failed[bookID]; ok && time.Since(t) < e.failTTL {
		e.mu.Unlock()
		return false
	}
	e.inflight[bookID] = true
	e.mu.Unlock()

	select {
	case ch <- bookID:
		e.mu.Lock()
		e.enqueued++
		e.mu.Unlock()
		return true
	default:
		e.mu.Lock()
		delete(e.inflight, bookID)
		e.mu.Unlock()
		return false
	}
}

// RequestMany enqueues several books on the background queue. Returns the
// count actually accepted.
func (e *Enricher) RequestMany(bookIDs []string) int {
	accepted := 0
	for _, id := range bookIDs {
		if e.Request(id) {
			accepted++
		}
	}
	return accepted
}

// QueueDepth returns the total pending count across both queues.
func (e *Enricher) QueueDepth() int {
	return len(e.queueHigh) + len(e.queueBg)
}

// Stats returns total accepted + total updated counters.
func (e *Enricher) Stats() (enqueued, updated int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.enqueued, e.updated
}

func (e *Enricher) run() {
	for {
		// Always drain the high-priority queue first. select-with-default
		// is non-blocking, so we fall through to the blocking select only
		// when both queues are momentarily empty.
		select {
		case id := <-e.queueHigh:
			e.process(id)
			time.Sleep(e.cooldown)
			continue
		default:
		}
		select {
		case id := <-e.queueHigh:
			e.process(id)
		case id := <-e.queueBg:
			e.process(id)
		}
		time.Sleep(e.cooldown)
	}
}

func (e *Enricher) process(bookID string) {
	defer e.markDone(bookID)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	book, err := e.store.GetBook(ctx, bookID)
	if err != nil {
		e.markFailed(bookID)
		return
	}

	updated := false
	if needsMetadata(book) {
		if found, ok := e.client.Lookup(ctx, book); ok {
			found.ID = book.ID
			if err := e.store.UpdateMetadata(ctx, found); err != nil {
				e.markFailed(bookID)
				return
			}
			// Subjects + Categories ride alongside the scalar metadata write
			// but live in their own join tables; UpdateMetadata doesn't touch
			// them, so we publish them explicitly.
			if len(found.Subjects) > 0 {
				_ = e.store.AddBookSubjects(ctx, bookID, found.Subjects, found.MetadataSource)
			}
			if len(found.Categories) > 0 {
				_ = e.store.AddBookCategories(ctx, bookID, found.Categories, found.MetadataSource)
			}
			updated = true
			// Refresh local view so the AI step sees the freshly-merged book.
			if refreshed, err := e.store.GetBook(ctx, bookID); err == nil {
				book = refreshed
			}
		}
	}

	if e.shouldRunAI(ctx, book) {
		if err := e.runAIEnrichment(ctx, book); err == nil {
			updated = true
		}
	}

	if updated {
		e.mu.Lock()
		e.updated++
		e.mu.Unlock()
		return
	}
	e.markFailed(bookID)
}

// shouldRunAI gates the LM Studio call: must have a configured client, the
// book must have indexed text to sample, and the book must not already have
// subjects on file (from Open Library / Google Books or a previous AI pass).
func (e *Enricher) shouldRunAI(ctx context.Context, book library.Book) bool {
	if e.ai == nil {
		return false
	}
	if book.IndexStatus != "indexed" {
		return false
	}
	if book.PassageCount == 0 {
		return false
	}
	has, err := e.store.HasSubjects(ctx, book.ID)
	if err == nil && has {
		return false
	}
	return true
}

// runAIEnrichment samples a few passages and asks LM Studio for structured
// metadata. Subjects/Categories land in their own join tables; a generated
// summary is folded into the book's description only when no description
// exists yet (we never overwrite the publisher-supplied prose).
func (e *Enricher) runAIEnrichment(ctx context.Context, book library.Book) error {
	if !e.ai.Available(ctx) {
		return fmt.Errorf("LM Studio unavailable")
	}
	samples, err := e.sampleBookPassages(ctx, book.ID, 4)
	if err != nil || len(samples) == 0 {
		if err == nil {
			err = fmt.Errorf("no sample passages")
		}
		return err
	}
	aictx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	meta, err := e.ai.ExtractMetadata(aictx, book.Title, book.Authors, samples)
	if err != nil {
		return err
	}
	if len(meta.Subjects) == 0 && len(meta.Categories) == 0 && meta.Summary == "" {
		return fmt.Errorf("AI returned no usable fields")
	}

	wrote := false
	if len(meta.Subjects) > 0 {
		if err := e.store.AddBookSubjects(ctx, book.ID, meta.Subjects, "lmstudio"); err == nil {
			wrote = true
		}
	}
	if len(meta.Categories) > 0 {
		if err := e.store.AddBookCategories(ctx, book.ID, meta.Categories, "lmstudio"); err == nil {
			wrote = true
		}
	}

	// Only fill the description if there isn't one already — the AI summary
	// is grounded in 4 sample passages, so it's lower-quality than the real
	// publisher description when one exists.
	if strings.TrimSpace(book.Description) == "" && strings.TrimSpace(meta.Summary) != "" {
		updated := book
		updated.Description = strings.TrimSpace(meta.Summary)
		updated.MetadataSource = appendSource(book.MetadataSource, "lmstudio")
		if err := e.store.UpdateMetadata(ctx, updated); err == nil {
			wrote = true
		}
	} else if book.MetadataSource != appendSource(book.MetadataSource, "lmstudio") {
		// Keep the source list in sync even when we didn't touch the description.
		updated := book
		updated.MetadataSource = appendSource(book.MetadataSource, "lmstudio")
		if err := e.store.UpdateMetadata(ctx, updated); err == nil {
			wrote = true
		}
	}
	if !wrote {
		return fmt.Errorf("AI returned no usable fields")
	}
	return nil
}

func (e *Enricher) sampleBookPassages(ctx context.Context, bookID string, n int) ([]string, error) {
	rows, err := e.store.BookPassages(ctx, bookID, 0, 200)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if n <= 0 {
		n = 4
	}
	// Filter to passages that look like real prose, not boilerplate. Front
	// matter, blank-page artefacts, copyright notices, and short TOC blurbs
	// drag the AI off the actual content; dropping them costs nothing and
	// reliably improves the subjects/categories the model returns.
	keep := rows[:0]
	for _, r := range rows {
		t := strings.TrimSpace(r.Text)
		if len(t) < 800 {
			continue
		}
		if looksLikeBoilerplate(t) {
			continue
		}
		keep = append(keep, r)
	}
	if len(keep) == 0 {
		// Fall back to whatever we have rather than returning nothing — a
		// noisy sample is still better than nothing for short books.
		keep = rows
	}
	if len(keep) <= n {
		out := make([]string, 0, len(keep))
		for _, r := range keep {
			out = append(out, r.Text)
		}
		return out, nil
	}
	// Skip the very first passage (often a title page) and evenly space the
	// rest across the remainder so we cover the actual work.
	span := keep
	if len(span) > n+1 {
		span = span[1:]
	}
	out := make([]string, 0, n)
	step := len(span) / n
	for i := 0; i < n; i++ {
		out = append(out, span[i*step].Text)
	}
	return out, nil
}

// looksLikeBoilerplate flags passages that are mostly front matter, legal
// notices, or table-of-contents fragments — text that wastes AI context
// without describing what the book is actually about.
func looksLikeBoilerplate(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"all rights reserved",
		"library of congress cataloging",
		"isbn-13",
		"printed in the united states",
		"table of contents",
		"copyright ©",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func appendSource(existing, additional string) string {
	if existing == "" {
		return additional
	}
	for _, s := range strings.Split(existing, ",") {
		if strings.TrimSpace(s) == additional {
			return existing
		}
	}
	return existing + "," + additional
}

func (e *Enricher) markDone(bookID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.inflight, bookID)
}

func (e *Enricher) markFailed(bookID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.failed[bookID] = time.Now()
}

func needsMetadata(b library.Book) bool {
	if strings.TrimSpace(b.MetadataSource) == "" {
		return true
	}
	if strings.TrimSpace(b.CoverURL) == "" {
		return true
	}
	if strings.TrimSpace(b.Description) == "" {
		return true
	}
	return false
}
