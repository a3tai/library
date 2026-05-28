package library

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

const (
	defaultLimit        = 80
	configDirName       = "A3T Library"
	legacyConfigDirName = "Books"
)

type Store struct {
	db *sql.DB
	// writeMu serialises all writers at the application level. SQLite's
	// WAL allows N concurrent readers + 1 writer; this mutex makes sure
	// our N goroutines (importer, indexer, enricher, UI metadata writes)
	// never collide on the writer slot, which would otherwise surface as
	// SQLITE_BUSY ("database is locked") errors despite the busy_timeout.
	writeMu sync.Mutex
}

func DefaultDBPath() (string, error) {
	if path := firstSetEnv("LIBRARY_DB"); path != "" {
		return path, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, configDirName, "library.db"), nil
}

func firstSetEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func Open(path string) (*Store, error) {
	if path == "" {
		defaultEnvPath := firstSetEnv("LIBRARY_DB")
		var err error
		path, err = DefaultDBPath()
		if err != nil {
			return nil, err
		}
		if defaultEnvPath == "" {
			if err := migrateLegacyDBIfNeeded(path); err != nil {
				return nil, err
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// Pragmas applied per-connection via the URL query string. WAL is the
	// foundation; synchronous=NORMAL is the canonical "fast but crash-safe"
	// setting (one-fsync-per-checkpoint instead of one-per-commit). The
	// 64 MiB page cache + 256 MiB mmap window keeps hot lookups in memory
	// across the read pool. busy_timeout=10s tolerates the occasional
	// reader/writer contention without surfacing SQLITE_BUSY to callers.
	db, err := sql.Open("sqlite",
		path+
			"?_pragma=journal_mode(WAL)"+
			"&_pragma=synchronous(NORMAL)"+
			"&_pragma=busy_timeout(30000)"+
			"&_pragma=foreign_keys(1)"+
			"&_pragma=temp_store(MEMORY)"+
			"&_pragma=cache_size(-65536)"+
			"&_pragma=mmap_size(268435456)")
	if err != nil {
		return nil, err
	}
	// WAL allows N readers + 1 writer concurrently. SetMaxOpenConns=1 was
	// strangling reads behind writes; 4 connections lets the UI poll
	// Stats / ListBooks while a long inventory commit is in flight.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func migrateLegacyDBIfNeeded(target string) error {
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	legacy := filepath.Join(base, legacyConfigDirName, "library.db")
	if legacy == target {
		return nil
	}
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(legacy); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	checkpointLegacyDB(legacy)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := copyFileIfExists(legacy+suffix, target+suffix); err != nil {
			return err
		}
	}
	return nil
}

func checkpointLegacyDB(path string) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(30000)")
	if err != nil {
		return
	}
	defer db.Close()
	_, _ = db.Exec("PRAGMA wal_checkpoint(FULL)")
}

func copyFileIfExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode().Perm())
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}
