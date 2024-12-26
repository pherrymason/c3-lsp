package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
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
	symbols []Symbol
}

func (s *SymbolTable) RegisterVariable(genDecl *ast.GenDecl, currentModule ast.Module) {
	s.symbols = append(s.symbols, Symbol{
		Name:     genDecl.Spec.(*ast.ValueSpec).Names[0].Name,
		Module:   []string{currentModule.Name},
		NodeDecl: genDecl,
		Range:    genDecl.Range,
	})
}
func (s *SymbolTable) RegisterType(typeDecl *ast.GenDecl, currentModule ast.Module) {
	s.symbols = append(s.symbols, Symbol{
		Name:     typeDecl.Spec.(*ast.TypeSpec).Name.Name,
		Module:   []string{currentModule.Name},
		NodeDecl: typeDecl,
		Range:    typeDecl.Range,
	})
}

type SymbolID uint

type Symbol struct {
	Name     string
	Module   ModuleName
	Range    lsp.Range
	NodeDecl ast.Node // Declaration node of this symbol
	Type     TypeDefinition
}

type ModuleName []string

type TypeDefinition struct {
	Name      string
	IsBuiltIn bool // Is it a built in type definition?
	NodeDecl  ast.Node
}

// ScopeTable represents the list of symbols declared in a scope
type ScopeTable struct {
	Range   lsp.Range
	Symbols []SymbolID
}

func BuildSymbolTable(astTree ast.Node) SymbolTable {
	visitor := symbolTableGenerator{}
	walk.Walk(&visitor, astTree)

	return visitor.table
}

type symbolTableGenerator struct {
	table SymbolTable

	// State properties to keep track
	currentModule ast.Module
}

func (v *symbolTableGenerator) Enter(node ast.Node) walk.Visitor {
	switch n := node.(type) {
	case ast.Module:
		v.currentModule = ast.Module{
			Name:              n.Name,
			GenericParameters: n.GenericParameters,
			NodeAttributes:    n.NodeAttributes,
		}

	case *ast.GenDecl:
		if n.Token == ast.VAR {
			v.table.RegisterVariable(n, v.currentModule)
		} else if n.Token == ast.ENUM || n.Token == ast.STRUCT {
			v.table.RegisterType(n, v.currentModule)
		}
	}

	return v
}

func (v *symbolTableGenerator) Exit(n ast.Node) {

}
