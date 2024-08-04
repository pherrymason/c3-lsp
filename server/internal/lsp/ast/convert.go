package ast

import "C"

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

func GetCST(sourceCode string) *sitter.Node {
	return cst.GetParsedTreeFromString(sourceCode).RootNode()
}

func ConvertToAST(cstNode *sitter.Node, sourceCode string) File {
	source := []byte(sourceCode)

	var prg File

	if cstNode.Type() == "source_file" {
		prg = convertSourceFile(cstNode, source)
	}

	for i := 0; i < int(cstNode.ChildCount()); i++ {
		var lastMod *Module
		if len(prg.Modules) > 0 {
			lastMod = &prg.Modules[len(prg.Modules)-1]
		}

		node := cstNode.Child(i)
		switch node.Type() {
		case "module":
			prg.Modules = append(prg.Modules, convert_module(node, source))

		case "import_declaration":
			lastMod.Imports = append(lastMod.Imports, convert_imports(node, source)...)

		case "global_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_global_declaration(node, source))

		case "enum_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_enum_declaration(node, source))

		case "struct_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_struct_declaration(node, source))

		case "bitstruct_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_bitstruct_declaration(node, source))

		case "fault_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_fault_declaration(node, source))
		}
	}

	return prg
}

func convertSourceFile(node *sitter.Node, source []byte) File {
	file := File{}
	file.SetPos(node.StartPoint(), node.EndPoint())

	return file
}

func convert_module(node *sitter.Node, source []byte) Module {
	module := Module{}
	module.Name = node.ChildByFieldName("path").Content(source)
	module.SetPos(node.StartPoint(), node.EndPoint())

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "generic_parameters", "generic_module_parameters":
			for g := 0; g < int(child.ChildCount()); g++ {
				gn := child.Child(g)
				if gn.Type() == "type_ident" {
					genericName := gn.Content(source)
					module.GenericParameters = append(module.GenericParameters, genericName)
				}
			}
		case "attributes":
			for a := 0; a < int(child.ChildCount()); a++ {
				gn := child.Child(a)
				module.Attributes = append(module.Attributes, gn.Content(source))
			}
		}
	}

	return module
}

func convert_imports(node *sitter.Node, source []byte) []string {
	imports := []string{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)

		switch n.Type() {
		case "path_ident":
			temp_mod := ""
			for m := 0; m < int(n.ChildCount()); m++ {
				sn := n.Child(m)
				if sn.Type() == "ident" || sn.Type() == "module_resolution" {
					temp_mod += sn.Content(source)
				}
			}
			imports = append(imports, temp_mod)
		}
	}

	return imports
}

func convert_global_declaration(node *sitter.Node, source []byte) VariableDecl {
	variable := VariableDecl{
		Names: []Identifier{},
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	for i := uint32(0); i < node.ChildCount(); i++ {
		n := node.Child(int(i))
		//fmt.Println(i, ":", n.Type(), ":: ", n.Content(sourceCode), ":: has errors: ", n.HasError())
		switch n.Type() {
		case "type":
			variable.Type = typeNodeToType(n, source)

		case "ident":
			variable.Names = append(
				variable.Names,
				Identifier{
					Name: n.Content(source),
					ASTNodeBase: NewBaseNodeBuilder().
						WithSitterPosRange(n.StartPoint(), n.EndPoint()).
						Build(),
				},
			)

		case ";":

		case "multi_declaration":
			for j := 0; j < int(n.ChildCount()); j++ {
				sub := n.Child(j)
				if sub.Type() == "ident" {
					variable.Names = append(
						variable.Names,
						Identifier{
							Name: sub.Content(source),
							ASTNodeBase: NewBaseNodeBuilder().
								WithSitterPosRange(sub.StartPoint(), sub.EndPoint()).
								Build(),
						},
					)
				}
			}

		case "integer_literal":
			/*
				initializer := &ast.Initializer{}
				initializer.SetPosition(child.StartPoint(), child.EndPoint())
				initializer.Value = child.Content()
				decl.Initializer = initializer
			*/
		}
	}

	return variable
}

func convert_enum_declaration(node *sitter.Node, sourceCode []byte) EnumDecl {
	enumDecl := EnumDecl{
		Name: node.ChildByFieldName("name").Content(sourceCode),
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "enum_spec":
			enumDecl.BaseType = typeNodeToType(n.Child(1), sourceCode)
			if n.ChildCount() >= 3 {
				param_list := n.Child(2)
				for p := 0; p < int(param_list.ChildCount()); p++ {
					paramNode := param_list.Child(p)
					if paramNode.Type() == "enum_param_declaration" {
						enumDecl.Properties = append(
							enumDecl.Properties,
							EnumProperty{
								ASTNodeBase: NewBaseNodeBuilder().
									WithSitterPosRange(paramNode.StartPoint(), paramNode.EndPoint()).
									Build(),
								Name: Identifier{
									Name: paramNode.Child(1).Content(sourceCode),
									ASTNodeBase: NewBaseNodeBuilder().
										WithSitterPosRange(paramNode.Child(1).StartPoint(), paramNode.Child(1).EndPoint()).
										Build(),
								},
								Type: typeNodeToType(paramNode.Child(0), sourceCode),
							},
						)
					}
				}
			}

		case "enum_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				enumeratorNode := n.Child(i)
				if enumeratorNode.Type() != "enum_constant" {
					continue
				}

				compositeLiteral := CompositeLiteral{}
				args := enumeratorNode.ChildByFieldName("args")
				if args != nil {
					for a := 0; a < int(args.ChildCount()); a++ {
						arg := args.Child(a)
						if arg.Type() == "arg" {
							compositeLiteral.Values = append(compositeLiteral.Values,
								convert_literal(arg.Child(0), sourceCode),
							)
						}
					}
				}

				name := enumeratorNode.ChildByFieldName("name")
				enumDecl.Members = append(enumDecl.Members,
					EnumMember{
						Name: Identifier{
							Name: name.Content(sourceCode),
							ASTNodeBase: NewBaseNodeBuilder().
								WithSitterPosRange(name.StartPoint(), name.EndPoint()).
								Build(),
						},
						Value: compositeLiteral,
						ASTNodeBase: NewBaseNodeBuilder().
							WithSitterPosRange(enumeratorNode.StartPoint(), enumeratorNode.EndPoint()).
							Build(),
					},
				)

			}
		}
	}

	return enumDecl
}

