package ast

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

/*
 * @dataProvider test all kind of literals here!
 */
func TestConvertToAST_declaration_with_assignment(t *testing.T) {
	cases := []struct {
		skip     bool
		literal  string
		expected ASTNode
	}{
		{
			literal:  "1",
			expected: IntegerLiteral{Value: "1"},
		},
		{
			literal:  "1.1",
			expected: RealLiteral{Value: "1.1"},
		},
		{
			literal:  "false",
			expected: BoolLiteral{Value: false},
		},
		{
			literal:  "true",
			expected: BoolLiteral{Value: true},
		},
		{
			literal:  "null",
			expected: Literal{Value: "null"},
		},
		{
			literal:  "\"hello\"",
			expected: Literal{Value: "\"hello\""},
		},
		{
			literal:  "`hello`",
			expected: Literal{Value: "`hello`"},
		},
		{
			literal:  "x'FF'",
			expected: Literal{Value: "x'FF'"},
		},
		{
			literal:  "x\"FF\"",
			expected: Literal{Value: "x\"FF\""},
		},
		{
			literal:  "x`FF`",
			expected: Literal{Value: "x`FF`"},
		},
		{
			literal:  "b64'FF'",
			expected: Literal{Value: "b64'FF'"},
		},
		{
			literal:  "b64\"FF\"",
			expected: Literal{Value: "b64\"FF\""},
		},
		{
			literal:  "b64`FF`",
			expected: Literal{Value: "b64`FF`"},
		},
		{
			literal:  "$$builtin",
			expected: Literal{Value: "$$builtin"},
		},
		// _ident_expr
		// - const_ident
		{
			literal:  "A_CONSTANT",
			expected: NewIdentifierBuilder().WithName("A_CONSTANT").WithStartEnd(2, 13, 2, 23).Build(),
		},
		// - ident
		{
			literal:  "ident",
			expected: NewIdentifierBuilder().WithName("ident").WithStartEnd(2, 13, 2, 18).Build(),
		},
		// - at_ident
		{
			literal:  "@ident",
			expected: NewIdentifierBuilder().WithName("@ident").WithStartEnd(2, 13, 2, 19).Build(),
		},
		// module_ident_expr:
		{
			literal:  "path::ident",
			expected: NewIdentifierBuilder().WithPath("path").WithName("ident").WithStartEnd(2, 13, 2, 24).Build(),
		},
		{
			literal:  "$_abc",
			expected: NewIdentifierBuilder().WithName("$_abc").WithStartEnd(2, 13, 2, 18).Build(),
		},
		{
			literal:  "#_abc",
			expected: NewIdentifierBuilder().WithName("#_abc").WithStartEnd(2, 13, 2, 18).Build(),
		},
		{
			literal: "&anotherVariable",
			expected: UnaryExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 29).Build(),
				Operator:    "&",
				Argument:    NewIdentifierBuilder().WithName("anotherVariable").WithStartEnd(2, 14, 2, 29).Build(),
			},
		},

		// seq($.type, $.initializer_list),
		{
			literal: "Type{1,2}",
			expected: InlineTypeWithInitizlization{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 22).Build(),
				Type: NewTypeInfoBuilder().
					WithName("Type").
					WithNameStartEnd(2, 13, 2, 17).
					WithStartEnd(2, 13, 2, 17).
					Build(),
				InitializerList: InitializerList{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 17, 2, 22).Build(),
					Args: []Expression{
						IntegerLiteral{Value: "1"},
						IntegerLiteral{Value: "2"},
					},
				},
			},
		},

		// $vacount
		{
			literal:  "$vacount",
			expected: Literal{Value: "$vacount"},
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

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			varDecl := ast.Modules[0].Declarations[0].(VariableDecl)
			assert.Equal(t, tt.expected, varDecl.Initializer)
		})
	}
}

