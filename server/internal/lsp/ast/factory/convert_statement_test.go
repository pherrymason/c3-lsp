package factory

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func TestConvertToAST_declaration_stmt_constant(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected ast.Node
	}{
		{
			input: "const int I;",
			expected: &ast.GenDecl{
				Token:          ast.Token(ast.CONST),
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 3, 1, 15).Build(),
				Spec: &ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdentifierBuilder().WithName("I").WithStartEnd(1, 13, 1, 14).BuildPtr()},
					Type: ast.NewTypeInfoBuilder().
						WithName("int").
						IsBuiltin().
						WithStartEnd(1, 9, 1, 12).
						WithNameStartEnd(1, 9, 1, 12).
						Build(),
				},
			},
		},
		{
			input: "const int I = 1;", // With initialization
			expected: &ast.GenDecl{
				Token:          ast.Token(ast.CONST),
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 3, 1, 19).Build(),
				Spec: &ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdentifierBuilder().WithName("I").WithStartEnd(1, 13, 1, 14).BuildPtr()},
					Type: ast.NewTypeInfoBuilder().
						WithName("int").
						IsBuiltin().
						WithStartEnd(1, 9, 1, 12).
						WithNameStartEnd(1, 9, 1, 12).
						Build(),
					Value: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 17, 1, 18).Build(),
						Kind:           ast.INT,
						Value:          "1",
					},
				},
			},
		},
		{
			input: "const I;", // Without type
			expected: &ast.GenDecl{
				Token:          ast.Token(ast.CONST),
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 3, 1, 11).Build(),
				Spec: &ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdentifierBuilder().WithName("I").WithStartEnd(1, 9, 1, 10).BuildPtr()},
					Type:  nil,
				},
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			varDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
			assert.Equal(t, tt.expected, varDecl)
		})
	}
}

func TestConvertToAST_declaration_stmt_local_variable(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected ast.Node
	}{
		{
			input: "int i;",
			expected: &ast.GenDecl{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 9).Build(),
				Token:          ast.VAR,
				Spec: &ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdentifierBuilder().WithName("i").WithStartEnd(2, 7, 2, 8).BuildPtr()},
					Type: ast.NewTypeInfoBuilder().
						WithName("int").
						IsBuiltin().
						WithStartEnd(2, 3, 2, 6).
						WithNameStartEnd(2, 3, 2, 6).
						Build(),
				},
			},
		},
		{
			input: "int i = 1;", // With initialization
			expected: &ast.GenDecl{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 13).Build(),
				Token:          ast.VAR,
				Spec: &ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdentifierBuilder().WithName("i").WithStartEnd(2, 7, 2, 8).BuildPtr()},
					Type: ast.NewTypeInfoBuilder().
						WithName("int").
						IsBuiltin().
						WithStartEnd(2, 3, 2, 6).
						WithNameStartEnd(2, 3, 2, 6).
						Build(),
					Value: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 11, 2, 12).Build(),
						Kind:           ast.INT,
						Value:          "1",
					},
				},
			},
		},
		{
			input: "static int i = 1;", // With initialization
			expected: &ast.GenDecl{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 20).Build(),
				Token:          ast.VAR,
				Spec: &ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdentifierBuilder().WithName("i").WithStartEnd(2, 14, 2, 15).BuildPtr()},
					Type: ast.NewTypeInfoBuilder().
						WithName("int").
						IsBuiltin().
						IsStatic().
						WithStartEnd(2, 10, 2, 13).
						WithNameStartEnd(2, 10, 2, 13).
						Build(),
					Value: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 18, 2, 19).Build(),
						Kind:           ast.INT,
						Value:          "1",
					},
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.DeclarationStmt).Decl) //(VariableDecl))
		})
	}
}

func TestConvertToAST_continue_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.ContinueStatement
	}{
		{
			input: "continue;",
			expected: &ast.ContinueStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 12).Build(),
				Label:          option.None[string](),
			},
		},
		{
			input: "continue FOO;", // With label
			expected: &ast.ContinueStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 16).Build(),
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.ContinueStatement))
		})
	}
}

