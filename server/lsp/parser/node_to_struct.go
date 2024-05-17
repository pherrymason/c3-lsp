package parser

import (
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
struct_declaration: $ => seq(

	$._struct_or_union,
	field('name', $.type_ident),
	optional($.interface_impl),
	optional($.attributes),
	field('body', $.struct_body),

),
_struct_or_union: _ => choice('struct', 'union'),
struct_body: $ => seq(

	  '{',
	  // NOTE Allowing empty struct to not be too strict.
	  repeat($.struct_member_declaration),
	  '}',
	),

struct_member_declaration: $ => choice(

	  seq(field('type', $.type), $.identifier_list, optional($.attributes), ';'),
	  seq($._struct_or_union, optional($.ident), optional($.attributes), field('body', $.struct_body)),
	  seq('bitstruct', optional($.ident), ':', $.type, optional($.attributes), field('body', $.bitstruct_body)),
	  seq('inline', field('type', $.type), optional($.ident), optional($.attributes), ';'),
	),
*/
func (p *Parser) nodeToStruct(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Struct {
	nameNode := node.ChildByFieldName("name")
	name := nameNode.Content(sourceCode)
	var interfaces []string
	isUnion := false

	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		switch child.Type() {
		case "union":
			isUnion = true
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
		}
	}

	// TODO parse attributes
	bodyNode := node.ChildByFieldName("body")
	structFields := make([]*idx.StructMember, 0)

	for i := uint32(0); i < bodyNode.ChildCount(); i++ {
		memberNode := bodyNode.Child(int(i))
		//fmt.Println("body child:", memberNode.Type())
		if memberNode.Type() != "struct_member_declaration" {
			continue
		}

		var fieldType string
		var identifiers []string
		var identifiersRange []idx.Range

		for x := uint32(0); x < memberNode.ChildCount(); x++ {
			n := memberNode.Child(int(x))
			//fmt.Println("child:", n.Type())
			switch n.Type() {
			case "type":
				fieldType = n.Content(sourceCode)
			case "identifier_list":
				for j := uint32(0); j < n.ChildCount(); j++ {
					identifiers = append(identifiers, n.Child(int(j)).Content(sourceCode))
					identifiersRange = append(identifiersRange,
						idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint()),
					)
				}
			case "attributes":
				// TODO
			case "bitstruct_body":
				bitStructs := p.nodeToBitStructMembers(n, moduleName, docId, sourceCode)
				structFields = append(structFields, bitStructs...)
			}
		}

		for y := 0; y < len(identifiers); y++ {
			structMember := idx.NewStructMember(
				identifiers[y],
				fieldType,
				option.None[[2]uint](),
				moduleName,
				docId,
				identifiersRange[y],
			)
			structFields = append(structFields, &structMember)
		}
	}

	var _struct idx.Struct
	if isUnion {
		_struct = idx.NewUnion(
			name,
			structFields,
			moduleName,
			docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
		)
	} else {
		_struct = idx.NewStruct(
			name,
			interfaces,
			structFields,
			moduleName,
			docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
		)
	}

	return _struct
}
