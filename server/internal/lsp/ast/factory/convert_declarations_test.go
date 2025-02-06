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
	t.Run("convert module", func(t *testing.T) {
		source := `module foo;
	int dummy;`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		assert.Equal(t, "foo", tree.Modules[0].Name)
		assert.Equal(t, lsp.NewRange(0, 0, 1, 11), tree.Modules[0].Range)
	})

	t.Run("Single anonymous module", func(t *testing.T) {
		source := `int number = 0;
		number + 2;`

		cv := NewASTConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "path/file/xxx.c3")

		assert.Equal(t, "path_file_xxx", tree.Modules[0].Name)
		assert.Equal(t, lsp.NewRange(0, 0, 1, 13), tree.Modules[0].Range)
	})

	t.Run("first module is anonymous, second is named", func(t *testing.T) {
		source := `
	int variable = 0;
	module foo;`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "path/file/xxx.c3")

		assert.Equal(t, "path_file_xxx", tree.Modules[0].Name)
		assert.Equal(t, lsp.NewRange(1, 1, 1, 18), tree.Modules[0].Range, "Wrong range in anonymous module")

		assert.Equal(t, "foo", tree.Modules[1].Name)
		assert.Equal(t, lsp.NewRange(2, 1, 2, 12), tree.Modules[1].Range, "Wrong range in module foo")
	})

	t.Run("multiple named modules", func(t *testing.T) {
		source := `<* foo is first module *>
	module foo;
	int variable = 0;

	<* app is second module *>
	module app;
	fn void main() {
	};`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "path/file/xxx.c3")

		assert.Equal(t, "foo", tree.Modules[0].Name)
		assert.Equal(t, lsp.NewRange(1, 1, 4, 27), tree.Modules[0].Range, "Range of module foo is wrong")
		assert.Equal(t, "foo is first module", tree.Modules[0].DocComment.Get().GetBody())

		assert.Equal(t, "app", tree.Modules[1].Name)
		assert.Equal(t, lsp.NewRange(5, 1, 7, 3), tree.Modules[1].Range, "Range of module app is wrong")
		assert.Equal(t, "app is second module", tree.Modules[1].DocComment.Get().GetBody())
	})

}

func TestConvertToAST_module_with_generics(t *testing.T) {
	source := `module foo(<TypeDescription>);`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

	assert.Equal(t, []string{"TypeDescription"}, tree.Modules[0].GenericParameters)
}

