package factory

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func aWithPos(startRow uint, startCol uint, endRow uint, endCol uint) ast.NodeAttributes {
	return ast.NewNodeAttributesBuilder().
		WithRangePositions(startRow, startCol, endRow, endCol).
		Build()
}

func TestConvertToAST_module(t *testing.T) {
	source := `module foo;
	int dummy;`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	assert.Equal(t, "foo", tree.Modules[0].Name)
	assert.Equal(t, lsp.NewRange(0, 0, 1, 11), tree.Modules[0].Range)
}

func TestConvertToAST_module_implicit(t *testing.T) {
	t.Run("First module is anonymous, second is named", func(t *testing.T) {
		source := `
	int variable = 0;
	module foo;`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source), source, "path/file/xxx.c3")

		assert.Equal(t, "path_file_xxx", tree.Modules[0].Name)
		assert.Equal(t, lsp.NewRange(1, 1, 2, 1), tree.Modules[0].Range)

		assert.Equal(t, "foo", tree.Modules[1].Name)
		assert.Equal(t, lsp.NewRange(2, 1, 2, 12), tree.Modules[1].Range)
	})

	t.Run("Single anonymous module", func(t *testing.T) {
		source := `int number = 0;
		number + 2;`

		cv := NewASTConverter()
		tree := cv.ConvertToAST(GetCST(source), source, "path/file/xxx.c3")

		assert.Equal(t, "path_file_xxx", tree.Modules[0].Name)
		assert.Equal(t, lsp.NewRange(0, 0, 1, 13), tree.Modules[0].Range)
	})
}

func TestConvertToAST_module_with_generics(t *testing.T) {
	source := `module foo(<TypeDescription>);`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	assert.Equal(t, []string{"TypeDescription"}, tree.Modules[0].GenericParameters)
}

func TestConvertToAST_module_with_attributes(t *testing.T) {
	source := `module foo @private;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	assert.Equal(t, []string{"@private"}, tree.Modules[0].Attributes)
}

func TestConvertToAST_module_with_imports(t *testing.T) {
	source := `module foo;
	import foo;
	import foo2::subfoo;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	assert.Equal(t, "foo", tree.Modules[0].Imports[0].Path.Name)
	assert.Equal(t, lsp.NewRange(1, 1, 1, 12), tree.Modules[0].Imports[0].Range)

	assert.Equal(t, "foo2::subfoo", tree.Modules[0].Imports[1].Path.Name)
	assert.Equal(t, lsp.NewRange(2, 1, 2, 21), tree.Modules[0].Imports[1].Range)
}

// -----------------------------------------------------
// Convert global variable declaration

func TestConvertToAST_global_variable_unitialized(t *testing.T) {
	source := `module foo;
	int hello;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
	spec := decl.Spec.(*ast.ValueSpec)

	assert.Equal(t, "hello", spec.Names[0].Name)
	assert.Equal(t, lsp.NewRange(1, 5, 1, 10), spec.Names[0].Range)
	assert.Equal(t, "int", spec.Type.Identifier.Name)
	assert.Equal(t, lsp.NewRange(1, 1, 1, 4), spec.Type.Identifier.Range)
}

func TestConvertToAST_global_variable_with_scalar_initialization(t *testing.T) {
	source := `module foo;
	int hello = 3;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)

	spec := decl.Spec.(*ast.ValueSpec)
	assert.Equal(t, "hello", spec.Names[0].Name)
	assert.Equal(t, &ast.BasicLit{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithRange(lsp.NewRange(1, 13, 1, 14)).Build(),
		Kind:           ast.INT,
		Value:          "3",
	}, spec.Value)
}

