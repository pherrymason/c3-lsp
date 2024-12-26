package factory

import "C"

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"log"
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
)

var ConvertDebug bool = false

// statements not implemented yet and we want to ignore
var ignoreStatements = [2]string{
	"line_comment",
	"block_comment",
}

func GetCST(sourceCode string) *sitter.Node {
	return cst.GetParsedTreeFromString(sourceCode).RootNode()
}

func ConvertToAST(cstNode *sitter.Node, sourceCode string, fileName string) ast.File {
	source := []byte(sourceCode)

	var prg ast.File
	if cstNode.Type() == "source_file" {
		prg = *ast.NewFile(
			fileName,
			lsp.NewRangeFromSitterNode(cstNode),
			[]ast.Module{},
		)
	}

	anonymousModule := false
	for i := 0; i < int(cstNode.ChildCount()); i++ {
		node := cstNode.Child(i)
		parsedModules := len(prg.Modules)
		if parsedModules == 0 && node.Type() != "module" {
			anonymousModule = true
			prg.AddModule(
				*ast.NewModule(
					symbols.NormalizeModuleName(fileName),
					lsp.NewRangeFromSitterNode(node),
					&prg,
				),
			)
			parsedModules = len(prg.Modules)
		}

		var lastMod *ast.Module
		if parsedModules > 0 {
			lastMod = &prg.Modules[len(prg.Modules)-1]
		}

		switch node.Type() {
		case "module":
			if anonymousModule {
				anonymousModule = false
				lastMod.NodeAttributes.Range.End = lsp.Position{Line: uint(node.StartPoint().Row), Column: uint(node.StartPoint().Column)}
			}

			module := convert_module(node, source)
			prg.AddModule(module)

		case "import_declaration":
			anImport := convert_imports(node, source).(*ast.Import)
			lastMod.Imports = append(lastMod.Imports, anImport)

		case "global_declaration":
			variable := convert_global_declaration(node, source)
			lastMod.Declarations = append(lastMod.Declarations, &variable)

		case "enum_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_enum_declaration(node, source))

		case "struct_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_struct_declaration(node, source))

		case "bitstruct_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_bitstruct_declaration(node, source))

		case "fault_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_fault_declaration(node, source))

		case "const_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_const_declaration(node, source))

		case "define_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_def_declaration(node, source))

		case "func_definition", "func_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_function_declaration(node, source))

		case "interface_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_interface_declaration(node, source))

		case "macro_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_macro_declaration(node, source))
		}
	}

	return prg
}

func convertSourceFile(node *sitter.Node, source []byte) ast.File {
	file := ast.File{}
	//file.SetPos(node.StartPoint(), node.EndPoint())
	ast.ChangeNodePosition(&file.NodeAttributes, node.StartPoint(), node.EndPoint())

	return file
}

func convert_module(node *sitter.Node, source []byte) ast.Module {
	module := ast.NewModule(
		node.ChildByFieldName("path").Content(source),
		lsp.NewRangeFromSitterNode(node),
		nil,
	)

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

	return *module
}

func convert_imports(node *sitter.Node, source []byte) ast.Statement {
	imports := &ast.Import{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithRange(lsp.NewRangeFromSitterNode(node)).
			Build(),
		Path: node.ChildByFieldName("path").Content(source),
	}

	return imports
}

func convert_global_declaration(node *sitter.Node, source []byte) ast.GenDecl {
	variable := ast.GenDecl{
		Token: ast.VAR,
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithRange(lsp.NewRangeFromSitterNode(node)).
			Build(),
	}

	valueSpec := &ast.ValueSpec{}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//debugNode(n, source)
		switch n.Type() {
		case "type":
			valueSpec.Type = convert_type(n, source)

		case "ident":
			valueSpec.Names = append(valueSpec.Names, convert_ident(n, source).(*ast.Ident))

		case ";":
			/*
				case "multi_declaration":
					for j := 0; j < int(n.ChildCount()); j++ {
						sub := n.Child(j)
						if sub.TypeDescription() == "ident" {
							variable.Names = append(
								variable.Names,
								convert_ident(sub, source).(*Ident),
							)
						}
					}*/
		}
	}
	variable.Spec = valueSpec

	// Check for initializer
	// _assign_right_expr

	right := node.ChildByFieldName("right")
	if right != nil {
		expr := convert_expression(right, source).(ast.Expression)
		variable.Spec.(*ast.ValueSpec).Value = expr

		//variable.Initializer = convert_expression(right, source).(Expression)
	}

	return variable
}

