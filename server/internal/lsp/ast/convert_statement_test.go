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
		{
			input: `
			switch (foo) {
				case int:
					hello;
			}`,
			expected: SwitchStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 6, 4).Build(),
				Label:       option.None[string](),
				Cases: []SwitchCase{
					{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 4, 5, 11).Build(),
						Value: NewTypeInfoBuilder().
							WithName("int").
							IsBuiltin().
							WithNameStartEnd(4, 9, 4, 12).
							WithStartEnd(4, 9, 4, 12).
							Build(),
						Statements: []Statement{
							NewIdentifierBuilder().WithName("hello").WithStartEnd(5, 5, 5, 10).Build(),
						},
					},
				},
			},
		},
		{
			input: `
			switch (foo) {
				case 1..10:
					hello;
			}`,
			expected: SwitchStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 6, 4).Build(),
				Label:       option.None[string](),
				Cases: []SwitchCase{
					{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 4, 5, 11).Build(),
						Value: SwitchCaseRange{
							ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 9, 4, 14).Build(),
							Start:       IntegerLiteral{Value: "1"},
							End:         IntegerLiteral{Value: "10"},
						},
						Statements: []Statement{
							NewIdentifierBuilder().WithName("hello").WithStartEnd(5, 5, 5, 10).Build(),
						},
					},
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

func TestConvertToAST_nextcase(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected Nextcase
	}{
		{
			input: `nextcase;`,
			expected: Nextcase{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 12).Build(),
				Label:       option.None[string](),
			},
		},
		{
			input: `nextcase 3;`,
			expected: Nextcase{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 14).Build(),
				Label:       option.None[string](),
				Value:       IntegerLiteral{Value: "3"},
			},
		},
		{
			input: `nextcase LABEL:3;`,
			expected: Nextcase{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 20).Build(),
				Label:       option.Some("LABEL"),
				Value:       IntegerLiteral{Value: "3"},
			},
		},
		{
			input: `nextcase rand();`,
			expected: Nextcase{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 19).Build(),
				Label:       option.None[string](),
				Value: FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 12, 2, 18).Build(),
					Identifier:  NewIdentifierBuilder().WithName("rand").WithStartEnd(2, 12, 2, 16).Build(),
					Arguments:   []Arg{},
				},
			},
		},
		{
			input: `nextcase default;`,
			expected: Nextcase{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 3, 2, 20).Build(),
				Label:       option.None[string](),
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("nextcase: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(Nextcase))
		})
	}
}

func TestConvertToAST_if_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected IfStatement
	}{
		{
			//skip: true,
			input: `
			if (true) {}`,
			expected: IfStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 15).Build(),
				Label:       option.None[string](),
				Condition:   []Expression{BoolLiteral{Value: true}},
			},
		},
		{
			skip: true,
			input: `
			if (c > 0) {}`,
			expected: IfStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 16).Build(),
				Label:       option.None[string](),
				Condition: []Expression{
					BinaryExpr{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 7, 3, 12).Build(),
						Left:        NewIdentifierBuilder().WithName("c").WithStartEnd(3, 7, 3, 8).Build(),
						Operator:    ">",
						Right:       IntegerLiteral{Value: "0"},
					},
				},
			},
		},
		{ // Comma separated conditions
			skip: true,
			input: `
			if (c > 0, c < 10) {}`,
			expected: IfStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 24).Build(),
				Label:       option.None[string](),
				Condition: []Expression{
					BinaryExpr{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 7, 3, 12).Build(),
						Left:        NewIdentifierBuilder().WithName("c").WithStartEnd(3, 7, 3, 8).Build(),
						Operator:    ">",
						Right:       IntegerLiteral{Value: "0"},
					},
					BinaryExpr{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 14, 3, 20).Build(),
						Left:        NewIdentifierBuilder().WithName("c").WithStartEnd(3, 14, 3, 15).Build(),
						Operator:    "<",
						Right:       IntegerLiteral{Value: "10"},
					},
				},
			},
		},
		{
			skip: false,
			input: `
			if (value) {}
			else {}`,
			expected: IfStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 4, 10).Build(),
				Label:       option.None[string](),
				Condition: []Expression{
					NewIdentifierBuilder().WithName("value").WithStartEnd(3, 7, 3, 12).Build(),
				},
				Else: ElseStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 3, 4, 10).Build(),
				},
			},
		},
		{
			input: `
			if (value){}
			else if (value2){}`,
			expected: IfStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 4, 21).Build(),
				Label:       option.None[string](),
				Condition: []Expression{
					NewIdentifierBuilder().WithName("value").WithStartEnd(3, 7, 3, 12).Build(),
				},
				Else: ElseStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 3, 4, 21).Build(),
					Statement: IfStatement{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 8, 4, 21).Build(),
						Label:       option.None[string](),
						Condition: []Expression{
							NewIdentifierBuilder().WithName("value2").WithStartEnd(4, 12, 4, 18).Build(),
						},
					},
				},
			},
		},
		{
			// Labeled IF: TODO
			skip: true,
			input: `
			if FOO: (i > 0)
			{
			}`,
			expected: IfStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Label:       option.Some("FOO"),
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("if stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(IfStatement))
		})
	}
}

