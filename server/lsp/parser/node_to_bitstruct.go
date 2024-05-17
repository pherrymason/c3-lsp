package parser

import (
	"strconv"

	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) nodeToBitStruct(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Bitstruct {
	nameNode := node.ChildByFieldName("name")
	name := nameNode.Content(sourceCode)
	var interfaces []string
	bakedType := ""
	structFields := []*idx.StructMember{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		//fmt.Println("type:", child.Type(), child.Content(sourceCode))

		switch child.Type() {
		case "interface_impl":
			// TODO
			for x := 0; x < int(child.ChildCount()); x++ {
				n := child.Child(x)
				if n.Type() == "interface" {
					interfaces = append(interfaces, n.Content(sourceCode))
				}
			}
		case "attributes":
			// TODO attributes
		case "type":
			bakedType = child.Child(0).Content(sourceCode)
		case "bitstruct_body":
			structFields = p.nodeToBitStructMembers(child, moduleName, docId, sourceCode)
		}
	}

	_struct := idx.NewBitstruct(
		name,
		bakedType,
		interfaces,
		structFields,
		moduleName,
		docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	return _struct
}

func (p *Parser) nodeToBitStructMembers(node *sitter.Node, moduleName string, docId string, sourceCode []byte) []*idx.StructMember {

	structFields := []*idx.StructMember{}
	for j := int(0); j < int(node.ChildCount()); j++ {
		bdefnode := node.Child(j)
		bType := bdefnode.Type()
		if bType == "bitstruct_def" {
			bitRanges := [2]uint{}
			lowBit, _ := strconv.ParseInt(bdefnode.Child(3).Content(sourceCode), 10, 32)
			bitRanges[0] = uint(lowBit)

			if bdefnode.ChildCount() >= 6 {
				highBit, _ := strconv.ParseInt(bdefnode.Child(5).Content(sourceCode), 10, 32)
				bitRanges[1] = uint(highBit)
			}

			member := idx.NewStructMember(
				bdefnode.Child(1).Content(sourceCode),
				bdefnode.Child(0).Content(sourceCode),
				option.Some(bitRanges),
				moduleName,
				docId,
				idx.NewRangeFromTreeSitterPositions(bdefnode.Child(1).StartPoint(), bdefnode.Child(1).EndPoint()),
				//idx.NewRangeFromTreeSitterPositions(child.StartPoint(), child.EndPoint()),
			)
			structFields = append(structFields, &member)
		} else if bType == "_bitstruct_simple_defs" {
			// Could not make examples with these to parse.
		}
	}

	return structFields
}
