package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/a3tai/library/internal/embedder"
	"github.com/a3tai/library/internal/library"
	"github.com/a3tai/library/internal/mcpserver"
)

func main() {
	dbPath := flag.String("db", "", "path to the A3T: Library SQLite database")
	flag.Parse()

	store, err := library.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// search_passages goes through the LM Studio embeddings client when
	// it's reachable; otherwise the server falls back to FTS5. The desktop
	// app and this stdio binary therefore behave identically. The
	// stdio binary has no settings UI, so DB-stored settings are folded in
	// at startup and the provider isn't swapped at runtime.
	stored, _ := store.GetSettings(context.Background())
	emb := embedder.NewFromConfig(embedder.Resolve(stored))
	mcpserver.NewWithEmbedder(store, embedder.NewProvider(emb)).ServeStdio(os.Stdin, os.Stdout)
}