func TestConvertToAST_for_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected ForStatement
	}{
		{
			skip: false,
			input: `
			for (int i=0; i<10; i++) {}`,
			expected: ForStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 30).Build(),
				Label:       option.None[string](),
				Initializer: []Expression{
					VariableDecl{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 8, 3, 15).Build(),
						Names: []Identifier{
							NewIdentifierBuilder().
								WithName("i").
								WithStartEnd(3, 12, 3, 13).
								Build(),
						},
						Type: NewTypeInfoBuilder().
							WithName("int").
							WithStartEnd(3, 8, 3, 11).
							WithNameStartEnd(3, 8, 3, 11).
							IsBuiltin().
							Build(),
						Initializer: IntegerLiteral{
							Value: "0",
						},
					},
				},
				Condition: BinaryExpr{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 17, 3, 21).Build(),
					Left:        NewIdentifierBuilder().WithName("i").WithStartEnd(3, 17, 3, 18).Build(),
					Right:       IntegerLiteral{Value: "10"},
					Operator:    "<",
				},
				Update: []Expression{
					UpdateExpression{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 23, 3, 26).Build(),
						Operator:    "++",
						Argument:    NewIdentifierBuilder().WithName("i").WithStartEnd(3, 23, 3, 24).Build(),
					},
				},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 28, 3, 30).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			for (int i=0, j=0; true; i++) {}`,
			expected: ForStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 35).Build(),
				Label:       option.None[string](),
				Initializer: []Expression{
					VariableDecl{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 8, 3, 20).Build(),
						Names: []Identifier{
							NewIdentifierBuilder().
								WithName("i").
								WithStartEnd(3, 12, 3, 13).
								Build(),
						},
						Type: NewTypeInfoBuilder().
							WithName("int").
							WithStartEnd(3, 8, 3, 11).
							WithNameStartEnd(3, 8, 3, 11).
							IsBuiltin().
							Build(),
						Initializer: IntegerLiteral{
							Value: "0",
						},
					},
					AssignmentStatement{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 17, 3, 20).Build(),
						Left: NewIdentifierBuilder().
							WithName("j").
							WithStartEnd(3, 17, 3, 18).
							Build(),
						Right:    IntegerLiteral{Value: "0"},
						Operator: "=",
					},
				},
				Condition: BoolLiteral{Value: true},
				Update: []Expression{
					UpdateExpression{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 28, 3, 31).Build(),
						Operator:    "++",
						Argument:    NewIdentifierBuilder().WithName("i").WithStartEnd(3, 28, 3, 29).Build(),
					},
				},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 33, 3, 35).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			for (int i=0; foo(); i++) {}`,
			expected: ForStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 31).Build(),
				Label:       option.None[string](),
				Initializer: []Expression{
					VariableDecl{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 8, 3, 15).Build(),
						Names: []Identifier{
							NewIdentifierBuilder().
								WithName("i").
								WithStartEnd(3, 12, 3, 13).
								Build(),
						},
						Type: NewTypeInfoBuilder().
							WithName("int").
							WithStartEnd(3, 8, 3, 11).
							WithNameStartEnd(3, 8, 3, 11).
							IsBuiltin().
							Build(),
						Initializer: IntegerLiteral{
							Value: "0",
						},
					},
				},
				Condition: FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 17, 3, 22).Build(),
					Identifier:  NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 17, 3, 20).Build(),
					Arguments:   []Arg{},
				},
				Update: []Expression{
					UpdateExpression{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 24, 3, 27).Build(),
						Operator:    "++",
						Argument:    NewIdentifierBuilder().WithName("i").WithStartEnd(3, 24, 3, 25).Build(),
					},
				},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 29, 3, 31).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			// Testing body
			skip: false,
			input: `
			for (;;) {
				int i = 0;
			}`,
			expected: ForStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Label:       option.None[string](),
				Initializer: nil,
				Condition:   nil,
				Update:      nil,
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 12, 5, 4).Build(),
					Statements: []Expression{
						VariableDecl{
							ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 4, 4, 14).Build(),
							Names: []Identifier{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).Build(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
							Initializer: IntegerLiteral{Value: "0"},
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("for stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(ForStatement))
		})
	}
}

