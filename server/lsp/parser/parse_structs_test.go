package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
)

func TestFindsGlobalStructs(t *testing.T) {
	source := `struct MyStruct{
	int data;
	char key;
}

fn void MyStruct.init(&self)
{
	*self = {
		.data = 4,
	};
}`

	module := "x"
	docId := "docId"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("finds struct", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		found := symbols.Structs["MyStruct"]
		assert.Equal(t, "MyStruct", found.GetName())
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

		member = found.GetMembers()[1]
		assert.Equal(t, "key", member.GetName())
		assert.Equal(t, "char", member.GetType())
		assert.Equal(t, idx.NewRange(2, 6, 2, 9), member.GetIdRange())
	})
}
