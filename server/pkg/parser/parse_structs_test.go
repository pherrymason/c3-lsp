package parser

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func TestFindsGlobalStructs(t *testing.T) {
	source := `
module x;
struct MyStruct (MyInterface, MySecondInterface) {
	int data;
	char key;
	raylib::Camera camera;
}

fn void MyStruct.init(&self)
{
	*self = {
		.data = 4,
	};
}
`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds struct", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Structs["MyStruct"]
		assert.Same(t, symbols.Get("x").Children()[0], found)
		assert.Equal(t, "MyStruct", found.GetName())
		assert.False(t, found.IsUnion())
		assert.Equal(t, idx.NewRange(2, 0, 6, 1), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(2, 7, 2, 15), found.GetIdRange())
	})

	t.Run("finds struct members", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Structs["MyStruct"]
		member := found.GetMembers()[0]
		assert.Equal(t, "data", member.GetName())
		assert.Equal(t, "int", member.GetType().GetName())
		assert.Equal(t, idx.NewRange(3, 5, 3, 9), member.GetIdRange())
		assert.Equal(t, "docId", member.GetDocumentURI())
		assert.Equal(t, "x", member.GetModuleString())
		assert.Same(t, found.Children()[0], member)

		member = found.GetMembers()[1]
		assert.Equal(t, "key", member.GetName())
		assert.Equal(t, "char", member.GetType().GetName())
		assert.Equal(t, idx.NewRange(4, 6, 4, 9), member.GetIdRange())
		assert.Equal(t, "docId", member.GetDocumentURI())
		assert.Equal(t, "x", member.GetModuleString())
		assert.Same(t, found.Children()[1], member)

		// Last member contains implicit module path. This should be reflected in its parsed Type
		member = found.GetMembers()[2]
		assert.Equal(t, "camera", member.GetName())
		assert.Equal(t, "Camera", member.GetType().GetName())
		assert.Equal(t, "raylib::Camera", member.GetType().GetFullQualifiedName())

		assert.Equal(t, idx.NewRange(5, 16, 5, 22), member.GetIdRange())
		assert.Equal(t, "docId", member.GetDocumentURI())
		assert.Equal(t, "x", member.GetModuleString())
		assert.Same(t, found.Children()[2], member)
	})

	t.Run("finds struct implementing interface", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Structs["MyStruct"]
		assert.Equal(t, "MyStruct", found.GetName())
		assert.Equal(t, []string{"MyInterface", "MySecondInterface"}, found.GetInterfaces())
	})
}

func TestParses_substructs(t *testing.T) {
	source := `
module x;
struct Foo {
  struct data {
    int print;
	char name;
  }
}
`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)

	found := symbols.Get("x").Structs["Foo"]
	assert.Same(t, symbols.Get("x").Children()[0], found)
	assert.Equal(t, "Foo", found.GetName())
	assert.False(t, found.IsUnion())
	assert.Equal(t, idx.NewRange(2, 0, 7, 1), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(2, 7, 2, 10), found.GetIdRange())

	member := found.GetMembers()[0]
	assert.Equal(t, "data", member.GetName())
	assert.Equal(t, true, member.IsStruct())
	assert.Equal(t, idx.NewRange(3, 9, 3, 13), member.GetIdRange())
	assert.Same(t, found.Children()[0], member)

	// Check inner subtype struct members
	substruct := member.Substruct().Get()
	member = substruct.GetMembers()[0]
	assert.Equal(t, "print", member.GetName())
	assert.Equal(t, "int", member.GetType().GetName())
	assert.Equal(t, idx.NewRange(4, 8, 4, 13), member.GetIdRange())
	assert.Same(t, substruct.Children()[0], member)

	member = substruct.GetMembers()[1]
	assert.Equal(t, "name", member.GetName())
	assert.Equal(t, "char", member.GetType().GetName())
	assert.Equal(t, idx.NewRange(5, 6, 5, 10), member.GetIdRange())
	assert.Same(t, substruct.Children()[1], member)
}

func TestParses_anonymous_substructs(t *testing.T) {
	source := `
module x;
struct Foo {
  struct {
    int print;
  }
}
`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)

	found := symbols.Get("x").Structs["Foo"]
	assert.Same(t, symbols.Get("x").Children()[0], found)
	assert.Equal(t, "Foo", found.GetName())
	assert.False(t, found.IsUnion())
	assert.Equal(t, idx.NewRange(2, 0, 6, 1), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(2, 7, 2, 10), found.GetIdRange())

	member := found.GetMembers()[0]
	assert.Equal(t, "print", member.GetName())
	assert.Equal(t, "int", member.GetType().GetName())
	assert.Equal(t, idx.NewRange(4, 8, 4, 13), member.GetIdRange())
	assert.Same(t, found.Children()[0], member)
}

