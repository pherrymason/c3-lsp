package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
)

func TestFindsGlobalStructs(t *testing.T) {
	source := `struct MyStruct (MyInterface, MySecondInterface) {
	int data;
	char key;
}

fn void MyStruct.init(&self)
{
	*self = {
		.data = 4,
	};
}
`

	module := "x"
	docId := "docId"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("finds struct", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		found := symbols.Structs["MyStruct"]
		assert.Equal(t, "MyStruct", found.GetName())
		assert.False(t, found.IsUnion())
		assert.Equal(t, idx.NewRange(0, 0, 3, 1), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(0, 7, 0, 15), found.GetIdRange())
	})

	t.Run("finds struct members", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		found := symbols.Structs["MyStruct"]
		member := found.GetMembers()[0]
		assert.Equal(t, "data", member.GetName())
		assert.Equal(t, "int", member.GetType())
		assert.Equal(t, idx.NewRange(1, 5, 1, 9), member.GetIdRange())
		assert.Equal(t, "docId", member.GetDocumentURI())
		assert.Equal(t, "x", member.GetModule())

		member = found.GetMembers()[1]
		assert.Equal(t, "key", member.GetName())
		assert.Equal(t, "char", member.GetType())
		assert.Equal(t, idx.NewRange(2, 6, 2, 9), member.GetIdRange())
		assert.Equal(t, "docId", member.GetDocumentURI())
		assert.Equal(t, "x", member.GetModule())
	})

	t.Run("finds struct implementing interface", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		found := symbols.Structs["MyStruct"]
		assert.Equal(t, "MyStruct", found.GetName())
		assert.Equal(t, []string{"MyInterface", "MySecondInterface"}, found.GetInterfaces())
	})
}

func TestParse_Unions(t *testing.T) {
	source := `union MyUnion{
		short as_short;
		int as_int;
	}`
	module := "x"
	docId := "docId"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("parses union", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		found := symbols.Structs["MyUnion"]
		assert.Equal(t, "MyUnion", found.GetName())
		assert.True(t, found.IsUnion())
		assert.Equal(t, idx.NewRange(0, 0, 3, 2), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(0, 6, 0, 13), found.GetIdRange())
	})
}