func TestConvertToAST_break_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.BreakStatement
	}{
		{
			input: "break;",
			expected: &ast.BreakStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 9).Build(),
				Label:          option.None[string](),
			},
		},
		{
			input: "break FOO;", // With label
			expected: &ast.BreakStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 13).Build(),
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.BreakStatement))
		})
	}
}

func TestConvertToAST_switch_stmt(t *testing.T) {

	expected := &ast.SwitchStatement{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 10, 4).Build(),
		Label:          option.None[string](),
		Condition: []*ast.DeclOrExpr{
			{Expr: ast.NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 11, 3, 14).BuildPtr()},
		},
		Cases: []ast.SwitchCase{
			{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 4, 5, 11).Build(),
				Value: &ast.ExpressionStmt{
					Expr: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 9, 4, 10).Build(),
						Kind:           ast.INT,
						Value:          "1",
					},
				},
				Statements: []ast.Statement{
					&ast.ExpressionStmt{
						Expr: ast.NewIdentifierBuilder().WithName("hello").WithStartEnd(5, 5, 5, 10).BuildPtr(),
					},
				},
			},
			{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(6, 4, 7, 9).Build(),
				Value: &ast.ExpressionStmt{Expr: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(6, 9, 6, 10).Build(),
					Kind:           ast.INT,
					Value:          "2",
				}},
				Statements: []ast.Statement{
					&ast.ExpressionStmt{Expr: ast.NewIdentifierBuilder().WithName("bye").WithStartEnd(7, 5, 7, 8).BuildPtr()},
				},
			},
		},
		Default: []ast.Statement{
			&ast.ExpressionStmt{Expr: ast.NewIdentifierBuilder().WithName("chirp").WithStartEnd(9, 5, 9, 10).BuildPtr()},
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

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
	assert.Equal(t, expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.SwitchStatement))
}

func TestConvertToAST_switch_with_ident_in_case(t *testing.T) {
	source := `module foo;
			fn void main(){
				switch (foo) {
					case int:
						hello;
				}
			}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	expected := ast.NewTypeInfoBuilder().
		WithName("int").
		IsBuiltin().
		WithNameStartEnd(3, 10, 3, 13).
		WithStartEnd(3, 10, 3, 13).
		Build()

	assert.Equal(t, expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.SwitchStatement).Cases[0].Value.(*ast.ExpressionStmt).Expr)
}

func TestConvertToAST_switch_with_range_in_case(t *testing.T) {
	source := `module foo;
			fn void main(){
				switch (foo) {
					case 1..10:
						hello;
				}
			}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	expected := &ast.SwitchCaseRange{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 10, 3, 15).Build(),
		Start: &ast.BasicLit{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 10, 3, 11).Build(),
			Kind:           ast.INT, Value: "1"},
		End: &ast.BasicLit{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 13, 3, 15).Build(),
			Kind:           ast.INT, Value: "10"},
	}

	assert.Equal(t, expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.SwitchStatement).Cases[0].Value.(*ast.SwitchCaseRange))
}

func TestConvertToAST_nextcase(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.Nextcase
	}{
		{
			skip:  false,
			input: `nextcase;`,
			expected: &ast.Nextcase{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 12).Build(),
				Label:          option.None[string](),
			},
		},
		{
			skip:  false,
			input: `nextcase 3;`,
			expected: &ast.Nextcase{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 14).Build(),
				Label:          option.None[string](),
				Value: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 12, 2, 13).Build(),
					Kind:           ast.INT,
					Value:          "3"},
			},
		},
		{
			skip:  false,
			input: `nextcase LABEL:3;`,
			expected: &ast.Nextcase{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 20).Build(),
				Label:          option.Some("LABEL"),
				Value: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 18, 2, 19).Build(),
					Kind:           ast.INT,
					Value:          "3",
				},
			},
		},
		{
			input: `nextcase rand();`,
			expected: &ast.Nextcase{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 19).Build(),
				Label:          option.None[string](),
				Value: &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 12, 2, 18).Build(),
					Identifier:     ast.NewIdentifierBuilder().WithName("rand").WithStartEnd(2, 12, 2, 16).BuildPtr(),
					Arguments:      []ast.Expression{},
				},
			},
		},
		{
			input: `nextcase default;`,
			expected: &ast.Nextcase{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 3, 2, 20).Build(),
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.Nextcase))
		})
	}
}

