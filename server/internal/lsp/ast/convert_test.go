package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToAST_module(t *testing.T) {
	source := `module foo;`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	expectedAst := File{
		Modules: []Module{
			Module{
				Name: "foo",
				ASTNodeBase: ASTNodeBase{
					Attributes: nil,
					StartPos:   Position{0, 0},
					EndPos:     Position{0, 11},
				},
			},
		},
		ASTNodeBase: ASTNodeBase{
			Attributes: nil,
			StartPos:   Position{0, 0},
			EndPos:     Position{0, 11},
		},
	}

	assert.Equal(t, expectedAst, ast)
}

func TestConvertToAST_module_with_generics(t *testing.T) {
	source := `module foo(<Type>);`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	expectedAst := File{
		Modules: []Module{
			Module{
				Name:              "foo",
				GenericParameters: []string{"Type"},
				ASTNodeBase: ASTNodeBase{
					Attributes: nil,
					StartPos:   Position{0, 0},
					EndPos:     Position{0, 19},
				},
			},
		},
		ASTNodeBase: ASTNodeBase{
			Attributes: nil,
			StartPos:   Position{0, 0},
			EndPos:     Position{0, 19},
		},
	}

	assert.Equal(t, expectedAst, ast)
}

func TestConvertToAST_module_with_attributes(t *testing.T) {
	source := `module foo @private;`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	expectedAst := File{
		Modules: []Module{
			Module{
				Name:              "foo",
				GenericParameters: nil,
				ASTNodeBase: ASTNodeBase{
					Attributes: []string{"@private"},
					StartPos:   Position{0, 0},
					EndPos:     Position{0, 20},
				},
			},
		},
		ASTNodeBase: ASTNodeBase{
			Attributes: nil,
			StartPos:   Position{0, 0},
			EndPos:     Position{0, 20},
		},
	}

	assert.Equal(t, expectedAst, ast)
}

func TestConvertToAST_module_with_imports(t *testing.T) {
	source := `module foo;
	import foo;
	import foo2;`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	assert.Equal(t, []string{"foo", "foo2"}, ast.Modules[0].Imports)
}

func TestConvertToAST_global_variables(t *testing.T) {
	source := `module foo;
	int hello = 3;
	int dog, cat, elephant;`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	expectedHello := VariableDecl{
		ASTNodeBase: NewBaseNodeBuilder().
			WithStartEnd(1, 1, 1, 15).
			Build(),
		Names: []Identifier{
			{
				Name: "hello",
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(1, 5, 1, 10).
					Build(),
			},
		},
		Type: TypeInfo{
			Name:    "int",
			BuiltIn: true,
			ASTNodeBase: NewBaseNodeBuilder().
				WithStartEnd(1, 1, 1, 4).
				Build(),
		},
	}
	assert.Equal(t, expectedHello, ast.Modules[0].Declarations[0])

	expectedAnimals := VariableDecl{
		ASTNodeBase: NewBaseNodeBuilder().
			WithStartEnd(2, 1, 2, 24).
			Build(),
		Names: []Identifier{
			{
				Name: "dog",
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(2, 5, 2, 8).
					Build(),
			},
			{
				Name: "cat",
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(2, 10, 2, 13).
					Build(),
			},
			{
				Name: "elephant",
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(2, 15, 2, 23).
					Build(),
			},
		},
		Type: TypeInfo{
			Name:    "int",
			BuiltIn: true,
			ASTNodeBase: NewBaseNodeBuilder().
				WithStartEnd(2, 1, 2, 4).
				Build(),
		},
	}
	assert.Equal(t, expectedAnimals, ast.Modules[0].Declarations[1])
}
