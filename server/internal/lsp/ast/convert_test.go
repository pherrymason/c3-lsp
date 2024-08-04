package ast

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func aWithPos(startRow uint, startCol uint, endRow uint, endCol uint) ASTNodeBase {
	return NewBaseNodeBuilder().
		WithStartEnd(startRow, startCol, endRow, endCol).
		Build()
}

func TestConvertToAST_module(t *testing.T) {
	source := `module foo;`

	ast := ConvertToAST(GetCST(source), source)

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

	ast := ConvertToAST(GetCST(source), source)

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

	ast := ConvertToAST(GetCST(source), source)

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

	ast := ConvertToAST(GetCST(source), source)

	assert.Equal(t, []string{"foo", "foo2"}, ast.Modules[0].Imports)
}

func TestConvertToAST_global_variables(t *testing.T) {
	source := `module foo;
	int hello = 3;
	int dog, cat, elephant;`

	ast := ConvertToAST(GetCST(source), source)

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
			Identifier: NewIdentifierBuilder().WithName("int").WithStartEnd(1, 1, 1, 4).Build(),
			BuiltIn:    true,
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
			Identifier:  NewIdentifierBuilder().WithName("int").WithStartEnd(2, 1, 2, 4).Build(),
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

	ast := ConvertToAST(GetCST(source), source)

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
			Identifier:  NewIdentifierBuilder().WithName("int").WithStartEnd(2, 18, 2, 21).Build(),
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
	enum State : int (String desc, bool active, char ke) {
		PENDING = {"pending start", false, 'c'},
		RUNNING = {"running", true, 'e'},
	}`

	ast := ConvertToAST(GetCST(source), source)

	// Test enum with associated parameters declaration
	row := uint(1)
	expected := EnumDecl{
		Name: "State",
		BaseType: TypeInfo{
			Identifier:  NewIdentifierBuilder().WithName("int").WithStartEnd(1, 14, 1, 17).Build(),
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
					Identifier:  NewIdentifierBuilder().WithName("String").WithStartEnd(1, 19, 1, 25).Build(),
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
					Identifier:  NewIdentifierBuilder().WithName("bool").WithStartEnd(1, 32, 1, 36).Build(),
					BuiltIn:     true,
					Optional:    false,
					ASTNodeBase: aWithPos(row, 32, row, 36),
				},
				ASTNodeBase: aWithPos(row, 32, row, 43),
			},
			{
				Name: Identifier{
					Name:        "ke",
					ASTNodeBase: aWithPos(row, 50, row, 52),
				},
				Type: TypeInfo{
					Identifier:  NewIdentifierBuilder().WithName("char").WithStartEnd(1, 45, 1, 49).Build(),
					BuiltIn:     true,
					Optional:    false,
					ASTNodeBase: aWithPos(row, 45, row, 49),
				},
				ASTNodeBase: aWithPos(row, 45, row, 52),
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
						Literal{Value: "c"},
					},
				},
				ASTNodeBase: aWithPos(row+1, 2, row+1, 41),
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
						Literal{Value: "e"},
					},
				},
				ASTNodeBase: aWithPos(row+2, 2, row+2, 34),
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0].Declarations[0])
}

func TestConvertToAST_struct_decl(t *testing.T) {
	source := `module foo;
	struct MyStruct {
		int data;
		char key;
		raylib::Camera camera;
	}`

	ast := ConvertToAST(GetCST(source), source)

	expected := StructDecl{
		ASTNodeBase: aWithPos(1, 1, 5, 2),
		Name:        "MyStruct",
		StructType:  StructTypeNormal,
		Members: []StructMemberDecl{
			{
				ASTNodeBase: aWithPos(2, 2, 2, 11),
				Names: []Identifier{
					{
						ASTNodeBase: aWithPos(2, 6, 2, 10),
						Name:        "data",
					},
				},
				Type: TypeInfo{
					ASTNodeBase: aWithPos(2, 2, 2, 5),
					Identifier:  NewIdentifierBuilder().WithName("int").WithStartEnd(2, 2, 2, 5).Build(),
					BuiltIn:     true,
				},
			},
			{
				ASTNodeBase: aWithPos(3, 2, 3, 11),
				Names: []Identifier{
					{
						ASTNodeBase: aWithPos(3, 7, 3, 10),
						Name:        "key",
					},
				},
				Type: TypeInfo{
					ASTNodeBase: aWithPos(3, 2, 3, 6),
					Identifier:  NewIdentifierBuilder().WithName("char").WithStartEnd(3, 2, 3, 6).Build(),
					BuiltIn:     true,
				},
			},
			{
				ASTNodeBase: aWithPos(4, 2, 4, 24),
				Names: []Identifier{
					{
						ASTNodeBase: aWithPos(4, 17, 4, 23),
						Name:        "camera",
					},
				},
				Type: TypeInfo{
					ASTNodeBase: aWithPos(4, 2, 4, 16),
					Identifier: Identifier{
						Path:        "raylib",
						Name:        "Camera",
						ASTNodeBase: aWithPos(4, 2, 4, 16),
					},
					BuiltIn: false,
				},
			},
		},
	}

	assert.Equal(t, expected, ast.Modules[0].Declarations[0])
}

func TestConvertToAST_struct_decl_with_interface(t *testing.T) {
	source := `module foo;
	struct MyStruct (MyInterface, MySecondInterface) {
		int data;
		char key;
		raylib::Camera camera;
	}`

	ast := ConvertToAST(GetCST(source), source)

	expected := []string{"MyInterface", "MySecondInterface"}

	structDecl := ast.Modules[0].Declarations[0].(StructDecl)
	assert.Equal(t, expected, structDecl.Implements)
}

func TestConvertToAST_struct_decl_with_anonymous_bitstructs(t *testing.T) {
	source := `module x;
	def Register16 = UInt16;
	struct Registers {
		bitstruct : Register16 @overlap {
			Register16 bc : 0..15;
			Register b : 8..15;
			Register c : 0..7;
		}
		Register16 sp;
		Register16 pc;
	}`

	ast := ConvertToAST(GetCST(source), source)
	structDecl := ast.Modules[0].Declarations[0].(StructDecl)

	assert.Equal(t, 5, len(structDecl.Members))

	assert.Equal(
		t,
		StructMemberDecl{
			ASTNodeBase: aWithPos(4, 3, 4, 25),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("bc").
					WithStartEnd(4, 14, 4, 16).
					Build(),
			},
			Type: TypeInfo{
				ASTNodeBase: aWithPos(4, 3, 4, 13),
				Identifier: NewIdentifierBuilder().
					WithName("Register16").
					WithStartEnd(4, 3, 4, 13).
					Build(),
			},
			BitRange: option.Some([2]uint{0, 15}),
		},
		structDecl.Members[0],
	)

	assert.Equal(
		t,
		StructMemberDecl{
			ASTNodeBase: aWithPos(5, 3, 5, 22),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("b").
					WithStartEnd(5, 12, 5, 13).
					Build(),
			},
			Type: TypeInfo{
				ASTNodeBase: aWithPos(5, 3, 5, 11),
				Identifier: NewIdentifierBuilder().
					WithName("Register").
					WithStartEnd(5, 3, 5, 11).
					Build(),
			},
			BitRange: option.Some([2]uint{8, 15}),
		},
		structDecl.Members[1],
	)

	assert.Equal(
		t,
		StructMemberDecl{
			ASTNodeBase: aWithPos(6, 3, 6, 21),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("c").
					WithStartEnd(6, 12, 6, 13).
					Build(),
			},
			Type: TypeInfo{
				ASTNodeBase: aWithPos(6, 3, 6, 11),
				Identifier: NewIdentifierBuilder().
					WithName("Register").
					WithStartEnd(6, 3, 6, 11).
					Build(),
			},
			BitRange: option.Some([2]uint{0, 7}),
		},
		structDecl.Members[2],
	)

	assert.Equal(
		t,
		StructMemberDecl{
			ASTNodeBase: aWithPos(8, 2, 8, 16),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("sp").
					WithStartEnd(8, 13, 8, 15).
					Build(),
			},
			Type: TypeInfo{
				ASTNodeBase: aWithPos(8, 2, 8, 12),
				Identifier: NewIdentifierBuilder().
					WithName("Register16").
					WithStartEnd(8, 2, 8, 12).
					Build(),
			},
		},
		structDecl.Members[3],
	)

	assert.Equal(
		t,
		StructMemberDecl{
			ASTNodeBase: aWithPos(9, 2, 9, 16),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("pc").
					WithStartEnd(9, 13, 9, 15).
					Build(),
			},
			Type: TypeInfo{
				ASTNodeBase: aWithPos(9, 2, 9, 12),
				Identifier: NewIdentifierBuilder().
					WithName("Register16").
					WithStartEnd(9, 2, 9, 12).
					Build(),
			},
		},
		structDecl.Members[4],
	)
}

func TestConvertToAST_struct_decl_with_inline_substructs(t *testing.T) {
	source := `module x;
	struct Person {
		int age;
		String name;
	}
	struct ImportantPerson {
		inline Person person;
		String title;
	}`

	ast := ConvertToAST(GetCST(source), source)
	structDecl := ast.Modules[0].Declarations[1].(StructDecl)

	assert.Equal(t, true, structDecl.Members[0].IsInlined)
}

func TestConvertToAST_union_decl(t *testing.T) {
	source := `module foo;
	union MyStruct {
		char data;
		char key;
	}`

	ast := ConvertToAST(GetCST(source), source)
	unionDecl := ast.Modules[0].Declarations[0].(StructDecl)

	assert.Equal(t, StructTypeUnion, int(unionDecl.StructType))
}
