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

func TestConvertToAST_enum_decl(t *testing.T) {
	source := `module foo;
	enum Colors { RED, BLUE, GREEN }
	enum TypedColors:int { RED, BLUE, GREEN } // Typed enums
	enum State : int (String desc, bool active) {
		PEDNING("pending start", false),
		RUNNING("running", true),
		TERMINATED("ended", false)
	}`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	// Test basic enum declaration
	expected := EnumDecl{
		Name: "Colors",
		ASTNodeBase: NewBaseNodeBuilder().
			WithStartEnd(1, 1, 1, 33).
			Build(),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name: "RED",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(1, 15, 1, 18).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(1, 15, 1, 18).
					Build(),
			},
			{
				Name: Identifier{
					Name: "BLUE",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(1, 20, 1, 24).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(1, 20, 1, 24).
					Build(),
			},
			{
				Name: Identifier{
					Name: "GREEN",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(1, 26, 1, 31).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(1, 26, 1, 31).
					Build(),
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0].Declarations[0])

	// Test typed enum declaration
	expected = EnumDecl{
		Name: "TypedColors",
		BaseType: TypeInfo{
			Name:     "int",
			BuiltIn:  true,
			Optional: false,
			ASTNodeBase: NewBaseNodeBuilder().
				WithStartEnd(2, 18, 2, 21).
				Build(),
		},
		ASTNodeBase: NewBaseNodeBuilder().
			WithStartEnd(2, 1, 2, 42).
			Build(),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name: "RED",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(2, 24, 2, 27).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(2, 24, 2, 27).
					Build(),
			},
			{
				Name: Identifier{
					Name: "BLUE",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(2, 29, 2, 33).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(2, 29, 2, 33).
					Build(),
			},
			{
				Name: Identifier{
					Name: "GREEN",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(2, 35, 2, 40).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(2, 35, 2, 40).
					Build(),
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0].Declarations[1])
}