func TestConvertToAST_declaration_with_initializer_list_assingment(t *testing.T) {
	cases := []struct {
		literal  string
		expected ASTNode
	}{
		{
			literal: "{[0] = 1, [2] = 2}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+18).Build(),
				Args: []Expression{
					ArgParamPathSet{
						Path: "[0]",
						Expr: IntegerLiteral{Value: "1"},
					},
					ArgParamPathSet{
						Path: "[2]",
						Expr: IntegerLiteral{Value: "2"},
					},
				},
			},
		},
		{
			literal: "{[0] = Type}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+12).Build(),
				Args: []Expression{
					ArgParamPathSet{
						Path: "[0]",
						Expr: NewTypeInfoBuilder().
							WithStartEnd(2, 13+7, 2, 13+11).
							WithName("Type").
							WithNameStartEnd(2, 13+7, 2, 13+11).
							Build(),
					},
				},
			},
		},
		{
			literal: "{[0..2] = 2}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 25).Build(),
				Args: []Expression{
					ArgParamPathSet{
						Path: "[0..2]",
						Expr: IntegerLiteral{Value: "2"},
					},
				},
			},
		},
		{
			literal: "{.a = 1}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+8).Build(),
				Args: []Expression{
					ArgFieldSet{
						//ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13+2, 2, 13+3).Build(),
						FieldName: "a",
						Expr:      IntegerLiteral{Value: "1"},
					},
				},
			},
		},
		{
			literal: "{Type}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+6).Build(),
				Args: []Expression{
					NewTypeInfoBuilder().
						WithStartEnd(2, 13+1, 2, 13+5).
						WithName("Type").
						WithNameStartEnd(2, 13+1, 2, 13+5).
						Build(),
				},
			},
		},
		{
			literal: "{$vasplat()}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 25).Build(),
				Args: []Expression{
					Literal{Value: "$vasplat()"},
				},
			},
		},
		{
			literal: "{$vasplat(0..1)}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 29).Build(),
				Args: []Expression{
					Literal{Value: "$vasplat(0..1)"},
				},
			},
		},
		{
			literal: "{...id}",
			expected: InitializerList{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 20).Build(),
				Args: []Expression{
					NewIdentifierBuilder().
						WithName("id").
						WithStartEnd(2, 13+4, 2, 13+6).Build(),
				},
			},
		},
		/*{
			literal:  "{[0..2] = Type}",
			expected: Literal{Value: "{[0..2] = Type}"},
		},*/ /*
			{
				literal:  "{.a = Type}",
				expected: Literal{Value: "{.a = Type}"},
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
		t.Run(fmt.Sprintf("compile_time_call: %s", tt.literal), func(t *testing.T) {
			source := `
			module foo;
			int var = ` + tt.literal + ";"

			ast := ConvertToAST(GetCST(source), source, "file.c3")

			varDecl := ast.Modules[0].Declarations[0].(VariableDecl)
			assert.Equal(t, tt.expected, varDecl.Initializer)
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

	ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call(t *testing.T) {
	t.Skip("TODO")
	source := `
	module foo;
	fn void main() {
		call();
	}`

	ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call_with_arguments(t *testing.T) {
	t.Skip("TODO")
	source := `
	module foo;
	fn void main() {
		call2(cat);
	}`

	ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call_chain(t *testing.T) {
	t.Skip("TODO")
	source := `
	module foo;
	fn void main() {
		object.call(1).call2(3);
		}`

	ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_compile_time_call(t *testing.T) {

	cases := []struct {
		skip             bool
		input            string
		expected         FunctionCall
		functionCallName string
		ArgumentTypeName string
		Argument         Expression
	}{
		{
			input:            "$(Type{1,2})", // type
			functionCallName: "$",
			ArgumentTypeName: "TypeInfo",
			Argument: NewTypeInfoBuilder().
				WithName("Type").
				WithNameStartEnd(1, 10, 1, 10+4).
				WithStartEnd(1, 10, 1, 10+4).
				Build(),
		},
		{
			input:            "$(10)", // literal
			ArgumentTypeName: "IntegerLiteral",
			Argument:         IntegerLiteral{Value: "10"},
		},
		{
			input:            "$(a[5])",
			ArgumentTypeName: "IndexAccess",
			Argument: IndexAccess{
				Array: NewIdentifierBuilder().WithName("a").WithStartEnd(1, 10, 1, 11).Build(),
				Index: "[5]",
			},
		},
		{
			input:            "$(a[5..6])",
			ArgumentTypeName: "RangeAccess",
			Argument: RangeAccess{
				Array:      NewIdentifierBuilder().WithName("a").WithStartEnd(1, 10, 1, 11).Build(),
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
					ast := ConvertToAST(GetCST(source), source, "file.c3")
					varDecl := ast.Modules[0].Declarations[0].(VariableDecl)
					initializer := varDecl.Initializer.(FunctionCall)

					assert.Equal(t, method, initializer.Identifier.(Identifier).Name)

					arg := initializer.Arguments[0]
					switch arg.(type) {
					case Literal:
						if tt.ArgumentTypeName != "Literal" {
							t.Errorf("Expected argument must be Literal. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(Literal)
						assert.Equal(t, e.Value, arg.(Literal).Value)
					case IntegerLiteral:
						if tt.ArgumentTypeName != "IntegerLiteral" {
							t.Errorf("Expected argument must be IntegerLiteral. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(IntegerLiteral)
						assert.Equal(t, e.Value, arg.(IntegerLiteral).Value)

					case TypeInfo:
						if tt.ArgumentTypeName != "TypeInfo" {
							t.Errorf("Expected argument must be TypeInfo. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(TypeInfo)
						assert.Equal(t, e.Identifier.Name, arg.(TypeInfo).Identifier.Name)

					case IndexAccess:
						if tt.ArgumentTypeName != "IndexAccess" {
							t.Errorf("Expected argument must be IndexAccess. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(IndexAccess)
						assert.Equal(t, e.Array.(Identifier).Name, arg.(IndexAccess).Array.(Identifier).Name)

					case RangeAccess:
						if tt.ArgumentTypeName != "RangeAccess" {
							t.Errorf("Expected argument must be RangeAccess. It was %s", tt.ArgumentTypeName)
						}
						e := tt.Argument.(RangeAccess)
						assert.Equal(t, e.Array.(Identifier).Name, arg.(RangeAccess).Array.(Identifier).Name)
						assert.Equal(t, e.RangeStart, arg.(RangeAccess).RangeStart)
						assert.Equal(t, e.RangeEnd, arg.(RangeAccess).RangeEnd)

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

				ast := ConvertToAST(GetCST(source), source, "file.c3")
				varDecl := ast.Modules[0].Declarations[0].(VariableDecl)
				initializer := varDecl.Initializer.(FunctionCall)

				assert.Equal(t, FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 12, 1, 16+length).Build(),
					Identifier:  NewIdentifierBuilder().WithName(method).WithStartEnd(1, 12, 1, 12+length).Build(),
					Arguments: []Arg{
						NewIdentifierBuilder().WithName("id").WithStartEnd(1, 12+length+1, 1, 14+length+1).Build(),
					},
				}, initializer)
			})
	}
}

func TestConvertToAST_compile_time_analyse(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected FunctionCall
	}{
		{
			input: "$eval(id)",
			expected: FunctionCall{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 12, 1, 21).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$eval").WithStartEnd(1, 12, 1, 17).Build(),
				Arguments: []Arg{
					NewIdentifierBuilder().WithName("id").WithStartEnd(1, 18, 1, 20).Build(),
				},
			},
		},
		/*
			{
				skip:  false,
				input: "$and(id)",
				expected: FunctionCall{
					ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 12, 1, 19).Build(),
					Identifier:  NewIdentifierBuilder().WithName("$and").WithStartEnd(1, 12, 1, 16).Build(),
					Arguments: []Arg{
						NewIdentifierBuilder().WithName("id").WithStartEnd(1, 17, 1, 19).Build(),
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

				ast := ConvertToAST(GetCST(source), source, "file.c3")

				assert.Equal(t, tt.expected, ast.Modules[0].Declarations[0].(VariableDecl).Initializer.(FunctionCall))
			})
	}
}

func TestConvertToAST_lambda_declaration(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected LambdaDeclaration
	}{
		{
			input: "int i = fn int (int a, int b){};",
			expected: LambdaDeclaration{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 8, 1, 29).Build(),
				ReturnType: option.Some[TypeInfo](NewTypeInfoBuilder().
					WithName("int").
					IsBuiltin().
					WithNameStartEnd(1, 11, 1, 14).
					WithStartEnd(1, 11, 1, 14).
					Build()),
				Parameters: []FunctionParameter{
					{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 16, 1, 21).Build(),
						Name:        NewIdentifierBuilder().WithName("a").WithStartEnd(1, 20, 1, 21).Build(),
						Type: NewTypeInfoBuilder().
							WithName("int").
							IsBuiltin().
							WithNameStartEnd(1, 16, 1, 19).
							WithStartEnd(1, 16, 1, 19).
							Build(),
					},
					{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 23, 1, 28).Build(),
						Name:        NewIdentifierBuilder().WithName("b").WithStartEnd(1, 27, 1, 28).Build(),
						Type: NewTypeInfoBuilder().
							WithName("int").
							IsBuiltin().
							WithNameStartEnd(1, 23, 1, 26).
							WithStartEnd(1, 23, 1, 26).
							Build(),
					},
				},
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 29, 1, 31).Build(),
					Statements:  []Expression{},
				},
			},
		},
		{
			input: "int i = fn (){};",
			expected: LambdaDeclaration{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 8, 1, 13).Build(),
				Parameters:  []FunctionParameter{},
				ReturnType:  option.None[TypeInfo](),
				Body: CompoundStatement{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 13, 1, 15).Build(),
					Statements:  []Expression{},
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

				ast := ConvertToAST(GetCST(source), source, "file.c3")

				assert.Equal(t, tt.expected, ast.Modules[0].Declarations[0].(VariableDecl).Initializer.(LambdaDeclaration))
			})
	}
}

func TestConvertToAST_asignment_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected AssignmentStatement
	}{
		{
			input: "i = 10;",
			expected: AssignmentStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 19, 1, 19+6).Build(),
				Left:        NewIdentifierBuilder().WithName("i").WithStartEnd(1, 19, 1, 20).Build(),
				Right:       IntegerLiteral{Value: "10"},
				Operator:    "=",
			},
		},
		{
			input: "$CompileTimeType = Type;",
			expected: AssignmentStatement{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 19, 1, 19+23).Build(),
				Left:        Literal{Value: "$CompileTimeType"},
				Right: NewTypeInfoBuilder().
					WithName("Type").
					WithNameStartEnd(1, 38, 1, 42).
					WithStartEnd(1, 38, 1, 42).
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
			fmt.Sprintf("lambda_declaration: %s", tt.input),
			func(t *testing.T) {
				source := `module foo;
				fn void main(){` + tt.input + `};`

				ast := ConvertToAST(GetCST(source), source, "file.c3")

				cmp_stmts := ast.Modules[0].Functions[0].(FunctionDecl).Body.(CompoundStatement)
				assert.Equal(t, tt.expected, cmp_stmts.Statements[0].(AssignmentStatement))
			})
	}
}

func TestConvertToAST_ternary_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected TernaryExpression
	}{
		{
			input: "i > 10 ? a:b;",
			expected: TernaryExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 19, 1, 19+12).Build(),
				Condition: BinaryExpr{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 19, 1, 25).Build(),
					Left:        NewIdentifierBuilder().WithName("i").WithStartEnd(1, 19, 1, 20).Build(),
					Operator:    ">",
					Right:       IntegerLiteral{Value: "10"},
				},
				Consequence: NewIdentifierBuilder().WithName("a").WithStartEnd(1, 19+9, 1, 19+10).Build(),
				Alternative: NewIdentifierBuilder().WithName("b").WithStartEnd(1, 19+11, 1, 19+12).Build(),
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

				ast := ConvertToAST(GetCST(source), source, "file.c3")

				cmp_stmts := ast.Modules[0].Functions[0].(FunctionDecl).Body.(CompoundStatement)
				assert.Equal(t, tt.expected, cmp_stmts.Statements[0].(TernaryExpression))
			})
	}
}

func TestConvertToAST_lambda_expr(t *testing.T) {
	source := `module foo;
	int i = fn int () => 10;`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	expected := LambdaDeclaration{
		ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 24).Build(),
		ReturnType: option.Some(NewTypeInfoBuilder().
			WithStartEnd(1, 12, 1, 15).
			WithName("int").
			WithNameStartEnd(1, 12, 1, 15).
			IsBuiltin().
			Build(),
		),
		Parameters: []FunctionParameter{},
		Body: ReturnStatement{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 22, 1, 24).Build(),
			Return:      option.Some(Expression(IntegerLiteral{Value: "10"})),
		},
	}

	lambda := ast.Modules[0].Declarations[0].(VariableDecl).Initializer.(LambdaDeclaration)
	assert.Equal(t, expected, lambda)

}

func TestConvertToAST_elvis_or_else_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected TernaryExpression
	}{
		{
			source: `module foo;
			int i = condition ?: 10;`,
			expected: TernaryExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 26).Build(),
				Condition:   NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Consequence: NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Alternative: IntegerLiteral{Value: "10"},
			},
		},
		{
			source: `module foo;
			int i = condition ?? 10;`,
			expected: TernaryExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 26).Build(),
				Condition:   NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Consequence: NewIdentifierBuilder().WithName("condition").WithStartEnd(1, 11, 1, 20).Build(),
				Alternative: IntegerLiteral{Value: "10"},
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
				ast := ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				varDecl := ast.Modules[0].Declarations[0].(VariableDecl)
				assert.Equal(t, tt.expected, varDecl.Initializer.(TernaryExpression))
			})
	}
}

func TestConvertToAST_optional_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected OptionalExpression
	}{
		{
			source: `module foo;
			int b = a + b?;`,
			expected: OptionalExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 17).Build(),
				Operator:    "?",
				Argument: BinaryExpr{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 16).Build(),
					Left:        NewIdentifierBuilder().WithName("a").WithStartEnd(1, 11, 1, 12).Build(),
					Right:       NewIdentifierBuilder().WithName("b").WithStartEnd(1, 15, 1, 16).Build(),
					Operator:    "+",
				},
			},
		},
		{
			source: `module foo;
			int b = a + b?!;`,
			expected: OptionalExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 18).Build(),
				Operator:    "?!",
				Argument: BinaryExpr{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 16).Build(),
					Left:        NewIdentifierBuilder().WithName("a").WithStartEnd(1, 11, 1, 12).Build(),
					Right:       NewIdentifierBuilder().WithName("b").WithStartEnd(1, 15, 1, 16).Build(),
					Operator:    "+",
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
				ast := ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				varDecl := ast.Modules[0].Declarations[0].(VariableDecl)
				assert.Equal(t, tt.expected, varDecl.Initializer.(OptionalExpression))
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

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	stmt := ast.Modules[0].Functions[0].(FunctionDecl).Body.(CompoundStatement).Statements[0]

	assert.Equal(t,
		UnaryExpression{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 2, 2, 5).Build(),
			Operator:    "++",
			Argument:    NewIdentifierBuilder().WithName("b").WithStartEnd(2, 4, 2, 5).Build(),
		},
		stmt.(UnaryExpression),
	)
}

func TestConvertToAST_cast_expr(t *testing.T) {
	source := `module foo;
	fn void main() {
		(int)b;
	}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	stmt := ast.Modules[0].Functions[0].(FunctionDecl).Body.(CompoundStatement).Statements[0]

	assert.Equal(t,
		CastExpression{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 2, 2, 8).Build(),
			Type: NewTypeInfoBuilder().
				WithName("int").
				WithNameStartEnd(2, 3, 2, 6).
				WithStartEnd(2, 3, 2, 6).
				IsBuiltin().
				Build(),
			Argument: NewIdentifierBuilder().WithName("b").WithStartEnd(2, 7, 2, 8).Build(),
		},
		stmt.(CastExpression),
	)
}

func TestConvertToAST_rethrow_expr(t *testing.T) {
	//t.Skip("Pending until implement call_expr")
	cases := []struct {
		skip     bool
		source   string
		expected RethrowExpression
	}{
		{
			source: `module foo;
			int b = foo_may_error()!;`,
			expected: RethrowExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 27).Build(),
				Operator:    "!",
				Argument: FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 26).Build(),
					Identifier:  NewIdentifierBuilder().WithName("foo_may_error").WithStartEnd(1, 11, 1, 24).Build(),
					Arguments:   []Arg{},
				},
			},
		},
		{
			source: `module foo;
			int b = foo_may_error()!!;`,
			expected: RethrowExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 28).Build(),
				Operator:    "!!",
				Argument: FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 11, 1, 26).Build(),
					Identifier:  NewIdentifierBuilder().WithName("foo_may_error").WithStartEnd(1, 11, 1, 24).Build(),
					Arguments:   []Arg{},
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
				ast := ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				varDecl := ast.Modules[0].Declarations[0].(VariableDecl)
				assert.Equal(t, tt.expected, varDecl.Initializer.(RethrowExpression))
			})
	}
}

func TestConvertToAST_call_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected FunctionCall
	}{
		{
			source: `module foo;
			fn void main() {
				simple();
			}`,
			expected: FunctionCall{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 12).Build(),
				Identifier:  NewIdentifierBuilder().WithName("simple").WithStartEnd(2, 4, 2, 10).Build(),
				Arguments:   []Arg{},
			},
		},
		{
			source: `module foo;
			fn void main() {
				simple(a, b);
			}`,
			expected: FunctionCall{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 16).Build(),
				Identifier:  NewIdentifierBuilder().WithName("simple").WithStartEnd(2, 4, 2, 10).Build(),
				Arguments: []Arg{
					NewIdentifierBuilder().WithName("a").WithStartEnd(2, 11, 2, 12).Build(),
					NewIdentifierBuilder().WithName("b").WithStartEnd(2, 14, 2, 15).Build(),
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
			expected: FunctionCall{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 16).Build(),
				Identifier:  NewIdentifierBuilder().WithName("simple").WithStartEnd(2, 4, 2, 10).Build(),
				Arguments: []Arg{
					NewIdentifierBuilder().WithName("a").WithStartEnd(2, 11, 2, 12).Build(),
					NewIdentifierBuilder().WithName("b").WithStartEnd(2, 14, 2, 15).Build(),
				},
			},
		},
		{
			// Trailing blocks: https://c3-lang.org/references/docs/macros/#capturing-a-trailing-block
			skip: false,
			source: `module foo;
			fn void main() {
				$simple(a, b)
				{
				    a = 1;
				};
			}`,
			expected: FunctionCall{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 5, 5).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$simple").WithStartEnd(2, 4, 2, 11).Build(),
				Arguments: []Arg{
					NewIdentifierBuilder().WithName("a").WithStartEnd(2, 12, 2, 13).Build(),
					NewIdentifierBuilder().WithName("b").WithStartEnd(2, 15, 2, 16).Build(),
				},
				TrailingBlock: option.Some(
					CompoundStatement{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(3, 4, 5, 5).Build(),
						Statements: []Expression{
							AssignmentStatement{
								ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(4, 8, 4, 13).Build(),
								Left:        NewIdentifierBuilder().WithName("a").WithStartEnd(4, 8, 4, 9).Build(),
								Right:       IntegerLiteral{Value: "1"},
								Operator:    "=",
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
				ast := ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				mainFnc := ast.Modules[0].Functions[0].(FunctionDecl)
				assert.Equal(t, tt.expected, mainFnc.Body.(CompoundStatement).Statements[0].(FunctionCall))
			})
	}
}

func TestConvertToAST_trailing_generic_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected FunctionCall
	}{
		{
			source: `module foo;
			fn void main(){
				test(<int, double>)(1.0, &g);
			}`,
			expected: FunctionCall{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 32).Build(),
				Identifier: Identifier{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 8).Build(),
					Name:        "test",
					Path:        "",
				},
				GenericArguments: option.Some([]Expression{
					NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(2, 10, 2, 13).WithStartEnd(2, 10, 2, 13).Build(),
					NewTypeInfoBuilder().WithName("double").IsBuiltin().WithNameStartEnd(2, 15, 2, 21).WithStartEnd(2, 15, 2, 21).Build(),
				}),
				Arguments: []Arg{
					RealLiteral{Value: "1.0"},
					UnaryExpression{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 29, 2, 31).Build(),
						Operator:    "&",
						Argument:    NewIdentifierBuilder().WithName("g").WithStartEnd(2, 30, 2, 31).Build(),
					},
				},
			},
		},
		{
			source: `module foo;
			fn void main(){
				foo_test::test(<int, double>)(1.0, &g);
			}`,
			expected: FunctionCall{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 42).Build(),
				Identifier: NewIdentifierBuilder().
					WithName("test").
					WithPath("foo_test").
					WithStartEnd(2, 4, 2, 18).
					Build(),
				GenericArguments: option.Some([]Expression{
					NewTypeInfoBuilder().WithName("int").IsBuiltin().WithNameStartEnd(2, 20, 2, 23).WithStartEnd(2, 20, 2, 23).Build(),
					NewTypeInfoBuilder().WithName("double").IsBuiltin().WithNameStartEnd(2, 25, 2, 31).WithStartEnd(2, 25, 2, 31).Build(),
				}),
				Arguments: []Arg{
					RealLiteral{Value: "1.0"},
					UnaryExpression{
						ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 39, 2, 41).Build(),
						Operator:    "&",
						Argument:    NewIdentifierBuilder().WithName("g").WithStartEnd(2, 40, 2, 41).Build(),
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
				ast := ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				assert.Equal(
					t,
					tt.expected,
					ast.Modules[0].Functions[0].(FunctionDecl).Body.(CompoundStatement).Statements[0].(FunctionCall),
				)
			},
		)
	}
}

func TestConvertToAST_update_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected UpdateExpression
	}{
		{
			source: `module foo;
			fn void main(){
				a++;
			}`,
			expected: UpdateExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 7).Build(),
				Operator:    "++",
				Argument: Identifier{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 5).Build(),
					Name:        "a",
				},
			},
		},
		{
			source: `module foo;
			fn void main(){
				a--;
			}`,
			expected: UpdateExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 7).Build(),
				Operator:    "--",
				Argument: NewIdentifierBuilder().
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
				ast := ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				assert.Equal(
					t,
					tt.expected,
					ast.Modules[0].Functions[0].(FunctionDecl).Body.(CompoundStatement).Statements[0].(UpdateExpression),
				)
			},
		)
	}
}

func TestConvertToAST_subscript_expr(t *testing.T) {
	cases := []struct {
		skip     bool
		source   string
		expected SubscriptExpression
	}{
		{
			source: `module foo;
			fn void main(){
				a[0];
			}`,
			expected: SubscriptExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 8).Build(),
				Argument:    NewIdentifierBuilder().WithName("a").WithStartEnd(2, 4, 2, 5).Build(),
				Index:       IntegerLiteral{Value: "0"},
			},
		},
		{
			source: `module foo;
				fn void main(){
					a[0..2];
				}`,
			expected: SubscriptExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 5, 2, 12).Build(),
				Argument:    NewIdentifierBuilder().WithName("a").WithStartEnd(2, 5, 2, 6).Build(),
				Index: RangeIndex{
					//ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 6, 2, 9).Build(),
					Start: option.Some(uint(0)),
					End:   option.Some(uint(2)),
				},
			},
		},
		{
			source: `module foo;
			fn void main(){
				a[id];
			}`,
			expected: SubscriptExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 9).Build(),
				Argument:    NewIdentifierBuilder().WithName("a").WithStartEnd(2, 4, 2, 5).Build(),
				Index:       NewIdentifierBuilder().WithName("id").WithStartEnd(2, 6, 2, 8).Build(),
			},
		},
		{
			source: `module foo;
			fn void main(){
				a[call()];
			}`,
			expected: SubscriptExpression{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 4, 2, 13).Build(),
				Argument:    NewIdentifierBuilder().WithName("a").WithStartEnd(2, 4, 2, 5).Build(),
				Index: FunctionCall{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 6, 2, 12).Build(),
					Identifier:  NewIdentifierBuilder().WithName("call").WithStartEnd(2, 6, 2, 10).Build(),
					Arguments:   []Arg{},
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
				ast := ConvertToAST(GetCST(tt.source), tt.source, "file.c3")

				assert.Equal(
					t,
					tt.expected,
					ast.Modules[0].Functions[0].(FunctionDecl).Body.(CompoundStatement).Statements[0].(SubscriptExpression),
				)
			},
		)
	}
}