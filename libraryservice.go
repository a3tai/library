package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/a3tai/library/internal/aimeta"
	"github.com/a3tai/library/internal/embedder"
	"github.com/a3tai/library/internal/importer"
	"github.com/a3tai/library/internal/library"
	"github.com/a3tai/library/internal/metadata"
)

type LibraryService struct {
	store    *library.Store
	importer *importer.Importer
	metadata *metadata.Client
	enricher *metadata.Enricher
	// LM Studio clients. aiPtr/embed are hot-swappable: the settings panel
	// rebuilds them when the user changes LM Studio URL/key/model. Read
	// aiPtr through s.aimeta() so writers and readers don't race.
	ai       *aimeta.Client
	embed    *embedder.Provider
	aiMu     sync.RWMutex
	mcp      *http.Server
	mcpURL   string
	mcpToken string
	mu       sync.Mutex
	busy     bool

	// Importer state guarded by impMu.
	impMu     sync.Mutex
	impState  importerState
	impCancel context.CancelFunc
	// Pending paths picked while another import was running. FIFO drained
	// in runImport after the current pass finishes. The Add Books button
	// stays enabled at all times — clicks during a run land here.
	impQueue []string

	// Indexer (Pass 2) state.
	idxWake  chan struct{}
	idxMu    sync.Mutex
	idxState indexerState

	// Enrichment scanner — wakes the worker when new books arrive.
	scanWake chan struct{}

	// Embedding backfill scanner — wakes after Pass 2 indexing or import
	// when there are books with passages but no vectors yet.
	embedWake chan struct{}

	// Cache for aggregation queries (authors / subjects / genres). These
	// scan the entire books table, so re-running them on every search
	// keystroke was the bulk of search latency. Invalidated when the
	// importer or indexer commits new rows.
	aggMu       sync.Mutex
	aggAuthors  aggCacheEntry
	aggSubject  aggCacheEntry
	aggCategory aggCacheEntry

	// Per-book TOC cache. The chat system prompt re-asks for TOC on every
	// turn — keeping the bytes byte-stable lets the local LLM reuse its
	// KV cache prefix instead of re-prefilling the whole prompt. The frontend
	// reader also benefits. Entries are dropped when the book is re-indexed.
	tocMu    sync.Mutex
	tocCache map[string][]library.TOCEntry

	// Chat context-window budget cache. Keyed by `baseURL|model` so a
	// switch to a different model triggers a fresh lookup. Populated
	// lazily by resolveChatBudget on first chat turn.
	chatBudgetMu    sync.Mutex
	chatBudgetCache map[string]int
}

type aggCacheEntry struct {
	limit   int
	expires time.Time
	data    []library.AggregateGroup
}

const aggCacheTTL = 60 * time.Second

type indexerState struct {
	Running bool   `json:"running"`
	Current string `json:"current"`
	Pending int    `json:"pending"`
	Indexed int    `json:"indexed"`
	Failed  int    `json:"failed"`
}

type importerState struct {
	Running      bool                   `json:"running"`
	Discovering  bool                   `json:"discovering"`
	Path         string                 `json:"path"`
	StartedAt    time.Time              `json:"startedAt,omitempty"`
	FinishedAt   time.Time              `json:"finishedAt,omitempty"`
	Total        int                    `json:"total"`
	Processed    int                    `json:"processed"`
	Imported     int                    `json:"imported"`
	Updated      int                    `json:"updated"`
	Skipped      int                    `json:"skipped"`
	Failed       int                    `json:"failed"`
	Current      string                 `json:"current"`
	RecentErrors []string               `json:"recentErrors"`
	Done         bool                   `json:"done"`
	Cancelled    bool                   `json:"cancelled"`
	Error        string                 `json:"error,omitempty"`
	Summary      *library.ImportSummary `json:"summary,omitempty"`
}

// ImporterStatus is the JSON-friendly view returned to the frontend.
type ImporterStatus struct {
	importerState
	DurationMs         int64        `json:"durationMs"`
	EnricherQueueDepth int          `json:"enricherQueueDepth"`
	Indexer            indexerState `json:"indexer"`
	// Paths the user picked while another import was running. They will
	// be processed in FIFO order once the current run finishes; surface
	// them so the UI can show "N folders queued."
	QueuedPaths []string `json:"queuedPaths"`
}