func convert_enum_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	enumType := &ast.EnumType{
		Fields: []ast.Expression{},
	}
	spec := &ast.TypeSpec{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithRange(lsp.NewRangeFromSitterNode(node)).
			Build(),
		Name: ast.NewIdentifierBuilder().
			WithName(node.ChildByFieldName("name").Content(sourceCode)).
			BuildPtr(),
		TypeDescription: enumType,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "enum_spec":
			enumType.BaseType = option.Some(convert_type(n.Child(1), sourceCode))
			if n.ChildCount() >= 3 {
				param_list := n.Child(2)
				for p := 0; p < int(param_list.ChildCount()); p++ {
					paramNode := param_list.Child(p)
					if paramNode.Type() == "enum_param_declaration" {
						convertType := convert_type(paramNode.Child(0), sourceCode)
						enumType.Fields = append(
							enumType.Fields,
							&ast.Field{
								NodeAttributes: ast.NewNodeAttributesBuilder().
									WithSitterPos(paramNode).
									Build(),
								Name: ast.NewIdentifierBuilder().
									WithName(paramNode.Child(1).Content(sourceCode)).
									WithSitterPos(paramNode.Child(1)).
									BuildPtr(),
								Type: convertType,
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

				compositeLiteral := &ast.CompositeLiteral{}
				args := enumeratorNode.ChildByFieldName("args")
				if args != nil && args.ChildCount() > 0 {
					lastChild := args.Child(int(args.ChildCount()) - 1)
					if is_literal(lastChild) {
						compositeLiteral.Values = append(compositeLiteral.Values,
							convert_literal(lastChild, sourceCode),
						)
					} else if lastChild.Type() == "initializer_list" {
						for a := 0; a < int(lastChild.ChildCount()); a++ {
							arg := lastChild.Child(a)
							if arg.Type() == "arg" {
								if !is_literal(arg.Child(0)) {
									// Exit early to ensure correspondence between
									// index of each value and index of each predefined
									// enum parameter
									break
								}
								compositeLiteral.Values = append(compositeLiteral.Values,
									convert_literal(arg.Child(0), sourceCode),
								)
							}
						}
					}
				}

				nameNode := enumeratorNode.ChildByFieldName("name")
				enumType.Values = append(enumType.Values,
					&ast.EnumValue{
						Name: ast.NewIdentifierBuilder().
							WithName(nameNode.Content(sourceCode)).
							WithSitterRange(nameNode).
							BuildPtr(),
						Value: compositeLiteral,
						/*NodeAttributes: NewNodeAttributesBuilder().
						WithSitterStartEnd(enumeratorNode.StartPoint(), enumeratorNode.EndPoint()).
						Build(),*/
					},
				)

			}
		}
	}

	return &ast.GenDecl{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithRange(lsp.NewRangeFromSitterNode(node)).Build(),
		Token:          ast.Token(ast.ENUM),
		Spec:           spec,
	}
}

func convert_struct_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	structDecl := ast.StructDecl{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterPos(node).
			Build(),
		StructType: ast.StructTypeNormal,
	}

	structDecl.Name = node.ChildByFieldName("name").Content(sourceCode)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "union":
			structDecl.StructType = ast.StructTypeUnion
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

		//fmt.Println("body child:", memberNode.TypeDescription())
		if memberNode.Type() != "struct_member_declaration" {
			continue
		}
		//fmt.Printf("%d - %s\n", i, memberNode.Content(sourceCode))

		fieldType := ast.TypeInfo{}
		member := ast.StructMemberDecl{
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithSitterPos(memberNode).
				Build(),
		}

		for x := 0; x < int(memberNode.ChildCount()); x++ {
			n := memberNode.Child(x)
			//debugNode(n, sourceCode)

			switch n.Type() {
			case "type":
				fieldType = convert_type(n, sourceCode)
				member.Type = fieldType
			case "identifier_list":
				for j := 0; j < int(n.ChildCount()); j++ {
					member.Names = append(
						member.Names,
						*(convert_ident(n.Child(j), sourceCode).(*ast.Ident)),
					)
				}
			case "attributes":
				// TODO

			case "bitstruct_body":
				bitStructsMembers := convert_bitstruct_members(n, sourceCode)
				structDecl.Members = append(structDecl.Members, bitStructsMembers...)
				//structFields = append(structFields, bitStructsMembers...)

			case "inline":
				member.IsInlined = true

			case "ident":
				member.Names = append(member.Names,
					*(convert_ident(n, sourceCode).(*ast.Ident)),
				)
			}
		}

		if len(member.Names) > 0 {
			structDecl.Members = append(structDecl.Members, member)
		}
	}

	return &structDecl
}

func convert_bitstruct_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	structDecl := ast.StructDecl{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
		StructType:     ast.StructTypeBitStruct,
	}

	membersNode := node.ChildByFieldName("body")
	structDecl.Members = convert_bitstruct_members(membersNode, sourceCode)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		//fmt.Println("type:", child.TypeDescription(), child.Content(sourceCode))

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
			structDecl.BackingType = option.Some(convert_type(child, sourceCode))

		case "bitstruct_body":
			structDecl.Members = convert_bitstruct_members(child, sourceCode)
		}
	}

	return &structDecl
}

func convert_bitstruct_members(node *sitter.Node, source []byte) []ast.StructMemberDecl {
	members := []ast.StructMemberDecl{}
	for i := 0; i < int(node.ChildCount()); i++ {
		bdefnode := node.Child(i)
		//debugNode(bdefnode, source)
		bType := bdefnode.Type()
		member := ast.StructMemberDecl{
			NodeAttributes: ast.NewNodeAttributesBuilder().
				WithSitterPos(bdefnode).
				Build(),
		}

		if bType == "bitstruct_member_declaration" {
			for x := 0; x < int(bdefnode.ChildCount()); x++ {
				xNode := bdefnode.Child(x)
				//fmt.Println(xNode.TypeDescription())
				switch xNode.Type() {
				case "base_type":
					// Note: here we consciously pass bdefnode because typeNodeToType expects a child node of base_type. If we send xNode it will not find it.
					member.Type = convert_type(bdefnode, source)
				case "ident":
					member.Names = append(
						member.Names,
						*(convert_ident(xNode, source).(*ast.Ident)),
					)
				}
			}

			bitRanges := [2]uint{}

			if bdefnode.ChildCount() >= 4 {
				lowBit, _ := strconv.ParseInt(bdefnode.Child(3).Content(sourceCode), 10, 32)
				bitRanges[0] = uint(lowBit)
			}

			if bdefnode.ChildCount() >= 6 {
				highBit, _ := strconv.ParseInt(bdefnode.Child(5).Content(source), 10, 32)
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
		}
	}

	return members
}

func convert_fault_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	// TODO parse attributes

	baseType := option.None[ast.TypeInfo]() // TODO Parse type!
	var constants []ast.FaultMember

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "fault_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				constantNode := n.Child(i)

				if constantNode.Type() == "const_ident" {
					constants = append(constants,
						ast.FaultMember{
							Name: ast.NewIdentifierBuilder().
								WithName(constantNode.Content(sourceCode)).
								WithSitterPos(constantNode).
								Build(),
							NodeAttributes: ast.NewNodeAttributesBuilder().
								WithSitterPos(constantNode).
								Build(),
						},
					)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	fault := &ast.FaultDecl{
		Name: ast.NewIdentifierBuilder().
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		BackingType: baseType,
		Members:     constants,
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterPos(node).
			Build(),
	}

	return fault
}

func convert_const_declaration(node *sitter.Node, source []byte) ast.Declaration {
	constant := &ast.ConstDecl{
		Names: []*ast.Ident{},
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterPos(node).
			Build(),
	}

	var idNode *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type":
			constant.Type = option.Some(convert_type(n, source))

		case "const_ident":
			idNode = n

		case "attributes":
			// TODO
		}
	}

	constant.Names = append(constant.Names,
		ast.NewIdentifierBuilder().
			WithName(idNode.Content(source)).
			WithSitterPos(idNode).
			BuildPtr(),
	)

	right := node.ChildByFieldName("right")
	if right != nil {
		constant.Initializer = convert_expression(right, source).(ast.Expression)
	}

	return constant
}

