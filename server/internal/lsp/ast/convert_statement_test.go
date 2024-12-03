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
		expected Node
	}{
		{
			input: "const int I;",
			expected: &ConstDecl{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(1, 3, 1, 15).Build(),
				Names:          []*Ident{NewIdentifierBuilder().WithName("I").WithStartEnd(1, 13, 1, 14).BuildPtr()},
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
			expected: &ConstDecl{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(1, 3, 1, 19).Build(),
				Names:          []*Ident{NewIdentifierBuilder().WithName("I").WithStartEnd(1, 13, 1, 14).BuildPtr()},
				Type: option.Some(NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithStartEnd(1, 9, 1, 12).
					WithNameStartEnd(1, 9, 1, 12).
					Build()),
				Initializer: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(1, 17, 1, 18).Build(),
					Kind:           INT,
					Value:          "1",
				},
			},
		},
		{
			input: "const I;", // Without type
			expected: &ConstDecl{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(1, 3, 1, 11).Build(),
				Names:          []*Ident{NewIdentifierBuilder().WithName("I").WithStartEnd(1, 9, 1, 10).BuildPtr()},
				Type:           option.None[TypeInfo](),
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

			varDecl := ast.Modules[0].Declarations[0].(*ConstDecl)
			assert.Equal(t, tt.expected, varDecl)
		})
	}
}

func TestConvertToAST_declaration_stmt_local_variable(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected Node
	}{
		{
			input: "int i;",
			expected: &VariableDecl{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 9).Build(),
				Names:          []*Ident{NewIdentifierBuilder().WithName("i").WithStartEnd(2, 7, 2, 8).BuildPtr()},
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
			expected: &VariableDecl{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 13).Build(),
				Names:          []*Ident{NewIdentifierBuilder().WithName("i").WithStartEnd(2, 7, 2, 8).BuildPtr()},
				Type: NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithStartEnd(2, 3, 2, 6).
					WithNameStartEnd(2, 3, 2, 6).
					Build(),
				Initializer: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 11, 2, 12).Build(),
					Kind:           INT,
					Value:          "1",
				},
			},
		},
		{
			input: "static int i = 1;", // With initialization
			expected: &VariableDecl{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 20).Build(),
				Names:          []*Ident{NewIdentifierBuilder().WithName("i").WithStartEnd(2, 14, 2, 15).BuildPtr()},
				Type: NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					IsStatic().
					WithStartEnd(2, 10, 2, 13).
					WithNameStartEnd(2, 10, 2, 13).
					Build(),
				Initializer: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 18, 2, 19).Build(),
					Kind:           INT,
					Value:          "1",
				},
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*DeclarationStmt).Decl) //(VariableDecl))
		})
	}
}

func TestConvertToAST_continue_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ContinueStatement
	}{
		{
			input: "continue;",
			expected: &ContinueStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 12).Build(),
				Label:          option.None[string](),
			},
		},
		{
			input: "continue FOO;", // With label
			expected: &ContinueStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 16).Build(),
				Label:          option.Some("FOO"),
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*ContinueStatement))
		})
	}
}

func TestConvertToAST_break_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *BreakStatement
	}{
		{
			input: "break;",
			expected: &BreakStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 9).Build(),
				Label:          option.None[string](),
			},
		},
		{
			input: "break FOO;", // With label
			expected: &BreakStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 13).Build(),
				Label:          option.Some("FOO"),
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*BreakStatement))
		})
	}
}