type LibrarySnapshot struct {
	Books     []library.Book `json:"books"`
	Stats     library.Stats  `json:"stats"`
	DBPath    string         `json:"dbPath"`
	Hydrating bool           `json:"hydrating"`
	MCP       MCPStatus      `json:"mcp"`
}

type MCPStatus struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
	Port    int    `json:"port"`
	Token   string `json:"token,omitempty"`
}

func NewLibraryService(dbPath string) (*LibraryService, error) {
	store, err := library.Open(dbPath)
	if err != nil {
		return nil, err
	}
	client := metadata.New()
	// Resolve LM Studio config from DB-backed settings first, falling back
	// to env vars + defaults. Settings load errors are non-fatal — we just
	// use the env path.
	stored, _ := loadStoredSettings(store)
	ai := aimeta.NewFromConfig(aimeta.Resolve(stored))
	emb := embedder.NewFromConfig(embedder.Resolve(stored))
	s := &LibraryService{
		store:     store,
		importer:  importer.New(store),
		metadata:  client,
		enricher:  metadata.NewEnricher(store, client, ai),
		ai:        ai,
		embed:     embedder.NewProvider(emb),
		idxWake:   make(chan struct{}, 1),
		scanWake:  make(chan struct{}, 1),
		embedWake: make(chan struct{}, 1),
	}
	go s.runIndexer()
	go s.runEnrichmentScanner()
	go s.runEmbeddingScanner()
	// Wake all workers in case a previous run left work pending.
	s.kickIndexer()
	s.kickScanner()
	s.kickEmbedder()
	return s, nil
}

// kickScanner is a non-blocking nudge for the enrichment scanner. Used
// by import completion and other code paths that just produced new work.
func (s *LibraryService) kickScanner() {
	select {
	case s.scanWake <- struct{}{}:
	default:
	}
}

// runEnrichmentScanner is the long-lived background agent that keeps the
// enricher fed. The enricher itself does the per-book work (Open Library,
// Google Books, then LM Studio AI extraction); this loop just decides
// what to enqueue and when. The cadence is:
//
//   - Active: when there is work to do, refill aggressively (every 6s).
//     The enricher's queue is bounded, so this is mostly a no-op for full
//     queues — RequestMany returns the count actually accepted.
//   - Idle:   when the DB reports nothing pending, sleep for 5 min before
//     re-checking. A kickScanner() call from import completion can shorten
//     the wait.
//
// The loop runs forever; it exits only when the process exits.
func (s *LibraryService) runEnrichmentScanner() {
	const (
		batchSize    = 64
		activeSleep  = 6 * time.Second
		idleSleep    = 5 * time.Minute
		queueHighWat = 200 // back off when the enricher queue is already deep
	)
	for {
		// Block until something asks us to scan, with a long fallback timer.
		// The fallback ensures we still re-check periodically even if no
		// kickScanner() ever fires.
		idleTimer := time.NewTimer(idleSleep)
		select {
		case <-s.scanWake:
			if !idleTimer.Stop() {
				<-idleTimer.C
			}
		case <-idleTimer.C:
		}

		// Drain the queue tightly while there is work and the enricher
		// hasn't fallen behind. Each iteration pulls one batch and waits
		// briefly so the enricher can chew through it.
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			pending, err := s.store.CountBooksNeedingEnrichment(ctx)
			cancel()
			if err != nil || pending == 0 {
				break
			}
			if s.enricher != nil && s.enricher.QueueDepth() >= queueHighWat {
				time.Sleep(activeSleep)
				continue
			}
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			books, err := s.store.BooksNeedingEnrichment(ctx, batchSize)
			cancel()
			if err != nil || len(books) == 0 {
				break
			}
			ids := make([]string, 0, len(books))
			for _, b := range books {
				ids = append(ids, b.ID)
			}
			s.enricher.RequestMany(ids)
			time.Sleep(activeSleep)
		}
	}
}

// kickEmbedder is a non-blocking nudge for the embedding backfill scanner.
// Called whenever new passages land (Pass 2 indexer commit, import commit).
func (s *LibraryService) kickEmbedder() {
	select {
	case s.embedWake <- struct{}{}:
	default:
	}
}

