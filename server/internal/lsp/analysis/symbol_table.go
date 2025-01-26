package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"strings"
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
	scopeTree map[string]*ModulesList // scope trees for each file

	moduleFileMap map[string][]FileModulePtr // List of files containing a given module
}

type FileModulePtr struct {
	fileName string
	module   ModuleName
	scope    *Scope
}

type ModulesList struct {
	modules map[string]*Scope
}

func (mg *ModulesList) GetModuleScope(name string) *Scope {
	scope, exists := mg.modules[name]
	if !exists {
		return nil
	}

	return scope
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		scopeTree:     make(map[string]*ModulesList),
		moduleFileMap: make(map[string][]FileModulePtr),
	}
}

func (s *SymbolTable) RegisterNewRootScope(file string, node ast.Module) *Scope {
	_, exists := s.scopeTree[file]
	if !exists {
		s.scopeTree[file] = &ModulesList{
			modules: make(map[string]*Scope),
		}
	}

	scope := &Scope{
		Module:  option.Some(NewModuleName(node.Name)),
		Range:   node.GetRange(),
		Imports: []ModuleName{},
	}
	s.scopeTree[file].modules[node.Name] = scope

	for _, imp := range node.Imports {
		scope.Imports = append(scope.Imports, NewModuleName(imp.Path.Name))
	}

	// Register it in moduleFileMap
	s.moduleFileMap[node.Name] = append(s.moduleFileMap[node.Name], FileModulePtr{
		fileName: file,
		module:   NewModuleName(node.Name),
		scope:    scope,
	})

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

type FromPosition struct {
	pos      lsp.Position
	fileName string
	module   ModuleName
}

// FindSymbolByPosition Searches a symbol with ident=identName.
// Takes into account current scope, defined by pos, fileName and module.
func (s *SymbolTable) FindSymbolByPosition(identName string, from FromPosition) option.Option[*Symbol] {
	type SymbolScope struct {
		symbol     *Symbol
		rangeScope lsp.Range
	}

	var symbolFound *Symbol

	moduleScope := s.scopeTree[from.fileName].GetModuleScope(from.module.String())

	visitedFiles := make(map[string]bool)
	toVisit := []FileModulePtr{
		{
			fileName: from.fileName,
			module:   from.module,
			scope:    moduleScope,
		},
	}

	iteration := 0
	for len(toVisit) > 0 {
		currentFile := toVisit[0].fileName
		moduleScope = toVisit[0].scope
		module := toVisit[0].module
		toVisit = toVisit[1:]

		visitedKey := currentFile + "+" + module.String()
		if visitedFiles[visitedKey] {
			continue
		}
		visitedFiles[visitedKey] = true

		// Search current scope
		var scope *Scope
		if iteration == 0 {
			scope = FindClosestScope(moduleScope, from.pos)
		} else {
			// other iterations, are searching in root scope of imported module, position is not relevant
			scope = moduleScope
		}
		if scope == nil {
			return option.None[*Symbol]()
		}

		// Search inside the scope and go up until find its declaration
		symbolFound = s.findSymbolInScope(identName, scope)
		if symbolFound == nil {
			// Check if there are imported modules
			toVisit, visitedFiles = findImportedFiles(s, scope, module, visitedFiles, toVisit)
		}
		iteration++
	}

	if symbolFound == nil {
		return option.None[*Symbol]()
	}
	return option.Some(symbolFound)
}

func findImportedFiles(s *SymbolTable, scope *Scope, currentModule ModuleName, visited map[string]bool, toVisit []FileModulePtr) ([]FileModulePtr, map[string]bool) {
	// look for implicit imports
	for _, importedModule := range scope.RootScope().Imports {
		for _, fileModule := range s.moduleFileMap[importedModule.String()] {
			key := fileModule.fileName + "+" + fileModule.module.String()
			if !visited[key] {
				toVisit = append(toVisit, fileModule)
			}
		}
	}

	// Look for submodules, as they are also implicitly imported
	for _, fileModuleCollection := range s.moduleFileMap {
		for _, fileModule := range fileModuleCollection {
			if fileModule.module.IsSubModuleOf(currentModule) {
				key := fileModule.fileName + "+" + fileModule.module.String()
				if !visited[key] {
					toVisit = append(toVisit, fileModule)
				}
			}
		}
	}

	return toVisit, visited
}

// SolveType Finds type of Symbol with `name` based on a position and a fileName.
// TODO Be able to specify module to which name belongs to. This will be needed to be able to find types imported from different modules
func (s *SymbolTable) SolveType(name string, ctxPosition lsp.Position, fileName string, currentModule ModuleName) *Symbol {
	// 1- Find the scope
	moduleGroup := s.scopeTree[fileName]
	scope := moduleGroup.GetModuleScope(currentModule.String())
	scope = FindClosestScope(scope, ctxPosition)
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
				typeName = spec.Type.Identifier.String() // TODO this will fail if type contains Module path
			}
		}

		// Second search, we need to search for symbol with typeName
		from := FromPosition{pos: symbolFound.Range.Start, fileName: fileName, module: currentModule}
		symbol := s.FindSymbolByPosition(typeName, from)
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
	URI      string
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

type ModuleName struct {
	tokens []string
}

func NewModuleName(module string) ModuleName {
	var tokens []string
	if len(module) > 0 {
		tokens = strings.Split(module, "::")
	}
	return ModuleName{tokens: tokens}
}

func (m ModuleName) IsEqual(other ModuleName) bool {
	if len(m.tokens) != len(other.tokens) {
		return false
	}

	for i, token := range m.tokens {
		if token != other.tokens[i] {
			return false
		}
	}

	return true
}

func (m ModuleName) IsSubModuleOf(parentModule ModuleName) bool {
	if len(m.tokens) < len(parentModule.tokens) {
		return false
	}

	isChild := true
	for i, pm := range parentModule.tokens {
		if i > len(m.tokens) {
			break
		}

		if m.tokens[i] != pm {
			isChild = false
			break
		}
	}

	return isChild
}

func (m ModuleName) String() string {
	return strings.Join(m.tokens, "::")
}

type TypeDefinition struct {
	Name      string
	IsBuiltIn bool // Is it a built-in type definition?
	NodeDecl  ast.Node
}

type RelationType string

const (
	Method RelationType = "method"   // It's a method of parent
	Field  RelationType = "property" // It's a field of parent
)

// Relation represents a relation between a symbol and its parent.
type Relation struct {
	Child *Symbol
	Tag   RelationType
}
