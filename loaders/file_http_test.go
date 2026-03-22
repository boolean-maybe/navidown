package loaders

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boolean-maybe/navidown/navidown"
)

func TestFileHTTP_EmptyURL(t *testing.T) {
	f := &FileHTTP{}
	content, err := f.FetchContent(navidown.NavElement{URL: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content, got %q", content)
	}
}

func TestFileHTTP_LocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	want := "# Hello\n\nWorld\n"
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	f := &FileHTTP{}
	got, err := f.FetchContent(navidown.NavElement{URL: path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFileHTTP_LocalFileNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.md")

	f := &FileHTTP{SearchRoots: []string{dir}}
	_, err := f.FetchContent(navidown.NavElement{URL: path})
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestFileHTTP_LocalFileRelative(t *testing.T) {
	dir := t.TempDir()

	source := filepath.Join(dir, "source.md")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "target.md")
	want := "# Target\n"
	if err := os.WriteFile(target, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	f := &FileHTTP{SearchRoots: []string{dir}}
	got, err := f.FetchContent(navidown.NavElement{
		URL:            "target.md",
		SourceFilePath: source,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFileHTTP_HTTPSuccess(t *testing.T) {
	want := "# Hello from HTTP"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(want))
	}))
	defer srv.Close()

	f := &FileHTTP{}
	got, err := f.FetchContent(navidown.NavElement{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFileHTTP_HTTPNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := &FileHTTP{}
	_, err := f.FetchContent(navidown.NavElement{URL: srv.URL})
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "non-200 status: 404") {
		t.Errorf("error should mention status code, got: %v", err)
	}
}

func TestFileHTTP_HTTPCustomClient(t *testing.T) {
	want := "custom client"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(want))
	}))
	defer srv.Close()

	f := &FileHTTP{Client: srv.Client()}
	got, err := f.FetchContent(navidown.NavElement{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFileHTTP_DirectoryTraversal(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.md")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := &FileHTTP{SearchRoots: []string{dir}}
	_, err := f.FetchContent(navidown.NavElement{
		URL:            "../../../../etc/passwd",
		SourceFilePath: source,
	})
	if err == nil {
		t.Fatal("expected error for directory traversal, got nil")
	}
}
