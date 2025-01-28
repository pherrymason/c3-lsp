package factory

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func TestConvertToAST_convert_type(t *testing.T) {
	cases := []struct {
		skip     bool
		Type     string
		expected ast.Node
	}{
		{
			Type: "int",
			expected: ast.NewTypeInfoBuilder().
				WithStartEnd(0, 0, 0, 3).
				WithName("int").
				WithNameStartEnd(0, 0, 0, 3).
				IsBuiltin().
				Build(),
		},
		{
			Type: "Custom",
			expected: ast.NewTypeInfoBuilder().
				WithStartEnd(0, 0, 0, 6).
				WithName("Custom").
				WithNameStartEnd(0, 0, 0, 6).
				Build(),
		},
		{
			Type: "int!",
			expected: ast.NewTypeInfoBuilder().
				WithStartEnd(0, 0, 0, 4).
				WithName("int").
				WithNameStartEnd(0, 0, 0, 3).
				IsBuiltin().
				IsOptional().
				Build(),
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("convert_type: %s", tt.Type), func(t *testing.T) {
			source := `` + tt.Type + ` aVar = something;`

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)

			spec := decl.Spec.(*ast.ValueSpec)
			assert.Equal(t, tt.expected, spec.Type)
		})
	}
}

/*
 * @dataProvider test all kind of literals here!
 */
func TestConvertToAST_declaration_with_assignment(t *testing.T) {
	cases := []struct {
		skip     bool
		literal  string
		expected ast.Node
	}{
		{
			literal: "1",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().
					WithRangePositions(2, 13, 2, 14).
					Build(),
				Kind:  ast.INT,
				Value: "1",
			},
		},
		{
			literal: "1.1",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 16).Build(),
				Kind:           ast.FLOAT,
				Value:          "1.1"},
		},
		{
			literal: "false",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 18).Build(),
				Kind:           ast.BOOLEAN,
				Value:          "false",
			},
		},
		{
			literal: "true",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 17).Build(),
				Kind:           ast.BOOLEAN,
				Value:          "true"},
		},
		{
			literal: "null",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 17).Build(),
				Kind:           ast.NULL,
				Value:          "null"},
		},
		{
			literal: "\"hello\"",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 20).Build(),
				Kind:           ast.STRING, Value: "\"hello\""},
		},
		{
			literal: "`hello`",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 20).Build(),
				Kind:           ast.STRING, Value: "`hello`"},
		},
		{
			literal: "x'FF'",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 18).Build(),
				Kind:           ast.STRING, Value: "x'FF'"},
		},
		{
			literal: "x\"FF\"",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 18).Build(),
				Kind:           ast.STRING, Value: "x\"FF\""},
		},
		{
			literal: "x`FF`",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 18).Build(),
				Kind:           ast.STRING, Value: "x`FF`"},
		},
		{
			literal: "b64'FF'",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 20).Build(),
				Kind:           ast.STRING, Value: "b64'FF'"},
		},
		{
			literal: "b64\"FF\"",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 20).Build(),
				Kind:           ast.STRING, Value: "b64\"FF\""},
		},
		{
			literal: "b64`FF`",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 20).Build(),
				Kind:           ast.STRING, Value: "b64`FF`"},
		},
		{
			literal: "$$builtin",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 22).Build(),
				Kind:           ast.STRING,
				Value:          "$$builtin"},
		},
		// _ident_expr
		// - const_ident
		{
			literal:  "A_CONSTANT",
			expected: ast.NewIdentifierBuilder().WithName("A_CONSTANT").WithStartEnd(2, 13, 2, 23).Build(),
		},
		// - ident
		{
			literal:  "ident",
			expected: ast.NewIdentifierBuilder().WithName("ident").WithStartEnd(2, 13, 2, 18).Build(),
		},
		// - at_ident
		{
			literal:  "@ident",
			expected: ast.NewIdentifierBuilder().WithName("@ident").WithStartEnd(2, 13, 2, 19).Build(),
		},
		// module_ident_expr:
		{
			literal: "path::ident",
			expected: &ast.Ident{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 24).Build(),
				ModulePath: ast.NewIdentifierBuilder().
					WithName("path").
					WithStartEnd(2, 13, 2, 19).
					Build(),
				Name: "ident",
			},
		},
		{
			literal:  "$_abc",
			expected: ast.NewIdentifierBuilder().WithName("$_abc").IsCompileTime(true).WithStartEnd(2, 13, 2, 18).Build(),
		},
		{
			literal:  "#_abc",
			expected: ast.NewIdentifierBuilder().WithName("#_abc").WithStartEnd(2, 13, 2, 18).Build(),
		},
		{
			literal: "&anotherVariable",
			expected: &ast.UnaryExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 29).Build(),
				Operator:       "&",
				Argument:       ast.NewIdentifierBuilder().WithName("anotherVariable").WithStartEnd(2, 14, 2, 29).Build(),
			},
		},
		{
			literal: "Enum.MEMBER",
			expected: &ast.SelectorExpr{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 24).Build(),
				X: ast.NewTypeInfoBuilder().
					WithName("Enum").
					WithNameStartEnd(2, 13, 2, 17).
					WithStartEnd(2, 13, 2, 17).
					Build(),
				Sel: ast.NewIdentifierBuilder().WithName("MEMBER").WithStartEnd(2, 18, 2, 24).Build(),
			},
		},
		{
			literal: "((int)1.0)",
			expected: &ast.ParenExpr{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 23).Build(),
				X: &ast.CastExpression{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 14, 2, 22).Build(),
					Type: ast.NewTypeInfoBuilder().
						WithName("int").
						WithStartEnd(2, 15, 2, 18).
						WithNameStartEnd(2, 15, 2, 18).
						IsBuiltin().
						Build(),
					Argument: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 19, 2, 22).Build(),
						Kind:           ast.FLOAT,
						Value:          "1.0",
					},
				},
			},
		},

		// seq($.type, $.initializer_list),
		{
			literal: "TypeDescription{1,2}",
			expected: &ast.InlineTypeWithInitialization{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 33).Build(),
				Type: ast.NewTypeInfoBuilder().
					WithName("TypeDescription").
					WithNameStartEnd(2, 13, 2, 28).
					WithStartEnd(2, 13, 2, 28).
					Build(),
				InitializerList: &ast.InitializerList{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 28, 2, 33).Build(),
					Args: []ast.Expression{
						&ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 29, 2, 30).Build(),
							Kind:           ast.INT,
							Value:          "1"},

						&ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 31, 2, 32).Build(),
							Kind:           ast.INT,
							Value:          "2"},
					},
				},
			},
		},

		// $vacount
		{
			literal: "$vacount",
			expected: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 21).Build(),
				Kind:           ast.STRING,
				Value:          "$vacount"},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("assignment ast: %s", tt.literal), func(t *testing.T) {
			source := `
			module foo;
			int var = ` + tt.literal + ";"

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
			assert.Equal(t, ast.Token(ast.VAR), decl.Token)

			spec := decl.Spec.(*ast.ValueSpec)
			assert.Equal(t, tt.expected, spec.Value)
		})
	}
}

