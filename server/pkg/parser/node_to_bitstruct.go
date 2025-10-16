package parser

import (
	"strconv"

	"github.com/pherrymason/c3-lsp/pkg/option"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) nodeToBitStruct(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Bitstruct {
	nameNode := node.ChildByFieldName("name")
	name := nameNode.Content(sourceCode)
	var interfaces []string
	var bakedType idx.Type

	fieldsNode := node.ChildByFieldName("body")
	structFields := p.nodeToBitStructMembers(fieldsNode, currentModule, docId, sourceCode)

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
			bakedType = p.typeNodeToType(child, currentModule, sourceCode)
		}
	}

	_struct := idx.NewBitstruct(
		name,
		bakedType,
		interfaces,
		structFields,
		currentModule.GetModuleString(),
		*docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	return _struct
}

func (p *Parser) nodeToBitStructMembers(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) []*idx.StructMember {

	structFields := []*idx.StructMember{}
	// node = bitstruct_body
	for i := 0; i < int(node.ChildCount()); i++ {
		bdefnode := node.Child(i)
		bType := bdefnode.Type()
		if bType == "bitstruct_member_declaration" {
			var memberType idx.Type
			var identity string
			if bdefnodeType := bdefnode.ChildByFieldName("type"); bdefnodeType != nil {
				memberType = p.typeNodeToType(bdefnodeType, currentModule, sourceCode)
			}
			for x := 0; x < int(bdefnode.ChildCount()); x++ {
				xNode := bdefnode.Child(x)
				//fmt.Println(xNode.Type())
				switch xNode.Type() {
				case "ident":
					identity = xNode.Content(sourceCode)
				}
			}

			bitRanges := [2]uint{}

			if bdefnode.ChildCount() >= 4 {
				lowBit, _ := strconv.ParseInt(bdefnode.Child(3).Content(sourceCode), 10, 32)
				bitRanges[0] = uint(lowBit)
			}

			if bdefnode.ChildCount() >= 6 {
				highBit, _ := strconv.ParseInt(bdefnode.Child(5).Content(sourceCode), 10, 32)
				bitRanges[1] = uint(highBit)
			}

			member := idx.NewStructMember(
				identity,
				memberType,
				option.Some(bitRanges),
				currentModule.GetModuleString(),
				*docId,
				idx.NewRangeFromTreeSitterPositions(bdefnode.Child(1).StartPoint(), bdefnode.Child(1).EndPoint()),
			)
			structFields = append(structFields, &member)
		}
	}

	return structFields
}
