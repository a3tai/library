package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/a3tai/library/internal/library"
)

func TestWalkSupportedSkipsSymlinks(t *testing.T) {
	dir := t.TempDir()
	safe := filepath.Join(dir, "safe.txt")
	if err := os.WriteFile(safe, []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "secret.txt")
	if err := os.Symlink(safe, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	info, err := importPathInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	var summary library.ImportSummary
	paths, err := walkSupported(context.Background(), dir, info, &summary, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != safe {
		t.Fatalf("paths = %#v, want only %q", paths, safe)
	}
	if _, err := ImportFile(link); err == nil {
		t.Fatal("ImportFile accepted a symlink")
	}
}

func TestInventoryFullHashSeesSameSizeMiddleChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.txt")
	if err := os.WriteFile(path, []byte("aaa-middle-zzz"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := Inventory(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("aaa-changed-zz"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Inventory(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.FileSize != second.FileSize {
		t.Fatalf("test setup changed file size: %d vs %d", first.FileSize, second.FileSize)
	}
	if first.FileHash == second.FileHash {
		t.Fatalf("same-size content change kept hash %q", first.FileHash)
	}
}