func TestConvertToAST_declaration_with_initializer_list_assignment(t *testing.T) {
	cases := []struct {
		incomplete bool
		name       string
		literal    string
		expected   ast.Node
	}{
		{
			name:    "testing arg: param_path = literal",
			literal: "{[0] = 1, [2] = 2}",
			expected: &ast.InitializerList{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 13+18).Build(),
				Args: []ast.Expression{
					&ast.ArgParamPathSet{
						Path: "[0]",
						Expr: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 20, 2, 21).Build(),
							Kind:           ast.INT,
							Value:          "1",
						},
					},
					&ast.ArgParamPathSet{
						Path: "[2]",
						Expr: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 29, 2, 30).Build(),
							Kind:           ast.INT,
							Value:          "2"},
					},
				},
			},
		},
		{
			name:    "testing arg: param_path [x..y] = literal",
			literal: "{[0..2] = 2}",
			expected: &ast.InitializerList{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 25).Build(),
				Args: []ast.Expression{
					&ast.ArgParamPathSet{
						Path: "[0..2]",
						Expr: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 23, 2, 24).Build(),
							Kind:           ast.INT,
							Value:          "2"},
					},
				},
			},
		},
		{
			// TODO: define what AST should be generated in ArgParamPathSet.Expr
			incomplete: true,
			name:       "testing arg: param_path = expr",
			literal:    "{[0] = TypeDescription{0}}",
			expected: &ast.InitializerList{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 36).Build(),
				Args: []ast.Expression{
					&ast.ArgParamPathSet{
						Path: "[0]",
						Expr: ast.NewTypeInfoBuilder().
							WithStartEnd(2, 13+7, 2, 35).
							WithName("TypeDescription").
							WithNameStartEnd(2, 13+7, 2, 35).
							Build(),
					},
				},
			},
		},
		{
			literal: "{.a = 1}",
			expected: &ast.InitializerList{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 13+8).Build(),
				Args: []ast.Expression{
					&ast.ArgFieldSet{
						FieldName: "a",
						Expr: &ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 19, 2, 20).Build(),
							Kind:           ast.INT,
							Value:          "1"},
					},
				},
			},
		},
		{
			// TODO: define what AST
			incomplete: true,
			literal:    "{TypeDescription}",
			expected: &ast.InitializerList{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 30).Build(),
				Args: []ast.Expression{
					ast.NewTypeInfoBuilder().
						WithStartEnd(2, 13+1, 2, 29).
						WithName("TypeDescription").
						WithNameStartEnd(2, 13+1, 2, 29).
						Build(),
				},
			},
		},
		{
			literal: "{$vasplat(0)}",
			expected: &ast.InitializerList{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 26).Build(),
				Args: []ast.Expression{
					&ast.FunctionCall{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 14, 2, 25).Build(),
						Identifier: ast.NewIdentifierBuilder().
							WithName("$vasplat").
							IsCompileTime(true).
							WithRange(lsp.NewRange(2, 14, 2, 22)).
							Build(),
						Arguments: []ast.Expression{&ast.BasicLit{
							NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 23, 2, 24).Build(),
							Kind:           ast.INT,
							Value:          "0"},
						},
					},
				},
			},
		},

		/* TODO move this test to call_arg testing
		{
			// No longer valid!
			literal: "{...id}",
			expected: &ast.InitializerList{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 20).Build(),
				Args: []ast.Expression{
					ast.NewIdentifierBuilder().
						WithName("id").
						WithStartEnd(2, 13+4, 2, 13+6).Build(),
				},
			},
		},
		*/
		/*{
			literal:  "{[0..2] = TypeDescription}",
			expected: Literal{Value: "{[0..2] = TypeDescription}"},
		},*/ /*
			{
				literal:  "{.a = TypeDescription}",
				expected: Literal{Value: "{.a = TypeDescription}"},
			},
			{
				literal:  "{$vasplat(0..2)}",
				expected: Literal{Value: "{$vasplat(0..2)}"},
			},
			{
				literal:  "{$vasplat()}",
				expected: Literal{Value: "{$vasplat()}"},
			},
			{
				literal:  "{...id}",
				expected: Literal{Value: "{...id}"},
			},*/
	}

	for _, tt := range cases {
		if tt.incomplete {
			continue
		}
		t.Run(fmt.Sprintf("compile_time_call: %s", tt.literal), func(t *testing.T) {
			source := `
			module foo;
			int var = ` + tt.literal + ";"

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

			decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
			spec := decl.Spec.(*ast.ValueSpec)
			assert.Equal(t, tt.expected, spec.Value)
		})
	}
}