func TestConvertToAST_declare_multiple_variables_in_single_statement(t *testing.T) {
	source := `module foo;
	int dog, cat, elephant;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
	assert.Len(t, decl.Spec.(*ast.ValueSpec).Names, 3)

	assert.Equal(t, "dog", decl.Spec.(*ast.ValueSpec).Names[0].Name)
	assert.Equal(t, lsp.NewRange(1, 5, 1, 8), decl.Spec.(*ast.ValueSpec).Names[0].Range)
	assert.Equal(t, "cat", decl.Spec.(*ast.ValueSpec).Names[1].Name)
	assert.Equal(t, lsp.NewRange(1, 10, 1, 13), decl.Spec.(*ast.ValueSpec).Names[1].Range)
	assert.Equal(t, "elephant", decl.Spec.(*ast.ValueSpec).Names[2].Name)
	assert.Equal(t, lsp.NewRange(1, 15, 1, 23), decl.Spec.(*ast.ValueSpec).Names[2].Range)
	assert.Equal(t, "int", decl.Spec.(*ast.ValueSpec).Type.Identifier.Name)
}

// -----------------------------------------------------

func TestConvertToAST_enum_decl(t *testing.T) {
	source := `module foo;
	enum Colors { RED, BLUE, GREEN }
	enum TypedColors:int { RED, BLUE, GREEN } // Typed enums`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	enumDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
	assert.Equal(t, ast.Token(ast.ENUM), enumDecl.Token)
	assert.Equal(t, "Colors", enumDecl.Spec.(*ast.TypeSpec).Name.Name)
	assert.Equal(t, lsp.NewRange(1, 1, 1, 33), enumDecl.Range)

	enumType := enumDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType)
	assert.Equal(t, option.None[ast.TypeInfo](), enumType.BaseType)

	assert.Equal(t, []ast.Expression{}, enumType.StaticValues, "No fields should be present")
	assert.Len(t, enumType.Values, 3)
	assert.Equal(t, "RED", enumType.Values[0].Name.Name)
	assert.Equal(t, lsp.NewRange(1, 15, 1, 18), enumType.Values[0].Name.Range)
	assert.Equal(t, "BLUE", enumType.Values[1].Name.Name)
	assert.Equal(t, lsp.NewRange(1, 20, 1, 24), enumType.Values[1].Name.Range)
	assert.Equal(t, "GREEN", enumType.Values[2].Name.Name)
	assert.Equal(t, lsp.NewRange(1, 26, 1, 31), enumType.Values[2].Name.Range)

	enumDecl = tree.Modules[0].Declarations[1].(*ast.GenDecl)
	assert.Equal(t, "int", enumDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType).BaseType.Get().Identifier.Name)
	return
}