func TestConvertToAST_foreach_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected ForeachStatement
	}{
		{
			skip: false,
			input: `
			foreach (int x : a) {}`,
			expected: ForeachStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 25).Build(),
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("x").WithStartEnd(3, 16, 3, 17).Build(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 20, 3, 21).Build(),
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 23, 3, 25).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int &x : a) {}`,
			expected: ForeachStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 26).Build(),
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().IsReference().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("x").WithStartEnd(3, 17, 3, 18).Build(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 21, 3, 22).Build(),
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 24, 3, 26).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int idx, char value : a) {}`,
			expected: ForeachStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 39).Build(),

				Index: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("idx").WithStartEnd(3, 16, 3, 19).Build(),
				},
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("char").IsBuiltin().WithNameStartEnd(3, 21, 3, 25).WithStartEnd(3, 21, 3, 25).Build(),
					Identifier: NewIdentifierBuilder().WithName("value").WithStartEnd(3, 26, 3, 31).Build(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 34, 3, 35).Build(),
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 37, 3, 39).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int x : a) {
				int i;
			}`,
			expected: ForeachStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("x").WithStartEnd(3, 16, 3, 17).Build(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 20, 3, 21).Build(),
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 23, 5, 4).Build(),
					Statements: []Expression{
						VariableDecl{
							ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 4, 4, 10).Build(),
							Names: []Identifier{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).Build(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("foreach stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(ForeachStatement))
		})
	}
}

func TestConvertToAST_while_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected WhileStatement
	}{
		{
			skip: false,
			input: `
			while (true) {}`,
			expected: WhileStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 18).Build(),
				Condition: []Expression{
					BoolLiteral{Value: true},
				},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 16, 3, 18).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			while (true) {
				int i;
			}`,
			expected: WhileStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Condition: []Expression{
					BoolLiteral{Value: true},
				},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 16, 5, 4).Build(),
					Statements: []Expression{
						VariableDecl{
							ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 4, 4, 10).Build(),
							Names: []Identifier{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).Build(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("while stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(WhileStatement))
		})
	}
}

func TestConvertToAST_do_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected DoStatement
	}{
		{
			skip: false,
			input: `
			do {};`,
			expected: DoStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 9).Build(),
				Condition:   nil,
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 6, 3, 8).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			do {} while(true);`,
			expected: DoStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 21).Build(),
				Condition:   BoolLiteral{Value: true},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 6, 3, 8).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			skip: false,
			input: `
			do {
				int i;
			} while(true);`,
			expected: DoStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 5, 17).Build(),
				Condition:   BoolLiteral{Value: true},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 6, 5, 4).Build(),
					Statements: []Expression{
						VariableDecl{
							ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 4, 4, 10).Build(),
							Names: []Identifier{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).Build(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("do stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(DoStatement))
		})
	}
}

func TestConvertToAST_defer_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected DeferStatement
	}{
		{
			skip: false,
			input: `
			defer foo();`,
			expected: DeferStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 15).Build(),
				Statement: FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 9, 3, 14).Build(),
					Identifier:  NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 9, 3, 12).Build(),
					Arguments:   []Arg{},
				},
			},
		},
		{
			skip: false,
			input: `
			defer try foo();`,
			expected: DeferStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 19).Build(),
				Statement: FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 13, 3, 18).Build(),
					Identifier:  NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 13, 3, 16).Build(),
					Arguments:   []Arg{},
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("defer stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(DeferStatement))
		})
	}
}

func TestConvertToAST_assert_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected AssertStatement
	}{
		{
			skip: false,
			input: `
			assert(true);`,
			expected: AssertStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 16).Build(),
				Assertions: []Expression{
					BoolLiteral{Value: true},
				},
			},
		},
		{
			skip: false,
			input: `
			assert(true,1);`,
			expected: AssertStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 3, 3, 18).Build(),
				Assertions: []Expression{
					BoolLiteral{Value: true},
					IntegerLiteral{Value: "1"},
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("assert stmt: %s", tt.input), func(t *testing.T) {
			source := `module foo;
			fn void main(){
			` + tt.input + `
			}`

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := ast.Modules[0].Functions[0].(FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(CompoundStatement).Statements[0].(AssertStatement))
		})
	}
}