/*
define_declaration [13, 0] - [13, 15]

	type_ident [13, 4] - [13, 8]
	typedef_type [13, 11] - [13, 14]
	type [13, 11] - [13, 14]
		base_type [13, 11] - [13, 14]
		base_type_name [13, 11] - [13, 14]
*/
func convert_def_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	defBuilder := ast.NewDefDeclBuilder().
		WithSitterPos(node)

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type_ident", "define_ident":
			defBuilder.WithName(n.Content(sourceCode)).
				WithIdentifierSitterPos(n)

		case "typedef_type":
			var _type ast.TypeInfo
			if n.Child(0).Type() == "type" {
				// Might contain module path
				_type = convert_type(n.Child(0), sourceCode)
				defBuilder.WithResolvesToType(_type)
			} else if n.Child(0).Type() == "func_typedef" {
				// TODO Parse full info of this func typedefinition
				defBuilder.WithResolvesTo(n.Content(sourceCode))
			}
		}
	}

	def := defBuilder.Build()
	return &def
}

func convert_interface_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	// TODO parse attributes
	var methods []ast.FunctionSignature
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "interface_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				m := n.Child(i)
				if m.Type() == "func_declaration" {
					fun := convert_function_signature(m, sourceCode)
					methods = append(methods, fun)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	_interface := &ast.InterfaceDecl{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
		Name:           ast.NewIdentifierBuilder().WithName(nameNode.Content(sourceCode)).WithSitterPos(nameNode).Build(),
		Methods:        methods,
	}

	return _interface
}

func convert_macro_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	var nameNode *sitter.Node

	var parameters []ast.FunctionParameter
	nodeParameters := node.Child(2)
	if nodeParameters.ChildCount() > 2 {
		for i := uint32(0); i < nodeParameters.ChildCount(); i++ {
			argNode := nodeParameters.Child(int(i))
			if argNode.Type() != "parameter" {
				continue
			}

			parameters = append(
				parameters,
				convert_function_parameter(argNode, option.None[ast.Ident](), sourceCode),
			)
		}
	}

	nameNode = node.Child(1).ChildByFieldName("name")
	macro := &ast.MacroDecl{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
		Signature: ast.MacroSignature{
			Name: ast.NewIdentifierBuilder().
				WithName(nameNode.Content(sourceCode)).
				WithSitterPos(nameNode).
				Build(),
			Parameters: parameters,
		},
	}
	/*
		if node.ChildByFieldName("body") != nil {
			variables := p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
			variables = append(arguments, variables...)
			macro.AddVariables(variables)
		}
	*/
	return macro
}

func is_literal(node *sitter.Node) bool {
	literals := []string{
		"string_literal", "char_literal", "raw_string_literal",
		"integer_literal", "real_literal",
		"bytes_literal",
		"true",
		"false",
		"null",
	}

	value := node.Type()
	for _, v := range literals {
		if v == value {
			return true
		}
	}
	return false
}

func convert_ident(node *sitter.Node, source []byte) ast.Expression {
	return ast.NewIdentifierBuilder().
		WithName(node.Content(source)).
		WithSitterRange(node).
		BuildPtr()
}

func convert_var_decl(node *sitter.Node, source []byte) ast.Declaration {
	//for i := 0; i < int(node.ChildCount()); i++ {

	//}
	decl := &ast.GenDecl{
		Token: ast.VAR,
	}

	return decl
}

func convert_type(node *sitter.Node, sourceCode []byte) ast.TypeInfo {
	typeInfo := ast.TypeInfo{
		Optional: false,
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterPos(node).
			Build(),
		Pointer: uint(0),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//fmt.Println(n.TypeDescription(), n.Content(sourceCode))
		switch n.Type() {
		case "base_type":
			typeInfo.NodeAttributes = ast.NewNodeAttributesBuilder().
				WithSitterPos(n).
				Build()

			for b := 0; b < int(n.ChildCount()); b++ {
				bn := n.Child(b)
				//fmt.Println("---"+bn.TypeDescription(), bn.Content(sourceCode))

				switch bn.Type() {
				case "base_type_name":
					typeInfo.Identifier = ast.NewIdentifierBuilder().
						WithName(bn.Content(sourceCode)).
						WithSitterPos(bn).
						WithSitterRange(bn).
						Build()
					typeInfo.BuiltIn = true
				case "type_ident":
					typeInfo.Identifier = ast.NewIdentifierBuilder().
						WithName(bn.Content(sourceCode)).
						WithSitterPos(bn).
						WithSitterRange(bn).
						Build()
				case "generic_arguments":
					for g := 0; g < int(bn.ChildCount()); g++ {
						gn := bn.Child(g)
						if gn.Type() == "type" {
							gType := convert_type(gn, sourceCode)
							typeInfo.Generics = append(typeInfo.Generics, gType)
						}
					}

				case "module_type_ident":
					//fmt.Println(bn)
					typeInfo.Identifier = ast.NewIdentifierBuilder().
						WithPath(strings.Trim(bn.Child(0).Content(sourceCode), ":")).
						WithName(bn.Child(1).Content(sourceCode)).
						WithSitterPos(bn).
						WithSitterRange(bn).
						Build()
				}
			}

		case "type_suffix":
			suffix := n.Content(sourceCode)
			if suffix == "*" {
				// TODO Only covers pointer to final value
				typeInfo.Pointer = 1
			}
		case "!":
			typeInfo.Optional = true
		}
	}
	return typeInfo
}