func convert_struct_declaration(node *sitter.Node, sourceCode []byte) StructDecl {
	structDecl := StructDecl{
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
		StructType: StructTypeNormal,
	}

	structDecl.Name = node.ChildByFieldName("name").Content(sourceCode)
	//membersNeedingSubtypingResolve := []string{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "union":
			structDecl.StructType = StructTypeUnion
		case "interface_impl":
			for x := 0; x < int(child.ChildCount()); x++ {
				n := child.Child(x)
				if n.Type() == "interface" {
					structDecl.Implements = append(structDecl.Implements, n.Content(sourceCode))
				}
			}
		case "attributes":
			// TODO attributes
		}
	}

	// TODO parse attributes
	bodyNode := node.ChildByFieldName("body")

	// Search Struct members
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		memberNode := bodyNode.Child(i)
		isInline := false

		//fmt.Println("body child:", memberNode.Type())
		if memberNode.Type() != "struct_member_declaration" {
			continue
		}
		fmt.Printf("%d - %s\n", i, memberNode.Content(sourceCode))

		fieldType := TypeInfo{}
		member := StructMemberDecl{
			ASTNodeBase: NewBaseNodeBuilder().
				WithSitterPosRange(memberNode.StartPoint(), memberNode.EndPoint()).
				Build(),
		}

		for x := 0; x < int(memberNode.ChildCount()); x++ {
			n := memberNode.Child(x)

			switch n.Type() {
			case "type":
				fieldType = typeNodeToType(n, sourceCode)
				member.Type = fieldType
				//fmt.Println(fieldType, n.Content(sourceCode))

				//fieldType = n.Content(sourceCode)
				if isInline {
					//	identifier = "dummy-subtyping"
				}
			case "identifier_list":
				for j := 0; j < int(n.ChildCount()); j++ {
					member.Names = append(member.Names,
						Identifier{
							ASTNodeBase: NewBaseNodeBuilder().WithSitterPosRange(n.Child(j).StartPoint(), n.Child(j).EndPoint()).Build(),
							Name:        n.Child(j).Content(sourceCode),
						},
					) /*
						identifiers = append(identifiers, n.Child(j).Content(sourceCode))
						identifiersRange = append(identifiersRange,
							idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint()),
						)*/
				}
			case "attributes":
				// TODO
			case "bitstruct_body":
				bitStructsMembers := convert_bitstruct_members(n, sourceCode)
				structDecl.Members = append(structDecl.Members, bitStructsMembers...)
				//structFields = append(structFields, bitStructsMembers...)

			case "inline":
				//isInline = true
				//fmt.Println("inline!: ", n.Content(sourceCode))
				//inlinedSubTyping = append(inlinedSubTyping, "1")
				member.IsInlined = true

			case "ident":
				member.Names = append(member.Names,
					Identifier{
						ASTNodeBase: NewBaseNodeBuilder().WithSitterPosRange(n.StartPoint(), n.EndPoint()).Build(),
						Name:        n.Content(sourceCode),
					},
				) /*
					identifier = n.Content(sourceCode)
					identifiersRange = append(identifiersRange,
						idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint()),
					)*/
			}
		}

		/*
			if len(identifiers) > 0 {
				for y := 0; y < len(identifiers); y++ {
					structMember := idx.NewStructMember(
						identifiers[y],
						fieldType, // TODO <--- this type parsing is too simple
						option.None[[2]uint](),
						currentModule.GetModuleString(),
						docId,
						identifiersRange[y],
					)
					structFields = append(structFields, &structMember)
				}
			} else if isInline {
				var structMember idx.StructMember
				membersNeedingSubtypingResolve = append(membersNeedingSubtypingResolve, fieldType)
				structMember = idx.NewInlineSubtype(
					identifier,
					fieldType,
					currentModule.GetModuleString(),
					docId,
					identifiersRange[0],
				)
				structFields = append(structFields, &structMember)
			} else if len(identifier) > 0 {
				structMember := idx.NewStructMember(
					identifier,
					fieldType,
					option.None[[2]uint](),
					currentModule.GetModuleString(),
					docId,
					identifiersRange[0],
				)

				structFields = append(structFields, &structMember)
			}*/

		if len(member.Names) > 0 {
			structDecl.Members = append(structDecl.Members, member)
		}
	}

	return structDecl
}

