package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close test store: %v", err)
		}
	})
	return store
}

func TestDefaultDBPathUsesLibraryDB(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom.db")
	t.Setenv("LIBRARY_DB", want)

	got, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	if got != want {
		t.Fatalf("DefaultDBPath = %q, want %q", got, want)
	}
}

func TestDefaultDBPathIgnoresLegacyBooksDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LIBRARY_DB", "")

	legacy := filepath.Join(home, "Library", "Application Support", legacyConfigDirName, "library.db")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("create legacy dir: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write legacy marker: %v", err)
	}

	got, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	want := filepath.Join(home, "Library", "Application Support", configDirName, "library.db")
	if got != want {
		t.Fatalf("DefaultDBPath = %q, want %q", got, want)
	}
}

func TestOpenMigratesLegacyDBToA3TLibrary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LIBRARY_DB", "")

	base := filepath.Join(home, "Library", "Application Support")
	legacy := filepath.Join(base, legacyConfigDirName, "library.db")
	next := filepath.Join(base, configDirName, "library.db")

	legacyStore, err := Open(legacy)
	if err != nil {
		t.Fatalf("open legacy store: %v", err)
	}
	_, err = legacyStore.UpsertImportedBook(context.Background(), ImportBook{
		Book: Book{
			ID:             "legacy-book",
			Title:          "Legacy Book",
			FilePath:       "/tmp/legacy.epub",
			Format:         "epub",
			FileHash:       "legacy-hash",
			FileSize:       12,
			MetadataSource: "test",
			CoverURL:       "https://example.test/legacy.jpg",
			Description:    "Migrated from the old app name.",
		},
	})
	if err != nil {
		t.Fatalf("seed legacy store: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("close legacy store: %v", err)
	}

	store, err := Open("")
	if err != nil {
		t.Fatalf("open migrated store: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(next); err != nil {
		t.Fatalf("new db not created at %q: %v", next, err)
	}
	stats, err := store.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Books != 1 {
		t.Fatalf("migrated stats books = %d, want 1", stats.Books)
	}
}

func TestOpenDoesNotMigrateLegacyDBWhenLibraryDBIsSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	custom := filepath.Join(t.TempDir(), "custom.db")
	t.Setenv("LIBRARY_DB", custom)

	base := filepath.Join(home, "Library", "Application Support")
	legacy := filepath.Join(base, legacyConfigDirName, "library.db")
	legacyStore, err := Open(legacy)
	if err != nil {
		t.Fatalf("open legacy store: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("close legacy store: %v", err)
	}

	store, err := Open("")
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	defer store.Close()

	stats, err := store.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Books != 0 {
		t.Fatalf("custom store books = %d, want empty database", stats.Books)
	}
}

func TestStoreImportSearchAndReadPaths(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)

	imported := ImportBook{
		Book: Book{
			ID:                  "book-1",
			Title:               "Practical Search",
			Authors:             "Ada Lovelace",
			Description:         "A field guide to local-first library search.",
			Publisher:           "A3T",
			PublishedDate:       "2026",
			Language:            "en",
			CoverURL:            "https://example.test/cover.jpg",
			FilePath:            "/tmp/practical-search.epub",
			Format:              "epub",
			FileHash:            "hash-1",
			FileSize:            42,
			MetadataSource:      "test",
			MetadataRefreshedAt: now,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
		Passages: []Passage{
			{ID: "p1", Label: "Chapter 1", ChunkIndex: 0, Text: "SQLite FTS makes local search fast."},
			{ID: "p2", Label: "Chapter 1", ChunkIndex: 1, Text: "Passage search should return snippets."},
			{ID: "p3", Label: "Chapter 2", ChunkIndex: 2, Text: "Metadata enrichment is optional."},
		},
	}

	updated, err := store.UpsertImportedBook(ctx, imported)
	if err != nil {
		t.Fatalf("UpsertImportedBook: %v", err)
	}
	if updated {
		t.Fatal("first import reported update")
	}

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Books != 1 || stats.Passages != 3 || stats.NeedsMetadata != 0 {
		t.Fatalf("Stats = %+v, want 1 book, 3 passages, 0 needs metadata", stats)
	}

	books, err := store.SearchBooks(ctx, "local-first", 10, 0)
	if err != nil {
		t.Fatalf("SearchBooks: %v", err)
	}
	if len(books) != 1 || books[0].ID != "book-1" || books[0].PassageCount != 3 {
		t.Fatalf("SearchBooks = %+v, want imported book with passage count", books)
	}

	passages, err := store.SearchPassages(ctx, "snippets", "", 10)
	if err != nil {
		t.Fatalf("SearchPassages: %v", err)
	}
	if len(passages) != 1 || passages[0].ID != "p2" || passages[0].BookTitle != "Practical Search" {
		t.Fatalf("SearchPassages = %+v, want p2 with book metadata", passages)
	}

	page, err := store.BookPassages(ctx, "book-1", 1, 2)
	if err != nil {
		t.Fatalf("BookPassages: %v", err)
	}
	if len(page) != 2 || page[0].ID != "p2" || page[1].ID != "p3" {
		t.Fatalf("BookPassages page = %+v, want p2,p3", page)
	}

	toc, err := store.BookTOC(ctx, "book-1")
	if err != nil {
		t.Fatalf("BookTOC: %v", err)
	}
	if len(toc) != 2 || toc[0].Label != "Chapter 1" || toc[0].Pages != 2 || toc[1].Label != "Chapter 2" {
		t.Fatalf("BookTOC = %+v, want grouped chapters", toc)
	}

	updated, err = store.UpsertImportedBook(ctx, imported)
	if err != nil {
		t.Fatalf("second UpsertImportedBook: %v", err)
	}
	if !updated {
		t.Fatal("second import did not report update")
	}
}
