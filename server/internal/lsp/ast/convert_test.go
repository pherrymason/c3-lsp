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
		PENDING("pending start", false),
		RUNNING("running", true),
	}`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	// Test basic enum declaration
	row := uint(1)
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
	row = 2
	expected = EnumDecl{
		Name: "TypedColors",
		BaseType: TypeInfo{
			Name:     "int",
			BuiltIn:  true,
			Optional: false,
			ASTNodeBase: NewBaseNodeBuilder().
				WithStartEnd(row, 18, row, 21).
				Build(),
		},
		ASTNodeBase: NewBaseNodeBuilder().
			WithStartEnd(row, 1, row, 42).
			Build(),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name: "RED",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row, 24, row, 27).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(row, 24, row, 27).
					Build(),
			},
			{
				Name: Identifier{
					Name: "BLUE",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row, 29, row, 33).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(row, 29, row, 33).
					Build(),
			},
			{
				Name: Identifier{
					Name: "GREEN",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row, 35, row, 40).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(row, 35, row, 40).
					Build(),
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0].Declarations[1])

	// Test enum with associated parameters declaration
	row = 3
	expected = EnumDecl{
		Name: "State",
		BaseType: TypeInfo{
			Name:     "int",
			BuiltIn:  true,
			Optional: false,
			ASTNodeBase: NewBaseNodeBuilder().
				WithStartEnd(row, 14, row, 17).
				Build(),
		},
		Properties: []EnumProperty{
			{
				Name: Identifier{
					Name: "desc",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row, 26, row, 30).
						Build(),
				},
				Type: TypeInfo{
					Name:     "String",
					BuiltIn:  false,
					Optional: false,
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row, 19, row, 25).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(row, 19, row, 30).
					Build(),
			},
			{
				Name: Identifier{
					Name: "active",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row, 37, row, 43).
						Build(),
				},
				Type: TypeInfo{
					Name:     "bool",
					BuiltIn:  true,
					Optional: false,
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row, 32, row, 36).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(row, 32, row, 43).
					Build(),
			},
		},
		ASTNodeBase: NewBaseNodeBuilder().
			WithStartEnd(row, 1, row+3, 2).
			Build(),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name: "PENDING",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row+1, 2, row+1, 9).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(row+1, 2, row+1, 33).
					Build(),
			},
			{
				Name: Identifier{
					Name: "RUNNING",
					ASTNodeBase: NewBaseNodeBuilder().
						WithStartEnd(row+2, 2, row+2, 9).
						Build(),
				},
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(row+2, 2, row+2, 26).
					Build(),
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0].Declarations[2])
}