func convert_bitstruct_declaration(node *sitter.Node, sourceCode []byte) StructDecl {
	structDecl := StructDecl{
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPosRange(node.StartPoint(), node.EndPoint()).Build(),
		StructType:  StructTypeBitStruct,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		//fmt.Println("type:", child.Type(), child.Content(sourceCode))

		switch child.Type() {
		case "interface_impl":
			// TODO
			for x := 0; x < int(child.ChildCount()); x++ {
				n := child.Child(x)
				if n.Type() == "interface" {
					structDecl.Implements = append(structDecl.Implements, n.Content(sourceCode))
				}
			}

		case "attributes":
			// TODO attributes

		case "type":
			structDecl.BackingType = option.Some(typeNodeToType(child, sourceCode))

		case "bitstruct_body":
			structDecl.Members = convert_bitstruct_members(child, sourceCode)
		}
	}

	return structDecl
}

func convert_bitstruct_members(node *sitter.Node, sourceCode []byte) []StructMemberDecl {
	members := []StructMemberDecl{}
	for i := 0; i < int(node.ChildCount()); i++ {
		bdefnode := node.Child(i)
		bType := bdefnode.Type()
		member := StructMemberDecl{
			ASTNodeBase: NewBaseNodeBuilder().
				WithSitterPosRange(bdefnode.StartPoint(), bdefnode.EndPoint()).
				Build(),
		}

		if bType == "bitstruct_def" {
			for x := 0; x < int(bdefnode.ChildCount()); x++ {
				xNode := bdefnode.Child(x)
				//fmt.Println(xNode.Type())
				switch xNode.Type() {
				case "base_type":
					// Note: here we consciously pass bdefnode because typeNodeToType expects a child node of base_type. If we send xNode it will not find it.
					member.Type = typeNodeToType(bdefnode, sourceCode)
				case "ident":
					member.Names = append(
						member.Names,
						NewIdentifierBuilder().
							WithName(xNode.Content(sourceCode)).
							WithSitterPos(xNode).
							Build(),
					)
				}
			}

			bitRanges := [2]uint{}
			lowBit, _ := strconv.ParseInt(bdefnode.Child(3).Content(sourceCode), 10, 32)
			bitRanges[0] = uint(lowBit)

			if bdefnode.ChildCount() >= 6 {
				highBit, _ := strconv.ParseInt(bdefnode.Child(5).Content(sourceCode), 10, 32)
				bitRanges[1] = uint(highBit)
			}
			member.BitRange = option.Some(bitRanges)

			/*member := idx.NewStructMember(
				identity,
				memberType,
				option.Some(bitRanges),
				currentModule.GetModuleString(),
				docId,
				idx.NewRangeFromTreeSitterPositions(bdefnode.Child(1).StartPoint(), bdefnode.Child(1).EndPoint()),
			)*/
			members = append(members, member)
		} else if bType == "_bitstruct_simple_defs" {
			// Could not make examples with these to parse.
		}
	}

	return members
}