func TestConvertToAST_if_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.IfStmt
	}{
		{
			//skip: true,
			input: `
			if (true) {}`,
			expected: &ast.IfStmt{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 15).Build(),
				Label:          option.None[string](),
				Condition: []*ast.DeclOrExpr{{Expr: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 7, 3, 11).Build(),
					Kind:           ast.BOOLEAN,
					Value:          "true",
				}}},
			},
		},
		{
			skip: false,
			input: `
			if (c > 0) {}`,
			expected: &ast.IfStmt{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 16).Build(),
				Label:          option.None[string](),
				Condition: []*ast.DeclOrExpr{
					{Expr: &ast.BinaryExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 7, 3, 12).Build(),
						Left:           ast.NewIdentifierBuilder().WithName("c").WithStartEnd(3, 7, 3, 8).BuildPtr(),
						Operator:       ">",
						Right: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 11, 3, 12).Build(),
							Kind:           ast.INT,
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
			expected: &ast.IfStmt{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 24).Build(),
				Label:          option.None[string](),
				Condition: []*ast.DeclOrExpr{
					{Expr: &ast.BinaryExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 7, 3, 12).Build(),
						Left:           ast.NewIdentifierBuilder().WithName("c").WithStartEnd(3, 7, 3, 8).BuildPtr(),
						Operator:       ">",
						Right: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 11, 3, 12).Build(),
							Kind:           ast.INT,
							Value:          "0",
						},
					}},
					{Expr: &ast.BinaryExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 14, 3, 20).Build(),
						Left:           ast.NewIdentifierBuilder().WithName("c").WithStartEnd(3, 14, 3, 15).BuildPtr(),
						Operator:       "<",
						Right: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 18, 3, 20).Build(),
							Kind:           ast.INT,
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
			expected: &ast.IfStmt{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 4, 10).Build(),
				Label:          option.None[string](),
				Condition: []*ast.DeclOrExpr{
					{Expr: ast.NewIdentifierBuilder().WithName("value").WithStartEnd(3, 7, 3, 12).BuildPtr()},
				},
				Else: ast.ElseStatement{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 3, 4, 10).Build(),
				},
			},
		},
		{
			input: `
			if (value){}
			else if (value2){}`,
			expected: &ast.IfStmt{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 4, 21).Build(),
				Label:          option.None[string](),
				Condition: []*ast.DeclOrExpr{
					{Expr: ast.NewIdentifierBuilder().WithName("value").WithStartEnd(3, 7, 3, 12).BuildPtr()},
				},
				Else: ast.ElseStatement{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 3, 4, 21).Build(),
					Statement: &ast.IfStmt{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 8, 4, 21).Build(),
						Label:          option.None[string](),
						Condition: []*ast.DeclOrExpr{
							{Expr: ast.NewIdentifierBuilder().WithName("value2").WithStartEnd(4, 12, 4, 18).BuildPtr()},
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
			expected: &ast.IfStmt{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 5, 4).Build(),
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.IfStmt))
		})
	}
}