func TestConvertToAST_function_statements_with_declarations(t *testing.T) {
	source := `
	module foo;
	fn void main() {
		int cat = 1;
		MyStruct object;
	}`

	cv := newTestAstConverter()
	cv.ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call(t *testing.T) {
	t.Skip("TODO")
	source := `
	module foo;
	fn void main() {
		call();
	}`

	cv := newTestAstConverter()
	cv.ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call_with_arguments(t *testing.T) {
	t.Skip("TODO")
	source := `
	module foo;
	fn void main() {
		call2(cat);
	}`

	cv := newTestAstConverter()
	cv.ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call_chain(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.FunctionCall
	}{
		{
			skip:  true,
			input: "object.call(1);",
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 2, 3, 16).Build(),
				Identifier: &ast.SelectorExpr{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 9, 3, 13).Build(),
					X:              ast.NewIdentifierBuilder().WithName("object").WithStartEnd(3, 2, 3, 8).Build(),
					Sel:            ast.NewIdentifierBuilder().WithName("call").WithStartEnd(3, 9, 3, 13).Build(),
				},
				Arguments: []ast.Expression{
					&ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 14, 3, 15).Build(),
						Kind:           ast.INT,
						Value:          "1"},
				},
			},
		},
		{
			skip:  true,
			input: "object.prop.call(1);",
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 2, 3, 21).Build(),
				Identifier: &ast.SelectorExpr{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 14, 3, 18).Build(),
					X: &ast.SelectorExpr{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 9, 3, 13).Build(),
						X:              ast.NewIdentifierBuilder().WithName("object").WithStartEnd(3, 2, 3, 8).Build(),
						Sel:            ast.NewIdentifierBuilder().WithName("prop").WithStartEnd(3, 9, 3, 13).Build(),
					},
					Sel: ast.NewIdentifierBuilder().WithName("call").WithStartEnd(3, 14, 3, 18).Build(),
				},
				Arguments: []ast.Expression{
					&ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 19, 3, 20).Build(),
						Kind:           ast.INT, Value: "1"},
				},
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("function_statements_call_chain[%d]", i), func(t *testing.T) {
			source := `
	module foo;
	fn void main() {
		` + tt.input + `
	}`

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
			compound := tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt)
			call := compound.Statements[0].(*ast.ExpressionStmt).Expr.(*ast.FunctionCall)

			assert.Equal(t, tt.expected, call)
		})
	}
}

func TestConvertToAST_field_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.SelectorExpr
	}{
		{
			skip:  false,
			input: "object.call().length;",
			expected: &ast.SelectorExpr{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 2, 3, 22).Build(),
				X: &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 2, 3, 15).Build(),
					Identifier: &ast.SelectorExpr{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 2, 3, 13).Build(),
						X:              ast.NewIdentifierBuilder().WithName("object").WithStartEnd(3, 2, 3, 8).Build(),
						Sel:            ast.NewIdentifierBuilder().WithName("call").WithStartEnd(3, 9, 3, 13).Build(),
					},
					Arguments:     []ast.Expression{},
					TrailingBlock: option.None[*ast.CompoundStmt](),
				},
				Sel: ast.NewIdentifierBuilder().
					WithStartEnd(3, 16, 3, 22).
					WithName("length").
					Build(),
			},
		},
	}
	for i, tt := range cases {
		if tt.skip {
			continue
		}
		t.Run(fmt.Sprintf("function_statements_call_chain[%d]", i), func(t *testing.T) {
			source := `
	module foo;
	fn void main() {
		` + tt.input + `
	}`

			cv := newTestAstConverter()
			tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
			compound := tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt)
			call := compound.Statements[0].(*ast.ExpressionStmt).Expr.(*ast.SelectorExpr)

			assert.Equal(t, tt.expected, call)
		})
	}
}