func convert_fault_declaration(node *sitter.Node, sourceCode []byte) Expression {
	// TODO parse attributes

	baseType := option.None[TypeInfo]() // TODO Parse type!
	var constants []FaultMember

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "fault_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				constantNode := n.Child(i)

				if constantNode.Type() == "const_ident" {
					constants = append(constants,
						FaultMember{
							Name: NewIdentifierBuilder().
								WithName(constantNode.Content(sourceCode)).
								WithSitterPos(constantNode).
								Build(),
							ASTNodeBase: NewBaseNodeBuilder().
								WithSitterPosRange(constantNode.StartPoint(), constantNode.EndPoint()).
								Build(),
						},
					)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	fault := FaultDecl{
		Name: NewIdentifierBuilder().
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		BackingType: baseType,
		Members:     constants,
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	return fault
}

func convert_literal(node *sitter.Node, sourceCode []byte) Expression {
	var literal Expression

	switch node.Type() {
	case "string_literal", "char_literal":
		literal = Literal{Value: node.Child(1).Content(sourceCode)}
	case "integer_literal", "real_literal":
		/*
			for i := 0; i < int(node.ChildCount()); i++ {
				fmt.Printf("Literal type not supported: %s\n", node.Child(i).Type())
			}
			fmt.Printf("Literal value: %s\n", node.Content(sourceCode))*/
		literal = Literal{
			Value: node.Content(sourceCode),
		}

	case "false":
		literal = BoolLiteral{Value: false}

	case "true":
		literal = BoolLiteral{Value: true}
	default:
		panic(fmt.Sprintf("Literal type not supported: %s\n", node.Type()))
	}

	return literal
}

func typeNodeToType(node *sitter.Node, sourceCode []byte) TypeInfo {
	if node.Type() == "optional_type" {
		return extTypeNodeToType(node.Child(0), true, sourceCode)
	}

	return extTypeNodeToType(node, false, sourceCode)
}

func extTypeNodeToType(
	node *sitter.Node,
	isOptional bool,
	sourceCode []byte,
) TypeInfo {
	/*
		baseTypeLanguage := false
		baseType := ""
		modulePath := ""
		generic_arguments := []TypeInfo{}
		pointerCount := 0*/

	typeInfo := TypeInfo{
		Optional: isOptional,
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//fmt.Println(n.Type(), n.Content(sourceCode))
		switch n.Type() {
		case "base_type":
			typeInfo.ASTNodeBase = NewBaseNodeBuilder().
				WithSitterPosRange(n.StartPoint(), n.EndPoint()).
				Build()

			for b := 0; b < int(n.ChildCount()); b++ {
				bn := n.Child(b)
				fmt.Println("---"+bn.Type(), bn.Content(sourceCode))

				switch bn.Type() {
				case "base_type_name":
					typeInfo.Identifier = NewIdentifierBuilder().
						WithName(bn.Content(sourceCode)).
						WithSitterPos(bn).
						Build()
					typeInfo.BuiltIn = true
				case "type_ident":
					typeInfo.Identifier = NewIdentifierBuilder().
						WithName(bn.Content(sourceCode)).
						WithSitterPos(bn).
						Build()
				case "generic_arguments":
					for g := 0; g < int(bn.ChildCount()); g++ {
						gn := bn.Child(g)
						if gn.Type() == "type" {
							gType := typeNodeToType(gn, sourceCode)
							typeInfo.Generics = append(typeInfo.Generics, gType)
						}
					}

				case "module_type_ident":
					//fmt.Println(bn)
					typeInfo.Identifier = NewIdentifierBuilder().
						WithPath(strings.Trim(bn.Child(0).Content(sourceCode), ":")).
						WithName(bn.Child(1).Content(sourceCode)).
						WithSitterPos(bn).
						Build()
				}
			}

		case "type_suffix":
			suffix := n.Content(sourceCode)
			if suffix == "*" {
				// TODO Only covers pointer to final value
				typeInfo.Pointer = 1
			}
		}
	}

	// Is baseType a module generic argument? Flag it.
	/*isGenericArgument := false
	for genericId, _ := range currentModule.GenericParameters {
		if genericId == baseType {
			isGenericArgument = true
		}
	}


	var parsedType symbols.Type
	if len(generic_arguments) == 0 {
		if isOptional {
			parsedType = symbols.NewOptionalType(baseTypeLanguage, baseType, pointerCount, isGenericArgument, modulePath)
		} else {
			parsedType = symbols.NewType(baseTypeLanguage, baseType, pointerCount, isGenericArgument, modulePath)
		}
	} else {
		// TODO Can a type with generic be itself a generic argument?
		parsedType = symbols.NewTypeWithGeneric(baseTypeLanguage, isOptional, baseType, pointerCount, generic_arguments, modulePath)
	}*/

	return typeInfo
}
