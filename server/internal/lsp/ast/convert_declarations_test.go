package ast

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func aWithPos(startRow uint, startCol uint, endRow uint, endCol uint) ASTBaseNode {
	return NewBaseNodeBuilder().
		WithStartEnd(startRow, startCol, endRow, endCol).
		Build()
}

func TestConvertToAST_module(t *testing.T) {
	source := `module foo;`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	expectedAst := File{
		ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(0, 0, 0, 11).Build(),
		Name:        "file.c3",
		Modules: []Module{
			{
				Name:        "foo",
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(0, 0, 0, 11).Build(),
			},
		},
	}

	assert.Equal(t, expectedAst, ast)
}

func TestConvertToAST_module_implicit(t *testing.T) {
	source := `
	int variable = 0;
	module foo;`

	ast := ConvertToAST(GetCST(source), source, "path/file/xxx.c3")

	expected := Module{
		Name:        "path_file_xxx",
		ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 1, 2, 1).Build(),
		Declarations: []Declaration{
			VariableDecl{
				ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 1, 1, 18).Build(),
				Names: []Identifier{
					NewIdentifierBuilder().WithName("variable").WithStartEnd(1, 5, 1, 13).Build(),
				},
				Type: NewTypeInfoBuilder().IsBuiltin().
					WithName("int").
					WithNameStartEnd(1, 1, 1, 4).
					WithStartEnd(1, 1, 1, 4).Build(),
				Initializer: IntegerLiteral{Value: "0"},
			},
		},
	}
	assert.Equal(t, expected, ast.Modules[0])

	expected = Module{
		Name:        "foo",
		ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 1, 2, 12).Build(),
	}
	assert.Equal(t, expected, ast.Modules[1])
}

func TestConvertToAST_module_with_generics(t *testing.T) {
	source := `module foo(<Type>);`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	expectedAst := File{
		Modules: []Module{
			{
				Name:              "foo",
				GenericParameters: []string{"Type"},
				ASTBaseNode: ASTBaseNode{
					Attributes: nil,
					StartPos:   Position{0, 0},
					EndPos:     Position{0, 19},
				},
			},
		},
		ASTBaseNode: ASTBaseNode{
			Attributes: nil,
			StartPos:   Position{0, 0},
			EndPos:     Position{0, 19},
		},
		Name: "file.c3",
	}

	assert.Equal(t, expectedAst, ast)
}

func TestConvertToAST_module_with_attributes(t *testing.T) {
	source := `module foo @private;`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	expectedAst := File{
		Modules: []Module{
			{
				Name:              "foo",
				GenericParameters: nil,
				ASTBaseNode: ASTBaseNode{
					Attributes: []string{"@private"},
					StartPos:   Position{0, 0},
					EndPos:     Position{0, 20},
				},
			},
		},
		ASTBaseNode: ASTBaseNode{
			Attributes: nil,
			StartPos:   Position{0, 0},
			EndPos:     Position{0, 20},
		},
		Name: "file.c3",
	}

	assert.Equal(t, expectedAst, ast)
}

func TestConvertToAST_module_with_imports(t *testing.T) {
	source := `module foo;
	import foo;
	import foo2::subfoo;`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	assert.Equal(t, []Import{
		{Path: "foo"},
		{Path: "foo2::subfoo"},
	}, ast.Modules[0].Imports)
}

