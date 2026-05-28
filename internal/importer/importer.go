package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a3tai/library/internal/library"
)

type Importer struct {
	store *library.Store
}

func New(store *library.Store) *Importer {
	return &Importer{store: store}
}

// Progress is the live state surfaced during a long-running import. Callers
// receive it via the Progress callback passed to ImportPathWithProgress.
type Progress struct {
	Discovering bool // true while the importer is still walking the tree
	Total       int  // files discovered so far (climbs during walk, frozen after)
	Processed   int  // files attempted (success + failure)
	Imported    int
	Updated     int
	Skipped     int
	Failed      int
	Current     string // path of the file currently being read
	LastError   string // most recent failure message, if any
}

// ProgressFunc receives a snapshot of the progress at well-defined moments:
// once after the pre-walk (so Total is known), and once after every file is
// attempted. Implementations should be cheap and non-blocking; the importer
// holds no mutex while calling.
type ProgressFunc func(Progress)

func (i *Importer) ImportPath(ctx context.Context, path string) (library.ImportSummary, error) {
	return i.ImportPathWithProgress(ctx, path, nil)
}

// InventoryPath is Pass 1 of the multi-pass import: walk the directory,
// hash + stat each supported file, peek at cheap embedded metadata, and
// upsert a book row marked index_status="queued". No passages, no FTS
// passage indexing, no online lookups. Returns when the entire walk is done.
func (i *Importer) InventoryPath(ctx context.Context, path string, onProgress ProgressFunc) (library.ImportSummary, error) {
	var summary library.ImportSummary
	path = strings.TrimSpace(path)
	if path == "" {
		return summary, fmt.Errorf("path is required")
	}
	info, err := importPathInfo(path)
	if err != nil {
		return summary, err
	}

	paths, err := walkSupported(ctx, path, info, &summary, onProgress)
	if err != nil {
		return summary, err
	}

	if len(paths) == 0 {
		summary.Skipped++
		summary.Messages = append(summary.Messages, "No supported EPUB, PDF, or TXT files found")
		if onProgress != nil {
			onProgress(progressFromSummary(summary, 0, ""))
		}
		return summary, nil
	}

	progress := Progress{Total: len(paths)}
	if onProgress != nil {
		onProgress(progress)
	}

	const batchSize = 64
	pending := make([]library.Book, 0, batchSize)
	pendingPaths := make([]string, 0, batchSize)
	flush := func() {
		if len(pending) == 0 {
			return
		}
		results, err := i.store.UpsertInventoryBooks(ctx, pending)
		if err != nil {
			// Whole-batch failure — degrade gracefully and surface the error
			// against each pending path so the user sees what happened.
			for _, p := range pendingPaths {
				summary.Failed++
				progress.Failed++
				progress.LastError = fmt.Sprintf("%s: %v", p, err)
				summary.Messages = append(summary.Messages, progress.LastError)
				progress.Processed++
			}
		} else {
			for idx, r := range results {
				if r.Err != nil {
					summary.Failed++
					progress.Failed++
					path := ""
					if idx < len(pendingPaths) {
						path = pendingPaths[idx]
					}
					progress.LastError = fmt.Sprintf("%s: %v", path, r.Err)
					summary.Messages = append(summary.Messages, progress.LastError)
				} else if r.Existed {
					summary.Updated++
					progress.Updated++
				} else {
					summary.Imported++
					progress.Imported++
				}
				progress.Processed++
			}
		}
		pending = pending[:0]
		pendingPaths = pendingPaths[:0]
		if onProgress != nil {
			onProgress(progress)
		}
	}

	for _, filePath := range paths {
		if ctx.Err() != nil {
			break
		}
		progress.Current = filePath
		if onProgress != nil {
			onProgress(progress)
		}

		book, err := Inventory(filePath)
		if err != nil {
			summary.Failed++
			progress.Failed++
			progress.LastError = fmt.Sprintf("%s: %v", filePath, err)
			summary.Messages = append(summary.Messages, progress.LastError)
			progress.Processed++
			if onProgress != nil {
				onProgress(progress)
			}
			continue
		}
		pending = append(pending, book)
		pendingPaths = append(pendingPaths, filePath)
		if len(pending) >= batchSize {
			flush()
		}
	}
	flush()
	return summary, ctx.Err()
}

// walkSupported recursively scans `path` for files the importer can handle
// and returns them in walk order. Throughout the walk it emits Progress
// updates with Discovering=true so the UI can show "Discovering: N files…"
// instead of a frozen dialog while a deep folder is enumerated. Updates
// are throttled (every 200 files or 100ms) so a 200k-file tree doesn't
// flood the IPC channel.
func walkSupported(ctx context.Context, path string, info os.FileInfo, summary *library.ImportSummary, onProgress ProgressFunc) ([]string, error) {
	paths := []string{}
	if !info.IsDir() {
		if supported(path) {
			paths = append(paths, path)
		}
		return paths, nil
	}

	const (
		emitEveryFiles = 200
		emitEveryDur   = 100 * time.Millisecond
	)
	lastEmit := time.Now()
	emitProgress := func(force bool) {
		if onProgress == nil {
			return
		}
		if !force && time.Since(lastEmit) < emitEveryDur && len(paths)%emitEveryFiles != 0 {
			return
		}
		onProgress(Progress{Discovering: true, Total: len(paths)})
		lastEmit = time.Now()
	}
	emitProgress(true) // immediate "Discovering: 0" so the dialog opens with intent

	err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			summary.Failed++
			summary.Messages = append(summary.Messages, fmt.Sprintf("%s: %v", filePath, walkErr))
			emitProgress(false)
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			emitProgress(false)
			return nil
		}
		if d.IsDir() {
			emitProgress(false)
			return nil
		}
		if supported(filePath) {
			paths = append(paths, filePath)
			emitProgress(false)
		}
		return nil
	})
	emitProgress(true) // final discovery snapshot before processing kicks in
	return paths, err
}