func convert_type2(node *sitter.Node, sourceCode []byte) ast.Expression {
	typeInfo := ast.TypeInfo{
		Optional: false,
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterPos(node).
			Build(),
		Pointer: uint(0),
	}
	//ident := &Ident{}
	//isBaseType := false

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//fmt.Println(n.TypeDescription(), n.Content(sourceCode))
		switch n.Type() {
		case "base_type":
			typeInfo.NodeAttributes = ast.NewNodeAttributesBuilder().
				WithSitterPos(n).
				WithRange(lsp.NewRangeFromSitterNode(n)).
				Build()

			for b := 0; b < int(n.ChildCount()); b++ {
				bn := n.Child(b)
				//fmt.Println("---"+bn.TypeDescription(), bn.Content(sourceCode))

				switch bn.Type() {
				case "base_type_name":
					/*ident = NewIdentifierBuilder().
						WithName(bn.Content(sourceCode)).
						WithSitterPos(bn).
						BuildPtr()
					isBaseType = true*/
					typeInfo.Identifier = ast.NewIdentifierBuilder().
						WithName(bn.Content(sourceCode)).
						WithSitterRange(bn).
						Build()
					typeInfo.BuiltIn = true
				case "type_ident":
					/*ident = NewIdentifierBuilder().
					WithName(bn.Content(sourceCode)).
					WithSitterPos(bn).
					BuildPtr()*/
					typeInfo.Identifier = ast.NewIdentifierBuilder().
						WithName(bn.Content(sourceCode)).
						WithSitterRange(bn).
						Build()
				case "generic_arguments":
					for g := 0; g < int(bn.ChildCount()); g++ {
						gn := bn.Child(g)
						if gn.Type() == "type" {
							gType := convert_type(gn, sourceCode)
							typeInfo.Generics = append(typeInfo.Generics, gType)
						}
					}

				case "module_type_ident":
					typeInfo.Identifier = ast.NewIdentifierBuilder().
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
		case "!":
			typeInfo.Optional = true
		}
	}

	return typeInfo
}

func convert_function_declaration(node *sitter.Node, source []byte) ast.Declaration {
	var typeIdentifier option.Option[ast.Ident]
	funcHeader := node.Child(1)
	//debugNode(funcHeader, source)

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = option.Some(ast.NewIdentifierBuilder().
			WithName(funcHeader.ChildByFieldName("method_type").Content(source)).
			WithSitterPos(funcHeader.ChildByFieldName("method_type")).
			Build())
	}
	signature := convert_function_signature(node, source)

	bodyNode := node.ChildByFieldName("body")
	var body ast.Node
	if bodyNode != nil {
		n := bodyNode.Child(0)
		// options
		// compound_stmt
		// => expr;
		if n.Type() == "compound_stmt" {
			body = convert_compound_stmt(n, source)
		} else {
			body = convert_expression(n.NextSibling(), source)
		}
	}

	funcDecl := &ast.FunctionDecl{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
		ParentTypeId:   typeIdentifier,
		Signature:      signature,
		Body:           body,
	}

	/*
		var variables []*idx.Variable
		if node.ChildByFieldName("body") != nil {
			variables = p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
		}

		variables = append(variables, parameters...)

		funcDecl.AddVariables(variables)
	*/
	return funcDecl
}

func convert_function_signature(node *sitter.Node, sourceCode []byte) ast.FunctionSignature {
	var typeIdentifier option.Option[ast.Ident]
	funcHeader := node.Child(1)
	nameNode := funcHeader.ChildByFieldName("name")

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = option.Some(ast.NewIdentifierBuilder().
			WithName(funcHeader.ChildByFieldName("method_type").Content(sourceCode)).
			WithSitterPos(funcHeader.ChildByFieldName("method_type")).
			Build())
	}

	signatureDecl := ast.FunctionSignature{
		Name: ast.NewIdentifierBuilder().
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		ReturnType: convert_type(funcHeader.ChildByFieldName("return_type"), sourceCode),
		Parameters: convert_function_parameter_list(node.Child(2), typeIdentifier, sourceCode),
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterPos(node).
			Build(),
	}

	return signatureDecl
}

func convert_function_parameter_list(node *sitter.Node, typeIdentifier option.Option[ast.Ident], source []byte) []ast.FunctionParameter {
	if node.Type() != "fn_parameter_list" {
		panic(
			fmt.Sprintf("Wrong node provided: Expected fn_parameter_list, provided %s", node.Type()),
		)
	}

	var parameters []ast.FunctionParameter
	if node.ChildCount() > 2 {
		for i := 0; i < int(node.ChildCount()); i++ {
			argNode := node.Child(i)
			if argNode.Type() != "parameter" {
				continue
			}

			parameters = append(
				parameters,
				convert_function_parameter(argNode, typeIdentifier, source),
			)
		}
	}

	return parameters
}