// runEmbeddingScanner backfills passage_embeddings for any indexed book that
// doesn't yet have vectors under the current embedding model. Cheap when no
// LM Studio is reachable — Available() short-circuits and the loop falls
// back to idle sleep. On embed failure (typically "no model loaded") the
// loop bails to idle sleep so we don't hammer LM Studio in tight cycles.
//
// Logging is rate-limited: identical errors collapse to one line every
// ~60s so a misconfigured endpoint doesn't flood the console.
func (s *LibraryService) runEmbeddingScanner() {
	const (
		batchBooks  = 4
		activeSleep = 2 * time.Second
		idleSleep   = 5 * time.Minute
	)
	var lastLog string
	var lastLogAt time.Time
	logOnce := func(msg string) {
		// Only emit the same message once a minute. Different messages
		// always print so a recovery (or new failure mode) is visible.
		if msg == lastLog && time.Since(lastLogAt) < 60*time.Second {
			return
		}
		lastLog = msg
		lastLogAt = time.Now()
		log.Print(msg)
	}
	for {
		idleTimer := time.NewTimer(idleSleep)
		select {
		case <-s.embedWake:
			if !idleTimer.Stop() {
				<-idleTimer.C
			}
		case <-idleTimer.C:
		}
		client := s.embed.Get()
		if client == nil {
			continue
		}
		probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		available := client.Available(probeCtx)
		cancel()
		if !available {
			continue
		}

		model := client.Model
		// Inner loop with a fail-fast break: a single failure bounces us
		// back to the outer idle wait instead of re-attempting the same
		// books at 2-second cadence.
		failed := false
		for !failed {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			pending, err := s.store.CountBooksWithoutEmbeddings(ctx, model)
			cancel()
			if err != nil || pending == 0 {
				break
			}
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			ids, err := s.store.BooksWithoutEmbeddings(ctx, model, batchBooks)
			cancel()
			if err != nil || len(ids) == 0 {
				break
			}
			for _, id := range ids {
				if err := s.embedBookPassages(context.Background(), id, model); err != nil {
					logOnce(fmt.Sprintf("embed backfill paused: %v", err))
					failed = true
					break
				}
			}
			if !failed {
				time.Sleep(activeSleep)
			}
		}
	}
}

// embedBookPassages reads a book's passages, sends them to LM Studio in
// fixed-size batches, and writes the resulting vectors back in one
// transaction per book.
func (s *LibraryService) embedBookPassages(ctx context.Context, bookID, model string) error {
	const batchSize = 32 // tune to LM Studio's per-request budget
	client := s.embed.Get()
	if client == nil {
		return errors.New("embedder is not configured")
	}
	passages, err := s.store.PassagesForEmbedding(ctx, bookID)
	if err != nil {
		return err
	}
	if len(passages) == 0 {
		// Empty write keeps the book out of BooksWithoutEmbeddings —
		// strictly speaking the SQL already filters by passage_count > 0,
		// but defensively avoid an infinite retry on edge cases.
		return s.store.UpsertPassageEmbeddings(ctx, bookID, model, nil)
	}
	var all []library.PassageEmbedding
	for i := 0; i < len(passages); i += batchSize {
		end := i + batchSize
		if end > len(passages) {
			end = len(passages)
		}
		batch := passages[i:end]
		inputs := make([]string, len(batch))
		for j, p := range batch {
			inputs[j] = p.Text
		}
		callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		vecs, err := client.Embed(callCtx, inputs)
		cancel()
		if err != nil {
			return err
		}
		for j, v := range vecs {
			if len(v) == 0 {
				continue
			}
			all = append(all, library.PassageEmbedding{
				PassageID: batch[j].ID,
				BookID:    bookID,
				Vector:    v,
			})
		}
	}
	return s.store.UpsertPassageEmbeddings(ctx, bookID, model, all)
}

// Settings is the JSON-friendly view returned to the settings panel. URL
// / model / chat-model values fall back to env vars + defaults when
// blank, so the UI shows what's actually in effect. KeyConfigured is true
// when an API key is set in DB or env — the value itself is never sent to
// the frontend.