func TestConvertToAST_expression_block(t *testing.T) {
	source := `module foo;
	fn void test()
	{
		{|
			int a;
		|};
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	_, ok := tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt).Statements[0].(*ast.ExpressionStmt).Expr.(*ast.BlockExpr)
	assert.True(t, ok, "Not BlockExpr found")
}

func TestConvertToAST_compile_time_call(t *testing.T) {

	cases := []struct {
		skip             bool
		input            string
		expected         ast.FunctionCall
		functionCallName string
		ArgumentTypeName string
		Argument         ast.Expression
	}{
		{
			input:            "$(TypeDescription{1,2})", // type
			functionCallName: "$",
			ArgumentTypeName: "TypeInfo",
			Argument: ast.NewTypeInfoBuilder().
				WithName("TypeDescription").
				WithNameStartEnd(1, 10, 1, 10+4).
				WithStartEnd(1, 10, 1, 10+4).
				Build(),
		},
		{
			input:            "$(10)", // literal
			ArgumentTypeName: "BasicLit",
			Argument: &ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 13, 2, 14).Build(),
				Kind:           ast.INT,
				Value:          "10"},
		},
		{
			input:            "$(a[5])",
			ArgumentTypeName: "IndexAccessExpr",
			Argument: &ast.IndexAccessExpr{
				Array: ast.NewIdentifierBuilder().WithName("a").WithStartEnd(1, 10, 1, 11).Build(),
				Index: "[5]",
			},
		},
		{
			input:            "$(a[5..6])",
			ArgumentTypeName: "RangeAccessExpr",
			Argument: &ast.RangeAccessExpr{
				Array:      ast.NewIdentifierBuilder().WithName("a").WithStartEnd(1, 10, 1, 11).Build(),
				RangeStart: 5,
				RangeEnd:   6,
			},
		},
	}

	methods := []string{
		"$alignof",
		"$extnameof",
		"$nameof",
		"$offsetof",
		"$qnameof",
	}

	for _, method := range methods {
		//fmt.Printf("*********: %s\n", method)
		for _, tt := range cases {
			if tt.skip {
				continue
			}

			input := strings.Replace(tt.input, "$", method, 1)

			t.Run(
				fmt.Sprintf(
					"assignment initializer list: %s",
					input,
				), func(t *testing.T) {
					source := `module foo;
	int x = ` + input + `;`
					cv := newTestAstConverter()
					tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
					decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
					spec := decl.Spec.(*ast.ValueSpec)
					initializer := spec.Value.(*ast.FunctionCall)

					assert.Equal(t, method, initializer.Identifier.(*ast.Ident).Name)

					arg := initializer.Arguments[0]
					switch arg.(type) {
					case *ast.BasicLit:
						if tt.ArgumentTypeName != "BasicLit" {
							t.Errorf("Expected argument must be BasicLit. It was %s", tt.ArgumentTypeName)
							break
						}
						e := tt.Argument.(*ast.BasicLit)
						assert.Equal(t, e.Value, arg.(*ast.BasicLit).Value)

					case *ast.TypeInfo:
						if tt.ArgumentTypeName != "TypeInfo" {
							t.Errorf("Expected argument must be TypeInfo. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(*ast.TypeInfo)
						assert.Equal(t, e.Identifier.Name, arg.(*ast.TypeInfo).Identifier.Name)

					case *ast.IndexAccessExpr:
						if tt.ArgumentTypeName != "IndexAccessExpr" {
							t.Errorf("Expected argument must be IndexAccessExpr. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(*ast.IndexAccessExpr)
						assert.Equal(t, e.Array.(*ast.Ident).Name, arg.(*ast.IndexAccessExpr).Array.(*ast.Ident).Name)

					case *ast.RangeAccessExpr:
						if tt.ArgumentTypeName != "RangeAccessExpr" {
							t.Errorf("Expected argument must be RangeAccessExpr. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(*ast.RangeAccessExpr)
						assert.Equal(t, e.Array.(*ast.Ident).Name, arg.(*ast.RangeAccessExpr).Array.(*ast.Ident).Name)
						assert.Equal(t, e.RangeStart, arg.(*ast.RangeAccessExpr).RangeStart)
						assert.Equal(t, e.RangeEnd, arg.(*ast.RangeAccessExpr).RangeEnd)

					default:
						t.Errorf("Expected argument wrong type.")
					}
				})
		}
	}
}

func TestConvertToAST_compile_time_argument_call(t *testing.T) {

	methods := []string{
		"$vaconst",
		"$vaarg",
		"$varef",
		"$vaexpr",
	}

	for _, method := range methods {
		//fmt.Printf("*********: %s\n", method)

		t.Run(
			fmt.Sprintf("compile_time_argument_call: %s", method),
			func(t *testing.T) {
				length := uint(len(method))
				source := `module foo;
				int x = ` + method + `(id);`

				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
				decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
				spec := decl.Spec.(*ast.ValueSpec)
				initializer := spec.Value.(*ast.FunctionCall)

				assert.Equal(t, &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 12, 1, 16+length).Build(),
					Identifier:     ast.NewIdentifierBuilder().WithName(method).WithStartEnd(1, 12, 1, 12+length).Build(),
					Arguments: []ast.Expression{
						ast.NewIdentifierBuilder().WithName("id").WithStartEnd(1, 12+length+1, 1, 14+length+1).Build(),
					},
				}, initializer)
			})
	}
}

func TestConvertToAST_compile_time_analyse(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.FunctionCall
	}{
		{
			input: "$eval(id)",
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 12, 1, 21).Build(),
				Identifier:     ast.NewIdentifierBuilder().WithName("$eval").WithStartEnd(1, 12, 1, 17).Build(),
				Arguments: []ast.Expression{
					ast.NewIdentifierBuilder().WithName("id").WithStartEnd(1, 18, 1, 20).Build(),
				},
			},
		},
		/*
			{
				skip:  false,
				input: "$and(id)",
				expected: FunctionCall{
					ASTNodeBase: NewNodeAttributesBuilder().WithRangePositions(1, 12, 1, 19).Build(),
					Ident:  NewIdentifierBuilder().WithName("$and").WithRangePositions(1, 12, 1, 16).Build(),
					Arguments: []Arg{
						NewIdentifierBuilder().WithName("id").WithRangePositions(1, 17, 1, 19).Build(),
					},
				},
			},*/
	}
	/*
		methods := []string{
			"$eval",
			"$defined",
			"$sizeof",
			"$stringify",
			"$is_const",
		}*/

	for _, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("compile_time_analyse: %s", tt.input),
			func(t *testing.T) {
				source := `module foo;
				int x = ` + tt.input + `;`

				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

				decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
				spec := decl.Spec.(*ast.ValueSpec)
				assert.Equal(t, tt.expected, spec.Value.(*ast.FunctionCall))
			})
	}
}

func TestConvertToAST_lambda_declaration(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.LambdaDeclarationExpr
	}{
		{
			input: "int i = fn int (int a, int b){};",
			expected: &ast.LambdaDeclarationExpr{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 8, 1, 29).Build(),
				ReturnType: option.Some[*ast.TypeInfo](ast.NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithNameStartEnd(1, 11, 1, 14).
					WithStartEnd(1, 11, 1, 14).
					Build()),
				Parameters: []*ast.FunctionParameter{
					{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 16, 1, 21).Build(),
						Name:           ast.NewIdentifierBuilder().WithName("a").WithStartEnd(1, 20, 1, 21).Build(),
						Type: ast.NewTypeInfoBuilder().
							WithName("int").
							IsBuiltin().
							WithNameStartEnd(1, 16, 1, 19).
							WithStartEnd(1, 16, 1, 19).
							Build(),
					},
					{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 23, 1, 28).Build(),
						Name:           ast.NewIdentifierBuilder().WithName("b").WithStartEnd(1, 27, 1, 28).Build(),
						Type: ast.NewTypeInfoBuilder().
							WithName("int").
							IsBuiltin().
							WithNameStartEnd(1, 23, 1, 26).
							WithStartEnd(1, 23, 1, 26).
							Build(),
					},
				},
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 29, 1, 31).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
		{
			input: "int i = fn (){};",
			expected: &ast.LambdaDeclarationExpr{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 8, 1, 13).Build(),
				//Parameters:     []FunctionParameter{},
				ReturnType: option.None[*ast.TypeInfo](),
				Body: &ast.CompoundStmt{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 13, 1, 15).Build(),
					Statements:     []ast.Statement{},
				},
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("lambda_declaration: %s", tt.input),
			func(t *testing.T) {
				source := `module foo;
` + tt.input

				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

				decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
				spec := decl.Spec.(*ast.ValueSpec)
				assert.Equal(t, tt.expected, spec.Value.(*ast.LambdaDeclarationExpr))
			})
	}
}

func TestConvertToAST_assignment_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.AssignmentExpression
	}{
		{
			input: "i = 10;",
			expected: &ast.AssignmentExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 19, 1, 19+6).Build(),
				Left:           ast.NewIdentifierBuilder().WithName("i").WithStartEnd(1, 19, 1, 20).Build(),
				Right: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 23, 1, 25).Build(),
					Kind:           ast.INT,
					Value:          "10",
				},
				Operator: "=",
			},
		},
		{
			input: "$CompileTimeType = TypeDescription;",
			expected: &ast.AssignmentExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 19, 1, 53).Build(),
				Left: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 19, 1, 35).Build(),
					Kind:           ast.STRING,
					Value:          "$CompileTimeType"},
				Right: ast.NewTypeInfoBuilder().
					WithName("TypeDescription").
					WithNameStartEnd(1, 38, 1, 53).
					WithStartEnd(1, 38, 1, 53).
					Build(),
				Operator: "=",
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("assignment_expr: %s", tt.input),
			func(t *testing.T) {
				source := `module foo;
				fn void main(){` + tt.input + `}`

				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

				cmp_stmts := tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt)
				assert.Equal(t, tt.expected, cmp_stmts.Statements[0].(*ast.ExpressionStmt).Expr.(*ast.AssignmentExpression))
			})
	}
}

func TestConvertToAST_ternary_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected *ast.TernaryExpression
	}{
		{
			input: "i > 10 ? a:b;",
			expected: &ast.TernaryExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 19, 1, 19+12).Build(),
				Condition: &ast.BinaryExpression{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 19, 1, 25).Build(),
					Left:           ast.NewIdentifierBuilder().WithName("i").WithStartEnd(1, 19, 1, 20).Build(),
					Operator:       ">",
					Right: &ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 23, 1, 25).Build(),
						Kind:           ast.INT,
						Value:          "10"},
				},
				Consequence: ast.NewIdentifierBuilder().WithName("a").WithStartEnd(1, 19+9, 1, 19+10).Build(),
				Alternative: ast.NewIdentifierBuilder().WithName("b").WithStartEnd(1, 19+11, 1, 19+12).Build(),
			},
		},
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("lambda_declaration: %s", tt.input),
			func(t *testing.T) {
				source := `module foo;
				fn void main(){` + tt.input + `};`

				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

				cmp_stmts := tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt)
				assert.Equal(t, tt.expected, cmp_stmts.Statements[0].(*ast.ExpressionStmt).Expr) //TernaryExpression
			})
	}
}

func TestConvertToAST_lambda_expr(t *testing.T) {
	source := `module foo;
	int i = fn int () => 10;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	expected := &ast.LambdaDeclarationExpr{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 9, 1, 24).Build(),
		ReturnType: option.Some(ast.NewTypeInfoBuilder().
			WithStartEnd(1, 12, 1, 15).
			WithName("int").
			WithNameStartEnd(1, 12, 1, 15).
			IsBuiltin().
			Build(),
		),
		//Parameters: []FunctionParameter{},
		Body: &ast.ReturnStatement{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 22, 1, 24).Build(),
			Return: option.Some[ast.Expression](&ast.BasicLit{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 22, 1, 24).Build(),
				Kind:           ast.INT,
				Value:          "10",
			}),
		},
	}

	decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
	spec := decl.Spec.(*ast.ValueSpec)
	lambda := spec.Value.(*ast.LambdaDeclarationExpr)
	assert.Equal(t, expected, lambda)

}

