package fs

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func TestConvertPathToURI(t *testing.T) {
	path := `D:\projects\c3-lsp\assets\c3-demo\foobar\foo.c3`

	uri := ConvertPathToURI(path, option.None[string]())

	assert.Equal(t, "file:///D:/projects/c3-lsp/assets/c3-demo/foobar/foo.c3", uri)
}

func TestScanForC3WithOptions_skips_ignored_dirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "build"), 0o755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "src", "main.c3"), []byte("module app;"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "build", "generated.c3"), []byte("module generated;"), 0o644); err != nil {
		t.Fatalf("failed to create generated file: %v", err)
	}

	files, stats, err := ScanForC3WithOptions(root, ScanOptions{IgnoreDirs: []string{"build"}})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Contains(t, files[0], filepath.Join("src", "main.c3"))
	assert.GreaterOrEqual(t, stats.SkippedDirs, 1)
}

func TestScanForC3WithOptions_prioritizes_requested_dirs(t *testing.T) {
	root := t.TempDir()
	primary := filepath.Join(root, "primary")
	secondary := filepath.Join(root, "secondary")
	if err := os.MkdirAll(primary, 0o755); err != nil {
		t.Fatalf("failed to create primary dir: %v", err)
	}
	if err := os.MkdirAll(secondary, 0o755); err != nil {
		t.Fatalf("failed to create secondary dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(secondary, "b.c3"), []byte("module app::b;"), 0o644); err != nil {
		t.Fatalf("failed to create b.c3: %v", err)
	}
	if err := os.WriteFile(filepath.Join(primary, "a.c3"), []byte("module app::a;"), 0o644); err != nil {
		t.Fatalf("failed to create a.c3: %v", err)
	}

	files, _, err := ScanForC3WithOptions(root, ScanOptions{PriorityDirs: []string{primary}})
	assert.NoError(t, err)
	if assert.GreaterOrEqual(t, len(files), 2) {
		assert.Contains(t, files[0], filepath.Join("primary", "a.c3"))
	}
}

func TestScanForC3WithOptions_includes_c3l_archives(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "sqlite3.c3l")

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	zw := zip.NewWriter(file)
	entry, err := zw.Create("sqlite.c3i")
	if err != nil {
		t.Fatalf("failed to create archive entry: %v", err)
	}
	if _, err := entry.Write([]byte("module sqlite3;")); err != nil {
		t.Fatalf("failed to write archive entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close archive file: %v", err)
	}

	files, _, err := ScanForC3WithOptions(root, ScanOptions{})
	assert.NoError(t, err)
	assert.Contains(t, files, archivePath)
}