// nodeToArgument Very similar to nodeToVariable, but arguments have optional identifiers (for example when using `self` for struct methods)
/*
	_parameter: $ => choice(
      seq($.type, $.ident, optional($.attributes)),			// 3
      seq($.type, '...', $.ident, optional($.attributes)),	// 3/4
      seq($.type, '...', $.ct_ident),						// 3
      seq($.type, $.ct_ident),								// 2
      seq($.type, '...', optional($.attributes)),			// 2/3
      seq($.type, $.hash_ident, optional($.attributes)),	// 2/3
      seq($.type, '&', $.ident, optional($.attributes)),	// 3/4
      seq($.type, optional($.attributes)),					// 1/2
      seq('&', $.ident, optional($.attributes)),			// 2/3
      seq($.hash_ident, optional($.attributes)),			// 1/2
      '...',												// 1
      seq($.ident, optional($.attributes)),					// 1/2
      seq($.ident, '...', optional($.attributes)),			// 2/3
      $.ct_ident,											// 1
      seq($.ct_ident, '...'),								// 2
    ),
*/
func convert_function_parameter(argNode *sitter.Node, methodIdentifier option.Option[ast.Ident], sourceCode []byte) ast.FunctionParameter {
	var identifier ast.Ident
	var argType ast.TypeInfo
	ampersandFound := false

	for i := 0; i < int(argNode.ChildCount()); i++ {
		n := argNode.Child(int(i))

		switch n.Type() {
		case "&":
			ampersandFound = true

		case "type":
			argType = convert_type(n, sourceCode)
		case "ident":
			identifier = ast.NewIdentifierBuilder().
				WithName(n.Content(sourceCode)).
				WithSitterPos(n).
				Build()

			// When detecting a self, the type is the Struct type
			if identifier.Name == "self" && methodIdentifier.IsSome() {
				pointer := uint(0)
				if ampersandFound {
					pointer = 1
				}

				argType = ast.TypeInfo{
					Identifier: ast.NewIdentifierBuilder().
						WithName(methodIdentifier.Get().Name).
						WithSitterPos(n).
						Build(),
					Pointer:        pointer,
					NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(argNode).Build(),
				}
			}
		}
	}

	variable := ast.FunctionParameter{
		Name:           identifier,
		Type:           argType,
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(argNode).Build(),
	}

	return variable
}

func convert_lambda_declaration(node *sitter.Node, source []byte) ast.Expression {
	rType := option.None[ast.TypeInfo]()
	var parameters []ast.FunctionParameter

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type", "optional_type":
			r := convert_type(n, source)
			rType = option.Some[ast.TypeInfo](r)
		case "fn_parameter_list":
			parameters = convert_function_parameter_list(n, option.None[ast.Ident](), source)
		case "attributes":
			// TODO
		}
	}

	return &ast.LambdaDeclarationExpr{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
		ReturnType:     rType,
		Parameters:     parameters,
	}
}

func convert_lambda_declaration_with_body(node *sitter.Node, source []byte) ast.Expression {
	expression := convert_lambda_declaration(node, source)

	lambda := expression.(*ast.LambdaDeclarationExpr)
	lambda.Body = convert_compound_stmt(node.NextSibling(), source).(*ast.CompoundStmt)

	return lambda
}

func convert_lambda_expr(node *sitter.Node, source []byte) ast.Expression {
	expr := convert_lambda_declaration(node.Child(0), source)

	lambda := expr.(*ast.LambdaDeclarationExpr)

	bodyNode := node.Child(1).ChildByFieldName("body")

	lambda.NodeAttributes.Range.End.Column = uint(bodyNode.EndPoint().Column)
	lambda.NodeAttributes.Range.End.Line = uint(bodyNode.EndPoint().Row)
	expression := convert_expression(bodyNode, source).(ast.Expression)
	lambda.Body = &ast.ReturnStatement{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(bodyNode).Build(),
		Return:         option.Some(expression),
	}

	return lambda
}

/*
$.assignment_expr,
$.ternary_expr,
$.lambda_expr,
$.elvis_orelse_expr,
$.suffix_expr,
$.binary_expr,
$.unary_expr,
$.cast_expr,
$.rethrow_expr,
$.trailing_generic_expr,
$.update_expr,
$.call_expr,
$.subscript_expr,
$.initializer_list,
$._base_expr,
*/
func convert_expression(node *sitter.Node, source []byte) ast.Expression {
	ConvertDebug = false
	if ConvertDebug {
		fmt.Print("trying to convert_expression:\n")
		debugNode(node, source, "convert_expression")
	}

	converted := anyOf("_expr", []NodeRule{
		NodeOfType("assignment_expr"),
		NodeOfType("ternary_expr"),
		NodeChildWithSequenceOf([]NodeRule{
			NodeOfType("lambda_declaration"),
			NodeOfType("implies_body"),
		}, "lambda_expr"),
		NodeOfType("elvis_orelse_expr"),
		NodeOfType("optional_expr"),
		NodeOfType("binary_expr"),
		NodeOfType("unary_expr"),
		NodeOfType("cast_expr"),
		NodeOfType("rethrow_expr"),
		NodeOfType("trailing_generic_expr"),
		NodeOfType("update_expr"),
		NodeOfType("call_expr"),
		NodeOfType("subscript_expr"),
		NodeOfType("initializer_list"),
		NodeTryConversionFunc("_base_expr"),
	}, node, source, ConvertDebug)

	return converted.(ast.Expression)
}

func convert_expr_stmt(node *sitter.Node, source []byte) ast.Statement {
	expr := convert_expression(node, source)

	return &ast.ExpressionStmt{
		Expr: expr,
	}
}

