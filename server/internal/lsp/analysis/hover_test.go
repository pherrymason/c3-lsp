package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

func TestBuildHover_Variable(t *testing.T) {
	symbol := &Symbol{
		Name:   "variable",
		Module: NewModuleName("app"),
		URI:    "file.c3",
		NodeDecl: &ast.GenDecl{
			Token: ast.VAR,
			Spec:  &ast.ValueSpec{},
		},
		Kind: ast.VAR,
		Type: TypeDefinition{
			Name: "int",
		},
	}

	hover := buildHover(symbol)

	assert.Equal(t, "```c3\nint variable\n```\n\nIn module **[app]**", hover.Contents.(protocol.MarkupContent).Value)
}

func TestBuildHover_Struct(t *testing.T) {

}

func TestBuildHover_Field(t *testing.T) {

}

func TestBuildHover_Function(t *testing.T) {

}

func TestBuildHover_Macro(t *testing.T) {
	t.Run("simple macro", func(t *testing.T) {
		symbol := &Symbol{
			Name:   "macro",
			Module: NewModuleName("app"),
			URI:    "file.c3",
			Kind:   ast.MACRO,
			NodeDecl: &ast.MacroDecl{
				Signature: &ast.MacroSignature{
					Name: &ast.Ident{Name: "foo"},
				},
			},
		}

		hover := buildHover(symbol)

		assert.Equal(t, "```c3\nmacro foo()\n```\n\nIn module **[app]**", hover.Contents.(protocol.MarkupContent).Value)
	})

	t.Run("macro method", func(t *testing.T) {
		symbol := &Symbol{
			Name:   "macro",
			Module: NewModuleName("app"),
			URI:    "file.c3",
			Kind:   ast.MACRO,
			NodeDecl: &ast.MacroDecl{
				Signature: &ast.MacroSignature{
					Name:         &ast.Ident{Name: "foo"},
					ParentTypeId: option.Some(&ast.Ident{Name: "Obj"}),
				},
			},
		}

		hover := buildHover(symbol)

		assert.Equal(t, "```c3\nmacro Obj.foo()\n```\n\nIn module **[app]**", hover.Contents.(protocol.MarkupContent).Value)
	})

	t.Run("macro with arguments", func(t *testing.T) {
		symbol := &Symbol{
			Name:   "macro",
			Module: NewModuleName("app"),
			URI:    "file.c3",
			Kind:   ast.MACRO,
			NodeDecl: &ast.MacroDecl{
				Signature: &ast.MacroSignature{
					Name: &ast.Ident{Name: "foo"},
					Parameters: []*ast.FunctionParameter{
						{
							Name: &ast.Ident{Name: "arg"},
							Type: &ast.TypeInfo{
								Identifier: &ast.Ident{Name: "int"},
							},
						},
					},
				},
			},
		}

		hover := buildHover(symbol)

		assert.Equal(t, "```c3\nmacro foo(int arg)\n```\n\nIn module **[app]**", hover.Contents.(protocol.MarkupContent).Value)
	})

	t.Run("macro with arguments and trailing param block", func(t *testing.T) {
		symbol := &Symbol{
			Name:   "macro",
			Module: NewModuleName("app"),
			URI:    "file.c3",
			Kind:   ast.MACRO,
			NodeDecl: &ast.MacroDecl{
				Signature: &ast.MacroSignature{
					Name: &ast.Ident{Name: "foo"},
					Parameters: []*ast.FunctionParameter{
						{
							Name: &ast.Ident{Name: "arg"},
							Type: &ast.TypeInfo{
								Identifier: &ast.Ident{Name: "int"},
							},
						},
					},
					TrailingBlockParam: &ast.TrailingBlockParam{
						Name: &ast.Ident{Name: "@body"},
					},
				},
			},
		}

		hover := buildHover(symbol)

		assert.Equal(t, "```c3\nmacro foo(int arg; @body)\n```\n\nIn module **[app]**", hover.Contents.(protocol.MarkupContent).Value)
	})

	t.Run("macro with arguments and trailing param block with args", func(t *testing.T) {
		symbol := &Symbol{
			Name:   "macro",
			Module: NewModuleName("app"),
			URI:    "file.c3",
			Kind:   ast.MACRO,
			NodeDecl: &ast.MacroDecl{
				Signature: &ast.MacroSignature{
					Name: &ast.Ident{Name: "foo"},
					Parameters: []*ast.FunctionParameter{
						{
							Name: &ast.Ident{Name: "arg"},
							Type: &ast.TypeInfo{
								Identifier: &ast.Ident{Name: "int"},
							},
						},
					},
					TrailingBlockParam: &ast.TrailingBlockParam{
						Name: &ast.Ident{Name: "@body"},
						Parameters: []*ast.FunctionParameter{
							{
								Name: &ast.Ident{Name: "it"},
							},
						},
					},
				},
			},
		}

		hover := buildHover(symbol)

		assert.Equal(t, "```c3\nmacro foo(int arg; @body(it))\n```\n\nIn module **[app]**", hover.Contents.(protocol.MarkupContent).Value)
	})
}
