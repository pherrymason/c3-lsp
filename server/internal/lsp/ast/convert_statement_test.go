package ast

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func TestConvertToAST_declaration_stmt_constant(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected ASTNode
	}{
		{
			input: "const int I;",
			expected: ConstDecl{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 3, 1, 15).Build(),
				Names:       []Identifier{NewIdentifierBuilder().WithName("I").WithStartEnd(1, 13, 1, 14).Build()},
				Type: option.Some(NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithStartEnd(1, 9, 1, 12).
					WithNameStartEnd(1, 9, 1, 12).
					Build()),
			},
		},
		{
			input: "const int I = 1;", // With initialization
			expected: ConstDecl{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 3, 1, 19).Build(),
				Names:       []Identifier{NewIdentifierBuilder().WithName("I").WithStartEnd(1, 13, 1, 14).Build()},
				Type: option.Some(NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithStartEnd(1, 9, 1, 12).
					WithNameStartEnd(1, 9, 1, 12).
					Build()),
				Initializer: IntegerLiteral{Value: "1"},
			},
		},
		{
			input: "const I;", // Without type
			expected: ConstDecl{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 3, 1, 11).Build(),
				Names:       []Identifier{NewIdentifierBuilder().WithName("I").WithStartEnd(1, 9, 1, 10).Build()},
				Type:        option.None[TypeInfo](),
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("declaration_stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			` + tt.input

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			varDecl := ast.Modules[0].Declarations[0].(ConstDecl)
			assert.Equal(t, tt.expected, varDecl)
		})
	}
}

func TestConvertToAST_declaration_stmt_local_variable(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected ASTNode
	}{
		{
			input: "int i;",
			expected: VariableDecl{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 9).Build(),
				Names:       []Identifier{NewIdentifierBuilder().WithName("i").WithStartEnd(2, 7, 2, 8).Build()},
				Type: NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithStartEnd(2, 3, 2, 6).
					WithNameStartEnd(2, 3, 2, 6).
					Build(),
			},
		},
		{
			input: "int i = 1;", // With initialization
			expected: VariableDecl{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 13).Build(),
				Names:       []Identifier{NewIdentifierBuilder().WithName("i").WithStartEnd(2, 7, 2, 8).Build()},
				Type: NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithStartEnd(2, 3, 2, 6).
					WithNameStartEnd(2, 3, 2, 6).
					Build(),
				Initializer: IntegerLiteral{Value: "1"},
			},
		},
		{
			input: "static int i = 1;", // With initialization
			expected: VariableDecl{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 20).Build(),
				Names:       []Identifier{NewIdentifierBuilder().WithName("i").WithStartEnd(2, 14, 2, 15).Build()},
				Type: NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					IsStatic().
					WithStartEnd(2, 10, 2, 13).
					WithNameStartEnd(2, 10, 2, 13).
					Build(),
				Initializer: IntegerLiteral{Value: "1"},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("declaration_stmt_local_variable: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(VariableDecl))
		})
	}
}

func TestConvertToAST_continue_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected ContinueStatement
	}{
		{
			input: "continue;",
			expected: ContinueStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 12).Build(),
				Label:       option.None[string](),
			},
		},
		{
			input: "continue FOO;", // With label
			expected: ContinueStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 16).Build(),
				Label:       option.Some("FOO"),
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("continue_stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(ContinueStatement))
		})
	}
}

func TestConvertToAST_break_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected BreakStatement
	}{
		{
			input: "break;",
			expected: BreakStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 9).Build(),
				Label:       option.None[string](),
			},
		},
		{
			input: "break FOO;", // With label
			expected: BreakStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 13).Build(),
				Label:       option.Some("FOO"),
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("break_stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(BreakStatement))
		})
	}
}

func TestConvertToAST_switch_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected SwitchStatement
	}{
		{
			input: `
			switch (foo) {
				case 1:
					hello;
				case 2:
					bye;
				default:
					chirp;
			}`,
			expected: SwitchStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 10, 4).Build(),
				Label:       option.None[string](),
				Cases: []SwitchCase{
					{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 4, 5, 11).Build(),
						Value:       IntegerLiteral{Value: "1"},
						Statements: []Statement{
							NewIdentifierBuilder().WithName("hello").WithStartEnd(5, 5, 5, 10).Build(),
						},
					},
					{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(6, 4, 7, 9).Build(),
						Value:       IntegerLiteral{Value: "2"},
						Statements: []Statement{
							NewIdentifierBuilder().WithName("bye").WithStartEnd(7, 5, 7, 8).Build(),
						},
					},
				},
				Default: []Statement{
					NewIdentifierBuilder().WithName("chirp").WithStartEnd(9, 5, 9, 10).Build(),
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("switch_stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(SwitchStatement))
		})
	}
}
