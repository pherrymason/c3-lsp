package document

import (
	"path/filepath"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
)

func TestDocumentStore_Get_accepts_canonical_path_without_uri_fallback(t *testing.T) {
	storage, err := fs.NewFileStorage("")
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

	store := NewDocumentStore(*storage)
	path := fs.GetCanonicalPath(filepath.Join(t.TempDir(), "main.c3"))
	doc := NewDocumentFromString(path, "module app;")
	store.Set(&doc)

	got, ok := store.Get(path)
	if !ok || got == nil {
		t.Fatalf("expected document lookup by canonical path to succeed")
	}
}

func TestDocumentStore_Get_returns_false_for_non_file_uri(t *testing.T) {
	storage, err := fs.NewFileStorage("")
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

	store := NewDocumentStore(*storage)
	path := fs.GetCanonicalPath(filepath.Join(t.TempDir(), "main.c3"))
	doc := NewDocumentFromString(path, "module app;")
	store.Set(&doc)

	got, ok := store.Get("not-a-file-uri")
	if ok || got != nil {
		t.Fatalf("expected invalid lookup to fail cleanly")
	}
}

func TestDocumentStore_Set_stores_file_uri_under_canonical_path(t *testing.T) {
	storage, err := fs.NewFileStorage("")
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

	store := NewDocumentStore(*storage)
	path := fs.GetCanonicalPath(filepath.Join(t.TempDir(), "main.c3"))
	uri := fs.ConvertPathToURI(path, option.None[string]())
	doc := NewDocumentFromString(uri, "module app;")
	store.Set(&doc)

	got, ok := store.Get(path)
	if !ok || got == nil {
		t.Fatalf("expected canonical lookup for URI-backed document to succeed")
	}
}