func TestConvertToAST_switch_stmt(t *testing.T) {

	expected := &SwitchStatement{
		NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 10, 4).Build(),
		Label:          option.None[string](),
		Condition: []*DeclOrExpr{
			{Expr: NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 11, 3, 14).BuildPtr()},
		},
		Cases: []SwitchCase{
			{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 4, 5, 11).Build(),
				Value: &ExpressionStmt{
					Expr: &BasicLit{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 9, 4, 10).Build(),
						Kind:           INT,
						Value:          "1",
					},
				},
				Statements: []Statement{
					&ExpressionStmt{
						Expr: NewIdentifierBuilder().WithName("hello").WithStartEnd(5, 5, 5, 10).BuildPtr(),
					},
				},
			},
			{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(6, 4, 7, 9).Build(),
				Value: &ExpressionStmt{Expr: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(6, 9, 6, 10).Build(),
					Kind:           INT,
					Value:          "2",
				}},
				Statements: []Statement{
					&ExpressionStmt{Expr: NewIdentifierBuilder().WithName("bye").WithStartEnd(7, 5, 7, 8).BuildPtr()},
				},
			},
		},
		Default: []Statement{
			&ExpressionStmt{Expr: NewIdentifierBuilder().WithName("chirp").WithStartEnd(9, 5, 9, 10).BuildPtr()},
		},
	}

	source := `module foo;
			fn void main(){
			
			switch (foo) {
				case 1:
					hello;
				case 2:
					bye;
				default:
					chirp;
			}
			}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
	assert.Equal(t, expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*SwitchStatement))
}

func TestConvertToAST_switch_with_ident_in_case(t *testing.T) {
	source := `module foo;
			fn void main(){
				switch (foo) {
					case int:
						hello;
				}
			}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)

	expected := NewTypeInfoBuilder().
		WithName("int").
		IsBuiltin().
		WithNameStartEnd(3, 10, 3, 13).
		WithStartEnd(3, 10, 3, 13).
		Build()

	assert.Equal(t, expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*SwitchStatement).Cases[0].Value.(*ExpressionStmt).Expr)
}