func TestParse_struct_with_anonymous_bitstructs(t *testing.T) {
	source := `module x;
	def Register16 = UInt16;
	struct Registers {
		bitstruct : Register16 @overlap {
			Register16 bc : 0..15;
			Register b : 8..15;
			Register c : 0..7;
		}
		bitstruct : Register16 @overlap {
			Register16 de : 0..15;
			Register d : 8..15;
			Register e : 0..7;
		}
		Register16 sp;
		Register16 pc;
	}`
	doc := document.NewDocument("docId", source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)

	found := symbols.Get("x").Structs["Registers"]
	assert.Equal(t, "Registers", found.GetName())

	// Check field a
	cases := []struct {
		name      string
		fieldType string
		bitRange  option.Option[[2]uint]
		idRange   idx.Range
	}{
		{"bc", "Register16", option.Some([2]uint{0, 15}), idx.NewRange(5, 14, 5, 16)},
		{"b", "Register", option.Some([2]uint{8, 15}), idx.NewRange(6, 12, 6, 13)},
		{"c", "Register", option.Some([2]uint{0, 7}), idx.NewRange(7, 12, 7, 13)},
		{"de", "Register16", option.Some([2]uint{0, 15}), idx.NewRange(10, 14, 10, 16)},
		{"d", "Register", option.Some([2]uint{8, 15}), idx.NewRange(11, 12, 11, 13)},
		{"e", "Register", option.Some([2]uint{0, 7}), idx.NewRange(12, 12, 12, 13)},
		{"sp", "Register16", option.None[[2]uint](), idx.NewRange(13, 13, 13, 15)},
		{"pc", "Register16", option.None[[2]uint](), idx.NewRange(14, 13, 14, 15)},
	}

	members := found.GetMembers()
	assert.Equal(t, 8, len(members))
	for i, member := range members {
		if i >= len(cases) {
			assert.Fail(t, fmt.Sprintf("An unexpected member was found: %s", member.GetName()))
			break
		}

		assert.Same(t, found.Children()[i], member)
		assert.Equal(t, cases[i].name, member.GetName())
		assert.Equal(t, cases[i].fieldType, member.GetType().GetName())
		if cases[i].bitRange.IsSome() {
			bitRange := cases[i].bitRange.Get()
			assert.Equal(t, bitRange, member.GetBitRange())
		}
	}
}

func TestParse_struct_subtyping_members_should_be_flagged(t *testing.T) {
	t.Run("should resolve subtyping when substruct is in same file", func(t *testing.T) {
		source := `module x;
	struct Person {
		int age;
		String name;
	}
	struct ImportantPerson {
		inline Person person;
		String title;
	}`
		doc := document.NewDocument("docId", source)
		parser := createParser()

		symbols, _ := parser.ParseSymbols(&doc)
		module := symbols.Get("x")

		strukt, ok := module.Structs["ImportantPerson"]
		assert.True(t, ok)

		// Check ImportantPersons contains Person members
		members := strukt.GetMembers()

		assert.Equal(t, "person", members[0].GetName())
		assert.Equal(t, "Person", members[0].GetType().GetName())
		assert.True(t, members[0].IsInlinePendingToResolve(), "Member should be flagged to be inlined")
		assert.Equal(t, idx.NewRange(6, 16, 6, 22), members[0].GetIdRange(), "Identifier range is wrong")
	})

	t.Run("should not break when inline substruct still has no identifier written", func(t *testing.T) {
		source := `module x;
	struct Person {
		int age;
		String name;
	}
	struct ImportantPerson {
		inline Person
	}`
		doc := document.NewDocument("docId", source)
		parser := createParser()

		symbols, _ := parser.ParseSymbols(&doc)
		module := symbols.Get("x")

		_, ok := module.Structs["ImportantPerson"]
		assert.True(t, ok)
	})
}

func TestParse_Unions(t *testing.T) {
	source := `module x; 
	union MyUnion{
		short as_short;
		int as_int;
	}`
	doc := document.NewDocument("docId", source)
	parser := createParser()

	t.Run("parses union", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		module := symbols.Get("x")
		found := module.Structs["MyUnion"]
		assert.Equal(t, "MyUnion", found.GetName())
		assert.True(t, found.IsUnion())
		assert.Equal(t, idx.NewRange(1, 1, 4, 2), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(1, 7, 1, 14), found.GetIdRange())
		assert.Same(t, module.Children()[0], found)
	})
}

func TestParse_bitstructs(t *testing.T) {
	source := `module x;
	bitstruct Test : uint
	{
		ushort a : 0..15;
		ushort b : 16..31;
		bool c : 7;
	}`
	doc := document.NewDocument("docId", source)
	parser := createParser()

	t.Run("parses bitstruct", func(t *testing.T) {

		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Bitstructs["Test"]
		assert.Same(t, symbols.Get("x").Children()[0], found)
		assert.Equal(t, "Test", found.GetName())
		assert.Equal(t, "uint", found.Type().GetName())

		members := found.Members()
		assert.Equal(t, 3, len(members))

		// Check field a
		member := members[0]
		assert.Equal(t, "a", member.GetName())
		assert.Equal(t, "ushort", members[0].GetType().GetName())
		assert.Equal(t, [2]uint{0, 15}, members[0].GetBitRange())
		assert.Equal(t, idx.NewRange(3, 9, 3, 10), members[0].GetIdRange())
		assert.Same(t, found.Children()[0], member)

		// Check field b
		assert.Equal(t, "b", members[1].GetName())
		assert.Equal(t, "ushort", members[1].GetType().GetName())
		assert.Equal(t, [2]uint{16, 31}, members[1].GetBitRange())
		assert.Equal(t, idx.NewRange(4, 9, 4, 10), members[1].GetIdRange())
		assert.Same(t, found.Children()[1], members[1])

		// Check field c
		assert.Equal(t, "c", members[2].GetName())
		assert.Equal(t, "bool", members[2].GetType().GetName())
		assert.Equal(t, [2]uint{7}, members[2].GetBitRange())
		assert.Equal(t, idx.NewRange(5, 7, 5, 8), members[2].GetIdRange())
		assert.Same(t, found.Children()[2], members[2])
	})
}