func convert_base_expression(node *sitter.Node, source []byte) ast.Expression {
	//debugNode(node, source)

	found := anyOf("_base_expr", []NodeRule{
		NodeOfType("string_literal"),
		NodeOfType("char_literal"),
		NodeOfType("raw_string_literal"),
		NodeOfType("integer_literal"),
		NodeOfType("real_literal"),
		NodeOfType("bytes_literal"),
		NodeOfType("true"),
		NodeOfType("false"),
		NodeOfType("null"),

		NodeOfType("ident"),
		NodeOfType("ct_ident"),
		NodeOfType("hash_ident"),
		NodeOfType("const_ident"),
		NodeOfType("at_ident"),
		NodeOfType("module_ident_expr"),
		NodeOfType("bytes_expr"),
		NodeOfType("builtin"),
		NodeOfType("unary_expr"),
		NodeOfType("initializer_list"),
		NodeSiblingsWithSequenceOf([]NodeRule{
			NodeOfType("type"),
			NodeOfType("initializer_list"),
		}, "..type_with_initializer_list.."),

		NodeOfType("field_expr"),       // TODO
		NodeOfType("type_access_expr"), // TODO
		NodeOfType("paren_expr"),
		NodeOfType("expr_block"), // TODO

		NodeOfType("$vacount"),

		NodeOfType("$alignof"),
		NodeOfType("$extnameof"),
		NodeOfType("$nameof"),
		NodeOfType("$offsetof"),
		NodeOfType("$qnameof"),

		NodeOfType("$vaconst"),
		NodeOfType("$vaarg"),
		NodeOfType("$varef"),
		NodeOfType("$vaexpr"),

		NodeOfType("$eval"),
		NodeOfType("$is_const"),
		NodeOfType("$sizeof"),
		NodeOfType("$stringify"),

		NodeOfType("$feature"),

		NodeOfType("$and"),
		NodeOfType("$append"),
		NodeOfType("$concat"),
		NodeOfType("$defined"),
		NodeOfType("$embed"),
		NodeOfType("$or"),

		NodeOfType("$assignable"), // TODO

		NodeSiblingsWithSequenceOf([]NodeRule{
			NodeOfType("lambda_declaration"),
			NodeOfType("compound_stmt"),
		}, "..lambda_declaration_with_body.."),
	}, node, source, false)

	return found.(ast.Expression)
}

// convert_assignment_expr
// TODO Naming does not match with return type
func convert_assignment_expr(node *sitter.Node, source []byte) ast.Expression {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	var left ast.Expression
	var right ast.Expression
	operator := ""
	if leftNode.Type() == "ct_type_ident" {
		left = convert_ct_type_ident(leftNode, source)
		right = convert_type(rightNode, source)
		operator = "="
	} else {
		ConvertDebug = true
		left = convert_expression(leftNode, source)
		right = convert_expression(rightNode, source)
		ConvertDebug = false
		operator = node.ChildByFieldName("operator").Content(source)
	}

	return &ast.AssignmentExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Left:           left,
		Right:          right,
		Operator:       operator,
	}
}

func convert_binary_expr(node *sitter.Node, source []byte) ast.Expression {
	left := convert_expression(node.ChildByFieldName("left"), source)
	operator := node.ChildByFieldName("operator").Content(source)
	right := convert_expression(node.ChildByFieldName("right"), source)

	return &ast.BinaryExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Left:           left,
		Operator:       operator,
		Right:          right,
	}
}

func convert_bytes_expr(node *sitter.Node, source []byte) ast.Expression {
	return convert_literal(node.Child(0), source)
}

func convert_ternary_expr(node *sitter.Node, source []byte) ast.Expression {
	expected := []NodeRule{
		NodeOfType("binary_expr"),
		NodeOfType("unary_expr"),
		NodeOfType("cast_expr"),
		NodeOfType("rethrow_expr"),
		NodeOfType("trailing_generic_expr"),
		NodeOfType("update_expr"),
		NodeOfType("call_expr"),
		NodeOfType("subscript_expr"),
		NodeOfType("initializer_list"),
		NodeOfType("_base_expr"),
	}
	condition := anyOf("ternary_expr", expected, node.ChildByFieldName("condition"), source, false)

	return &ast.TernaryExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Condition:      condition.(ast.Expression),
		Consequence:    convert_expression(node.ChildByFieldName("consequence"), source),
		Alternative:    convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_type_access_expr(node *sitter.Node, source []byte) ast.Expression {
	var x ast.Expression
	var y *ast.Ident

	argumentNode := node.ChildByFieldName("argument")
	x = convert_type(argumentNode, source)
	y = choice([]string{"access_ident", "const_ident"}, node.ChildByFieldName("field"), source, false).(*ast.Ident)

	return &ast.SelectorExpr{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
		X:              x,
		Sel:            y,
	}
}

func convert_field_expr(node *sitter.Node, source []byte) ast.Expression {
	debugNode(node, source, "field_expr")
	argument := node.ChildByFieldName("argument")
	var argumentNode ast.Expression
	if argument.Type() == "ident" {
		argumentNode = ast.NewIdentifierBuilder().
			WithName(argument.Content(source)).
			WithSitterPos(argument).
			BuildPtr()
	} else {
		argumentNode = convert_field_expr(argument, source)
	}
	field := node.ChildByFieldName("field")

	return &ast.SelectorExpr{
		X: argumentNode,
		Sel: ast.NewIdentifierBuilder().
			WithName(field.Content(source)).
			WithSitterPos(field).
			BuildPtr(),
	}
}

func convert_elvis_orelse_expr(node *sitter.Node, source []byte) ast.Expression {
	conditionNode := node.ChildByFieldName("condition")

	return &ast.TernaryExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Condition:      convert_expression(conditionNode, source),
		Consequence:    convert_ident(conditionNode, source),
		Alternative:    convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_optional_expr(node *sitter.Node, source []byte) ast.Expression {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(source)
	if operatorNode.NextSibling() != nil && operatorNode.NextSibling().Type() == "!" {
		operator += "!"
	}

	argumentNode := node.ChildByFieldName("argument")
	return &ast.OptionalExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Operator:       operator,
		Argument:       convert_expression(argumentNode, source),
	}
}

func convert_unary_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.UnaryExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Operator:       node.ChildByFieldName("operator").Content(source),
		Argument:       convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_update_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.UpdateExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Operator:       node.ChildByFieldName("operator").Content(source),
		Argument:       convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_subscript_expr(node *sitter.Node, source []byte) ast.Expression {
	var index ast.Expression
	indexNode := node.ChildByFieldName("index")
	if indexNode != nil {
		index = convert_expression(indexNode, source)
	} else {
		rangeNode := node.ChildByFieldName("range")
		if rangeNode != nil {
			index = convert_range_expr(rangeNode, source)
		}
	}

	return &ast.SubscriptExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Index:          index,
		Argument:       convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_cast_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.CastExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Type:           convert_type(node.ChildByFieldName("type"), source),
		Argument:       convert_expression(node.ChildByFieldName("value"), source),
	}
}

