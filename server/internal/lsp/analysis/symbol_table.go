package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/option"
)

// SymbolTable stores list of symbols defined in the project.
// Each symbol has an "address". This address allows to now where the symbol
/*
// Ideas:
// module.[global|scope(funName)].symbolName
// this would cover:
//  	module foo;
//      int symbol;
//      fn void foo(){ int symbol; }
// This does not cover:
//      module foo;
//      fn void foo(){
//     		int symbol; // #1
//			{
//				int symbol; // #2
//			}
//			{
//				char symbol; // #3
//				{
//					float symbol; //#4
//				}
//			}
//    	}
//
// Ideas:
// #1: foo.foo.symbol    <-- declared in root scope
// #2: foo.foo[1].symbol <-- First sub scope
// #3: foo.foo[2].symbol <-- Second sub scope
// #4: foo.foo[2][0].symbol <-- Second sub scope, First sub scope
//
// Idea 2. Not sure if this will allow to find them, but maybe this scope
// location would be useful to have to disambiguate, better storing it
// in a different column

 	symbol table
 	--------------
	id: primary key
	symbol_name: name of the symbol
	module: module name where this is defined
	path: full route to reach this symbol
	scope_path: Example [2][0]. Helps determining what is the scope this is defined.
*/
type SymbolTable struct {
	// Each position inside symbols is the ID of the symbol which can be referenced in other index tables.
	symbols   []*Symbol
	scopeTree map[string]*Scope // scope trees for each file
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		symbols:   []*Symbol{},
		scopeTree: make(map[string]*Scope),
	}
}

func (s *SymbolTable) RegisterNewRootScope(file string, Range lsp.Range) *Scope {
	scope := &Scope{
		Range: Range,
	}
	s.scopeTree[file] = scope

	return scope
}

func (s *SymbolTable) RegisterSymbol(name string, nRange lsp.Range, n ast.Node, module ast.Module, scope *Scope, filePath string) (SymbolID, *Symbol) {
	symbol := &Symbol{
		Name:     name,
		Module:   ModuleName(module.Name),
		FilePath: filePath,
		NodeDecl: n,
		Range:    nRange,
		Scope:    scope,
	}
	s.symbols = append(s.symbols, symbol)

	return SymbolID(len(s.symbols)), symbol
}

func (s *SymbolTable) FindSymbol2(
	pos lsp.Position,
	fileName string,
	name string,
	module ModuleName,
	usageScopeStack []lsp.Range,
	hint ast.Token) option.Option[*Symbol] {
	type SymbolScope struct {
		symbol     *Symbol
		rangeScope lsp.Range
	}
	symbolFound := option.None[SymbolScope]()
	// Search current scope
	scope := FindScope(s.scopeTree[fileName], pos)
	if scope == nil {
		return option.None[*Symbol]()
	}

	// Search inside the scope and go up until find its declaration
	currentScope := scope
	found := false
	for {
		for _, symbol := range currentScope.Symbols {
			if symbol.Name != name {
				continue
			}
			if symbol.Module != module {
				continue
			}

			if hint == ast.STRUCT {
				if _, ok := symbol.NodeDecl.(*ast.StructDecl); !ok {
					continue
				}
			}

			symbolFound = option.Some(SymbolScope{
				symbol:     symbol,
				rangeScope: currentScope.Range,
			})
			found = true
			break
		}

		if found {
			break
		} else if currentScope.Parent.IsSome() {
			currentScope = currentScope.Parent.Get()
		} else {
			break
		}
	}

	if symbolFound.IsNone() {
		return option.None[*Symbol]()
	}

	return option.Some(symbolFound.Get().symbol)
}

// FindSymbol searches a symbol by name, module and a given scope
// name: name of the symbol. Exact match
// module: module name where the symbol should be defined.
// usageScopeStack: scopeTree where the symbol is being used. This helps to discard unrelated scopes.
// hint: If we are only interested in a type of symbol,
func (s *SymbolTable) FindSymbol(name string, module ModuleName, usageScopeStack []lsp.Range, hint ast.Token) option.Option[*Symbol] {
	type SymbolScope struct {
		symbol *Symbol
		scope  lsp.Range
	}
	symbolFound := option.None[SymbolScope]()
	checkScope := len(usageScopeStack) > 0

	for _, symbol := range s.symbols {
		if symbol.Name != name {
			continue
		}
		if symbol.Module != module {
			continue
		}

		matchedScope := option.None[lsp.Range]()
		if checkScope {
			for i := len(usageScopeStack) - 1; i >= 0; i-- {
				if symbol.Scope.Range.IsInside(usageScopeStack[i]) == true {
					matchedScope = option.Some(usageScopeStack[i])
					break
				}
			}

			if matchedScope.IsNone() {
				continue
			}
		}

		if hint == 1 {
			if _, ok := symbol.NodeDecl.(*ast.StructDecl); !ok {
				continue
			}
		}

		if checkScope {
			replace := false
			if symbolFound.IsSome() {
				// Check if matchedScope is deeper in provided usageScopeStack
				if symbolFound.Get().scope.IsInside(matchedScope.Get()) {
					// Ignore, already found symbol is in a closer scope
				} else {
					replace = true
				}
			} else {
				replace = true
			}

			if replace {
				symbolFound = option.Some(SymbolScope{
					symbol: symbol,
					scope:  matchedScope.Get(),
				})
			}
		} else {
			symbolFound = option.Some(SymbolScope{
				symbol: symbol,
			})
		}
	}

	if symbolFound.IsNone() {
		return option.None[*Symbol]()
	}

	return option.Some(symbolFound.Get().symbol)
}

type SymbolID int

type Symbol struct {
	Name     string
	Module   ModuleName
	FilePath string
	Range    lsp.Range
	NodeDecl ast.Node // Declaration node of this symbol
	Type     TypeDefinition
	Children []Relation
	Scope    *Scope
}

func (s *Symbol) AppendChild(id SymbolID, relationType RelationType) {
	s.Children = append(s.Children, Relation{id, relationType})
}

type ModuleName string

type TypeDefinition struct {
	Name      string
	IsBuiltIn bool // Is it a built-in type definition?
	NodeDecl  ast.Node
}

type RelationType string

const (
	Method   RelationType = "method"   // It's a method of parent
	Property RelationType = "property" // It's a property of parent
)

// Relation represents a relation between a symbol and its parent.
type Relation struct {
	SymbolID SymbolID
	Tag      RelationType
}