func TestConvertToAST_enum_decl_with_associated_params(t *testing.T) {
	source := `module foo;
	enum State : int (String desc, bool active, char ke) {
		PENDING = {"pending start", false, 'c'},
		RUNNING = {"running", true, 'e'},
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	enumDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
	enumType := enumDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType)
	assert.Len(t, enumType.StaticValues, 3)

	assert.Equal(t,
		ast.NewIdentifierBuilder().
			WithName("desc").
			WithStartEnd(1, 26, 1, 30).BuildPtr(),
		enumType.StaticValues[0].(*ast.Field).Name,
	)
	assert.Equal(t, "String", enumType.StaticValues[0].(*ast.Field).Type.Identifier.String())
	assert.Equal(t, lsp.NewRange(1, 19, 1, 25), enumType.StaticValues[0].(*ast.Field).Type.Identifier.Range)

	assert.Equal(t, ast.NewIdentifierBuilder().
		WithName("active").
		WithStartEnd(1, 37, 1, 43).BuildPtr(),
		enumType.StaticValues[1].(*ast.Field).Name,
	)
	assert.Equal(t, "bool", enumType.StaticValues[1].(*ast.Field).Type.Identifier.String())

	assert.Equal(t, ast.NewIdentifierBuilder().
		WithName("ke").
		WithStartEnd(1, 50, 1, 52).BuildPtr(),
		enumType.StaticValues[2].(*ast.Field).Name,
	)
	assert.Equal(t, "char", enumType.StaticValues[2].(*ast.Field).Type.Identifier.String())
}

func TestConvertToAST_struct_decl(t *testing.T) {
	t.Run("Test basic struct declaration", func(t *testing.T) {
		source := `module foo;
	struct MyStruct {
		int data;
		char key;
		raylib::Camera camera;
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

		decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		spec := decl.Spec.(*ast.TypeSpec)
		structType := spec.TypeDescription.(*ast.StructType)

		structRange := lsp.NewRange(1, 1, 5, 2)
		assert.Equal(t, structRange, decl.Range)
		assert.Equal(t, structRange, spec.Range)
		assert.Equal(t, structRange, structType.Range)
		assert.Equal(t, "MyStruct", spec.Name.Name)
		assert.Equal(t, lsp.NewRange(1, 8, 1, 16), spec.Name.Range)

		assert.Equal(t,
			&ast.StructField{
				NodeAttributes: aWithPos(2, 2, 2, 11),
				Names: []*ast.Ident{
					{
						NodeAttributes: aWithPos(2, 6, 2, 10),
						Name:           "data",
					},
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(2, 2, 2, 5),
					Identifier:     ast.NewIdentifierBuilder().WithName("int").WithStartEnd(2, 2, 2, 5).BuildPtr(),
					BuiltIn:        true,
				},
			}, structType.Fields[0])
		assert.Equal(t,
			&ast.StructField{
				NodeAttributes: aWithPos(3, 2, 3, 11),
				Names: []*ast.Ident{
					{
						NodeAttributes: aWithPos(3, 7, 3, 10),
						Name:           "key",
					},
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(3, 2, 3, 6),
					Identifier:     ast.NewIdentifierBuilder().WithName("char").WithStartEnd(3, 2, 3, 6).BuildPtr(),
					BuiltIn:        true,
				},
			}, structType.Fields[1])
		assert.Equal(t,
			&ast.StructField{
				NodeAttributes: aWithPos(4, 2, 4, 24),
				Names: []*ast.Ident{
					{
						NodeAttributes: aWithPos(4, 17, 4, 23),
						Name:           "camera",
					},
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(4, 2, 4, 16),
					Identifier: &ast.Ident{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(4, 2, 4, 16).Build(),
						ModulePath: ast.NewIdentifierBuilder().
							WithName("raylib").
							WithStartEnd(4, 2, 4, 10).
							BuildPtr(),
						Name: "Camera",
					},
					BuiltIn: false,
				},
			}, structType.Fields[2])
	})

	t.Run("struct with interface", func(t *testing.T) {
		source := `module foo;
	struct MyStruct (MyInterface, MySecondInterface) {
		int data;
		char key;
		raylib::Camera camera;
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

		structDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		structType := structDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)
		assert.Equal(t, "MyInterface", structType.Implements[0].Name)
		assert.Equal(t, lsp.NewRange(1, 18, 1, 29), structType.Implements[0].Range)

		assert.Equal(t, "MySecondInterface", structType.Implements[1].Name)
		assert.Equal(t, lsp.NewRange(1, 31, 1, 48), structType.Implements[1].Range)
	})

	t.Run("struct with anonymous sub struct", func(t *testing.T) {
		source := `module x;
		struct MyStruct {
			struct data {
			  int a;
			}
		}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

		structDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		structType := structDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)

		assert.Equal(t, "data", structType.Fields[0].Names[0].Name)
		assert.Equal(t, lsp.NewRange(2, 3, 4, 4), structType.Fields[0].Range)

		fieldType := structType.Fields[0].Type.(*ast.StructType)
		assert.Equal(t, lsp.NewRange(2, 3, 4, 4), fieldType.Range)
		assert.Equal(t, ast.StructTypeNormal, int(fieldType.Type))

		subField := fieldType.Fields[0]
		assert.Equal(t, lsp.NewRange(3, 5, 3, 11), subField.Range)
		assert.Equal(t, "a", subField.Names[0].Name)
		assert.Equal(t, lsp.NewRange(3, 9, 3, 10), subField.Names[0].Range)
		assert.Equal(t, "int", subField.Type.(ast.TypeInfo).Identifier.Name)
		assert.Equal(t, true, subField.Type.(ast.TypeInfo).BuiltIn)
		assert.Equal(t, lsp.NewRange(3, 5, 3, 8), subField.Type.(ast.TypeInfo).Range)
	})

	t.Run("struct with anonymous bit-structs", func(t *testing.T) {
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

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
		structDecl := tree.Modules[0].Declarations[1].(*ast.GenDecl)
		members := structDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType).Fields

		assert.Equal(t, 5, len(members))

		assert.Equal(
			t,
			&ast.StructField{
				NodeAttributes: aWithPos(4, 3, 4, 25),
				Names: []*ast.Ident{
					ast.NewIdentifierBuilder().
						WithName("bc").
						WithStartEnd(4, 14, 4, 16).
						BuildPtr(),
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(4, 3, 4, 13),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register16").
						WithStartEnd(4, 3, 4, 13).
						BuildPtr(),
				},
				BitRange: option.Some([2]uint{0, 15}),
			},
			members[0],
		)

		assert.Equal(
			t,
			&ast.StructField{
				NodeAttributes: aWithPos(5, 3, 5, 22),
				Names: []*ast.Ident{
					ast.NewIdentifierBuilder().
						WithName("b").
						WithStartEnd(5, 12, 5, 13).
						BuildPtr(),
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(5, 3, 5, 11),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register").
						WithStartEnd(5, 3, 5, 11).
						BuildPtr(),
				},
				BitRange: option.Some([2]uint{8, 15}),
			},
			members[1],
		)

		assert.Equal(
			t,
			&ast.StructField{
				NodeAttributes: aWithPos(6, 3, 6, 21),
				Names: []*ast.Ident{
					ast.NewIdentifierBuilder().
						WithName("c").
						WithStartEnd(6, 12, 6, 13).
						BuildPtr(),
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(6, 3, 6, 11),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register").
						WithStartEnd(6, 3, 6, 11).
						BuildPtr(),
				},
				BitRange: option.Some([2]uint{0, 7}),
			},
			members[2],
		)

		assert.Equal(
			t,
			&ast.StructField{
				NodeAttributes: aWithPos(8, 2, 8, 16),
				Names: []*ast.Ident{
					ast.NewIdentifierBuilder().
						WithName("sp").
						WithStartEnd(8, 13, 8, 15).
						BuildPtr(),
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(8, 2, 8, 12),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register16").
						WithStartEnd(8, 2, 8, 12).
						BuildPtr(),
				},
			},
			members[3],
		)

		assert.Equal(
			t,
			&ast.StructField{
				NodeAttributes: aWithPos(9, 2, 9, 16),
				Names: []*ast.Ident{
					ast.NewIdentifierBuilder().
						WithName("pc").
						WithStartEnd(9, 13, 9, 15).
						BuildPtr(),
				},
				Type: ast.TypeInfo{
					NodeAttributes: aWithPos(9, 2, 9, 12),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register16").
						WithStartEnd(9, 2, 9, 12).
						BuildPtr(),
				},
			},
			members[4],
		)
	})

	t.Run("struct with subtype", func(t *testing.T) {
		source := `module x;
	struct Person {
		int age;
		String name;
	}
	struct ImportantPerson {
		inline Person person;
		String title;
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
		structDecl := tree.Modules[0].Declarations[1].(*ast.GenDecl)
		members := structDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType).Fields

		assert.Equal(t, true, members[0].Inlined)
		assert.Equal(t, "person", members[0].Names[0].Name)
		assert.Equal(t, lsp.NewRange(6, 16, 6, 22), members[0].Names[0].Range)
		memberType := members[0].Type.(ast.TypeInfo)
		assert.Equal(t, "Person", memberType.Identifier.Name)
		assert.Equal(t, lsp.NewRange(6, 9, 6, 15), memberType.Identifier.Range)
	})
}

func TestConvertToAST_union_decl(t *testing.T) {
	source := `module foo;
	union MyStruct {
		char data;
		char key;
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	unionDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)

	assert.Equal(t, ast.StructTypeUnion, int(unionDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType).Type))
}