func TestConvertToAST_elvis_or_else_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected *ast.TernaryExpression
	}{
		{
			source: `module foo;
			int i = condition ?: 10;`,
			expected: &ast.TernaryExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 26).Build(),
				Condition:      ast.NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Consequence:    ast.NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Alternative: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 24, 1, 26).Build(),
					Kind:           ast.INT,
					Value:          "10",
				},
			},
		},
		{
			source: `module foo;
			int i = condition ?? 10;`,
			expected: &ast.TernaryExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 26).Build(),
				Condition:      ast.NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Consequence:    ast.NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Alternative: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 24, 1, 26).Build(),
					Kind:           ast.INT,
					Value:          "10"},
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("elivs_or_else_expr: %d", i),
			func(t *testing.T) {
				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
				spec := decl.Spec.(*ast.ValueSpec)
				assert.Equal(t, tt.expected, spec.Value.(*ast.TernaryExpression))
			})
	}
}

func TestConvertToAST_optional_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected *ast.OptionalExpression
	}{
		{
			source: `module foo;
			int b = a + b?;`,
			expected: &ast.OptionalExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 17).Build(),
				Operator:       "?",
				Argument: &ast.BinaryExpression{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 16).Build(),
					Left:           ast.NewIdentifierBuilder().WithName("a").WithStartEnd(1, 11, 1, 12).Build(),
					Right:          ast.NewIdentifierBuilder().WithName("b").WithStartEnd(1, 15, 1, 16).Build(),
					Operator:       "+",
				},
			},
		},
		{
			source: `module foo;
			int b = a + b?!;`,
			expected: &ast.OptionalExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 18).Build(),
				Operator:       "?!",
				Argument: &ast.BinaryExpression{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 16).Build(),
					Left:           ast.NewIdentifierBuilder().WithName("a").WithStartEnd(1, 11, 1, 12).Build(),
					Right:          ast.NewIdentifierBuilder().WithName("b").WithStartEnd(1, 15, 1, 16).Build(),
					Operator:       "+",
				},
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("optional_expr: %d", i),
			func(t *testing.T) {
				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
				spec := decl.Spec.(*ast.ValueSpec)
				assert.Equal(t, tt.expected, spec.Value.(*ast.OptionalExpression))
			})
	}
}