func TestConvertToAST_for_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.ForStatement
	}{
		{
			skip: false,
			input: `
			for (int i=0; i<10; i++) {}`,
			expected: &ast.ForStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 30).Build(),
				Label:          option.None[string](),
				Initializer: []*ast.DeclOrExpr{
					{
						Stmt: &ast.DeclarationStmt{
							//NodeAttributes: NewNodeAttributesBuilder().WithRangePositions(3, 8, 3, 15).Build(),
							Decl: &ast.GenDecl{
								NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 8, 3, 15).Build(),
								Token:          ast.VAR,
								Spec: &ast.ValueSpec{
									Names: []*ast.Ident{
										ast.NewIdentifierBuilder().
											WithName("i").
											WithStartEnd(3, 12, 3, 13).
											BuildPtr(),
									},
									Type: ast.NewTypeInfoBuilder().
										WithName("int").
										WithStartEnd(3, 8, 3, 11).
										WithNameStartEnd(3, 8, 3, 11).
										IsBuiltin().
										Build(),
									Value: &ast.BasicLit{
										NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 14, 3, 15).Build(),
										Kind:           ast.INT,
										Value:          "0",
									},
								},
							},
						},
					},
				},
				Condition: &ast.DeclOrExpr{
					Expr: &ast.BinaryExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 17, 3, 21).Build(),
						Left:           ast.NewIdentifierBuilder().WithName("i").WithStartEnd(3, 17, 3, 18).BuildPtr(),
						Right: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 19, 3, 21).Build(),
							Kind:           ast.INT,
							Value:          "10",
						},
						Operator: "<",
					},
				},
				Update: []*ast.DeclOrExpr{
					{
						Expr: &ast.UpdateExpression{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 23, 3, 26).Build(),
							Operator:       "++",
							Argument:       ast.NewIdentifierBuilder().WithName("i").WithStartEnd(3, 23, 3, 24).BuildPtr(),
						},
					},
				},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 28, 3, 30).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			for (int i=0, j=0; true; i++) {}`,
			expected: &ast.ForStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 35).Build(),
				Label:          option.None[string](),
				Initializer: []*ast.DeclOrExpr{
					{
						Stmt: &ast.DeclarationStmt{
							Decl: &ast.GenDecl{
								NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 8, 3, 20).Build(),
								Token:          ast.VAR,
								Spec: &ast.ValueSpec{
									Names: []*ast.Ident{
										ast.NewIdentifierBuilder().
											WithName("i").
											WithStartEnd(3, 12, 3, 13).
											BuildPtr(),
									},
									Type: ast.NewTypeInfoBuilder().
										WithName("int").
										WithStartEnd(3, 8, 3, 11).
										WithNameStartEnd(3, 8, 3, 11).
										IsBuiltin().
										Build(),
									Value: &ast.BasicLit{
										NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 14, 3, 15).Build(),
										Kind:           ast.INT,
										Value:          "0",
									},
								},
							},
						},
					},
					{Expr: &ast.AssignmentExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 17, 3, 20).Build(),
						Left: ast.NewIdentifierBuilder().
							WithName("j").
							WithStartEnd(3, 17, 3, 18).
							BuildPtr(),
						Right: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 19, 3, 20).Build(),
							Kind:           ast.INT,
							Value:          "0",
						},
						Operator: "=",
					},
					},
				},
				Condition: &ast.DeclOrExpr{
					Expr: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 22, 3, 26).Build(),
						Kind:           ast.BOOLEAN,
						Value:          "true"},
				},
				Update: []*ast.DeclOrExpr{
					{
						Expr: &ast.UpdateExpression{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 28, 3, 31).Build(),
							Operator:       "++",
							Argument:       ast.NewIdentifierBuilder().WithName("i").WithStartEnd(3, 28, 3, 29).BuildPtr(),
						},
					},
				},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 33, 3, 35).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			for (int i=0; foo(); i++) {}`,
			expected: &ast.ForStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 31).Build(),
				Label:          option.None[string](),
				Initializer: []*ast.DeclOrExpr{
					{Stmt: &ast.DeclarationStmt{Decl: &ast.GenDecl{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 8, 3, 15).Build(),
						Token:          ast.VAR,
						Spec: &ast.ValueSpec{
							Names: []*ast.Ident{
								ast.NewIdentifierBuilder().
									WithName("i").
									WithStartEnd(3, 12, 3, 13).
									BuildPtr(),
							},
							Type: ast.NewTypeInfoBuilder().
								WithName("int").
								WithStartEnd(3, 8, 3, 11).
								WithNameStartEnd(3, 8, 3, 11).
								IsBuiltin().
								Build(),
							Value: &ast.BasicLit{
								NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 14, 3, 15).Build(),
								Kind:           ast.INT,
								Value:          "0"},
						},
					},
					},
					},
				},
				Condition: &ast.DeclOrExpr{
					Expr: &ast.FunctionCall{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 17, 3, 22).Build(),
						Identifier:     ast.NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 17, 3, 20).BuildPtr(),
						Arguments:      []ast.Expression{},
					},
				},
				Update: []*ast.DeclOrExpr{
					{Expr: &ast.UpdateExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 24, 3, 27).Build(),
						Operator:       "++",
						Argument:       ast.NewIdentifierBuilder().WithName("i").WithStartEnd(3, 24, 3, 25).BuildPtr(),
					},
					},
				},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 29, 3, 31).Build(),
					Statements:     []ast.Statement{},
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
			expected: &ast.ForStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 5, 4).Build(),
				Label:          option.None[string](),
				Initializer:    nil,
				Condition:      nil,
				Update:         nil,
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 12, 5, 4).Build(),
					Statements: []ast.Statement{
						&ast.DeclarationStmt{Decl: &ast.GenDecl{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 4, 4, 14).Build(),
							Token:          ast.VAR,
							Spec: &ast.ValueSpec{
								Names: []*ast.Ident{
									ast.NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
								},
								Type: ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().
									WithStartEnd(4, 4, 4, 7).
									WithNameStartEnd(4, 4, 4, 7).Build(),
								Value: &ast.BasicLit{
									NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 12, 4, 13).Build(),
									Kind:           ast.INT,
									Value:          "0"},
							},
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.ForStatement))
		})
	}
}

