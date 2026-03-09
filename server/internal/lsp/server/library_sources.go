package server

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/fs"
)

const archiveEntrySeparator = "!"

func isPlainC3SourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".c3" || ext == ".c3i"
}

func isC3LibraryArchive(path string) bool {
	if strings.ToLower(filepath.Ext(path)) != ".c3l" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func isC3SourcePath(path string) bool {
	return isPlainC3SourceFile(path) || strings.EqualFold(filepath.Ext(path), ".c3l")
}

func loadSourceDocuments(path string) ([]loadedDocument, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		files, _, err := fs.ScanForC3WithOptions(path, fs.ScanOptions{IgnoreDirs: fs.DefaultC3ScanIgnoreDirs()})
		if err != nil {
			return nil, err
		}

		docs := make([]loadedDocument, 0, len(files))
		for _, file := range files {
			entries, err := loadSourceDocuments(file)
			if err != nil {
				docs = append(docs, loadedDocument{path: file, readErr: err})
				continue
			}
			docs = append(docs, entries...)
		}

		return docs, nil
	}

	if isPlainC3SourceFile(path) {
		content, err := os.ReadFile(path)
		return []loadedDocument{{path: path, content: string(content), readErr: err}}, nil
	}

	if !isC3LibraryArchive(path) {
		return nil, nil
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	entries := make([]loadedDocument, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if !isPlainC3SourceFile(file.Name) {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			entries = append(entries, loadedDocument{path: archiveEntryPath(path, file.Name), readErr: err})
			continue
		}

		data, readErr := readAllAndClose(rc)
		entries = append(entries, loadedDocument{
			path:    archiveEntryPath(path, file.Name),
			content: string(data),
			readErr: readErr,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	return entries, nil
}

func archiveEntryPath(archivePath string, entryName string) string {
	return archivePath + archiveEntrySeparator + filepath.ToSlash(entryName)
}

func (s *Server) dropIndexedArchiveEntries(archivePath string) {
	prefix := archivePath + archiveEntrySeparator
	for docURI := range s.state.GetAllUnitModules() {
		docID := string(docURI)
		if strings.HasPrefix(docID, prefix) {
			s.state.DeleteDocument(docID)
		}
	}
}

func readAllAndClose(rc interface {
	Read([]byte) (int, error)
	Close() error
}) ([]byte, error) {
	defer func() {
		_ = rc.Close()
	}()
	return io.ReadAll(rc)
}
