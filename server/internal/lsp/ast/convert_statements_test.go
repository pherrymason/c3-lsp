package ast

import (
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
			literal:  "\"hello\"",
			expected: Literal{Value: "\"hello\""},
		},
	}

	for _, tt := range cases {
		t.Run("assignment ast", func(t *testing.T) {
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
