package project_state

import (
	"sort"
	"strings"

	trie "github.com/pherrymason/c3-lsp/internal/lsp/symbol_trie"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type ProjectSnapshot struct {
	revision           uint64
	allUnitModules     map[protocol.DocumentUri]symbols_table.UnitModules
	fqnIndex           *trie.Trie
	modulesByName      map[string][]*symbols.Module
	docsByModule       map[string][]protocol.DocumentUri
	moduleNamesByShort map[string][]string
	scopeIndex         *ScopeCompletionIndex
}

func buildSnapshotIndexes(all map[protocol.DocumentUri]symbols_table.UnitModules) (map[string][]*symbols.Module, map[string][]protocol.DocumentUri, map[string][]string) {
	modulesByName := make(map[string][]*symbols.Module)
	docsByModule := make(map[string][]protocol.DocumentUri)
	moduleNamesByShort := make(map[string][]string)

	for docURI, unitModules := range all {
		for _, module := range unitModules.Modules() {
			name := module.GetName()
			modulesByName[name] = append(modulesByName[name], module)
			docsByModule[name] = append(docsByModule[name], docURI)

			short := name
			if idx := strings.LastIndex(name, "::"); idx >= 0 && idx+2 < len(name) {
				short = name[idx+2:]
			}
			moduleNamesByShort[short] = append(moduleNamesByShort[short], name)
		}
	}

	for key := range docsByModule {
		sort.Slice(docsByModule[key], func(i, j int) bool {
			return docsByModule[key][i] < docsByModule[key][j]
		})
	}

	for key := range moduleNamesByShort {
		sort.Strings(moduleNamesByShort[key])
	}

	return modulesByName, docsByModule, moduleNamesByShort
}

func (s *ProjectSnapshot) Revision() uint64 {
	if s == nil {
		return 0
	}
	return s.revision
}

func (s *ProjectSnapshot) GetUnitModulesByDoc(docId string) *symbols_table.UnitModules {
	if s == nil {
		return nil
	}
	v, ok := s.allUnitModules[protocol.DocumentUri(docId)]
	if !ok {
		normalized := utils.NormalizePath(docId)
		if normalized != "" {
			v, ok = s.allUnitModules[protocol.DocumentUri(normalized)]
		}
	}
	if !ok {
		return nil
	}
	copy := v
	return &copy
}

func (s *ProjectSnapshot) GetAllUnitModules() map[protocol.DocumentUri]symbols_table.UnitModules {
	if s == nil {
		return map[protocol.DocumentUri]symbols_table.UnitModules{}
	}

	cpy := make(map[protocol.DocumentUri]symbols_table.UnitModules, len(s.allUnitModules))
	for k, v := range s.allUnitModules {
		cpy[k] = v
	}

	return cpy
}

func (s *ProjectSnapshot) AllUnitModulesView() map[protocol.DocumentUri]symbols_table.UnitModules {
	if s == nil {
		return map[protocol.DocumentUri]symbols_table.UnitModules{}
	}

	return s.allUnitModules
}

func (s *ProjectSnapshot) UnitModulesByDocValue(docId string) (symbols_table.UnitModules, bool) {
	if s == nil {
		return symbols_table.UnitModules{}, false
	}

	v, ok := s.allUnitModules[protocol.DocumentUri(docId)]
	if !ok {
		return symbols_table.UnitModules{}, false
	}

	return v, true
}

func (s *ProjectSnapshot) SearchByFQN(query string) []symbols.Indexable {
	if s == nil || s.fqnIndex == nil {
		return nil
	}

	return s.fqnIndex.Search(query)
}

func (s *ProjectSnapshot) ModulesByName(name string) []*symbols.Module {
	if s == nil {
		return nil
	}

	modules := s.modulesByName[name]
	if len(modules) == 0 {
		return nil
	}

	return append([]*symbols.Module(nil), modules...)
}

func (s *ProjectSnapshot) DocsByModule(name string) []protocol.DocumentUri {
	if s == nil {
		return nil
	}

	docs := s.docsByModule[name]
	if len(docs) == 0 {
		return nil
	}

	return append([]protocol.DocumentUri(nil), docs...)
}

func (s *ProjectSnapshot) ScopeIndex() *ScopeCompletionIndex {
	if s == nil {
		return nil
	}
	return s.scopeIndex
}

func (s *ProjectSnapshot) ModuleNamesByShort(shortName string) []string {
	if s == nil {
		return nil
	}

	names := s.moduleNamesByShort[shortName]
	if len(names) == 0 {
		return nil
	}

	return append([]string(nil), names...)
}

// ForEachModule calls fn for every module in every document in the snapshot.
// It is nil-safe and replaces the repeated double-loop:
//
//	for _, unitModules := range snapshot.AllUnitModulesView() {
//	    for _, module := range unitModules.Modules() { ... }
//	}
func (s *ProjectSnapshot) ForEachModule(fn func(module *symbols.Module)) {
	if s == nil {
		return
	}

	for _, unitModules := range s.allUnitModules {
		for _, module := range unitModules.Modules() {
			if module == nil {
				continue
			}
			fn(module)
		}
	}
}

// ForEachModuleUntil calls fn for every module and stops early when fn returns true.
// It returns true if fn signalled early termination, false if all modules were visited.
func (s *ProjectSnapshot) ForEachModuleUntil(fn func(module *symbols.Module) bool) bool {
	if s == nil {
		return false
	}

	for _, unitModules := range s.allUnitModules {
		for _, module := range unitModules.Modules() {
			if module == nil {
				continue
			}
			if fn(module) {
				return true
			}
		}
	}
	return false
}
