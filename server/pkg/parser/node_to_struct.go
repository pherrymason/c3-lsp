package parser

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
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
func (p *Parser) nodeToStruct(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) (idx.Struct, []idx.Type) {
	nameNode := node.ChildByFieldName("name")
	name := nameNode.Content(sourceCode)
	var interfaces []string
	isUnion := false

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "union":
			isUnion = true
		case "interface_impl_list":
			// TODO
			for x := 0; x < int(child.ChildCount()); x++ {
				n := child.Child(x)
				if n.IsNamed() {
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

	structFields, membersNeedingSubtypingResolve := p.parse_struct_body(bodyNode, currentModule, docId, sourceCode)

	var _struct idx.Struct
	if isUnion {
		_struct = idx.NewUnion(
			name,
			structFields,
			currentModule.GetModuleString(),
			*docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
		)
	} else {
		_struct = idx.NewStruct(
			name,
			interfaces,
			structFields,
			currentModule.GetModuleString(),
			*docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
		)
	}

	return _struct, membersNeedingSubtypingResolve
}

func (p *Parser) parse_struct_body(bodyNode *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) ([]*idx.StructMember, []idx.Type) {
	structFields := make([]*idx.StructMember, 0)
	membersNeedingSubtypingResolve := []idx.Type{}

	// Iterate through struct_member_declaration nodes
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		memberNode := bodyNode.Child(i)
		isInline := false
		isSubStruct := false

		//fmt.Println("body child:", memberNode.Type())
		if memberNode.Type() != "struct_member_declaration" {
			continue
		}

		var fieldType idx.Type
		var identifiers []string
		var identifier string
		var identifiersRange []idx.Range
		var innerStructBody []*idx.StructMember
		var innerMembersNeedingSubtypingResolve []idx.Type

		// Iterate through children of struct_member_declaration
		for x := 0; x < int(memberNode.ChildCount()); x++ {
			n := memberNode.Child(x)
			// fmt.Println("child:", n.Type(), "::", memberNode.Content(sourceCode))
			switch n.Type() {
			case "type":
				fieldType = p.typeNodeToType(n, currentModule, sourceCode)
				if isInline {
					identifier = "dummy-subtyping"
				}

			case "identifier_list":
				for j := 0; j < int(n.ChildCount()); j++ {
					identifiers = append(identifiers, n.Child(j).Content(sourceCode))
					identifiersRange = append(identifiersRange,
						idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint()),
					)
				}
			case "attributes":
				// TODO
			case "bitstruct_body":
				bitStructsMembers := p.nodeToBitStructMembers(n, currentModule, docId, sourceCode)
				structFields = append(structFields, bitStructsMembers...)

			case "struct_body":
				isSubStruct = true
				innerStructBody, innerMembersNeedingSubtypingResolve = p.parse_struct_body(n, currentModule, docId, sourceCode)
				if len(identifier) == 0 {
					isInline = true
				}

				membersNeedingSubtypingResolve = append(membersNeedingSubtypingResolve, innerMembersNeedingSubtypingResolve...)

			case "inline":
				isInline = true

			case "ident":
				identifier = n.Content(sourceCode)
				identifiersRange = append(identifiersRange,
					idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint()),
				)
			}
		}

		if isSubStruct {
			if len(identifier) > 0 {
				structMember := idx.NewSubstructMember(identifier, innerStructBody, currentModule.GetModuleString(), *docId, identifiersRange[0])
				structFields = append(structFields, &structMember)
			} else {
				// Inline member
				for _, member := range innerStructBody {
					inlineMember := idx.NewInlineSubtype(
						member.GetName(),
						*member.GetType(),
						member.GetModuleString(),
						member.GetDocumentURI(),
						member.GetIdRange(),
					)
					structFields = append(structFields, &inlineMember)
				}
			}
		} else if len(identifiers) > 0 {
			for y := 0; y < len(identifiers); y++ {
				structMember := idx.NewStructMember(
					identifiers[y],
					fieldType, // TODO <--- this type parsing is too simple
					option.None[[2]uint](),
					currentModule.GetModuleString(),
					*docId,
					identifiersRange[y],
				)
				structFields = append(structFields, &structMember)
			}
		} else if isInline {
			if len(identifiersRange) > 0 {
				membersNeedingSubtypingResolve = append(membersNeedingSubtypingResolve, fieldType)
				structMember := idx.NewInlineSubtype(
					identifier,
					fieldType,
					currentModule.GetModuleString(),
					*docId,
					identifiersRange[0],
				)
				structFields = append(structFields, &structMember)
			}
		} else if len(identifier) > 0 {
			structMember := idx.NewStructMember(
				identifier,
				fieldType,
				option.None[[2]uint](),
				currentModule.GetModuleString(),
				*docId,
				identifiersRange[0],
			)

			structFields = append(structFields, &structMember)
		}
	}

	return structFields, membersNeedingSubtypingResolve
}
