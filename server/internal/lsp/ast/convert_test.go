package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func aWithPos(startRow uint, startCol uint, endRow uint, endCol uint) ASTNodeBase {
	return NewBaseNodeBuilder().
		WithStartEnd(startRow, startCol, endRow, endCol).
		Build()
}

func TestConvertToAST_module(t *testing.T) {
	source := `module foo;`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	expectedAst := File{
		Modules: []Module{
			{
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
		ASTNodeBase: aWithPos(2, 1, 2, 24),
		Names: []Identifier{
			{
				Name:        "dog",
				ASTNodeBase: aWithPos(2, 5, 2, 8),
			},
			{
				Name:        "cat",
				ASTNodeBase: aWithPos(2, 10, 2, 13),
			},
			{
				Name:        "elephant",
				ASTNodeBase: aWithPos(2, 15, 2, 23),
			},
		},
		Type: TypeInfo{
			Name:        "int",
			BuiltIn:     true,
			ASTNodeBase: aWithPos(2, 1, 2, 4),
		},
	}
	assert.Equal(t, expectedAnimals, ast.Modules[0].Declarations[1])
}

func TestConvertToAST_enum_decl(t *testing.T) {
	source := `module foo;
	enum Colors { RED, BLUE, GREEN }
	enum TypedColors:int { RED, BLUE, GREEN } // Typed enums`
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
			Name:        "int",
			BuiltIn:     true,
			Optional:    false,
			ASTNodeBase: aWithPos(row, 18, row, 21),
		},
		ASTNodeBase: aWithPos(row, 1, row, 42),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name:        "RED",
					ASTNodeBase: aWithPos(row, 24, row, 27),
				},
				ASTNodeBase: aWithPos(row, 24, row, 27),
			},
			{
				Name: Identifier{
					Name:        "BLUE",
					ASTNodeBase: aWithPos(row, 29, row, 33),
				},
				ASTNodeBase: aWithPos(row, 29, row, 33),
			},
			{
				Name: Identifier{
					Name:        "GREEN",
					ASTNodeBase: aWithPos(row, 35, row, 40),
				},
				ASTNodeBase: aWithPos(row, 35, row, 40),
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0].Declarations[1])
}

func TestConvertToAST_enum_decl_with_associated_params(t *testing.T) {
	source := `module foo;
	enum State : int (String desc, bool active) {
		PENDING = {"pending start", false},
		RUNNING = {"running", true},
	}`
	cst := GetCST(source)

	ast := ConvertToAST(cst, source)

	// Test enum with associated parameters declaration
	row := uint(1)
	expected := EnumDecl{
		Name: "State",
		BaseType: TypeInfo{
			Name:        "int",
			BuiltIn:     true,
			Optional:    false,
			ASTNodeBase: aWithPos(row, 14, row, 17),
		},
		Properties: []EnumProperty{
			{
				Name: Identifier{
					Name:        "desc",
					ASTNodeBase: aWithPos(row, 26, row, 30),
				},
				Type: TypeInfo{
					Name:        "String",
					BuiltIn:     false,
					Optional:    false,
					ASTNodeBase: aWithPos(row, 19, row, 25),
				},
				ASTNodeBase: aWithPos(row, 19, row, 30),
			},
			{
				Name: Identifier{
					Name:        "active",
					ASTNodeBase: aWithPos(row, 37, row, 43),
				},
				Type: TypeInfo{
					Name:        "bool",
					BuiltIn:     true,
					Optional:    false,
					ASTNodeBase: aWithPos(row, 32, row, 36),
				},
				ASTNodeBase: aWithPos(row, 32, row, 43),
			},
		},
		ASTNodeBase: aWithPos(row, 1, row+3, 2),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name:        "PENDING",
					ASTNodeBase: aWithPos(row+1, 2, row+1, 9),
				},
				Value: CompositeLiteral{
					Values: []Expression{
						Literal{Value: "pending start"},
						BoolLiteral{Value: false},
					},
				},
				ASTNodeBase: aWithPos(row+1, 2, row+1, 36),
			},
			{
				Name: Identifier{
					Name:        "RUNNING",
					ASTNodeBase: aWithPos(row+2, 2, row+2, 9),
				},
				Value: CompositeLiteral{
					Values: []Expression{
						Literal{Value: "running"},
						BoolLiteral{Value: true},
					},
				},
				ASTNodeBase: aWithPos(row+2, 2, row+2, 29),
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0].Declarations[0])
}
