package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) nodeToStruct(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Struct {
	nameNode := node.Child(1)
	name := nameNode.Content(sourceCode)
	// TODO parse attributes
	bodyNode := node.Child(2)

	fields := make([]idx.StructMember, 0)

	for i := uint32(0); i < bodyNode.ChildCount(); i++ {
		child := bodyNode.Child(int(i))
		switch child.Type() {
		case "field_declaration":
			fieldName := child.ChildByFieldName("name").Content(sourceCode)
			fieldType := child.ChildByFieldName("type").Content(sourceCode)
			fields = append(fields, idx.NewStructMember(fieldName, fieldType, idx.NewRangeFromSitterPositions(child.StartPoint(), child.EndPoint())))

		case "field_struct_declaration":
		case "field_union_declaration":
		}
	}

	_struct := idx.NewStruct(name, fields, doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()))

	return _struct
}