func TestConvertToAST_binary_expr(t *testing.T) {
	t.Skip("TODO")
}

func TestConvertToAST_unary_expr(t *testing.T) {
	source := `module foo;
	fn void main() {
		++b;
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	stmt := tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt).Statements[0]

	assert.Equal(t,
		&ast.UnaryExpression{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 2, 2, 5).Build(),
			Operator:       "++",
			Argument:       ast.NewIdentifierBuilder().WithName("b").WithStartEnd(2, 4, 2, 5).Build(),
		},
		stmt.(*ast.ExpressionStmt).Expr,
	)
}

func TestConvertToAST_cast_expr(t *testing.T) {
	source := `module foo;
	fn void main() {
		(int)b;
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	stmt := tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt).Statements[0]

	assert.Equal(t,
		&ast.CastExpression{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 2, 2, 8).Build(),
			Type: ast.NewTypeInfoBuilder().
				WithName("int").
				WithNameStartEnd(2, 3, 2, 6).
				WithStartEnd(2, 3, 2, 6).
				IsBuiltin().
				Build(),
			Argument: ast.NewIdentifierBuilder().WithName("b").WithStartEnd(2, 7, 2, 8).Build(),
		},
		stmt.(*ast.ExpressionStmt).Expr, //CastExpression),
	)
}