func TestConvertToAST_bitstruct_decl(t *testing.T) {
	source := `module x;
	bitstruct Test (AnInterface) : uint
	{
		ushort a : 0..15;
		ushort b : 16..31;
		bool c : 7;
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	bitstructDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
	structType := bitstructDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)

	assert.Equal(t, ast.StructTypeBitStruct, int(structType.Type))
	assert.Equal(t, true, structType.BackingType.IsSome())

	expectedType := ast.TypeInfo{
		NodeAttributes: aWithPos(1, 32, 1, 36),
		BuiltIn:        true,
		Identifier: ast.NewIdentifierBuilder().
			WithName("uint").
			WithStartEnd(1, 32, 1, 36).
			BuildPtr(),
	}
	assert.Equal(t, expectedType, structType.BackingType.Get())
	assert.Equal(t, 1, len(structType.Implements))
	assert.Equal(t, "AnInterface", structType.Implements[0].Name)

	expect := &ast.StructField{
		NodeAttributes: aWithPos(3, 2, 3, 19),
		Names: []*ast.Ident{
			ast.NewIdentifierBuilder().
				WithName("a").
				WithStartEnd(3, 9, 3, 10).
				BuildPtr(),
		},
		Type: ast.TypeInfo{
			NodeAttributes: aWithPos(3, 2, 3, 8),
			BuiltIn:        true,
			Identifier: ast.NewIdentifierBuilder().
				WithName("ushort").
				WithStartEnd(3, 2, 3, 8).
				BuildPtr(),
		},
		BitRange: option.Some([2]uint{0, 15}),
	}
	assert.Equal(t, expect, structType.Fields[0])
}

func TestConvertToAST_fault_decl(t *testing.T) {
	source := `module x;
	fault IOResult
	{
	  IO_ERROR,
	  PARSE_ERROR
	};`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	faultDecl := tree.Modules[0].Declarations[0].(*ast.FaultDecl)

	assert.Equal(
		t,
		ast.NewIdentifierBuilder().
			WithName("IOResult").
			WithStartEnd(1, 7, 1, 15).
			Build(),
		faultDecl.Name,
	)
	assert.Equal(t, lsp.Position{1, 1}, faultDecl.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{5, 2}, faultDecl.NodeAttributes.Range.End)

	assert.Equal(t, false, faultDecl.BackingType.IsSome())
	assert.Equal(t, 2, len(faultDecl.Members))

	assert.Equal(t,
		ast.FaultMember{
			Name: ast.NewIdentifierBuilder().
				WithName("IO_ERROR").
				WithStartEnd(3, 3, 3, 11).
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(3, 3, 3, 11).
				Build(),
		},
		faultDecl.Members[0],
	)

	assert.Equal(t,
		ast.FaultMember{
			Name: ast.NewIdentifierBuilder().
				WithName("PARSE_ERROR").
				WithStartEnd(4, 3, 4, 14).
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(4, 3, 4, 14).
				Build(),
		},
		faultDecl.Members[1],
	)
}

func TestConvertToAST_def_declares_type(t *testing.T) {
	source := `def Kilo = int;`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	assert.Equal(t,
		&ast.DefDecl{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(0, 0, 0, 15).Build(),
			Name:           ast.NewIdentifierBuilder().WithName("Kilo").WithStartEnd(0, 4, 0, 8).Build(),
			Expr: ast.NewTypeInfoBuilder().
				WithName("int").
				WithNameStartEnd(0, 11, 0, 14).
				IsBuiltin().
				WithStartEnd(0, 11, 0, 14).
				Build(),
		}, tree.Modules[0].Declarations[0])
}

func TestConvertToAST_def_declares_function_type(t *testing.T) {
	source := `def Kilo = fn void (int);`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	assert.Equal(t,
		&ast.FuncType{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(0, 11, 0, 24).Build(),
			ReturnType: ast.NewTypeInfoBuilder().
				WithName("void").
				WithStartEnd(0, 14, 0, 18).
				WithNameStartEnd(0, 14, 0, 18).
				IsBuiltin().
				Build(),
			Params: []ast.FunctionParameter{
				{
					NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(0, 20, 0, 23).Build(),
					Type: ast.NewTypeInfoBuilder().
						WithName("int").
						WithStartEnd(0, 20, 0, 23).
						WithNameStartEnd(0, 20, 0, 23).
						IsBuiltin().
						Build(),
				},
			},
		}, tree.Modules[0].Declarations[0].(*ast.DefDecl).Expr)
}

func TestConvertToAST_function_declaration(t *testing.T) {
	source := `module foo;
	fn void test() {
		return 1;
	}`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
	assert.Equal(t, lsp.Position{1, 1}, fnDecl.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{3, 2}, fnDecl.NodeAttributes.Range.End)

	assert.Equal(t, "test", fnDecl.Signature.Name.Name, "Function name")
	assert.Equal(t, lsp.Position{1, 9}, fnDecl.Signature.Name.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{1, 13}, fnDecl.Signature.Name.NodeAttributes.Range.End)

	assert.Equal(t, "void", fnDecl.Signature.ReturnType.Identifier.Name, "Return type")
	assert.Equal(t, lsp.Position{1, 4}, fnDecl.Signature.ReturnType.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{1, 8}, fnDecl.Signature.ReturnType.NodeAttributes.Range.End)
}

func TestConvertToAST_function_declaration_one_line(t *testing.T) {
	source := `module foo;
	fn void init_window(int width, int height, char* title) @extern("InitWindow");`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	assert.Equal(t, "init_window", fnDecl.Signature.Name.Name, "Function name")
	assert.Equal(t, lsp.Position{1, 1}, fnDecl.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{1, 79}, fnDecl.NodeAttributes.Range.End)
}

func TestConvertToAST_Function_returning_optional_type_declaration(t *testing.T) {
	source := `module foo;
	fn usz! test() {
		return 1;
	}`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	assert.Equal(t, "usz", fnDecl.Signature.ReturnType.Identifier.Name, "Return type")
	assert.Equal(t, true, fnDecl.Signature.ReturnType.Optional, "Return type should be optional")
}

func TestConvertToAST_function_with_arguments_declaration(t *testing.T) {
	source := `module foo;
	fn void test(int number, char ch, int* pointer) {
		return 1;
	}`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	assert.Equal(t, 3, len(fnDecl.Signature.Parameters))
	assert.Equal(t,
		ast.FunctionParameter{
			Name: ast.NewIdentifierBuilder().WithName("number").WithStartEnd(1, 18, 1, 24).Build(),
			Type: ast.NewTypeInfoBuilder().
				WithName("int").WithNameStartEnd(1, 14, 1, 17).
				IsBuiltin().
				WithStartEnd(1, 14, 1, 17).
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(1, 14, 1, 24).Build(),
		},
		fnDecl.Signature.Parameters[0],
	)
	assert.Equal(t,
		ast.FunctionParameter{
			Name: ast.NewIdentifierBuilder().WithName("ch").WithStartEnd(1, 31, 1, 33).Build(),
			Type: ast.NewTypeInfoBuilder().
				WithName("char").WithNameStartEnd(1, 26, 1, 30).
				IsBuiltin().
				WithStartEnd(1, 26, 1, 30).
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(1, 26, 1, 33).Build(),
		},
		fnDecl.Signature.Parameters[1],
	)
	assert.Equal(t,
		ast.FunctionParameter{
			Name: ast.NewIdentifierBuilder().WithName("pointer").WithStartEnd(1, 40, 1, 47).Build(),
			Type: ast.NewTypeInfoBuilder().
				WithName("int").WithNameStartEnd(1, 35, 1, 38).
				IsBuiltin().
				IsPointer().
				WithStartEnd(1, 35, 1, 38).
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(1, 35, 1, 47).Build(),
		},
		fnDecl.Signature.Parameters[2],
	)
}

func TestConvertToAST_method_declaration(t *testing.T) {
	source := `module foo;
	fn Object* UserStruct.method(self, int* pointer) {
		return 1;
	}`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")

	methodDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	assert.Equal(t, lsp.Position{1, 1}, methodDecl.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{3, 2}, methodDecl.NodeAttributes.Range.End)

	assert.Equal(t, true, methodDecl.ParentTypeId.IsSome(), "Function is flagged as method")
	assert.Equal(t, "method", methodDecl.Signature.Name.Name, "Function name")
	assert.Equal(t, lsp.Position{1, 23}, methodDecl.Signature.Name.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{1, 29}, methodDecl.Signature.Name.NodeAttributes.Range.End)

	assert.Equal(t, "Object", methodDecl.Signature.ReturnType.Identifier.Name, "Return type")
	assert.Equal(t, uint(1), methodDecl.Signature.ReturnType.Pointer, "Return type is pointer")
	assert.Equal(t, lsp.Position{1, 4}, methodDecl.Signature.ReturnType.NodeAttributes.Range.Start)
	assert.Equal(t, lsp.Position{1, 10}, methodDecl.Signature.ReturnType.NodeAttributes.Range.End)

	assert.Equal(t, 2, len(methodDecl.Signature.Parameters))
	assert.Equal(t,
		ast.FunctionParameter{
			Name: ast.NewIdentifierBuilder().WithName("self").WithStartEnd(1, 30, 1, 34).Build(),
			Type: ast.NewTypeInfoBuilder().
				WithName("UserStruct").WithNameStartEnd(1, 30, 1, 34).
				WithStartEnd(1, 30, 1, 34).
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(1, 30, 1, 34).Build(),
		},
		methodDecl.Signature.Parameters[0],
	)
	assert.Equal(t,
		ast.FunctionParameter{
			Name: ast.NewIdentifierBuilder().WithName("pointer").WithStartEnd(1, 41, 1, 48).Build(),
			Type: ast.NewTypeInfoBuilder().
				WithName("int").WithNameStartEnd(1, 36, 1, 39).
				IsBuiltin().
				IsPointer().
				WithStartEnd(1, 36, 1, 39).
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(1, 36, 1, 48).Build(),
		},
		methodDecl.Signature.Parameters[1],
	)
}

func TestConvertToAST_method_declaration_mutable(t *testing.T) {
	source := `module foo;
	fn Object* UserStruct.method(&self, int* pointer) {
		return 1;
	}`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	methodDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	assert.Equal(t,
		ast.FunctionParameter{
			Name: ast.NewIdentifierBuilder().WithName("self").WithStartEnd(1, 31, 1, 35).Build(),
			Type: ast.NewTypeInfoBuilder().
				WithName("UserStruct").WithNameStartEnd(1, 31, 1, 35).
				WithStartEnd(1, 30, 1, 35).
				IsPointer().
				Build(),
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithRangePositions(1, 30, 1, 35).Build(),
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

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	interfaceDecl := tree.Modules[0].Declarations[0].(*ast.InterfaceDecl)

	assert.Equal(t, 2, len(interfaceDecl.Methods))
}

func TestConvertToAST_macro_decl(t *testing.T) {
	source := `module foo;
	macro m(x) {
    	return x + 2;
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source), source, "file.c3")
	macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

	assert.Equal(t, lsp.Position{1, 1}, macroDecl.Range.Start)
	assert.Equal(t, lsp.Position{3, 2}, macroDecl.Range.End)

	assert.Equal(t, "m", macroDecl.Signature.Name.Name)
	assert.Equal(t, lsp.Position{1, 7}, macroDecl.Signature.Name.Range.Start)
	assert.Equal(t, lsp.Position{1, 8}, macroDecl.Signature.Name.Range.End)

	assert.Equal(t, 1, len(macroDecl.Signature.Parameters))
	assert.Equal(
		t,
		ast.FunctionParameter{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(1, 9, 1, 10).Build(),
			Name:           ast.NewIdentifierBuilder().WithName("x").WithStartEnd(1, 9, 1, 10).Build(),
		},
		macroDecl.Signature.Parameters[0],
	)
}
