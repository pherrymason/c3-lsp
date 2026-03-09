package server

import (
	"regexp"
	"sort"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var moduleDeclPattern = regexp.MustCompile(`(?m)^\s*module\s+([A-Za-z_][A-Za-z0-9_:]*)(?:\s+[^;\n]*)?;`)

func (s *Server) preloadImportedRootModulesForURI(uri protocol.DocumentUri) {
	s.preloadImportedRootModulesForURIWithForce(uri, false)
}

func (s *Server) preloadImportedRootModulesForURIForce(uri protocol.DocumentUri) {
	s.preloadImportedRootModulesForURIWithForce(uri, true)
}

func (s *Server) preloadImportedRootModulesForURIWithForce(uri protocol.DocumentUri, force bool) {
	docID := utils.NormalizePath(uri)
	doc := s.state.GetDocument(docID)
	if doc == nil {
		return
	}
	unit := s.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return
	}

	importRoots := map[string]bool{}
	for _, module := range unit.Modules() {
		if module == nil {
			continue
		}
		for _, imp := range module.Imports {
			root := imp
			if sep := strings.Index(root, "::"); sep >= 0 {
				root = root[:sep]
			}
			root = strings.TrimSpace(root)
			if root == "" {
				continue
			}
			recurse := !module.IsImportNoRecurse(imp)
			if existing, ok := importRoots[root]; !ok {
				importRoots[root] = recurse
			} else {
				importRoots[root] = existing || recurse
			}
		}
	}
	if len(importRoots) == 0 {
		return
	}

	root := s.resolveProjectRootForURI(&uri)
	root = fs.GetCanonicalPath(root)
	if root == "" || !isBuildableProjectRoot(root) {
		return
	}

	scanRoots := []string{root}
	scanRoots = append(scanRoots, s.workspaceDependencyDirs...)

	for importRoot, recurse := range importRoots {
		if !s.shouldPreloadImportRoot(root, importRoot, recurse, force) {
			continue
		}
		s.preloadImportRootFromScanRoots(scanRoots, importRoot, recurse)
	}
}

func (s *Server) shouldPreloadImportRoot(projectRoot string, importRoot string, recurse bool, force bool) bool {
	if force {
		return true
	}

	mode := "norecurse"
	if recurse {
		mode = "recurse"
	}
	key := projectRoot + "|" + importRoot + "|" + mode
	s.importPreloadMu.Lock()
	defer s.importPreloadMu.Unlock()
	if s.importPreloadDone == nil {
		s.importPreloadDone = make(map[string]struct{})
	}
	if _, ok := s.importPreloadDone[key]; ok {
		return false
	}
	s.importPreloadDone[key] = struct{}{}
	return true
}

func (s *Server) preloadImportRootFromScanRoots(scanRoots []string, importRoot string, recurse bool) {
	if importRoot == "" {
		return
	}

	seenFiles := map[string]struct{}{}
	allFiles := []string{}
	for _, scanRoot := range scanRoots {
		if scanRoot == "" {
			continue
		}
		files, _, _ := fs.ScanForC3WithOptions(scanRoot, fs.ScanOptions{IgnoreDirs: fs.DefaultC3ScanIgnoreDirs()})
		for _, file := range files {
			canonical := fs.GetCanonicalPath(file)
			if canonical == "" {
				continue
			}
			if _, ok := seenFiles[canonical]; ok {
				continue
			}
			seenFiles[canonical] = struct{}{}
			allFiles = append(allFiles, canonical)
		}
	}

	acceptModule := func(moduleName string) bool {
		return moduleName == importRoot || (recurse && strings.HasPrefix(moduleName, importRoot+"::"))
	}
	sort.Strings(allFiles)
	for _, file := range allFiles {
		s.loadFilterAndIndex(file, acceptModule)
	}
}

func extractDeclaredModuleName(source []byte) string {
	m := moduleDeclPattern.FindSubmatch(source)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}