func TestConvertToAST_global_variables(t *testing.T) {
	source := `module foo;
	int hello = 3;
	int dog, cat, elephant;`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	expectedHello := VariableDecl{
		ASTBaseNode: NewBaseNodeBuilder().
			WithStartEnd(1, 1, 1, 15).
			Build(),
		Names: []Identifier{
			{
				Name: "hello",
				ASTBaseNode: NewBaseNodeBuilder().
					WithStartEnd(1, 5, 1, 10).
					Build(),
			},
		},
		Type: TypeInfo{
			Identifier: NewIdentifierBuilder().WithName("int").WithStartEnd(1, 1, 1, 4).Build(),
			BuiltIn:    true,
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(1, 1, 1, 4).
				Build(),
		},
		Initializer: IntegerLiteral{Value: "3"},
	}
	assert.Equal(t, expectedHello, ast.Modules[0].Declarations[0])

	expectedAnimals := VariableDecl{
		ASTBaseNode: aWithPos(2, 1, 2, 24),
		Names: []Identifier{
			{
				Name:        "dog",
				ASTBaseNode: aWithPos(2, 5, 2, 8),
			},
			{
				Name:        "cat",
				ASTBaseNode: aWithPos(2, 10, 2, 13),
			},
			{
				Name:        "elephant",
				ASTBaseNode: aWithPos(2, 15, 2, 23),
			},
		},
		Type: TypeInfo{
			Identifier:  NewIdentifierBuilder().WithName("int").WithStartEnd(2, 1, 2, 4).Build(),
			BuiltIn:     true,
			ASTBaseNode: aWithPos(2, 1, 2, 4),
		},
	}
	assert.Equal(t, expectedAnimals, ast.Modules[0].Declarations[1])
}