func TestConvertToAST_foreach_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.ForeachStatement
	}{
		{
			skip: false,
			input: `
			foreach (int x : a) {}`,
			expected: &ast.ForeachStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 25).Build(),
				Value: ast.ForeachValue{
					Type:       ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: ast.NewIdentifierBuilder().WithName("x").WithStartEnd(3, 16, 3, 17).BuildPtr(),
				},
				Collection: ast.NewIdentifierBuilder().WithName("a").WithStartEnd(3, 20, 3, 21).BuildPtr(),
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 23, 3, 25).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int &x : a) {}`,
			expected: &ast.ForeachStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 26).Build(),
				Value: ast.ForeachValue{
					Type:       ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().IsReference().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: ast.NewIdentifierBuilder().WithName("x").WithStartEnd(3, 17, 3, 18).BuildPtr(),
				},
				Collection: ast.NewIdentifierBuilder().WithName("a").WithStartEnd(3, 21, 3, 22).BuildPtr(),
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 24, 3, 26).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int idx, char value : a) {}`,
			expected: &ast.ForeachStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 39).Build(),

				Index: ast.ForeachValue{
					Type:       ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: ast.NewIdentifierBuilder().WithName("idx").WithStartEnd(3, 16, 3, 19).BuildPtr(),
				},
				Value: ast.ForeachValue{
					Type:       ast.NewTypeInfoBuilder().WithName("char").IsBuiltin().WithNameStartEnd(3, 21, 3, 25).WithStartEnd(3, 21, 3, 25).Build(),
					Identifier: ast.NewIdentifierBuilder().WithName("value").WithStartEnd(3, 26, 3, 31).BuildPtr(),
				},
				Collection: ast.NewIdentifierBuilder().WithName("a").WithStartEnd(3, 34, 3, 35).BuildPtr(),
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 37, 3, 39).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			foreach (int x : a) {
				int i;
			}`,
			expected: &ast.ForeachStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 5, 4).Build(),
				Value: ast.ForeachValue{
					Type:       ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(3, 12, 3, 15).WithStartEnd(3, 12, 3, 15).Build(),
					Identifier: ast.NewIdentifierBuilder().WithName("x").WithStartEnd(3, 16, 3, 17).BuildPtr(),
				},
				Collection: ast.NewIdentifierBuilder().WithName("a").WithStartEnd(3, 20, 3, 21).BuildPtr(),
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 23, 5, 4).Build(),
					Statements: []ast.Statement{
						&ast.DeclarationStmt{Decl: &ast.GenDecl{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 4, 4, 10).Build(),
							Token:          ast.VAR,
							Spec: &ast.ValueSpec{
								Names: []*ast.Ident{
									ast.NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
								},
								Type: ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().
									WithStartEnd(4, 4, 4, 7).
									WithNameStartEnd(4, 4, 4, 7).Build(),
							},
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.ForeachStatement))
		})
	}
}

func TestConvertToAST_while_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.WhileStatement
	}{
		{
			skip: false,
			input: `
			while (true) {}`,
			expected: &ast.WhileStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 18).Build(),
				Condition: []*ast.DeclOrExpr{
					{Expr: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 10, 3, 14).Build(),
						Kind:           ast.BOOLEAN, Value: "true"}},
				},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 16, 3, 18).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			while (true) {
				int i;
			}`,
			expected: &ast.WhileStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 5, 4).Build(),
				Condition: []*ast.DeclOrExpr{
					{Expr: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 10, 3, 14).Build(),
						Kind:           ast.BOOLEAN,
						Value:          "true",
					},
					},
				},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 16, 5, 4).Build(),
					Statements: []ast.Statement{
						&ast.DeclarationStmt{Decl: &ast.GenDecl{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 4, 4, 10).Build(),
							Token:          ast.VAR,
							Spec: &ast.ValueSpec{
								Names: []*ast.Ident{
									ast.NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
								},
								Type: ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().
									WithStartEnd(4, 4, 4, 7).
									WithNameStartEnd(4, 4, 4, 7).Build(),
							},
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.WhileStatement))
		})
	}
}