func TestConvertToAST_module_with_attributes(t *testing.T) {
	source := `module foo @private;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

	assert.Equal(t, []string{"@private"}, tree.Modules[0].Attributes)
}

func TestConvertToAST_module_with_imports(t *testing.T) {
	source := `module foo;
	import foo;
	import foo2::subfoo;`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

	assert.Equal(t, "foo", tree.Modules[0].Imports[0].Path.Name)
	assert.Equal(t, lsp.NewRange(1, 1, 1, 12), tree.Modules[0].Imports[0].Range)

	assert.Equal(t, "foo2::subfoo", tree.Modules[0].Imports[1].Path.Name)
	assert.Equal(t, lsp.NewRange(2, 1, 2, 21), tree.Modules[0].Imports[1].Range)
}

// -----------------------------------------------------
// Convert global variable declaration

func TestConvertToAST_global_variable(t *testing.T) {
	t.Run("parses global variable uninitialized", func(t *testing.T) {
		source := `module foo;
	<* abc *>
	int hello;`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		spec := decl.Spec.(*ast.ValueSpec)

		assert.Equal(t, "hello", spec.Names[0].Name)
		assert.Equal(t, lsp.NewRange(2, 5, 2, 10), spec.Names[0].Range)
		assert.Equal(t, "int", spec.Type.Identifier.Name)
		assert.Equal(t, lsp.NewRange(2, 1, 2, 4), spec.Type.Identifier.Range)
		assert.Equal(t, "abc", decl.DocComment.Get().GetBody())
	})

	t.Run("parses global variable with scalar initialization", func(t *testing.T) {
		source := `module foo;
	int hello = 3;`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)

		spec := decl.Spec.(*ast.ValueSpec)
		assert.Equal(t, "hello", spec.Names[0].Name)
		assert.Equal(t, &ast.BasicLit{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRange(lsp.NewRange(1, 13, 1, 14)).Build(),
			Kind:           ast.INT,
			Value:          "3",
		}, spec.Value)
	})

	t.Run("parses global multiple variables in single statement", func(t *testing.T) {
		source := `module foo;
	int dog, cat, elephant;`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		assert.Len(t, decl.Spec.(*ast.ValueSpec).Names, 3)

		assert.Equal(t, "dog", decl.Spec.(*ast.ValueSpec).Names[0].Name)
		assert.Equal(t, lsp.NewRange(1, 5, 1, 8), decl.Spec.(*ast.ValueSpec).Names[0].Range)
		assert.Equal(t, "cat", decl.Spec.(*ast.ValueSpec).Names[1].Name)
		assert.Equal(t, lsp.NewRange(1, 10, 1, 13), decl.Spec.(*ast.ValueSpec).Names[1].Range)
		assert.Equal(t, "elephant", decl.Spec.(*ast.ValueSpec).Names[2].Name)
		assert.Equal(t, lsp.NewRange(1, 15, 1, 23), decl.Spec.(*ast.ValueSpec).Names[2].Range)
		assert.Equal(t, "int", decl.Spec.(*ast.ValueSpec).Type.Identifier.Name)
	})

	t.Run("parses constant declaration", func(t *testing.T) {
		source := `module foo;
	<* abc *>
	const int HELLO = 3;`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		assert.Equal(t, ast.Token(ast.CONST), decl.Token)
		assert.Equal(t, "abc", decl.DocComment.Get().GetBody())

		spec := decl.Spec.(*ast.ValueSpec)
		assert.Equal(t, "HELLO", spec.Names[0].Name)
		assert.Equal(t, &ast.BasicLit{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithRange(lsp.NewRange(2, 19, 2, 20)).Build(),
			Kind:           ast.INT,
			Value:          "3",
		}, spec.Value)
	})
}

// -----------------------------------------------------

func TestConvertToAST_enum_decl(t *testing.T) {
	t.Run("parses enum decl", func(t *testing.T) {
		source := `module foo;
	<* colors enum *>
	enum Colors { RED, BLUE, GREEN }
	enum TypedColors:int { RED, BLUE, GREEN } // Typed enums`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		enumDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		assert.Equal(t, ast.Token(ast.ENUM), enumDecl.Token)
		assert.Equal(t, "Colors", enumDecl.Spec.(*ast.TypeSpec).Name.Name)
		assert.Equal(t, lsp.NewRange(2, 1, 2, 33), enumDecl.Range)
		assert.Equal(t, "colors enum", enumDecl.DocComment.Get().GetBody())

		enumType := enumDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType)
		assert.Equal(t, option.None[*ast.TypeInfo](), enumType.BaseType)

		assert.Equal(t, []ast.Expression{}, enumType.StaticValues, "No fields should be present")
		assert.Len(t, enumType.Values, 3)
		assert.Equal(t, "RED", enumType.Values[0].Name.Name)
		assert.Equal(t, lsp.NewRange(2, 15, 2, 18), enumType.Values[0].Name.Range)
		assert.Equal(t, "BLUE", enumType.Values[1].Name.Name)
		assert.Equal(t, lsp.NewRange(2, 20, 2, 24), enumType.Values[1].Name.Range)
		assert.Equal(t, "GREEN", enumType.Values[2].Name.Name)
		assert.Equal(t, lsp.NewRange(2, 26, 2, 31), enumType.Values[2].Name.Range)

		enumDecl = tree.Modules[0].Declarations[1].(*ast.GenDecl)
		assert.Equal(t, "int", enumDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType).BaseType.Get().Identifier.Name)
	})
}