func TestConvertToAST_switch_with_range_in_case(t *testing.T) {
	source := `module foo;
			fn void main(){
				switch (foo) {
					case 1..10:
						hello;
				}
			}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)

	expected := &SwitchCaseRange{
		NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 10, 3, 15).Build(),
		Start: &BasicLit{
			NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 10, 3, 11).Build(),
			Kind:           INT, Value: "1"},
		End: &BasicLit{
			NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 13, 3, 15).Build(),
			Kind:           INT, Value: "10"},
	}

	assert.Equal(t, expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*SwitchStatement).Cases[0].Value.(*SwitchCaseRange))
}

func TestConvertToAST_nextcase(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *Nextcase
	}{
		{
			skip:  true,
			input: `nextcase;`,
			expected: &Nextcase{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 12).Build(),
				Label:          option.None[string](),
			},
		},
		{
			skip:  true,
			input: `nextcase 3;`,
			expected: &Nextcase{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 14).Build(),
				Label:          option.None[string](),
				Value: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 12, 2, 13).Build(),
					Kind:           INT,
					Value:          "3"},
			},
		},
		{
			skip:  true,
			input: `nextcase LABEL:3;`,
			expected: &Nextcase{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 20).Build(),
				Label:          option.Some("LABEL"),
				Value: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 18, 2, 19).Build(),
					Kind:           INT,
					Value:          "3",
				},
			},
		},
		{
			input: `nextcase rand();`,
			expected: &Nextcase{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 19).Build(),
				Label:          option.None[string](),
				Value: &FunctionCall{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 12, 2, 18).Build(),
					Identifier:     NewIdentifierBuilder().WithName("rand").WithStartEnd(2, 12, 2, 16).BuildPtr(),
					Arguments:      []Expression{},
				},
			},
		},
		{
			input: `nextcase default;`,
			expected: &Nextcase{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(2, 3, 2, 20).Build(),
				Label:          option.None[string](),
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*Nextcase))
		})
	}
}

func TestConvertToAST_if_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *IfStmt
	}{
		{
			//skip: true,
			input: `
			if (true) {}`,
			expected: &IfStmt{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 15).Build(),
				Label:          option.None[string](),
				Condition: []*DeclOrExpr{{Expr: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 7, 3, 11).Build(),
					Kind:           BOOLEAN,
					Value:          "true",
				}}},
			},
		},
		{
			skip: false,
			input: `
			if (c > 0) {}`,
			expected: &IfStmt{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 16).Build(),
				Label:          option.None[string](),
				Condition: []*DeclOrExpr{
					{Expr: &BinaryExpression{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 7, 3, 12).Build(),
						Left:           NewIdentifierBuilder().WithName("c").WithStartEnd(3, 7, 3, 8).BuildPtr(),
						Operator:       ">",
						Right: &BasicLit{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 11, 3, 12).Build(),
							Kind:           INT,
							Value:          "0",
						},
					},
					},
				},
			},
		},
		{ // Comma separated conditions
			skip: false,
			input: `
			if (c > 0, c < 10) {}`,
			expected: &IfStmt{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 24).Build(),
				Label:          option.None[string](),
				Condition: []*DeclOrExpr{
					{Expr: &BinaryExpression{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 7, 3, 12).Build(),
						Left:           NewIdentifierBuilder().WithName("c").WithStartEnd(3, 7, 3, 8).BuildPtr(),
						Operator:       ">",
						Right: &BasicLit{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 11, 3, 12).Build(),
							Kind:           INT,
							Value:          "0",
						},
					}},
					{Expr: &BinaryExpression{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 14, 3, 20).Build(),
						Left:           NewIdentifierBuilder().WithName("c").WithStartEnd(3, 14, 3, 15).BuildPtr(),
						Operator:       "<",
						Right: &BasicLit{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 18, 3, 20).Build(),
							Kind:           INT,
							Value:          "10",
						},
					}},
				},
			},
		},
		{
			skip: false,
			input: `
			if (value) {}
			else {}`,
			expected: &IfStmt{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 4, 10).Build(),
				Label:          option.None[string](),
				Condition: []*DeclOrExpr{
					{Expr: NewIdentifierBuilder().WithName("value").WithStartEnd(3, 7, 3, 12).BuildPtr()},
				},
				Else: ElseStatement{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 3, 4, 10).Build(),
				},
			},
		},
		{
			input: `
			if (value){}
			else if (value2){}`,
			expected: &IfStmt{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 4, 21).Build(),
				Label:          option.None[string](),
				Condition: []*DeclOrExpr{
					{Expr: NewIdentifierBuilder().WithName("value").WithStartEnd(3, 7, 3, 12).BuildPtr()},
				},
				Else: ElseStatement{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 3, 4, 21).Build(),
					Statement: &IfStmt{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 8, 4, 21).Build(),
						Label:          option.None[string](),
						Condition: []*DeclOrExpr{
							{Expr: NewIdentifierBuilder().WithName("value2").WithStartEnd(4, 12, 4, 18).BuildPtr()},
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
			expected: &IfStmt{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Label:          option.Some("FOO"),
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*IfStmt))
		})
	}
}

func TestConvertToAST_for_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ForStatement
	}{
		{
			skip: false,
			input: `
			for (int i=0; i<10; i++) {}`,
			expected: &ForStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 30).Build(),
				Label:          option.None[string](),
				Initializer: []*DeclOrExpr{
					{
						Stmt: &DeclarationStmt{
							//NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 8, 3, 15).Build(),
							Decl: &VariableDecl{
								NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 8, 3, 15).Build(),
								Names: []*Ident{
									NewIdentifierBuilder().
										WithName("i").
										WithStartEnd(3, 12, 3, 13).
										BuildPtr(),
								},
								Type: NewTypeInfoBuilder().
									WithName("int").
									WithStartEnd(3, 8, 3, 11).
									WithNameStartEnd(3, 8, 3, 11).
									IsBuiltin().
									Build(),
								Initializer: &BasicLit{
									NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 14, 3, 15).Build(),
									Kind:           INT,
									Value:          "0",
								},
							},
						},
					},
				},
				Condition: &DeclOrExpr{
					Expr: &BinaryExpression{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 17, 3, 21).Build(),
						Left:           NewIdentifierBuilder().WithName("i").WithStartEnd(3, 17, 3, 18).BuildPtr(),
						Right: &BasicLit{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 19, 3, 21).Build(),
							Kind:           INT,
							Value:          "10",
						},
						Operator: "<",
					},
				},
				Update: []*DeclOrExpr{
					{
						Expr: &UpdateExpression{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 23, 3, 26).Build(),
							Operator:       "++",
							Argument:       NewIdentifierBuilder().WithName("i").WithStartEnd(3, 23, 3, 24).BuildPtr(),
						},
					},
				},
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 28, 3, 30).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			for (int i=0, j=0; true; i++) {}`,
			expected: &ForStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 35).Build(),
				Label:          option.None[string](),
				Initializer: []*DeclOrExpr{
					{
						Stmt: &DeclarationStmt{
							Decl: &VariableDecl{
								NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 8, 3, 20).Build(),
								Names: []*Ident{
									NewIdentifierBuilder().
										WithName("i").
										WithStartEnd(3, 12, 3, 13).
										BuildPtr(),
								},
								Type: NewTypeInfoBuilder().
									WithName("int").
									WithStartEnd(3, 8, 3, 11).
									WithNameStartEnd(3, 8, 3, 11).
									IsBuiltin().
									Build(),
								Initializer: &BasicLit{
									NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 14, 3, 15).Build(),
									Kind:           INT,
									Value:          "0",
								},
							},
						},
					},
					{Expr: &AssignmentExpression{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 17, 3, 20).Build(),
						Left: NewIdentifierBuilder().
							WithName("j").
							WithStartEnd(3, 17, 3, 18).
							BuildPtr(),
						Right: &BasicLit{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 19, 3, 20).Build(),
							Kind:           INT,
							Value:          "0",
						},
						Operator: "=",
					},
					},
				},
				Condition: &DeclOrExpr{
					Expr: &BasicLit{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 22, 3, 26).Build(),
						Kind:           BOOLEAN,
						Value:          "true"},
				},
				Update: []*DeclOrExpr{
					{
						Expr: &UpdateExpression{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 28, 3, 31).Build(),
							Operator:       "++",
							Argument:       NewIdentifierBuilder().WithName("i").WithStartEnd(3, 28, 3, 29).BuildPtr(),
						},
					},
				},
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 33, 3, 35).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			for (int i=0; foo(); i++) {}`,
			expected: &ForStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 31).Build(),
				Label:          option.None[string](),
				Initializer: []*DeclOrExpr{
					{Stmt: &DeclarationStmt{Decl: &VariableDecl{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 8, 3, 15).Build(),
						Names: []*Ident{
							NewIdentifierBuilder().
								WithName("i").
								WithStartEnd(3, 12, 3, 13).
								BuildPtr(),
						},
						Type: NewTypeInfoBuilder().
							WithName("int").
							WithStartEnd(3, 8, 3, 11).
							WithNameStartEnd(3, 8, 3, 11).
							IsBuiltin().
							Build(),
						Initializer: &BasicLit{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 14, 3, 15).Build(),
							Kind:           INT,
							Value:          "0"},
					},
					},
					},
				},
				Condition: &DeclOrExpr{
					Expr: &FunctionCall{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 17, 3, 22).Build(),
						Identifier:     NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 17, 3, 20).BuildPtr(),
						Arguments:      []Expression{},
					},
				},
				Update: []*DeclOrExpr{
					{Expr: &UpdateExpression{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 24, 3, 27).Build(),
						Operator:       "++",
						Argument:       NewIdentifierBuilder().WithName("i").WithStartEnd(3, 24, 3, 25).BuildPtr(),
					},
					},
				},
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 29, 3, 31).Build(),
					Statements:     []Statement{},
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
			expected: &ForStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Label:          option.None[string](),
				Initializer:    nil,
				Condition:      nil,
				Update:         nil,
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 12, 5, 4).Build(),
					Statements: []Statement{
						&DeclarationStmt{Decl: &VariableDecl{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 4, 4, 14).Build(),
							Names: []*Ident{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
							Initializer: &BasicLit{
								NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 12, 4, 13).Build(),
								Kind:           INT,
								Value:          "0"},
						},
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*ForStatement))
		})
	}
}

func TestConvertToAST_foreach_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ForeachStatement
	}{
		{
			skip: false,
			input: `
			foreach (int x : a) {}`,
			expected: &ForeachStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 25).Build(),
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("x").WithStartEnd(3, 16, 3, 17).BuildPtr(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 20, 3, 21).BuildPtr(),
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 23, 3, 25).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int &x : a) {}`,
			expected: &ForeachStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 26).Build(),
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().IsReference().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("x").WithStartEnd(3, 17, 3, 18).BuildPtr(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 21, 3, 22).BuildPtr(),
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 24, 3, 26).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int idx, char value : a) {}`,
			expected: &ForeachStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 39).Build(),

				Index: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("idx").WithStartEnd(3, 16, 3, 19).BuildPtr(),
				},
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("char").IsBuiltin().WithNameStartEnd(3, 21, 3, 25).WithStartEnd(3, 21, 3, 25).Build(),
					Identifier: NewIdentifierBuilder().WithName("value").WithStartEnd(3, 26, 3, 31).BuildPtr(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 34, 3, 35).BuildPtr(),
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 37, 3, 39).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int x : a) {
				int i;
			}`,
			expected: &ForeachStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Value: ForeachValue{
					Type:       NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: NewIdentifierBuilder().WithName("x").WithStartEnd(3, 16, 3, 17).BuildPtr(),
				},
				Collection: NewIdentifierBuilder().WithName("a").WithStartEnd(3, 20, 3, 21).BuildPtr(),
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 23, 5, 4).Build(),
					Statements: []Statement{
						&DeclarationStmt{Decl: &VariableDecl{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 4, 4, 10).Build(),
							Names: []*Ident{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
						},
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*ForeachStatement))
		})
	}
}

