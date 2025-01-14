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
	scopeTree map[string]*Scope // scope trees for each file
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
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

func (s *SymbolTable) findSymbolInScope(name string, scope *Scope) *Symbol {
	var symbolFound *Symbol
	currentScope := scope
	found := false
	for {
		for _, symbol := range currentScope.Symbols {
			if symbol.Name != name {
				continue
			}

			found = true
			symbolFound = symbol
			break
		}

		if !found && currentScope.Parent.IsSome() {
			currentScope = currentScope.Parent.Get()
		} else {
			break
		}
	}

	return symbolFound
}

func (s *SymbolTable) FindSymbolByPosition(pos lsp.Position, fileName string, name string, module ModuleName, hint ast.Token) option.Option[*Symbol] {
	type SymbolScope struct {
		symbol     *Symbol
		rangeScope lsp.Range
	}

	// Search current scope
	scope := FindScope(s.scopeTree[fileName], pos)
	if scope == nil {
		return option.None[*Symbol]()
	}

	// Search inside the scope and go up until find its declaration
	symbolFound := s.findSymbolInScope(name, scope)

	if symbolFound == nil {
		return option.None[*Symbol]()
	}

	return option.Some(symbolFound)
}

// SolveType Finds type of Symbol with `name` based on a position and a fileName.
// TODO Be able to specify module to which name belongs to. This will be needed to be able to find types imported from different modules
func (s *SymbolTable) SolveType(name string, ctxPosition lsp.Position, fileName string) *Symbol {
	// 1- Find the scope
	scope := FindScope(s.scopeTree[fileName], ctxPosition)
	// TODO If `module` is specified, check if scope belongs to that module, else, see if there are any imports to select the proper scope.

	// 2- Try to find the symbol in the scope stack
	symbolFound := s.findSymbolInScope(name, scope)

	if symbolFound == nil {
		// Search on imports
		// TODO
	}

	if symbolFound != nil {
		// Extract type info
		var typeName string
		switch n := symbolFound.NodeDecl.(type) {
		case *ast.GenDecl:
			switch spec := n.Spec.(type) {
			case *ast.ValueSpec:
				typeName = spec.Type.Identifier.Name
			}
		}

		// Second search, we need to search for symbol with typeName
		symbol := s.FindSymbolByPosition(symbolFound.Range.Start, fileName, typeName, "", 0)
		if symbol.IsNone() {
			return nil
		} else {
			return symbol.Get()
		}
	}

	return symbolFound
}

type SymbolID int

type Symbol struct {
	Name     string
	Module   ModuleName
	FilePath string
	Range    lsp.Range
	NodeDecl ast.Node // Declaration node of this symbol
	Kind     ast.Token
	Type     TypeDefinition
	Children []Relation
	Scope    *Scope
}

func (s *Symbol) AppendChild(child *Symbol, relationType RelationType) {
	s.Children = append(s.Children, Relation{child, relationType})
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
	Child *Symbol
	Tag   RelationType
}
