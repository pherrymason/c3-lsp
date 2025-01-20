package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/option"
)

type ScopeID int

// Scope defines a range position where it has effect, and the list of symbols defined inside.
// It also contains relations to parent and children scopes.
type Scope struct {
	Range    lsp.Range
	Module   option.Option[ModuleName]
	Parent   option.Option[*Scope] // Parent Scope (if any)
	Imports  []ModuleName          // Module imports (if any)
	Children []*Scope              // Children Scope (if any)
	Symbols  []*Symbol             // List of symbols defined in this scope
}

func (s *Scope) pushScope(Range lsp.Range) *Scope {
	newScope := &Scope{Range: Range, Parent: option.Some(s), Children: nil}
	s.Children = append(s.Children, newScope)

	return newScope
}

func (s *Scope) RegisterSymbol(name string, nRange lsp.Range, n ast.Node, module ast.Module, filePath string, kind ast.Token) (SymbolID, *Symbol) {
	symbol := &Symbol{
		Name:     name,
		Module:   NewModuleName(module.Name),
		URI:      filePath,
		NodeDecl: n,
		Range:    nRange,
		Scope:    s,
		Kind:     kind,
	}
	s.Symbols = append(s.Symbols, symbol)

	return SymbolID(len(s.Symbols)), symbol
}

func (s *Scope) RootScope() *Scope {
	if s.Parent.IsNone() {
		return s
	}

	return s.Parent.Get().RootScope()
}

// FindClosestScope given a scope, searches inside to select the closes to a given position
func FindClosestScope(scope *Scope, pos lsp.Position) *Scope {
	if !scope.Range.HasPosition(pos) {
		// If the position is not within the current scope, return nil
		return nil
	}

	// Check children scopes
	for _, childScope := range scope.Children {
		if found := FindClosestScope(childScope, pos); found != nil {
			return found
		}
	}

	// If no child contains the position, the current scope is the best match
	return scope
}