func TestConvertToAST_enum_decl_with_associated_params(t *testing.T) {
	source := `module foo;
	enum State : int (String desc, bool active, char ke) {
		PENDING = {"pending start", false, 'c'},
		RUNNING = {"running", true, 'e'},
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

	enumDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
	enumType := enumDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType)
	assert.Len(t, enumType.StaticValues, 3)

	assert.Equal(t,
		ast.NewIdentifierBuilder().
			WithName("desc").
			WithStartEnd(1, 26, 1, 30).Build(),
		enumType.StaticValues[0].(*ast.Field).Name,
	)
	assert.Equal(t, "String", enumType.StaticValues[0].(*ast.Field).Type.Identifier.String())
	assert.Equal(t, lsp.NewRange(1, 19, 1, 25), enumType.StaticValues[0].(*ast.Field).Type.Identifier.Range)

	assert.Equal(t, ast.NewIdentifierBuilder().
		WithName("active").
		WithStartEnd(1, 37, 1, 43).Build(),
		enumType.StaticValues[1].(*ast.Field).Name,
	)
	assert.Equal(t, "bool", enumType.StaticValues[1].(*ast.Field).Type.Identifier.String())

	assert.Equal(t, ast.NewIdentifierBuilder().
		WithName("ke").
		WithStartEnd(1, 50, 1, 52).Build(),
		enumType.StaticValues[2].(*ast.Field).Name,
	)
	assert.Equal(t, "char", enumType.StaticValues[2].(*ast.Field).Type.Identifier.String())
}

func TestConvertToAST_struct_decl(t *testing.T) {
	t.Run("Test basic struct declaration", func(t *testing.T) {
		source := `module foo;
	<* abc *>
	struct MyStruct {
		int data;
		char key;
		raylib::Camera camera;
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		decl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		spec := decl.Spec.(*ast.TypeSpec)
		structType := spec.TypeDescription.(*ast.StructType)

		structRange := lsp.NewRange(2, 1, 6, 2)
		assert.Equal(t, structRange, decl.Range)
		assert.Equal(t, structRange, structType.Range)
		assert.Equal(t, "MyStruct", spec.Name.Name)
		assert.Equal(t, lsp.NewRange(2, 8, 2, 16), spec.Name.Range)
		assert.Equal(t, "abc", decl.DocComment.Get().GetBody())

		assert.Equal(t,
			&ast.StructField{
				NodeAttributes: aWithPos(3, 2, 3, 11),
				Names: []*ast.Ident{
					{
						NodeAttributes: aWithPos(3, 6, 3, 10),
						Name:           "data",
					},
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(3, 2, 3, 5),
					Identifier:     ast.NewIdentifierBuilder().WithName("int").WithStartEnd(3, 2, 3, 5).Build(),
					BuiltIn:        true,
				},
			}, structType.Fields[0])
		assert.Equal(t,
			&ast.StructField{
				NodeAttributes: aWithPos(4, 2, 4, 11),
				Names: []*ast.Ident{
					{
						NodeAttributes: aWithPos(4, 7, 4, 10),
						Name:           "key",
					},
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(4, 2, 4, 6),
					Identifier:     ast.NewIdentifierBuilder().WithName("char").WithStartEnd(4, 2, 4, 6).Build(),
					BuiltIn:        true,
				},
			}, structType.Fields[1])
		assert.Equal(t,
			&ast.StructField{
				NodeAttributes: aWithPos(5, 2, 5, 24),
				Names: []*ast.Ident{
					{
						NodeAttributes: aWithPos(5, 17, 5, 23),
						Name:           "camera",
					},
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(5, 2, 5, 16),
					Identifier: &ast.Ident{
						NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(5, 2, 5, 16).Build(),
						ModulePath: ast.NewIdentifierBuilder().
							WithName("raylib").
							WithStartEnd(5, 2, 5, 10).
							Build(),
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
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

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
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

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
		assert.Equal(t, "int", subField.Type.(*ast.TypeInfo).Identifier.Name)
		assert.Equal(t, true, subField.Type.(*ast.TypeInfo).BuiltIn)
		assert.Equal(t, lsp.NewRange(3, 5, 3, 8), subField.Type.(*ast.TypeInfo).Range)
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
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
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
						Build(),
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(4, 3, 4, 13),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register16").
						WithStartEnd(4, 3, 4, 13).
						Build(),
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
						Build(),
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(5, 3, 5, 11),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register").
						WithStartEnd(5, 3, 5, 11).
						Build(),
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
						Build(),
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(6, 3, 6, 11),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register").
						WithStartEnd(6, 3, 6, 11).
						Build(),
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
						Build(),
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(8, 2, 8, 12),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register16").
						WithStartEnd(8, 2, 8, 12).
						Build(),
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
						Build(),
				},
				Type: &ast.TypeInfo{
					NodeAttributes: aWithPos(9, 2, 9, 12),
					Identifier: ast.NewIdentifierBuilder().
						WithName("Register16").
						WithStartEnd(9, 2, 9, 12).
						Build(),
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
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		structDecl := tree.Modules[0].Declarations[1].(*ast.GenDecl)
		members := structDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType).Fields

		assert.Equal(t, true, members[0].Inlined)
		assert.Equal(t, "person", members[0].Names[0].Name)
		assert.Equal(t, lsp.NewRange(6, 16, 6, 22), members[0].Names[0].Range)
		memberType := members[0].Type.(*ast.TypeInfo)
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
	tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
	unionDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)

	assert.Equal(t, ast.StructTypeUnion, int(unionDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType).Type))
}

func TestConvertToAST_bitstruct_decl(t *testing.T) {
	t.Run("parses bitstruct", func(t *testing.T) {
		source := `module x;
	<* abc *>
	bitstruct Test (AnInterface) : uint
	{
		ushort a : 0..15;
		ushort b : 16..31;
		bool c : 7;
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		bitStructDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		structType := bitStructDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)

		assert.Equal(t, ast.StructTypeBitStruct, int(structType.Type))
		assert.Equal(t, true, structType.BackingType.IsSome())
		assert.Equal(t, "abc", bitStructDecl.DocComment.Get().GetBody())

		expectedType := &ast.TypeInfo{
			NodeAttributes: aWithPos(2, 32, 2, 36),
			BuiltIn:        true,
			Identifier: ast.NewIdentifierBuilder().
				WithName("uint").
				WithStartEnd(2, 32, 2, 36).
				Build(),
		}
		assert.Equal(t, expectedType, structType.BackingType.Get())
		assert.Equal(t, 1, len(structType.Implements))
		assert.Equal(t, "AnInterface", structType.Implements[0].Name)

		expect := &ast.StructField{
			NodeAttributes: aWithPos(4, 2, 4, 19),
			Names: []*ast.Ident{
				ast.NewIdentifierBuilder().
					WithName("a").
					WithStartEnd(4, 9, 4, 10).
					Build(),
			},
			Type: &ast.TypeInfo{
				NodeAttributes: aWithPos(4, 2, 4, 8),
				BuiltIn:        true,
				Identifier: ast.NewIdentifierBuilder().
					WithName("ushort").
					WithStartEnd(4, 2, 4, 8).
					Build(),
			},
			BitRange: option.Some([2]uint{0, 15}),
		}
		assert.Equal(t, expect, structType.Fields[0])
	})

	t.Run("parses incomplete bitstruct", func(t *testing.T) {
		source := `module x;
	bitstruct Test : uint {
		ushort a;
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		bitStructDecl := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		structType := bitStructDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)

		assert.Equal(t, ast.StructTypeBitStruct, int(structType.Type))
		assert.Equal(t, true, structType.BackingType.IsSome())

		expect := &ast.StructField{
			NodeAttributes: aWithPos(2, 2, 2, 11),
			Names: []*ast.Ident{
				ast.NewIdentifierBuilder().
					WithName("a").
					WithStartEnd(2, 9, 2, 10).
					Build(),
			},
			Type: &ast.TypeInfo{
				NodeAttributes: aWithPos(2, 2, 2, 8),
				BuiltIn:        true,
				Identifier: ast.NewIdentifierBuilder().
					WithName("ushort").
					WithStartEnd(2, 2, 2, 8).
					Build(),
			},
			BitRange: option.None[[2]uint](),
		}
		assert.Equal(t, expect, structType.Fields[0])
	})
}

func TestConvertToAST_fault_decl(t *testing.T) {
	t.Run("parses fault", func(t *testing.T) {
		source := `module x;
	<* abc *>
	fault IOResult
	{
	  IO_ERROR,
	  PARSE_ERROR
	};`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		faultDecl := tree.Modules[0].Declarations[0].(*ast.FaultDecl)

		assert.Equal(
			t,
			ast.NewIdentifierBuilder().
				WithName("IOResult").
				WithStartEnd(2, 7, 2, 15).
				Build(),
			faultDecl.Name,
		)
		assert.Equal(t, lsp.Position{2, 1}, faultDecl.NodeAttributes.Range.Start)
		assert.Equal(t, lsp.Position{6, 2}, faultDecl.NodeAttributes.Range.End)
		assert.Equal(t, "abc", faultDecl.DocComment.Get().GetBody())

		assert.Equal(t, false, faultDecl.BackingType.IsSome())
		assert.Equal(t, 2, len(faultDecl.Members))

		assert.Equal(t,
			&ast.FaultMember{
				Name: ast.NewIdentifierBuilder().
					WithName("IO_ERROR").
					WithStartEnd(4, 3, 4, 11).
					Build(),
				NodeAttributes: ast.NewNodeAttributesBuilder().
					WithRangePositions(4, 3, 4, 11).
					Build(),
			},
			faultDecl.Members[0],
		)

		assert.Equal(t,
			&ast.FaultMember{
				Name: ast.NewIdentifierBuilder().
					WithName("PARSE_ERROR").
					WithStartEnd(5, 3, 5, 14).
					Build(),
				NodeAttributes: ast.NewNodeAttributesBuilder().
					WithRangePositions(5, 3, 5, 14).
					Build(),
			},
			faultDecl.Members[1],
		)
	})
}

func TestConvertToAST_def_declares(t *testing.T) {
	t.Run("parses def: global var alias", func(t *testing.T) {
		source := `
		int global_var = 10;
		def aliased_global = global_var;
		def aliased_global2 = ext::global_var;
		`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		// aliased global
		def := tree.Modules[0].Declarations[1].(*ast.GenDecl)
		assert.Equal(t, "aliased_global", def.Spec.(*ast.DefSpec).Name.Name)
		line := uint(2)
		assert.Equal(t, lsp.NewRange(line, 2, line, 34), def.Range)
		assert.Equal(t, lsp.NewRange(line, 6, line, 20), def.Spec.(*ast.DefSpec).Name.Range)
		assert.Equal(t, "global_var", def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Name)
		assert.Equal(t, lsp.NewRange(line, 23, line, 33), def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Range)

		// aliased imported global
		line = uint(3)
		def = tree.Modules[0].Declarations[2].(*ast.GenDecl)
		assert.Equal(t, "aliased_global2", def.Spec.(*ast.DefSpec).Name.Name)
		assert.Equal(t, lsp.NewRange(line, 2, line, 40), def.Range)
		assert.Equal(t, lsp.NewRange(line, 6, line, 21), def.Spec.(*ast.DefSpec).Name.Range)
		assert.Equal(t, "global_var", def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Name)
		assert.Equal(t, "ext::global_var", def.Spec.(*ast.DefSpec).Value.(*ast.Ident).String())
		assert.Equal(t, lsp.NewRange(line, 24, line, 39), def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Range)
	})

	t.Run("parses def: const alias", func(t *testing.T) {
		source := `
		const int MY_CONST = 5;
		def CONST_ALIAS = MY_CONST;
		def CONST_ALIAS2 = app::MY_CONST;
		`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		// aliased const
		line := uint(2)
		def := tree.Modules[0].Declarations[1].(*ast.GenDecl)
		assert.Equal(t, "CONST_ALIAS", def.Spec.(*ast.DefSpec).Name.Name)
		assert.Equal(t, lsp.NewRange(line, 2, line, 29), def.Range)
		assert.Equal(t, lsp.NewRange(line, 6, line, 17), def.Spec.(*ast.DefSpec).Name.Range)
		assert.Equal(t, "MY_CONST", def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Name)
		assert.Equal(t, lsp.NewRange(line, 20, line, 28), def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Range)

		// exported const
		line = uint(3)
		def = tree.Modules[0].Declarations[2].(*ast.GenDecl)
		assert.Equal(t, "CONST_ALIAS2", def.Spec.(*ast.DefSpec).Name.Name)
		assert.Equal(t, lsp.NewRange(line, 2, line, 35), def.Range)
		assert.Equal(t, lsp.NewRange(line, 6, line, 18), def.Spec.(*ast.DefSpec).Name.Range)
		assert.Equal(t, "MY_CONST", def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Name)
		assert.Equal(t, "app", def.Spec.(*ast.DefSpec).Value.(*ast.Ident).ModulePath.Name)
		assert.Equal(t, lsp.NewRange(line, 21, line, 34), def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Range)
	})

	t.Run("parses def: func alias with generics", func(t *testing.T) {
		source := `
		def func = a(<String,int>);`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		line := uint(1)
		def := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		assert.Equal(t, "func", def.Spec.(*ast.DefSpec).Name.Name)
		assert.Equal(t, lsp.NewRange(line, 2, line, 29), def.Range)
		assert.Equal(t, lsp.NewRange(line, 6, line, 10), def.Spec.(*ast.DefSpec).Name.Range)
		assert.Equal(t, "a", def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Name)
		assert.Equal(t, lsp.NewRange(line, 13, line, 14), def.Spec.(*ast.DefSpec).Value.(*ast.Ident).Range)
		assert.Equal(t, 2, len(def.Spec.(*ast.DefSpec).GenericParameters))
	})

	t.Run("parses def: alias to imported struct method", func(t *testing.T) {
		source := `
		def x = Type.hello;
		def y = app::Type.hello;
		`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		line := uint(1)
		def := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		assert.Equal(t, "x", def.Spec.(*ast.DefSpec).Name.Name)
		assert.Equal(t, lsp.NewRange(line, 2, line, 21), def.Range)
		assert.Equal(t, lsp.NewRange(line, 6, line, 7), def.Spec.(*ast.DefSpec).Name.Range)

		selector := def.Spec.(*ast.DefSpec).Value.(*ast.SelectorExpr)
		assert.Equal(t, "hello", selector.Sel.Name)
		assert.Equal(t, lsp.NewRange(line, 15, line, 20), selector.Sel.Range)
		assert.Equal(t, "Type", selector.X.(*ast.Ident).String())
		assert.Equal(t, lsp.NewRange(line, 10, line, 14), selector.X.(*ast.Ident).Range)

		// Imported struct method
		line = uint(2)
		def = tree.Modules[0].Declarations[1].(*ast.GenDecl)
		assert.Equal(t, "y", def.Spec.(*ast.DefSpec).Name.Name)
		assert.Equal(t, lsp.NewRange(line, 2, line, 26), def.Range)
		assert.Equal(t, lsp.NewRange(line, 6, line, 7), def.Spec.(*ast.DefSpec).Name.Range)

		selector = def.Spec.(*ast.DefSpec).Value.(*ast.SelectorExpr)
		assert.Equal(t, "hello", selector.Sel.Name)
		assert.Equal(t, "app::Type", selector.X.(*ast.Ident).String())
		assert.Equal(t, lsp.NewRange(line, 10, line, 19), selector.X.(*ast.Ident).Range)
	})

	t.Run("parses def declaring function", func(t *testing.T) {
		source := `def Kilo = fn void (int);`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		declaration := tree.Modules[0].Declarations[0].(*ast.GenDecl)
		assert.Equal(t,
			&ast.FuncType{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(0, 11, 0, 24).Build(),
				ReturnType: ast.NewTypeInfoBuilder().
					WithName("void").
					WithStartEnd(0, 14, 0, 18).
					WithNameStartEnd(0, 14, 0, 18).
					IsBuiltin().
					Build(),
				Params: []*ast.FunctionParameter{
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
			}, declaration.Spec.(*ast.DefSpec).Value)
	})
}

func TestConvertToAST_function_declaration(t *testing.T) {
	t.Run("parses function declaration", func(t *testing.T) {
		source := `module foo;
	<*
		Hello world.
		Hello world.
		@pure
		@param [in] pointer
		@require number > 0, number < 1000 : "invalid number"
		@ensure return == 1
	*>
	fn void test() {
		return 1;
	}`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)
		assert.Equal(t, lsp.Position{Line: 9, Column: 1}, fnDecl.NodeAttributes.Range.Start)
		assert.Equal(t, lsp.Position{Line: 11, Column: 2}, fnDecl.NodeAttributes.Range.End)
		assert.Equal(t, `Hello world.
Hello world.`, fnDecl.NodeAttributes.DocComment.Get().GetBody())
		assert.Equal(t, `Hello world.
Hello world.

**@pure**

**@param** [in] pointer

**@require** number > 0, number < 1000 : "invalid number"

**@ensure** return == 1`, fnDecl.NodeAttributes.DocComment.Get().DisplayBodyWithContracts())

		assert.Equal(t, "test", fnDecl.Signature.Name.Name, "Function name")
		assert.Equal(t, lsp.Position{Line: 9, Column: 9}, fnDecl.Signature.Name.NodeAttributes.Range.Start)
		assert.Equal(t, lsp.Position{Line: 9, Column: 13}, fnDecl.Signature.Name.NodeAttributes.Range.End)

		assert.Equal(t, "void", fnDecl.Signature.ReturnType.Identifier.Name, "Return type")
		assert.Equal(t, lsp.Position{Line: 9, Column: 4}, fnDecl.Signature.ReturnType.NodeAttributes.Range.Start)
		assert.Equal(t, lsp.Position{Line: 9, Column: 8}, fnDecl.Signature.ReturnType.NodeAttributes.Range.End)
	})

	t.Run("parses function with simple doc comment", func(t *testing.T) {
		source := `<*
		abc

		def

		ghi
		jkl
		*>
		fn void test(int number, char ch, int* pointer) {
			return 1;
		}`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

		expectedDoc := `abc

def

ghi
jkl`
		assert.Equal(t, expectedDoc, fnDecl.NodeAttributes.DocComment.Get().GetBody())
		assert.Equal(t, expectedDoc, fnDecl.NodeAttributes.DocComment.Get().DisplayBodyWithContracts())
	})

	t.Run("parses function with only contracts", func(t *testing.T) {
		source := `<*
			@pure
			@param [in] pointer
			@require number > 0, number < 1000 : "invalid number"
			@ensure return == 1
		*>
		fn void test(int number, char ch, int* pointer) {
			return 1;
		}`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

		expectedDoc := `**@pure**

**@param** [in] pointer

**@require** number > 0, number < 1000 : "invalid number"

**@ensure** return == 1`
		assert.Equal(t, "", fnDecl.NodeAttributes.DocComment.Get().GetBody())
		assert.Equal(t, expectedDoc, fnDecl.NodeAttributes.DocComment.Get().DisplayBodyWithContracts())
	})

	t.Run("parses extern function declaration", func(t *testing.T) {
		source := `module foo;
	fn void init_window(int width, int height, char* title) @extern("InitWindow");`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

		assert.Equal(t, "init_window", fnDecl.Signature.Name.Name, "Function name")
		assert.Equal(t, lsp.Position{Line: 1, Column: 1}, fnDecl.NodeAttributes.Range.Start)
		assert.Equal(t, lsp.Position{Line: 1, Column: 79}, fnDecl.NodeAttributes.Range.End)
	})

	t.Run("parses function returning optional", func(t *testing.T) {
		// TODO TestConvertToAST_convert_type should already cover this
		source := `module foo;
	fn usz! test() {
		return 1;
	}`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

		assert.Equal(t, "usz", fnDecl.Signature.ReturnType.Identifier.Name, "Return type")
		assert.Equal(t, true, fnDecl.Signature.ReturnType.Optional, "Return type should be optional")
	})

	t.Run("parses function with arguments", func(t *testing.T) {
		source := `module foo;
	fn void test(int number, char ch, int* pointer) {
		return 1;
	}`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		fnDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

		assert.Equal(t, 3, len(fnDecl.Signature.Parameters))
		assert.Equal(t,
			&ast.FunctionParameter{
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
			&ast.FunctionParameter{
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
			&ast.FunctionParameter{
				Name: ast.NewIdentifierBuilder().WithName("pointer").WithStartEnd(1, 40, 1, 47).Build(),
				Type: ast.NewTypeInfoBuilder().
					WithStartEnd(1, 35, 1, 39).
					WithName("int").WithNameStartEnd(1, 35, 1, 38).
					IsBuiltin().
					IsPointer().
					Build(),
				NodeAttributes: ast.NewNodeAttributesBuilder().
					WithRangePositions(1, 35, 1, 47).Build(),
			},
			fnDecl.Signature.Parameters[2],
		)
	})
}

func TestConvertToAST_method_declaration(t *testing.T) {
	t.Run("basic method declaration", func(t *testing.T) {
		source := `module foo;
	fn Object* UserStruct.method(self, int* pointer) {
		return 1;
	}`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

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
		assert.Equal(t, lsp.Position{1, 11}, methodDecl.Signature.ReturnType.NodeAttributes.Range.End)

		assert.Equal(t, 2, len(methodDecl.Signature.Parameters))
		assert.Equal(t,
			&ast.FunctionParameter{
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
			&ast.FunctionParameter{
				Name: ast.NewIdentifierBuilder().WithName("pointer").WithStartEnd(1, 41, 1, 48).Build(),
				Type: ast.NewTypeInfoBuilder().
					WithName("int").WithNameStartEnd(1, 36, 1, 39).
					IsBuiltin().
					IsPointer().
					WithStartEnd(1, 36, 1, 40).
					Build(),
				NodeAttributes: ast.NewNodeAttributesBuilder().
					WithRangePositions(1, 36, 1, 48).Build(),
			},
			methodDecl.Signature.Parameters[1],
		)
	})

	t.Run("external module method declaration", func(t *testing.T) {
		source := `module foo;
	fn Object* foo2::UserStruct.method(self, int* pointer) {
		return 1;
	}`
		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")

		methodDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

		assert.Equal(t, "foo2", methodDecl.ParentTypeId.Get().ModulePath.Name)
		assert.Equal(t, "UserStruct", methodDecl.ParentTypeId.Get().Name)
	})
}

func TestConvertToAST_method_declaration_mutable(t *testing.T) {
	source := `module foo;
	fn Object* UserStruct.method(&self, int* pointer) {
		return 1;
	}`
	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
	methodDecl := tree.Modules[0].Declarations[0].(*ast.FunctionDecl)

	assert.Equal(t,
		&ast.FunctionParameter{
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
	<* abc *>
	interface MyInterface {
		fn void method1();
		fn int method2(char arg);
	}`

	cv := newTestAstConverter()
	tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
	interfaceDecl := tree.Modules[0].Declarations[0].(*ast.InterfaceDecl)

	assert.Equal(t, lsp.NewRange(2, 1, 5, 2), interfaceDecl.GetRange())
	assert.Equal(t, "abc", interfaceDecl.DocComment.Get().GetBody())
	assert.Equal(t, 2, len(interfaceDecl.Methods))
}

func TestConvertToAST_macro_decl(t *testing.T) {
	t.Run("parser macro decl", func(t *testing.T) {
		source := `module foo;
	<* abc *>
	macro m(x) {
    	return x + 2;
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

		startRow := uint(2)
		endRow := uint(4)
		assert.Equal(t, lsp.Position{startRow, 1}, macroDecl.Range.Start)
		assert.Equal(t, lsp.Position{endRow, 2}, macroDecl.Range.End)
		assert.Equal(t, "abc", macroDecl.DocComment.Get().GetBody())

		assert.Equal(t, "m", macroDecl.Signature.Name.Name)
		assert.Equal(t, lsp.Position{startRow, 7}, macroDecl.Signature.Name.Range.Start)
		assert.Equal(t, lsp.Position{startRow, 8}, macroDecl.Signature.Name.Range.End)

		assert.Equal(t, 1, len(macroDecl.Signature.Parameters))
		assert.Equal(
			t,
			&ast.FunctionParameter{
				NodeAttributes: ast.NewNodeAttributesBuilder().WithRangePositions(startRow, 9, startRow, 10).Build(),
				Name:           ast.NewIdentifierBuilder().WithName("x").WithStartEnd(startRow, 9, startRow, 10).Build(),
			},
			macroDecl.Signature.Parameters[0],
		)
	})

	t.Run("parse macro with return type", func(t *testing.T) {
		source := `
		<* docs *>
		macro int m(int x) {
			return x + 2;
		}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

		assert.Equal(t, "m", macroDecl.Signature.Name.Name)
		assert.Equal(t, "int", macroDecl.Signature.ReturnType.Identifier.Name)
		assert.Equal(t, lsp.NewRange(2, 8, 2, 11), macroDecl.Signature.ReturnType.GetRange())
	})

	t.Run("parse macro struct member", func(t *testing.T) {
		source := `macro Object* UserStruct.@method() {
			@body();
			return 1;
		}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

		assert.Equal(t, lsp.NewRange(0, 0, 3, 3), macroDecl.Range)
		assert.Equal(t, "@method", macroDecl.Signature.Name.Name)
		assert.Equal(t, "Object", macroDecl.Signature.ReturnType.Identifier.Name)
		assert.Equal(t, uint(1), macroDecl.Signature.ReturnType.Pointer)
		assert.Equal(t, lsp.NewRange(0, 6, 0, 13), macroDecl.Signature.ReturnType.GetRange())
		assert.True(t, macroDecl.Signature.ParentTypeId.IsSome())
		assert.Equal(t, "UserStruct", macroDecl.Signature.ParentTypeId.Get().Name)
	})

	t.Run("parse macro struct member with arguments", func(t *testing.T) {
		source := `macro Object* UserStruct.@method(self, int* pointer; @body) {
			@body();
			return 1;
		}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

		assert.Equal(t, 2, len(macroDecl.Signature.Parameters))
		assert.Equal(t, lsp.NewRange(0, 33, 0, 37), macroDecl.Signature.Parameters[0].Range)
		assert.Equal(t, "self", macroDecl.Signature.Parameters[0].Name.Name)
		assert.Equal(t, lsp.NewRange(0, 33, 0, 37), macroDecl.Signature.Parameters[0].Name.Range)
		assert.Nil(t, macroDecl.Signature.Parameters[0].Type)

		assert.Equal(t, lsp.NewRange(0, 39, 0, 51), macroDecl.Signature.Parameters[1].Range)
		assert.Equal(t, "pointer", macroDecl.Signature.Parameters[1].Name.Name)
		assert.Equal(t, lsp.NewRange(0, 44, 0, 51), macroDecl.Signature.Parameters[1].Name.Range)
		assert.Equal(t, "int", macroDecl.Signature.Parameters[1].Type.Identifier.Name)
		assert.Equal(t, lsp.NewRange(0, 39, 0, 43), macroDecl.Signature.Parameters[1].Type.Range)
	})

	t.Run("parse macro with trailing @body", func(t *testing.T) {
		source := `macro void method(int* pointer; @body) {
			@body();
			return 1;
		}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

		bodyParam := macroDecl.Signature.TrailingBlockParam
		assert.Equal(t, lsp.NewRange(0, 32, 0, 37), bodyParam.Range)
		assert.Equal(t, "@body", bodyParam.Name.Name)
	})

	t.Run("parse macro with trailing @body with params", func(t *testing.T) {
		source := `macro void method(int* pointer; @body(it)) {
			@body();
			return 1;
		}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

		bodyParam := macroDecl.Signature.TrailingBlockParam
		assert.Equal(t, lsp.NewRange(0, 32, 0, 41), bodyParam.Range)
		assert.Equal(t, "@body", bodyParam.Name.Name)
		assert.Equal(t, 1, len(macroDecl.Signature.TrailingBlockParam.Parameters))
	})

	t.Run("parse invalid macro signature", func(t *testing.T) {
		source := `
	<* docs *>
	macro fn void scary() {
	}`

		cv := newTestAstConverter()
		tree := cv.ConvertToAST(GetCST(source).RootNode(), source, "file.c3")
		macroDecl := tree.Modules[0].Declarations[0].(*ast.MacroDecl)

		assert.NotNil(t, macroDecl)
		assert.True(t, macroDecl.Error)
		assert.Nil(t, macroDecl.Signature.Name)
	})
}