func (i *Importer) ImportPathWithProgress(ctx context.Context, path string, onProgress ProgressFunc) (library.ImportSummary, error) {
	var summary library.ImportSummary
	path = strings.TrimSpace(path)
	if path == "" {
		return summary, fmt.Errorf("path is required")
	}
	info, err := importPathInfo(path)
	if err != nil {
		return summary, err
	}

	paths, err := walkSupported(ctx, path, info, &summary, onProgress)
	if err != nil {
		return summary, err
	}

	if len(paths) == 0 {
		summary.Skipped++
		summary.Messages = append(summary.Messages, "No supported EPUB, PDF, or TXT files found")
		if onProgress != nil {
			onProgress(progressFromSummary(summary, 0, ""))
		}
		return summary, nil
	}

	progress := Progress{Total: len(paths)}
	if onProgress != nil {
		onProgress(progress)
	}

	for _, filePath := range paths {
		if ctx.Err() != nil {
			break
		}
		progress.Current = filePath
		if onProgress != nil {
			onProgress(progress)
		}

		imported, err := ImportFile(filePath)
		if err != nil {
			summary.Failed++
			progress.Failed++
			progress.LastError = fmt.Sprintf("%s: %v", filePath, err)
			summary.Messages = append(summary.Messages, progress.LastError)
		} else if updated, err := i.store.UpsertImportedBook(ctx, imported); err != nil {
			summary.Failed++
			progress.Failed++
			progress.LastError = fmt.Sprintf("%s: %v", filePath, err)
			summary.Messages = append(summary.Messages, progress.LastError)
		} else if updated {
			summary.Updated++
			progress.Updated++
		} else {
			summary.Imported++
			progress.Imported++
		}
		progress.Processed++
		if onProgress != nil {
			onProgress(progress)
		}
	}
	return summary, ctx.Err()
}

func progressFromSummary(s library.ImportSummary, total int, current string) Progress {
	return Progress{
		Total:     total,
		Processed: s.Imported + s.Updated + s.Skipped + s.Failed,
		Imported:  s.Imported,
		Updated:   s.Updated,
		Skipped:   s.Skipped,
		Failed:    s.Failed,
		Current:   current,
	}
}

func ImportFile(path string) (library.ImportBook, error) {
	if _, err := importFileInfo(path); err != nil {
		return library.ImportBook{}, err
	}
	var (
		out library.ImportBook
		err error
	)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub":
		out, err = ImportEPUB(path)
	case ".pdf":
		out, err = ImportPDF(path)
	case ".txt":
		out, err = ImportText(path)
	default:
		return library.ImportBook{}, fmt.Errorf("unsupported file type: %s", filepath.Ext(path))
	}
	if err != nil {
		return out, err
	}
	if out.Book.FileSize == 0 {
		if info, statErr := os.Stat(path); statErr == nil {
			out.Book.FileSize = info.Size()
		}
	}
	return out, nil
}

func supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub", ".pdf", ".txt":
		return true
	default:
		return false
	}
}

// fileHash computes a full-content SHA-256 of the file. Used by the
// passage-extracting Pass 2 path where we already have the file open and
// fully read for text extraction, so the hash is effectively free.
func fileHash(path string) (string, error) {
	if _, err := importFileInfo(path); err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// partialFileHash is kept for older tests/migrations that may need the former
// fast identity algorithm. The importer now uses full SHA-256 so middle-of-file
// edits cannot be missed by the inventory pass. This helper hashes
// (file size, first 64 KiB, last 64 KiB) into a SHA-256 instead of the
// entire file.
//
// The returned string is prefixed with "p1:" so existing rows whose
// file_hash holds the legacy full SHA-256 stay distinct from new inventory
// rows. Re-imports of the same file always produce the same partial hash,
// so the UNIQUE constraint on books.file_hash continues to act as a dedup.
func partialFileHash(path string) (string, error) {
	if _, err := importFileInfo(path); err != nil {
		return "", err
	}
	const window = 64 * 1024
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := info.Size()
	if size > maxImportFileBytes {
		return "", fmt.Errorf("file is too large: %d bytes (limit %d)", size, maxImportFileBytes)
	}
	h := sha256.New()
	// Mix the size in first so two files that share head/tail bytes but
	// differ in length still produce different hashes.
	var sizeBuf [8]byte
	for i := 0; i < 8; i++ {
		sizeBuf[i] = byte(size >> (8 * i))
	}
	h.Write(sizeBuf[:])

	if size <= 2*window {
		// Small file — just hash the whole thing once.
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		return "p1:" + hex.EncodeToString(h.Sum(nil)), nil
	}

	head := make([]byte, window)
	if _, err := io.ReadFull(f, head); err != nil {
		return "", err
	}
	h.Write(head)

	tail := make([]byte, window)
	if _, err := f.ReadAt(tail, size-window); err != nil && err != io.EOF {
		return "", err
	}
	h.Write(tail)
	return "p1:" + hex.EncodeToString(h.Sum(nil)), nil
}

func importPathInfo(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symlinks are not supported for import: %s", path)
	}
	if info.IsDir() {
		return info, nil
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	if info.Size() > maxImportFileBytes {
		return nil, fmt.Errorf("file is too large: %d bytes (limit %d)", info.Size(), maxImportFileBytes)
	}
	return info, nil
}

func importFileInfo(path string) (os.FileInfo, error) {
	info, err := importPathInfo(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("expected a file, got directory: %s", path)
	}
	return info, nil
}

func titleFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.Join(strings.Fields(base), " ")
	if base == "" {
		return "Untitled"
	}
	return base
}
