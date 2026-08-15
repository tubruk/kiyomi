package webui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// openFromFS mounts the file system on a minimal http handler so tests
// can request files through the standard library's net/http stack —
// the same way Echo's static middleware ultimately serves them.
func openFromFS(t *testing.T, fs http.FileSystem, name string) (int, []byte) {
	t.Helper()
	srv := httptest.NewServer(http.FileServer(fs))
	defer srv.Close()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/"+name, nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, body
}

func TestFSEmptyDevDirReturnsEmbedded(t *testing.T) {
	fsys, src, err := FS("")
	if err != nil {
		t.Fatalf("FS(\"\") error: %v", err)
	}
	if src != SourceEmbedded {
		t.Fatalf("source = %q, want %q", src, SourceEmbedded)
	}
	if fsys == nil {
		t.Fatal("fsys is nil")
	}

	// The placeholder index.html committed in dist/ must always be
	// served from the embed.
	status, body := openFromFS(t, fsys, "index.html")
	if status != http.StatusOK {
		t.Fatalf("GET index.html status = %d, want 200", status)
	}
	if len(body) == 0 {
		t.Fatal("index.html body is empty")
	}
}

func TestFSRelativeDevDirResolvedAgainstCaller(t *testing.T) {
	// Caller is responsible for any HOME-relative resolution; FS()
	// treats the devDir argument as already-resolved path. This test
	// uses an absolute path so the semantics under test are: the
	// directory's contents are served, not the embedded bundle.
	dir := t.TempDir()
	marker := []byte("<html>dev-marker</html>")
	if err := os.WriteFile(filepath.Join(dir, "index.html"), marker, 0o644); err != nil {
		t.Fatal(err)
	}

	fsys, src, err := FS(dir)
	if err != nil {
		t.Fatalf("FS(devDir) error: %v", err)
	}
	if src != SourceDevLocal {
		t.Fatalf("source = %q, want %q", src, SourceDevLocal)
	}

	status, body := openFromFS(t, fsys, "index.html")
	if status != http.StatusOK {
		t.Fatalf("GET index.html status = %d, want 200", status)
	}
	if string(body) != string(marker) {
		t.Fatalf("body = %q, want %q (dev dir must override embed)", body, marker)
	}
}

func TestFSMissingDevDirFallsBackToEmbedded(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	fsys, src, err := FS(missing)
	if err == nil {
		t.Fatal("expected an error for missing dev dir")
	}
	if src != SourceEmbedded {
		t.Fatalf("source = %q, want %q (fallback)", src, SourceEmbedded)
	}
	if fsys == nil {
		t.Fatal("fsys must still be returned on fallback")
	}
}

func TestFSNonDirectoryDevDirFallsBackToEmbedded(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	fsys, src, err := FS(file)
	if err == nil {
		t.Fatal("expected an error when devDir is a regular file")
	}
	if src != SourceEmbedded {
		t.Fatalf("source = %q, want %q (fallback)", src, SourceEmbedded)
	}
	if fsys == nil {
		t.Fatal("fsys must still be returned on fallback")
	}
}

func TestIndexBytesEmbedded(t *testing.T) {
	body, err := IndexBytes("")
	if err != nil {
		t.Fatalf("IndexBytes(\"\") error: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("IndexBytes returned empty body")
	}
}

func TestIndexBytesDevDir(t *testing.T) {
	dir := t.TempDir()
	want := []byte("<html>dev-index</html>")
	if err := os.WriteFile(filepath.Join(dir, "index.html"), want, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := IndexBytes(dir)
	if err != nil {
		t.Fatalf("IndexBytes(devDir) error: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("IndexBytes = %q, want %q", got, want)
	}
}