func TestConvertToAST_do_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.DoStatement
	}{
		{
			skip: false,
			input: `
			do {};`,
			expected: &ast.DoStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 9).Build(),
				Condition:      nil,
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 6, 3, 8).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			do {} while(true);`,
			expected: &ast.DoStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 21).Build(),
				Condition: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 15, 3, 19).Build(),
					Kind:           ast.BOOLEAN,
					Value:          "true"},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 6, 3, 8).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			skip: false,
			input: `
			do {
				int i;
			} while(true);`,
			expected: &ast.DoStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 5, 17).Build(),
				Condition: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(5, 11, 5, 15).Build(),
					Kind:           ast.BOOLEAN,
					Value:          "true"},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 6, 5, 4).Build(),
					Statements: []ast.Statement{
						&ast.DeclarationStmt{Decl: &ast.GenDecl{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 4, 4, 10).Build(),
							Token:          ast.VAR,
							Spec: &ast.ValueSpec{
								Names: []*ast.Ident{
									ast.NewIdentifierBuilder().WithName("i").WithStartEnd(4, 8, 4, 9).BuildPtr(),
								},
								Type: ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().
									WithStartEnd(4, 4, 4, 7).
									WithNameStartEnd(4, 4, 4, 7).Build(),
							},
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.DoStatement))
		})
	}
}

func TestConvertToAST_defer_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.DeferStatement
	}{
		{
			skip: false,
			input: `
			defer foo();`,
			expected: &ast.DeferStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 15).Build(),
				Statement: &ast.ExpressionStmt{Expr: &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 9, 3, 14).Build(),
					Identifier:     ast.NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 9, 3, 12).BuildPtr(),
					Arguments:      []ast.Expression{},
				},
				},
			},
		},
		{
			skip: false,
			input: `
			defer try foo();`,
			expected: &ast.DeferStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 19).Build(),
				Statement: &ast.ExpressionStmt{Expr: &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 13, 3, 18).Build(),
					Identifier:     ast.NewIdentifierBuilder().WithName("foo").WithStartEnd(3, 13, 3, 16).BuildPtr(),
					Arguments:      []ast.Expression{},
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.DeferStatement))
		})
	}
}

func TestConvertToAST_assert_stmt(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.AssertStatement
	}{
		{
			skip: false,
			input: `
			assert(true);`,
			expected: &ast.AssertStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 16).Build(),
				Assertions: []ast.Expression{
					&ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 10, 3, 14).Build(),
						Kind:           ast.BOOLEAN,
						Value:          "true"},
				},
			},
		},
		{
			skip: false,
			input: `
			assert(true,1);`,
			expected: &ast.AssertStatement{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 3, 3, 18).Build(),
				Assertions: []ast.Expression{
					&ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 10, 3, 14).Build(),
						Kind:           ast.BOOLEAN,
						Value:          "true",
					},
					&ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 15, 3, 16).Build(),
						Kind:           ast.INT,
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

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			funcDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
			assert.Equal(t, tt.expected, funcDecl.Body.(*ast.CompoundStmt).Statements[0].(*ast.AssertStatement))
		})
	}
}
