package language

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

type TrackedModules map[string]int
type SearchResult struct {
	result option.Option[symbols.Indexable]
	//trackedModules   map[string]int
	traversedModules map[string]bool
}

func (s SearchResult) TraversedModules() map[string]bool {
	return s.traversedModules
}

func (s SearchResult) IsSome() bool {
	return s.result.IsSome()
}

func (s SearchResult) IsNone() bool {
	return s.result.IsNone()
}

func (s SearchResult) Get() symbols.Indexable {
	return s.result.Get()
}

func (s *SearchResult) Set(symbol symbols.Indexable) {
	s.result = option.Some(symbol)
}

const (
	Nullo int = iota
	LockStatusReady
	LockStatusLocked
)

func (s *SearchResult) TrackTraversedModule(module string) {
	s.traversedModules[module] = true

	/*mt, ok := s.trackedModules[module]
	trackValue := LockStatusReady
	if ok && mt == LockStatusLocked {
		return false
	} else if mt == LockStatusReady {
		trackValue = LockStatusLocked
	}
	s.trackedModules[module] = trackValue

	return true*/
}

func NewSearchResultEmpty(trackedModules TrackedModules) SearchResult {
	return _NewSearchResult(
		option.None[symbols.Indexable](),
		trackedModules,
	)
}

func NewSearchResult(trackedModules TrackedModules) SearchResult {

	return _NewSearchResult(
		option.None[symbols.Indexable](),
		trackedModules,
	)
}

func _NewSearchResult(result option.Option[symbols.Indexable], trackedModules TrackedModules) SearchResult {
	traversedModules := make(map[string]bool)
	for moduleName, _ := range trackedModules {
		traversedModules[moduleName] = true
	}
	return SearchResult{
		result:           option.None[symbols.Indexable](),
		traversedModules: traversedModules,
	}
}
