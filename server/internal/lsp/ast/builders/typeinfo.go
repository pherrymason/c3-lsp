package ast_builders

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
)

// --
// TypeInfoBuilder
// --
type TypeInfoBuilder struct {
	typeInfo *ast.TypeInfo
}

func NewTypeInfoBuilder() *TypeInfoBuilder {
	return &TypeInfoBuilder{
		typeInfo: &ast.TypeInfo{
			NodeAttributes: ast.NodeAttributes{},
			Identifier: &ast.Ident{
				ModulePath: nil,
			},
		},
	}
}

func (b *TypeInfoBuilder) IsOptional() *TypeInfoBuilder {
	b.typeInfo.Optional = true
	return b
}

func (b *TypeInfoBuilder) IsReference() *TypeInfoBuilder {
	b.typeInfo.Reference = true
	return b
}
func (b *TypeInfoBuilder) IsBuiltin() *TypeInfoBuilder {
	b.typeInfo.BuiltIn = true
	return b
}
func (b *TypeInfoBuilder) IsStatic() *TypeInfoBuilder {
	b.typeInfo.Static = true
	return b
}
func (b *TypeInfoBuilder) IsPointer() *TypeInfoBuilder {
	b.typeInfo.Pointer = 1
	return b
}

func (b *TypeInfoBuilder) WithGeneric(name string, startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.typeInfo.Generics = append(
		b.typeInfo.Generics,
		NewTypeInfoBuilder().
			WithName(name).
			WithNameStartEnd(startRow, startCol, endRow, endCol).
			WithStartEnd(startRow, startCol, endRow, endCol).
			Build(),
	)

	return b
}

func (b *TypeInfoBuilder) WithName(name string) *TypeInfoBuilder {
	b.typeInfo.Identifier.Name = name

	return b
}

func (b *TypeInfoBuilder) WithPath(path *ast.Ident) *TypeInfoBuilder {
	b.typeInfo.Identifier.ModulePath = path

	return b
}

func (b *TypeInfoBuilder) WithNameStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.typeInfo.Identifier.Range.Start = lsp.Position{startRow, startCol}
	b.typeInfo.Identifier.Range.End = lsp.Position{endRow, endCol}
	return b
}

func (b *TypeInfoBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.typeInfo.NodeAttributes.Range.Start = lsp.Position{startRow, startCol}
	b.typeInfo.NodeAttributes.Range.End = lsp.Position{endRow, endCol}
	return b
}

func (b *TypeInfoBuilder) Build() *ast.TypeInfo {
	return b.typeInfo
}
