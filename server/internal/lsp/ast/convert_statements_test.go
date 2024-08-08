package ast

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
 * @dataProvider test all kind of literals here!
 */
func TestConvertToAST_declaration_with_assignment(t *testing.T) {
	cases := []struct {
		literal  string
		expected ASTNode
	}{
		{
			literal:  "1",
			expected: Literal{Value: "1"},
		},
		{
			literal:  "1.1",
			expected: Literal{Value: "1.1"},
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
				Operator:   "&",
				Expression: NewIdentifierBuilder().WithName("anotherVariable").WithStartEnd(2, 14, 2, 29).Build(),
			},
		},

		// seq($.type, $.initializer_list),
		{
			literal: "Type{1,2}",
			expected: InlineTypeWithInitizlization{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 22).Build(),
				Type: NewTypeInfoBuilder().
					WithName("Type").
					WithNameStartEnd(2, 13, 2, 17).
					WithStartEnd(2, 13, 2, 17).
					Build(),
				InitializerList: InitializerList{
					ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 17, 2, 22).Build(),
					Args: []Expression{
						Literal{Value: "1"},
						Literal{Value: "2"},
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
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+18).Build(),
				Args: []Expression{
					ArgParamPathSet{
						Path: "[0]",
						Expr: Literal{Value: "1"},
					},
					ArgParamPathSet{
						Path: "[2]",
						Expr: Literal{Value: "2"},
					},
				},
			},
		},
		{
			literal: "{[0] = Type}",
			expected: InitializerList{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+12).Build(),
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
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 25).Build(),
				Args: []Expression{
					ArgParamPathSet{
						Path: "[0..2]",
						Expr: Literal{Value: "2"},
					},
				},
			},
		},
		{
			literal: "{.a = 1}",
			expected: InitializerList{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+8).Build(),
				Args: []Expression{
					ArgFieldSet{
						//ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13+2, 2, 13+3).Build(),
						FieldName: "a",
						Expr:      Literal{Value: "1"},
					},
				},
			},
		},
		{
			literal: "{Type}",
			expected: InitializerList{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 13+6).Build(),
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
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 25).Build(),
				Args: []Expression{
					Literal{Value: "$vasplat()"},
				},
			},
		},
		{
			literal: "{$vasplat(0..1)}",
			expected: InitializerList{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 29).Build(),
				Args: []Expression{
					Literal{Value: "$vasplat(0..1)"},
				},
			},
		},
		{
			literal: "{...id}",
			expected: InitializerList{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(2, 13, 2, 20).Build(),
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
		t.Run(fmt.Sprintf("assignment initializer list: %s", tt.literal), func(t *testing.T) {
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
	source := `
	module foo;
	fn void main() {
		call();
	}`

	ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call_with_arguments(t *testing.T) {
	source := `
	module foo;
	fn void main() {
		call2(cat);
	}`

	ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_function_statements_call_chain(t *testing.T) {
	source := `
	module foo;
	fn void main() {
		object.call(1).call2(3);
		}`

	ConvertToAST(GetCST(source), source, "file.c3")
}

func TestConvertToAST_compile_time_call(t *testing.T) {
	cases := []struct {
		skip     bool
		input    string
		expected Expression
	}{
		{
			skip:  true,
			input: "$alignof(Type)", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 24).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$alignof").WithStartEnd(1, 9, 1, 17).Build(),
				Arguments: []Arg{
					NewTypeInfoBuilder().
						WithName("Type").
						WithNameStartEnd(1, 9+9, 1, 9+13).
						WithStartEnd(1, 9+9, 1, 9+13).
						Build(),
				},
			},
		},
		{
			skip:  true,
			input: "$alignof(10)", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 21).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$alignof").WithStartEnd(1, 9, 1, 17).Build(),
				Arguments:   []Arg{Literal{Value: "10"}},
			},
		},
		{
			skip:  true,
			input: "$alignof(a[5])", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 23).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$alignof").WithStartEnd(1, 9, 1, 17).Build(),
				Arguments: []Arg{
					IndexAccess{
						Array: NewIdentifierBuilder().WithName("a").WithStartEnd(1, 18, 1, 19).Build(),
						Index: "[5]",
					},
				},
			},
		},
		{
			skip:  true,
			input: "$alignof(a[5..6])", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 26).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$alignof").WithStartEnd(1, 9, 1, 17).Build(),
				Arguments: []Arg{
					RangeAccess{
						Array:      NewIdentifierBuilder().WithName("a").WithStartEnd(1, 18, 1, 19).Build(),
						RangeStart: 5,
						RangeEnd:   6,
					},
				},
			},
		},
		// "$extnameof",
		{
			input: "$extnameof(Type)", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 25).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$extnameof").WithStartEnd(1, 9, 1, 19).Build(),
				Arguments: []Arg{
					NewTypeInfoBuilder().
						WithName("Type").
						WithNameStartEnd(1, 9+11, 1, 9+15).
						WithStartEnd(1, 9+11, 1, 9+15).
						Build(),
				},
			},
		},
		{
			input: "$extnameof(10)", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 23).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$extnameof").WithStartEnd(1, 9, 1, 19).Build(),
				Arguments:   []Arg{Literal{Value: "10"}},
			},
		},
		{
			input: "$extnameof(a[5])", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 25).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$extnameof").WithStartEnd(1, 9, 1, 19).Build(),
				Arguments: []Arg{
					IndexAccess{
						Array: NewIdentifierBuilder().WithName("a").WithStartEnd(1, 20, 1, 21).Build(),
						Index: "[5]",
					},
				},
			},
		},
		{
			input: "$extnameof(a[5..6])", // type
			expected: FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 28).Build(),
				Identifier:  NewIdentifierBuilder().WithName("$extnameof").WithStartEnd(1, 9, 1, 19).Build(),
				Arguments: []Arg{
					RangeAccess{
						Array:      NewIdentifierBuilder().WithName("a").WithStartEnd(1, 20, 1, 21).Build(),
						RangeStart: 5,
						RangeEnd:   6,
					},
				},
			},
		},
		/*
			"$extnameof",
			"$nameof",
			"$offsetof",
			"$qnameof",
		*/
	}

	for _, tt := range cases {
		if tt.skip {
			continue
		}

		t.Run(fmt.Sprintf("assignment initializer list: %s", tt.input), func(t *testing.T) {
			source := `module foo;
	int x = ` + tt.input + `;`
			ast := ConvertToAST(GetCST(source), source, "file.c3")
			varDecl := ast.Modules[0].Declarations[0].(VariableDecl)

			assert.Equal(t, tt.expected, varDecl.Initializer)
		})
	}
}
