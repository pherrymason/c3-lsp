package search

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

type TrackedModules map[string]int
type SearchResult struct {
	// Whether the members of the found indexable are readable.
	// This is `true` if we were searching for a type and got a type, so its
	// members can be accessed, as well as its methods.
	// This is `false` if we were searching for a variable and got a type, so
	// that variable's type cannot be accessed, only its methods.
	membersReadable  bool
	result           option.Option[symbols.Indexable]
	traversedModules map[string]bool
	//trackedModules map[string]int
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

func (s *SearchResult) AreMembersReadable() bool {
	return s.membersReadable
}

func (s SearchResult) Get() symbols.Indexable {
	return s.result.Get()
}

func (s *SearchResult) SetMembersReadable(membersReadable bool) {
	s.membersReadable = membersReadable
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

func NewSearchResultEmptyWithTraversedModules(traversedModules map[string]bool) SearchResult {
	return SearchResult{
		membersReadable:  true,
		result:           option.None[symbols.Indexable](),
		traversedModules: traversedModules,
	}
}

func _NewSearchResult(result option.Option[symbols.Indexable], trackedModules TrackedModules) SearchResult {
	traversedModules := make(map[string]bool)
	for moduleName, _ := range trackedModules {
		traversedModules[moduleName] = true
	}
	return SearchResult{
		// Default members readable to 'true' as, usually, we're searching for the type
		// itself rather than a variable with that type. On a few cases, however, we
		// convert a variable into its type to search for more information. In those cases,
		// we should explicitly set `membersReadable` as appropriate, but only at the
		// conversion step. After advancing further into the access chain, if we do so,
		// for example, `membersReadable` would not necessarily remain `false`, as it just
		// refers to the immediate result.
		//
		// That is, `CoolEnum.VALUE` works because `CoolEnum` would imply a `membersReadable: true`
		// search, but `CoolEnum.VALUE.VALUE2` wouldn't work as `CoolEnum.VALUE` would imply
		// `membersReadable: false`.
		membersReadable:  true,
		result:           option.None[symbols.Indexable](),
		traversedModules: traversedModules,
	}
}