func convert_rethrow_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.RethrowExpression{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Operator:       node.ChildByFieldName("operator").Content(source),
		Argument:       convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_call_expr(node *sitter.Node, source []byte) ast.Expression {

	invocationNode := node.ChildByFieldName("arguments")
	args := []ast.Expression{}
	for i := 0; i < int(invocationNode.ChildCount()); i++ {
		n := invocationNode.Child(i)
		if n.Type() == "arg" {
			args = append(args, convert_arg(n, source))
		}
	}

	trailingNode := node.ChildByFieldName("trailing")
	compoundStmt := option.None[*ast.CompoundStmt]()
	if trailingNode != nil {
		compoundStmt = option.Some(convert_compound_stmt(trailingNode, source).(*ast.CompoundStmt))
	}

	expr := convert_expression(node.ChildByFieldName("function"), source)
	var identifier ast.Expression
	genericArguments := option.None[[]ast.Expression]()
	switch expr.(type) {
	case *ast.SelectorExpr:
		identifier = expr.(*ast.SelectorExpr)
	case *ast.Ident:
		identifier = expr.(*ast.Ident)
	case *ast.TrailingGenericsExpr:
		identifier = expr.(*ast.TrailingGenericsExpr).Identifier
		ga := expr.(*ast.TrailingGenericsExpr).GenericArguments
		genericArguments = option.Some(ga)
	}

	return &ast.FunctionCall{
		NodeAttributes:   ast.NewBaseNodeFromSitterNode(node),
		Identifier:       identifier,
		GenericArguments: genericArguments,
		Arguments:        args,
		TrailingBlock:    compoundStmt,
	}
}

/*
trailing_generic_expr: $ => prec.right(PREC.TRAILING, seq(
field('argument', $._expr),
field('operator', $.generic_arguments),
)),
*/
func convert_trailing_generic_expr(node *sitter.Node, source []byte) ast.Expression {
	argNode := node.ChildByFieldName("argument")
	expr := convert_expression(argNode, source)

	operator := convert_generic_arguments(node.ChildByFieldName("operator"), source)

	return &ast.TrailingGenericsExpr{
		NodeAttributes:   ast.NewBaseNodeFromSitterNode(node),
		Identifier:       expr.(*ast.Ident),
		GenericArguments: operator,
	}
}

func convert_generic_arguments(node *sitter.Node, source []byte) []ast.Expression {
	var args []ast.Expression
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)

		switch n.Type() {
		case "(<", ">)", ",":
			//ignore
		case "type":
			args = append(args, convert_type(n, source))

		default:
			args = append(args, convert_expression(n, source))
		}
	}

	return args
}

func convert_type_with_initializer_list(node *sitter.Node, source []byte) ast.Expression {
	baseExpr := convert_base_expression(node.NextNamedSibling(), source)
	initList, ok := baseExpr.(*ast.InitializerList)
	if !ok {
		initList = &ast.InitializerList{}
	}

	expression := &ast.InlineTypeWithInitialization{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithRangePositions(
				uint(node.StartPoint().Row),
				uint(node.StartPoint().Column),
				initList.NodeAttributes.Range.End.Line,
				initList.NodeAttributes.Range.End.Column,
			).Build(),
		Type:            convert_type(node, source),
		InitializerList: initList,
	}

	return expression
}

func convert_module_ident_expr(node *sitter.Node, source []byte) ast.Expression {
	return ast.NewIdentifierBuilder().
		WithName(node.ChildByFieldName("ident").Content(source)).
		WithPath(node.Child(0).Child(0).Content(source)).
		WithSitterPos(node).BuildPtr()
}

func convert_literal(node *sitter.Node, sourceCode []byte) ast.Expression {
	basicLiteral := ast.BasicLit{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		Value:          node.Content(sourceCode),
	}

	//fmt.Printf("Converting literal %s\n", node.TypeDescription())
	switch node.Type() {
	case "string_literal", "raw_string_literal", "bytes_literal":
		basicLiteral.Kind = ast.STRING
	case "char_literal":
		basicLiteral.Kind = ast.CHAR
	case "integer_literal":
		basicLiteral.Kind = ast.INT
	case "real_literal":
		basicLiteral.Kind = ast.FLOAT
	case "false", "true":
		basicLiteral.Kind = ast.BOOLEAN
	case "null":
		basicLiteral.Kind = ast.NULL
	default:
		panic(fmt.Sprintf("Literal type not supported: %s\n", node.Type()))
	}

	return &basicLiteral
}

func convert_as_literal(node *sitter.Node, source []byte) ast.Expression {
	return &ast.BasicLit{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
		Kind:           ast.STRING,
		Value:          node.Content(source),
	}
}

func convert_initializer_list(node *sitter.Node, source []byte) ast.Expression {
	initList := &ast.InitializerList{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "arg" {
			initList.Args = append(initList.Args, convert_arg(n, source))
		}
	}

	return initList
}

