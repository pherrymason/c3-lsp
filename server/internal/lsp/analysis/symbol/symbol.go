package symbols

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"strings"
)

type Location struct {
	FileName string
	Position lsp.Position
	Module   ModuleName
}

func NewLocation(fileName string, position lsp.Position, module ModuleName) Location {
	return Location{
		FileName: fileName,
		Position: position,
		Module:   module,
	}
}

type Symbol struct {
	Label      option.Option[string]
	Identifier string
	Module     ModuleName
	URI        string
	Range      lsp.Range
	Kind       ast.Token
	Public     bool     // only accessible through parent symbol. For example, enum assoc values
	NodeDecl   ast.Node // Declaration AST node of this symbol
	TypeDef    TypeDefinition
	TypeSymbol *Symbol
	Children   []Relation
	Scope      *Scope
}

type TypeDefinition struct {
	Name              string
	Module            option.Option[string]
	IsBuiltIn         bool // Is it a built-in type definition?
	Pointer           uint
	Optional          bool
	BuiltIn           bool
	Static            bool
	Reference         bool
	IsAnonymous       bool
	GenericParameters []TypeDefinition
	Symbol            *Symbol  // This points to the registered symbol if found.
	NodeDecl          ast.Node // deprecated
}

func TypeDefinitionFromASTTypeInfo(node *ast.TypeInfo) TypeDefinition {
	td := TypeDefinition{
		Name:              node.Identifier.Name,
		Module:            node.Module(),
		Pointer:           node.Pointer,
		Optional:          node.Optional,
		IsBuiltIn:         node.IsBuiltIn,
		Static:            node.Static,
		Reference:         node.Reference,
		GenericParameters: []TypeDefinition{},
		NodeDecl:          node,
	}

	for _, param := range node.GenericsParameters {
		td.GenericParameters = append(
			td.GenericParameters,
			TypeDefinition{
				Name:   param.Identifier.Name,
				Module: param.Module(),
			})
	}

	return td
}

func (t *TypeDefinition) String() string {
	id := t.Name
	if t.Pointer > 0 {
		id += strings.Repeat("*", int(t.Pointer))
	}
	if t.Optional {
		id += "!"
	}
	if len(t.GenericParameters) > 0 {
		id += "(<"
		list := []string{}
		for _, gn := range t.GenericParameters {
			list = append(list, gn.String())
		}
		id += strings.Join(list, ", ") + ">)"
	}

	return id
}

func (s *Symbol) AppendChild(child *Symbol, relationType RelationType) {
	s.Children = append(s.Children, Relation{child, relationType})
}

func (s *Symbol) GetLabel() string {
	if s.Label.IsSome() {
		return s.Label.Get()
	}

	return s.Identifier
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

type RelationType string

const (
	RelatedMethod RelationType = "method" // It's a method of parent
	RelatedField  RelationType = "field"  // It's a field of parent
)

// Relation represents a relation between a symbol and its parent.
type Relation struct {
	Symbol *Symbol
	Tag    RelationType
}