func TestConvertToAST_rethrow_expr(t *testing.T) {
	//t.Skip("Pending until implement call_expr")
	cases := []struct {
		skip     bool
		source   string
		expected *ast.RethrowExpression
	}{
		{
			source: `module foo;
			int b = foo_may_error()!;`,
			expected: &ast.RethrowExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 27).Build(),
				Operator:       "!",
				Argument: &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 26).Build(),
					Identifier:     ast.NewIdentifierBuilder().WithName("foo_may_error").WithStartEnd(1, 11, 1, 24).Build(),
					Arguments:      []ast.Expression{},
				},
			},
		},
		{
			source: `module foo;
			int b = foo_may_error()!!;`,
			expected: &ast.RethrowExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 28).Build(),
				Operator:       "!!",
				Argument: &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 11, 1, 26).Build(),
					Identifier:     ast.NewIdentifierBuilder().WithName("foo_may_error").WithStartEnd(1, 11, 1, 24).Build(),
					Arguments:      []ast.Expression{},
				},
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("rethrow_expr: %d", i),
			func(t *testing.T) {
				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
				spec := decl.Spec.(*ast.ValueSpec)
				assert.Equal(t, tt.expected, spec.Value.(*ast.RethrowExpression))
			})
	}
}

func TestConvertToAST_call_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected *ast.FunctionCall
	}{
		{
			skip: false,
			source: `module foo;
			fn void main() {
				simple();
			}`,
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 12).Build(),
				Identifier:     ast.NewIdentifierBuilder().WithName("simple").WithStartEnd(2, 4, 2, 10).Build(),
				Arguments:      []ast.Expression{},
			},
		},
		{
			skip: false,
			source: `module foo;
			fn void main() {
				simple(a, b);
			}`,
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 16).Build(),
				Identifier:     ast.NewIdentifierBuilder().WithName("simple").WithStartEnd(2, 4, 2, 10).Build(),
				Arguments: []ast.Expression{
					ast.NewIdentifierBuilder().WithName("a").WithStartEnd(2, 11, 2, 12).Build(),
					ast.NewIdentifierBuilder().WithName("b").WithStartEnd(2, 14, 2, 15).Build(),
				},
			},
		},
		{
			// TODO implement attributes after argument list
			skip: true,
			source: `module foo;
			fn void main() {
				simple(a, b) @attributes;
			}`,
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 28).Build(),
				Identifier:     ast.NewIdentifierBuilder().WithName("simple").WithStartEnd(2, 4, 2, 10).Build(),
				Arguments: []ast.Expression{
					ast.NewIdentifierBuilder().WithName("a").WithStartEnd(2, 11, 2, 12).Build(),
					ast.NewIdentifierBuilder().WithName("b").WithStartEnd(2, 14, 2, 15).Build(),
				},
			},
		},
		{
			// Trailing blocks: https://c3-lang.org/references/docs/macros/#capturing-a-trailing-block
			skip: true,
			source: `module foo;
			fn void main() {
				$simple(a, b)
				{
				    a = 1;
				};
			}`,
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 5, 5).Build(),
				Identifier:     ast.NewIdentifierBuilder().WithName("$simple").WithStartEnd(2, 4, 2, 11).Build(),
				Arguments: []ast.Expression{
					ast.NewIdentifierBuilder().WithName("a").WithStartEnd(2, 12, 2, 13).Build(),
					ast.NewIdentifierBuilder().WithName("b").WithStartEnd(2, 15, 2, 16).Build(),
				},
				TrailingBlock: option.Some(
					&ast.CompoundStmt{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(3, 4, 5, 5).Build(),
						Statements: []ast.Statement{
							&ast.ExpressionStmt{
								Expr: &ast.AssignmentExpression{
									NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 8, 4, 13).Build(),
									Left:           ast.NewIdentifierBuilder().WithName("a").WithStartEnd(4, 8, 4, 9).Build(),
									Right: &ast.BasicLit{
										NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 12, 4, 13).Build(),
										Kind:           ast.INT,
										Value:          "1",
									},
									Operator: "=",
								},
							},
						},
					},
				),
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("call_expr: %d", i),
			func(t *testing.T) {
				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				mainFnc := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
				assert.Equal(t, tt.expected, mainFnc.Body.(*ast.CompoundStmt).Statements[0].(*ast.ExpressionStmt).Expr) //FunctionCall))
			})
	}
}