func convert_arg(node *sitter.Node, source []byte) ast.Expression {
	childCount := int(node.ChildCount())

	if is_literal(node.Child(0)) {
		return convert_literal(node.Child(0), source)
	}

	switch node.Child(0).Type() {
	case "param_path":
		param_path := node.Child(0)
		var arg ast.Expression
		param_path_element := param_path.Child(0)

		argType := 0
		for p := 0; p < int(param_path_element.ChildCount()); p++ {
			pnode := param_path_element.Child(p)
			if pnode.IsNamed() {
				if pnode.Type() == "ident" {
					argType = 1
				} else {
					argType = 0
				}
			}
		}

		if argType == 1 {
			arg = &ast.ArgFieldSet{
				FieldName: param_path_element.Child(1).Content(source),
			}
		} else {
			arg = &ast.ArgParamPathSet{
				Path: node.Child(0).Content(source),
			}
		}

		for j := 1; j < childCount; j++ {
			//fmt.Print("\t       ")
			n := node.Child(j)
			var expr ast.Expression
			if n.Type() == "type" {
				expr = convert_type(n, source)
			} else if n.Type() != "=" {
				expr = convert_expression(n, source)
			}

			switch v := arg.(type) {
			case *ast.ArgParamPathSet:
				v.Expr = expr
				arg = v
			case *ast.ArgFieldSet:
				v.Expr = expr
				arg = v
			}
		}
		return arg

	case "type":
		return ast.Expression(convert_type(node.Child(0), source))
	case "$vasplat":
		return &ast.BasicLit{
			NodeAttributes: ast.NewNodeAttributesBuilder().WithSitterPos(node).Build(),
			Kind:           ast.STRING,
			Value:          node.Content(source),
		}
	case "...":
		return convert_expression(node.Child(1), source)
	default:
		// try expr
		expr := convert_expression(node.Child(0), source)
		if expr != nil {
			return expr
		}
	}

	return nil
}

func convert_param_path(param_path *sitter.Node, source []byte) ast.Path {
	var path ast.Path
	param_path_element := param_path.Child(0)

	pathType := ast.PathTypeIndexed
	for p := 0; p < int(param_path_element.ChildCount()); p++ {
		pnode := param_path_element.Child(p)
		if pnode.IsNamed() {
			if pnode.Type() == "ident" {
				pathType = ast.PathTypeField
			}
		} else if pnode.Type() == ".." {
			pathType = ast.PathTypeRange
		}
	}

	path = ast.Path{
		PathType: pathType,
	}
	if pathType == ast.PathTypeField {
		path.FieldName = param_path_element.Child(1).Content(source)
	} else if pathType == ast.PathTypeRange {
		path.PathStart = param_path_element.Child(1).Content(source)
		path.PathEnd = param_path_element.Child(3).Content(source)

	} else {
		path.Path = param_path.Child(0).Content(source)
	}

	return path
}

func convert_flat_path(node *sitter.Node, source []byte) ast.Expression {
	node = node.Child(0)

	if node.Type() == "type" {
		return convert_type(node, source)
	}

	base_expr := convert_base_expression(node, source)

	next := node.NextSibling()
	if next != nil {
		path := convert_param_path(next, source)
		switch path.PathType {
		case ast.PathTypeIndexed:
			return &ast.IndexAccessExpr{
				Array: base_expr,
				Index: path.Path,
			}
		case ast.PathTypeField:
			return &ast.FieldAccessExpr{
				Object: base_expr,
				Field:  path,
			}
		case ast.PathTypeRange:
			return &ast.RangeAccessExpr{
				Array:      base_expr,
				RangeStart: utils.StringToUint(path.PathStart),
				RangeEnd:   utils.StringToUint(path.PathEnd),
			}
		}
	}

	return base_expr
}

func convert_range_expr(node *sitter.Node, source []byte) ast.Expression {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")

	left := option.None[uint]()
	right := option.None[uint]()
	if leftNode != nil {
		left = option.Some(utils.StringToUint(leftNode.Content(source)))
	}
	if rightNode != nil {
		right = option.Some(utils.StringToUint(rightNode.Content(source)))
	}

	return &ast.RangeIndexExpr{
		Start: left,
		End:   right,
	}
}

func cast_expressions_to_args(expressions []ast.Expression) []ast.Expression {
	var args []ast.Expression

	for _, expr := range expressions {
		if arg, ok := expr.(ast.Expression); ok {
			args = append(args, arg)
		} else {
			// Si algún elemento no puede convertirse, retornamos un error
			panic(fmt.Sprintf("no se pudo convertir %v a Arg", expr))
		}
	}

	return args
}

type NodeConverterSeparated func(node *sitter.Node, source []byte) (ast.Expression, int)

func convert_token_separated(node *sitter.Node, separator string, source []byte, convert_func nodeConverter) []ast.Node {
	var nodes []ast.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == separator {
			continue
		}
		expr := convert_func(n, source)

		if expr != nil {
			nodes = append(nodes, expr)
		}
	}

	return nodes
}

func convert_paren_expr(node *sitter.Node, source []byte) ast.Expression {
	child := node.Child(0)
	if child.Type() != "(" {
		panic(
			fmt.Sprintf("convert_paren_expr: Incorrect type. Expected \"(\": %s", node.Type()),
		)
	}

	next := child.NextSibling()
	return &ast.ParenExpr{
		NodeAttributes: ast.NewBaseNodeFromSitterNode(node),
		X:              convert_expression(next, source),
	}
}

func debugNode(node *sitter.Node, source []byte, tag string) {
	if node == nil {
		log.Printf("Node is nil\n")
		return
	}

	log.Printf("%s: %s: %s\n----- %s\n", tag, node.Type(), node.Content(source), node)
}