func TestConvertToAST_while_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *WhileStatement
	}{
		{
			skip: false,
			input: `
			while (true) {}`,
			expected: &WhileStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 18).Build(),
				Condition: []*DeclOrExpr{
					{Expr: &BasicLit{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 10, 3, 14).Build(),
						Kind:           BOOLEAN, Value: "true"}},
				},
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 16, 3, 18).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			while (true) {
				int i;
			}`,
			expected: &WhileStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 5, 4).Build(),
				Condition: []*DeclOrExpr{
					{Expr: &BasicLit{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 10, 3, 14).Build(),
						Kind:           BOOLEAN,
						Value:          "true",
					},
					},
				},
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 16, 5, 4).Build(),
					Statements: []Statement{
						&DeclarationStmt{Decl: &VariableDecl{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 4, 4, 10).Build(),
							Names: []*Ident{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
						},
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*WhileStatement))
		})
	}
}

func TestConvertToAST_do_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *DoStatement
	}{
		{
			skip: false,
			input: `
			do {};`,
			expected: &DoStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 9).Build(),
				Condition:      nil,
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 6, 3, 8).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			do {} while(true);`,
			expected: &DoStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 21).Build(),
				Condition: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 15, 3, 19).Build(),
					Kind:           BOOLEAN,
					Value:          "true"},
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 6, 3, 8).Build(),
					Statements:     []Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			do {
				int i;
			} while(true);`,
			expected: &DoStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 5, 17).Build(),
				Condition: &BasicLit{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(5, 11, 5, 15).Build(),
					Kind:           BOOLEAN,
					Value:          "true"},
				Body: &CompoundStmt{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 6, 5, 4).Build(),
					Statements: []Statement{
						&DeclarationStmt{Decl: &VariableDecl{
							NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(4, 4, 4, 10).Build(),
							Names: []*Ident{
								NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
							},
							Type: NewTypeInfoBuilder().WithName("int").IsBuiltin().
								WithStartEnd(4, 4, 4, 7).
								WithNameStartEnd(4, 4, 4, 7).Build(),
						},
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*DoStatement))
		})
	}
}

func TestConvertToAST_defer_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *DeferStatement
	}{
		{
			skip: false,
			input: `
			defer foo();`,
			expected: &DeferStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 15).Build(),
				Statement: &ExpressionStmt{Expr: &FunctionCall{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 9, 3, 14).Build(),
					Identifier:     NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 9, 3, 12).BuildPtr(),
					Arguments:      []Expression{},
				},
				},
			},
		},
		{
			skip: false,
			input: `
			defer try foo();`,
			expected: &DeferStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 19).Build(),
				Statement: &ExpressionStmt{Expr: &FunctionCall{
					NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 13, 3, 18).Build(),
					Identifier:     NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 13, 3, 16).BuildPtr(),
					Arguments:      []Expression{},
				},
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*DeferStatement))
		})
	}
}

func TestConvertToAST_assert_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *AssertStatement
	}{
		{
			skip: false,
			input: `
			assert(true);`,
			expected: &AssertStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 16).Build(),
				Assertions: []Expression{
					&BasicLit{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 10, 3, 14).Build(),
						Kind:           BOOLEAN,
						Value:          "true"},
				},
			},
		},
		{
			skip: false,
			input: `
			assert(true,1);`,
			expected: &AssertStatement{
				NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 3, 3, 18).Build(),
				Assertions: []Expression{
					&BasicLit{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 10, 3, 14).Build(),
						Kind:           BOOLEAN,
						Value:          "true",
					},
					&BasicLit{
						NodeAttributes: NewNodeAttributesBuilder().WithStartEnd(3, 15, 3, 16).Build(),
						Kind:           INT,
						Value:          "1"},
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

			funcDecl := ast.Modules[0].Declarations[0].(*FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*CompoundStmt).Statements[0].(*AssertStatement))
		})
	}
}