func TestConvertToAST_enum_decl(t *testing.T) {
	source := `module foo;
	enum Colors { RED, BLUE, GREEN }
	enum TypedColors:int { RED, BLUE, GREEN } // Typed enums`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	// Test basic enum declaration
	row := uint(1)
	expected := EnumDecl{
		Name: "Colors",
		ASTBaseNode: NewBaseNodeBuilder().
			WithStartEnd(1, 1, 1, 33).
			Build(),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name: "RED",
					ASTBaseNode: NewBaseNodeBuilder().
						WithStartEnd(1, 15, 1, 18).
						Build(),
				},
				ASTBaseNode: NewBaseNodeBuilder().
					WithStartEnd(1, 15, 1, 18).
					Build(),
			},
			{
				Name: Identifier{
					Name: "BLUE",
					ASTBaseNode: NewBaseNodeBuilder().
						WithStartEnd(1, 20, 1, 24).
						Build(),
				},
				ASTBaseNode: NewBaseNodeBuilder().
					WithStartEnd(1, 20, 1, 24).
					Build(),
			},
			{
				Name: Identifier{
					Name: "GREEN",
					ASTBaseNode: NewBaseNodeBuilder().
						WithStartEnd(1, 26, 1, 31).
						Build(),
				},
				ASTBaseNode: NewBaseNodeBuilder().
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
			ASTBaseNode: aWithPos(row, 18, row, 21),
		},
		ASTBaseNode: aWithPos(row, 1, row, 42),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name:        "RED",
					ASTBaseNode: aWithPos(row, 24, row, 27),
				},
				ASTBaseNode: aWithPos(row, 24, row, 27),
			},
			{
				Name: Identifier{
					Name:        "BLUE",
					ASTBaseNode: aWithPos(row, 29, row, 33),
				},
				ASTBaseNode: aWithPos(row, 29, row, 33),
			},
			{
				Name: Identifier{
					Name:        "GREEN",
					ASTBaseNode: aWithPos(row, 35, row, 40),
				},
				ASTBaseNode: aWithPos(row, 35, row, 40),
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

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	// Test enum with associated parameters declaration
	row := uint(1)
	expected := EnumDecl{
		Name: "State",
		BaseType: TypeInfo{
			Identifier:  NewIdentifierBuilder().WithName("int").WithStartEnd(1, 14, 1, 17).Build(),
			BuiltIn:     true,
			Optional:    false,
			ASTBaseNode: aWithPos(row, 14, row, 17),
		},
		Properties: []EnumProperty{
			{
				Name: Identifier{
					Name:        "desc",
					ASTBaseNode: aWithPos(row, 26, row, 30),
				},
				Type: TypeInfo{
					Identifier:  NewIdentifierBuilder().WithName("String").WithStartEnd(1, 19, 1, 25).Build(),
					BuiltIn:     false,
					Optional:    false,
					ASTBaseNode: aWithPos(row, 19, row, 25),
				},
				ASTBaseNode: aWithPos(row, 19, row, 30),
			},
			{
				Name: Identifier{
					Name:        "active",
					ASTBaseNode: aWithPos(row, 37, row, 43),
				},
				Type: TypeInfo{
					Identifier:  NewIdentifierBuilder().WithName("bool").WithStartEnd(1, 32, 1, 36).Build(),
					BuiltIn:     true,
					Optional:    false,
					ASTBaseNode: aWithPos(row, 32, row, 36),
				},
				ASTBaseNode: aWithPos(row, 32, row, 43),
			},
			{
				Name: Identifier{
					Name:        "ke",
					ASTBaseNode: aWithPos(row, 50, row, 52),
				},
				Type: TypeInfo{
					Identifier:  NewIdentifierBuilder().WithName("char").WithStartEnd(1, 45, 1, 49).Build(),
					BuiltIn:     true,
					Optional:    false,
					ASTBaseNode: aWithPos(row, 45, row, 49),
				},
				ASTBaseNode: aWithPos(row, 45, row, 52),
			},
		},
		ASTBaseNode: aWithPos(row, 1, row+3, 2),
		Members: []EnumMember{
			{
				Name: Identifier{
					Name:        "PENDING",
					ASTBaseNode: aWithPos(row+1, 2, row+1, 9),
				},
				Value: CompositeLiteral{
					Values: []Expression{
						Literal{Value: "\"pending start\""},
						BoolLiteral{Value: false},
						Literal{Value: "'c'"},
					},
				},
				ASTBaseNode: aWithPos(row+1, 2, row+1, 41),
			},
			{
				Name: Identifier{
					Name:        "RUNNING",
					ASTBaseNode: aWithPos(row+2, 2, row+2, 9),
				},
				Value: CompositeLiteral{
					Values: []Expression{
						Literal{Value: "\"running\""},
						BoolLiteral{Value: true},
						Literal{Value: "'e'"},
					},
				},
				ASTBaseNode: aWithPos(row+2, 2, row+2, 34),
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

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	expected := StructDecl{
		ASTBaseNode: aWithPos(1, 1, 5, 2),
		Name:        "MyStruct",
		StructType:  StructTypeNormal,
		Members: []StructMemberDecl{
			{
				ASTBaseNode: aWithPos(2, 2, 2, 11),
				Names: []Identifier{
					{
						ASTBaseNode: aWithPos(2, 6, 2, 10),
						Name:        "data",
					},
				},
				Type: TypeInfo{
					ASTBaseNode: aWithPos(2, 2, 2, 5),
					Identifier:  NewIdentifierBuilder().WithName("int").WithStartEnd(2, 2, 2, 5).Build(),
					BuiltIn:     true,
				},
			},
			{
				ASTBaseNode: aWithPos(3, 2, 3, 11),
				Names: []Identifier{
					{
						ASTBaseNode: aWithPos(3, 7, 3, 10),
						Name:        "key",
					},
				},
				Type: TypeInfo{
					ASTBaseNode: aWithPos(3, 2, 3, 6),
					Identifier:  NewIdentifierBuilder().WithName("char").WithStartEnd(3, 2, 3, 6).Build(),
					BuiltIn:     true,
				},
			},
			{
				ASTBaseNode: aWithPos(4, 2, 4, 24),
				Names: []Identifier{
					{
						ASTBaseNode: aWithPos(4, 17, 4, 23),
						Name:        "camera",
					},
				},
				Type: TypeInfo{
					ASTBaseNode: aWithPos(4, 2, 4, 16),
					Identifier: Identifier{
						Path:        "raylib",
						Name:        "Camera",
						ASTBaseNode: aWithPos(4, 2, 4, 16),
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

	ast := ConvertToAST(GetCST(source), source, "file.c3")

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

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	structDecl := ast.Modules[0].Declarations[1].(StructDecl)

	assert.Equal(t, 5, len(structDecl.Members))

	assert.Equal(
		t,
		StructMemberDecl{
			ASTBaseNode: aWithPos(4, 3, 4, 25),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("bc").
					WithStartEnd(4, 14, 4, 16).
					Build(),
			},
			Type: TypeInfo{
				ASTBaseNode: aWithPos(4, 3, 4, 13),
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
			ASTBaseNode: aWithPos(5, 3, 5, 22),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("b").
					WithStartEnd(5, 12, 5, 13).
					Build(),
			},
			Type: TypeInfo{
				ASTBaseNode: aWithPos(5, 3, 5, 11),
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
			ASTBaseNode: aWithPos(6, 3, 6, 21),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("c").
					WithStartEnd(6, 12, 6, 13).
					Build(),
			},
			Type: TypeInfo{
				ASTBaseNode: aWithPos(6, 3, 6, 11),
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
			ASTBaseNode: aWithPos(8, 2, 8, 16),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("sp").
					WithStartEnd(8, 13, 8, 15).
					Build(),
			},
			Type: TypeInfo{
				ASTBaseNode: aWithPos(8, 2, 8, 12),
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
			ASTBaseNode: aWithPos(9, 2, 9, 16),
			Names: []Identifier{
				NewIdentifierBuilder().
					WithName("pc").
					WithStartEnd(9, 13, 9, 15).
					Build(),
			},
			Type: TypeInfo{
				ASTBaseNode: aWithPos(9, 2, 9, 12),
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

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	structDecl := ast.Modules[0].Declarations[1].(StructDecl)

	assert.Equal(t, true, structDecl.Members[0].IsInlined)
}

func TestConvertToAST_union_decl(t *testing.T) {
	source := `module foo;
	union MyStruct {
		char data;
		char key;
	}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	unionDecl := ast.Modules[0].Declarations[0].(StructDecl)

	assert.Equal(t, StructTypeUnion, int(unionDecl.StructType))
}

func TestConvertToAST_bitstruct_decl(t *testing.T) {
	source := `module x;
	bitstruct Test (AnInterface) : uint
	{
		ushort a : 0..15;
		ushort b : 16..31;
		bool c : 7;
	}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	bitstructDecl := ast.Modules[0].Declarations[0].(StructDecl)

	assert.Equal(t, StructTypeBitStruct, int(bitstructDecl.StructType))
	assert.Equal(t, true, bitstructDecl.BackingType.IsSome())

	expectedType := TypeInfo{
		ASTBaseNode: aWithPos(1, 32, 1, 36),
		BuiltIn:     true,
		Identifier: NewIdentifierBuilder().
			WithName("uint").
			WithStartEnd(1, 32, 1, 36).
			Build(),
	}
	assert.Equal(t, expectedType, bitstructDecl.BackingType.Get())
	assert.Equal(t, []string{"AnInterface"}, bitstructDecl.Implements)

	expect := StructMemberDecl{
		ASTBaseNode: aWithPos(3, 2, 3, 19),
		Names: []Identifier{
			NewIdentifierBuilder().
				WithName("a").
				WithStartEnd(3, 9, 3, 10).
				Build(),
		},
		Type: TypeInfo{
			ASTBaseNode: aWithPos(3, 2, 3, 8),
			BuiltIn:     true,
			Identifier: NewIdentifierBuilder().
				WithName("ushort").
				WithStartEnd(3, 2, 3, 8).
				Build(),
		},
		BitRange: option.Some([2]uint{0, 15}),
	}
	assert.Equal(t, expect, bitstructDecl.Members[0])
}

func TestConvertToAST_fault_decl(t *testing.T) {
	source := `module x;
	fault IOResult
	{
	  IO_ERROR,
	  PARSE_ERROR
	};`

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	faultDecl := ast.Modules[0].Declarations[0].(FaultDecl)

	assert.Equal(
		t,
		NewIdentifierBuilder().
			WithName("IOResult").
			WithStartEnd(1, 7, 1, 15).
			Build(),
		faultDecl.Name,
	)
	assert.Equal(t, Position{1, 1}, faultDecl.ASTBaseNode.StartPos)
	assert.Equal(t, Position{5, 2}, faultDecl.ASTBaseNode.EndPos)

	assert.Equal(t, false, faultDecl.BackingType.IsSome())
	assert.Equal(t, 2, len(faultDecl.Members))

	assert.Equal(t,
		FaultMember{
			Name: NewIdentifierBuilder().
				WithName("IO_ERROR").
				WithStartEnd(3, 3, 3, 11).
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(3, 3, 3, 11).
				Build(),
		},
		faultDecl.Members[0],
	)

	assert.Equal(t,
		FaultMember{
			Name: NewIdentifierBuilder().
				WithName("PARSE_ERROR").
				WithStartEnd(4, 3, 4, 14).
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(4, 3, 4, 14).
				Build(),
		},
		faultDecl.Members[1],
	)
}

func TestConvertToAST_def_decl(t *testing.T) {
	source := `module foo;
	def Kilo = int;
	def KiloPtr = Kilo*;
	def MyFunction = fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*); // TODO
	def MyMap = HashMap(<String, Feature>);
	def Camera = raylib::Camera;`
	ast := ConvertToAST(GetCST(source), source, "file.c3")
	row := uint(0)

	assert.Equal(t,
		DefDecl{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 1, 1, 16).Build(),
			Name:        NewIdentifierBuilder().WithName("Kilo").WithStartEnd(1, 5, 1, 9).Build(),
			resolvesToType: option.Some(
				NewTypeInfoBuilder().
					WithName("int").WithNameStartEnd(1, 12, 1, 15).
					IsBuiltin().
					WithStartEnd(1, 12, 1, 15).
					Build(),
			),
		}, ast.Modules[0].Declarations[0])

	assert.Equal(t,
		DefDecl{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(2, 1, 2, 21).Build(),
			Name:        NewIdentifierBuilder().WithName("KiloPtr").WithStartEnd(2, 5, 2, 12).Build(),
			resolvesToType: option.Some(
				NewTypeInfoBuilder().
					WithName("Kilo").WithNameStartEnd(2, 15, 2, 19).
					IsPointer().
					WithStartEnd(2, 15, 2, 19).
					Build(),
			),
		}, ast.Modules[0].Declarations[1])

	// Def with generics
	row = 4
	assert.Equal(t,
		DefDecl{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(row, 1, row, 40).Build(),
			Name:        NewIdentifierBuilder().WithName("MyMap").WithStartEnd(row, 5, row, 10).Build(),
			resolvesToType: option.Some(
				NewTypeInfoBuilder().
					WithName("HashMap").WithNameStartEnd(row, 13, row, 20).
					WithStartEnd(row, 13, row, 39).
					WithGeneric("String", row, 22, row, 28).
					WithGeneric("Feature", row, 30, row, 37).
					Build(),
			),
		}, ast.Modules[0].Declarations[3])

	// Def with Identifier path
	row = 5
	assert.Equal(t,
		DefDecl{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(row, 1, row, 29).Build(),
			Name:        NewIdentifierBuilder().WithName("Camera").WithStartEnd(row, 5, row, 11).Build(),
			resolvesToType: option.Some(
				NewTypeInfoBuilder().
					WithPath("raylib").
					WithName("Camera").WithNameStartEnd(row, 14, row, 28).
					WithStartEnd(row, 14, row, 28).
					Build(),
			),
		}, ast.Modules[0].Declarations[4])

}

func TestConvertToAST_function_declaration(t *testing.T) {
	source := `module foo;
	fn void test() {
		return 1;
	}`
	ast := ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := ast.Modules[0].Functions[0].(FunctionDecl)
	assert.Equal(t, Position{1, 1}, fnDecl.ASTBaseNode.StartPos)
	assert.Equal(t, Position{3, 2}, fnDecl.ASTBaseNode.EndPos)

	assert.Equal(t, "test", fnDecl.Signature.Name.Name, "Function name")
	assert.Equal(t, Position{1, 9}, fnDecl.Signature.Name.ASTBaseNode.StartPos)
	assert.Equal(t, Position{1, 13}, fnDecl.Signature.Name.ASTBaseNode.EndPos)

	assert.Equal(t, "void", fnDecl.Signature.ReturnType.Identifier.Name, "Return type")
	assert.Equal(t, Position{1, 4}, fnDecl.Signature.ReturnType.ASTBaseNode.StartPos)
	assert.Equal(t, Position{1, 8}, fnDecl.Signature.ReturnType.ASTBaseNode.EndPos)
}

func TestConvertToAST_function_declaration_one_line(t *testing.T) {
	source := `module foo;
	fn void init_window(int width, int height, char* title) @extern("InitWindow");`
	ast := ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := ast.Modules[0].Functions[0].(FunctionDecl)

	assert.Equal(t, "init_window", fnDecl.Signature.Name.Name, "Function name")
	assert.Equal(t, Position{1, 1}, fnDecl.ASTBaseNode.StartPos)
	assert.Equal(t, Position{1, 79}, fnDecl.ASTBaseNode.EndPos)
}

func TestConvertToAST_Function_returning_optional_type_declaration(t *testing.T) {
	source := `module foo;
	fn usz! test() {
		return 1;
	}`
	ast := ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := ast.Modules[0].Functions[0].(FunctionDecl)

	assert.Equal(t, "usz", fnDecl.Signature.ReturnType.Identifier.Name, "Return type")
	assert.Equal(t, true, fnDecl.Signature.ReturnType.Optional, "Return type should be optional")
}

func TestConvertToAST_function_with_arguments_declaration(t *testing.T) {
	source := `module foo;
	fn void test(int number, char ch, int* pointer) {
		return 1;
	}`
	ast := ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := ast.Modules[0].Functions[0].(FunctionDecl)

	assert.Equal(t, 3, len(fnDecl.Signature.Parameters))
	assert.Equal(t,
		FunctionParameter{
			Name: NewIdentifierBuilder().WithName("number").WithStartEnd(1, 18, 1, 24).Build(),
			Type: NewTypeInfoBuilder().
				WithName("int").WithNameStartEnd(1, 14, 1, 17).
				IsBuiltin().
				WithStartEnd(1, 14, 1, 17).
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(1, 14, 1, 24).Build(),
		},
		fnDecl.Signature.Parameters[0],
	)
	assert.Equal(t,
		FunctionParameter{
			Name: NewIdentifierBuilder().WithName("ch").WithStartEnd(1, 31, 1, 33).Build(),
			Type: NewTypeInfoBuilder().
				WithName("char").WithNameStartEnd(1, 26, 1, 30).
				IsBuiltin().
				WithStartEnd(1, 26, 1, 30).
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(1, 26, 1, 33).Build(),
		},
		fnDecl.Signature.Parameters[1],
	)
	assert.Equal(t,
		FunctionParameter{
			Name: NewIdentifierBuilder().WithName("pointer").WithStartEnd(1, 40, 1, 47).Build(),
			Type: NewTypeInfoBuilder().
				WithName("int").WithNameStartEnd(1, 35, 1, 38).
				IsBuiltin().
				IsPointer().
				WithStartEnd(1, 35, 1, 38).
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(1, 35, 1, 47).Build(),
		},
		fnDecl.Signature.Parameters[2],
	)
}

func TestConvertToAST_method_declaration(t *testing.T) {
	source := `module foo;
	fn Object* UserStruct.method(self, int* pointer) {
		return 1;
	}`
	ast := ConvertToAST(GetCST(source), source, "file.c3")

	methodDecl := ast.Modules[0].Functions[0].(FunctionDecl)

	assert.Equal(t, Position{1, 1}, methodDecl.ASTBaseNode.StartPos)
	assert.Equal(t, Position{3, 2}, methodDecl.ASTBaseNode.EndPos)

	assert.Equal(t, true, methodDecl.ParentTypeId.IsSome(), "Function is flagged as method")
	assert.Equal(t, "method", methodDecl.Signature.Name.Name, "Function name")
	assert.Equal(t, Position{1, 23}, methodDecl.Signature.Name.ASTBaseNode.StartPos)
	assert.Equal(t, Position{1, 29}, methodDecl.Signature.Name.ASTBaseNode.EndPos)

	assert.Equal(t, "Object", methodDecl.Signature.ReturnType.Identifier.Name, "Return type")
	assert.Equal(t, uint(1), methodDecl.Signature.ReturnType.Pointer, "Return type is pointer")
	assert.Equal(t, Position{1, 4}, methodDecl.Signature.ReturnType.ASTBaseNode.StartPos)
	assert.Equal(t, Position{1, 10}, methodDecl.Signature.ReturnType.ASTBaseNode.EndPos)

	assert.Equal(t, 2, len(methodDecl.Signature.Parameters))
	assert.Equal(t,
		FunctionParameter{
			Name: NewIdentifierBuilder().WithName("self").WithStartEnd(1, 30, 1, 34).Build(),
			Type: NewTypeInfoBuilder().
				WithName("UserStruct").WithNameStartEnd(1, 30, 1, 34).
				WithStartEnd(1, 30, 1, 34).
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(1, 30, 1, 34).Build(),
		},
		methodDecl.Signature.Parameters[0],
	)
	assert.Equal(t,
		FunctionParameter{
			Name: NewIdentifierBuilder().WithName("pointer").WithStartEnd(1, 41, 1, 48).Build(),
			Type: NewTypeInfoBuilder().
				WithName("int").WithNameStartEnd(1, 36, 1, 39).
				IsBuiltin().
				IsPointer().
				WithStartEnd(1, 36, 1, 39).
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(1, 36, 1, 48).Build(),
		},
		methodDecl.Signature.Parameters[1],
	)
}

func TestConvertToAST_method_declaration_mutable(t *testing.T) {
	source := `module foo;
	fn Object* UserStruct.method(&self, int* pointer) {
		return 1;
	}`
	ast := ConvertToAST(GetCST(source), source, "file.c3")
	methodDecl := ast.Modules[0].Functions[0].(FunctionDecl)

	assert.Equal(t,
		FunctionParameter{
			Name: NewIdentifierBuilder().WithName("self").WithStartEnd(1, 31, 1, 35).Build(),
			Type: NewTypeInfoBuilder().
				WithName("UserStruct").WithNameStartEnd(1, 31, 1, 35).
				WithStartEnd(1, 30, 1, 35).
				IsPointer().
				Build(),
			ASTBaseNode: NewBaseNodeBuilder().
				WithStartEnd(1, 30, 1, 35).Build(),
		},
		methodDecl.Signature.Parameters[0],
	)
}

func TestConvertToAST_interface_decl(t *testing.T) {
	source := `module foo;
	interface MyInterface {
		fn void method1();
		fn int method2(char arg);
	}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	interfaceDecl := ast.Modules[0].Declarations[0].(InterfaceDecl)

	assert.Equal(t, 2, len(interfaceDecl.Methods))
}

func TestConvertToAST_macro_decl(t *testing.T) {
	source := `module foo;
	macro m(x) {
    	return x + 2;
	}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")
	macroDecl := ast.Modules[0].Macros[0].(MacroDecl)

	assert.Equal(t, Position{1, 1}, macroDecl.StartPos)
	assert.Equal(t, Position{3, 2}, macroDecl.EndPos)

	assert.Equal(t, "m", macroDecl.Signature.Name.Name)
	assert.Equal(t, Position{1, 7}, macroDecl.Signature.Name.StartPos)
	assert.Equal(t, Position{1, 8}, macroDecl.Signature.Name.EndPos)

	assert.Equal(t, 1, len(macroDecl.Signature.Parameters))
	assert.Equal(
		t,
		FunctionParameter{
			ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(1, 9, 1, 10).Build(),
			Name:        NewIdentifierBuilder().WithName("x").WithStartEnd(1, 9, 1, 10).Build(),
		},
		macroDecl.Signature.Parameters[0],
	)
}