func TestConvertToAST_trailing_generic_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected *ast.FunctionCall
	}{
		{
			source: `module foo;
			fn void main(){
				test(<int, double>)(1.0, &g);
			}`,
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 32).Build(),
				Identifier: ast.NewIdentifierBuilder().
					WithStartEnd(2, 4, 2, 8).
					WithName("test").
					Build(),
				GenericArguments: option.Some([]ast.Expression{
					ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(2, 10, 2, 13).WithStartEnd(2, 10, 2, 13).Build(),
					ast.NewTypeInfoBuilder().WithName("double").IsBuiltin().WithNameStartEnd(2, 15, 2, 21).WithStartEnd(2, 15, 2, 21).Build(),
				}),
				Arguments: []ast.Expression{
					&ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 24, 2, 27).Build(),
						Kind:           ast.FLOAT,
						Value:          "1.0"},
					&ast.UnaryExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 29, 2, 31).Build(),
						Operator:       "&",
						Argument:       ast.NewIdentifierBuilder().WithName("g").WithStartEnd(2, 30, 2, 31).Build(),
					},
				},
			},
		},
		{
			source: `module foo;
			fn void main(){
				foo_test::test(<int, double>)(1.0, &g);
			}`,
			expected: &ast.FunctionCall{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 42).Build(),
				Identifier: &ast.Ident{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 18).Build(),
					ModulePath: ast.NewIdentifierBuilder().
						WithName("foo_test").
						WithStartEnd(2, 4, 2, 14).
						Build(),
					Name: "test",
				},
				GenericArguments: option.Some([]ast.Expression{
					ast.NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(2, 20, 2, 23).WithStartEnd(2, 20, 2, 23).Build(),
					ast.NewTypeInfoBuilder().WithName("double").IsBuiltin().WithNameStartEnd(2, 25, 2, 31).WithStartEnd(2, 25, 2, 31).Build(),
				}),
				Arguments: []ast.Expression{
					&ast.BasicLit{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 34, 2, 37).Build(),
						Kind:           ast.FLOAT,
						Value:          "1.0"},
					&ast.UnaryExpression{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 39, 2, 41).Build(),
						Operator:       "&",
						Argument:       ast.NewIdentifierBuilder().WithName("g").WithStartEnd(2, 40, 2, 41).Build(),
					},
				},
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("trailing_generic_expr: %d", i),
			func(t *testing.T) {
				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				assert.Equal(
					t,
					tt.expected,
					tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt).Statements[0].(*ast.ExpressionStmt).Expr, //FunctionCall),
				)
			},
		)
	}
}

func TestConvertToAST_update_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected *ast.UpdateExpression
	}{
		{
			source: `module foo;
			fn void main(){
				a++;
			}`,
			expected: &ast.UpdateExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 7).Build(),
				Operator:       "++",
				Argument: &ast.Ident{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 5).Build(),
					Name:           "a",
				},
			},
		},
		{
			source: `module foo;
			fn void main(){
				a--;
			}`,
			expected: &ast.UpdateExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 7).Build(),
				Operator:       "--",
				Argument: ast.NewIdentifierBuilder().
					WithName("a").
					WithStartEnd(2, 4, 2, 5).
					Build(),
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("update_expr: %d", i),
			func(t *testing.T) {
				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				assert.Equal(
					t,
					tt.expected,
					tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt).Statements[0].(*ast.ExpressionStmt).Expr, //(UpdateExpression),
				)
			},
		)
	}
}

func TestConvertToAST_subscript_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected *ast.SubscriptExpression
	}{
		{
			source: `module foo;
			fn void main(){
				a[0];
			}`,
			expected: &ast.SubscriptExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 8).Build(),
				Argument:       ast.NewIdentifierBuilder().WithName("a").WithStartEnd(2, 4, 2, 5).Build(),
				Index: &ast.BasicLit{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 6, 2, 7).Build(),
					Kind:           ast.INT,
					Value:          "0"},
			},
		},
		{
			source: `module foo;
				fn void main(){
					a[0..2];
				}`,
			expected: &ast.SubscriptExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 5, 2, 12).Build(),
				Argument:       ast.NewIdentifierBuilder().WithName("a").WithStartEnd(2, 5, 2, 6).Build(),
				Index: &ast.RangeIndexExpr{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 7, 2, 11).Build(),
					Start:          option.Some(uint(0)),
					End:            option.Some(uint(2)),
				},
			},
		},
		{
			source: `module foo;
			fn void main(){
				a[id];
			}`,
			expected: &ast.SubscriptExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 9).Build(),
				Argument:       ast.NewIdentifierBuilder().WithName("a").WithStartEnd(2, 4, 2, 5).Build(),
				Index:          ast.NewIdentifierBuilder().WithName("id").WithStartEnd(2, 6, 2, 8).Build(),
			},
		},
		{
			source: `module foo;
			fn void main(){
				a[call()];
			}`,
			expected: &ast.SubscriptExpression{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 4, 2, 13).Build(),
				Argument:       ast.NewIdentifierBuilder().WithName("a").WithStartEnd(2, 4, 2, 5).Build(),
				Index: &ast.FunctionCall{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(2, 6, 2, 12).Build(),
					Identifier:     ast.NewIdentifierBuilder().WithName("call").WithStartEnd(2, 6, 2, 10).Build(),
					Arguments:      []ast.Expression{},
				},
			},
		},
	}

	for i, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(
			fmt.Sprintf("subscript_expr: %d", i),
			func(t *testing.T) {
				cv := newTestAstConverter()
				tree := cv.ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				assert.Equal(
					t,
					tt.expected,
					tree.Modules[0].Declarations[0].(*ast.FunctionDecl).Body.(*ast.CompoundStmt).Statements[0].(*ast.ExpressionStmt).Expr, //SubscriptExpression),
				)
			},
		)
	}
}
