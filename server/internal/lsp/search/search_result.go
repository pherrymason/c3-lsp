package search

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

// Whether the search result was transformed from a distinct,
// and, if so, whether it was inline.
const (
	// The search result was not transformed from a distinct.
	NotFromDistinct = iota
	// The search result was transformed from a non-inline distinct.
	NonInlineDistinct
	// The search result was transformed from an inline distinct.
	InlineDistinct
)

type TrackedModules map[string]int
type SearchResult struct {
	// Whether the members of the found indexable are readable.
	// This is `true` if we were searching for a type and got a type, so its
	// members can be accessed, as well as its methods.
	// This is `false` if we were searching for a variable and got a type, so
	// that variable's type cannot be accessed, only its methods.
	membersReadable bool
	// Information on whether the result was transformed from a distinct,
	// and whether it was inline if so.
	// This is important as it determines whether certain type members can be
	// accessed. In particular, members may only be accessed on distinct instances,
	// if supported (for example, an instance of a distinct aliasing a struct may
	// access its members and an instance of a distinct enum may access its associated
	// values, but you cannot access enum constants themselves on the distinct type).
	// In addition, you may only access methods of the aliased distinct type if it is
	// inline.
	//
	// This may be one of 'NotFromDistinct', 'NonInlineDistinct' or 'InlineDistinct'.
	fromDistinct     int
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

func (s *SearchResult) SetFromDistinct(fromDistinct int) {
	s.fromDistinct = fromDistinct
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
		fromDistinct:     NotFromDistinct,
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
		fromDistinct:     NotFromDistinct,
		result:           option.None[symbols.Indexable](),
		traversedModules: traversedModules,
	}
}
