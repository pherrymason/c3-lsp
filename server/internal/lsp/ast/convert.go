package ast

import "C"

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
)

func GetCST(sourceCode string) *sitter.Node {
	return cst.GetParsedTreeFromString(sourceCode).RootNode()
}

func ConvertToAST(cstNode *sitter.Node, sourceCode string, fileName string) File {
	source := []byte(sourceCode)

	var prg File
	//fmt.Print(cstNode)
	if cstNode.Type() == "source_file" {
		prg = File{
			Name:        fileName,
			ASTBaseNode: NewBaseNodeBuilder().WithSitterPos(cstNode).Build(),
		}
	}

	anonymousModule := false
	for i := 0; i < int(cstNode.ChildCount()); i++ {
		node := cstNode.Child(i)
		parsedModules := len(prg.Modules)
		if parsedModules == 0 && node.Type() != "module" {
			anonymousModule = true
			prg.Modules = append(prg.Modules,
				Module{
					ASTBaseNode: NewBaseNodeBuilder().WithStartEnd(uint(node.StartPoint().Row), uint(node.StartPoint().Column), 0, 0).Build(),
					Name:        symbols.NormalizeModuleName(fileName),
				},
			)
			parsedModules = len(prg.Modules)
		}

		var lastMod *Module
		if parsedModules > 0 {
			lastMod = &prg.Modules[len(prg.Modules)-1]
		}

		switch node.Type() {
		case "module":
			if anonymousModule {
				anonymousModule = false
				lastMod.ASTBaseNode.EndPos = Position{uint(node.StartPoint().Row), uint(node.StartPoint().Column)}
			}

			prg.Modules = append(prg.Modules, convert_module(node, source))

		case "import_declaration":
			lastMod.Imports = append(lastMod.Imports, convert_imports(node, source).(Import))

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

		case "const_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_const_declaration(node, source))

		case "define_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_def_declaration(node, source))

		case "func_definition", "func_declaration":
			lastMod.Functions = append(lastMod.Functions, convert_function_declaration(node, source))

		case "interface_declaration":
			lastMod.Declarations = append(lastMod.Declarations, convert_interface_declaration(node, source))

		case "macro_declaration":
			lastMod.Macros = append(lastMod.Macros, convert_macro_declaration(node, source))
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

func convert_imports(node *sitter.Node, source []byte) Expression {
	imports := Import{
		Path: node.ChildByFieldName("path").Content(source),
	}

	return imports
}

func convert_global_declaration(node *sitter.Node, source []byte) VariableDecl {
	variable := VariableDecl{
		Names: []Identifier{},
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//debugNode(n, source)
		switch n.Type() {
		case "type":
			variable.Type = convert_type(n, source).(TypeInfo)

		case "ident":
			variable.Names = append(
				variable.Names,
				convert_ident(n, source).(Identifier),
			)

		case ";":

		case "multi_declaration":
			for j := 0; j < int(n.ChildCount()); j++ {
				sub := n.Child(j)
				if sub.Type() == "ident" {
					variable.Names = append(
						variable.Names,
						convert_ident(sub, source).(Identifier),
					)
				}
			}

			/*default:
			if n.IsNamed() {
				variable.Initializer = convert_expression(n, source)
			}*/
		}
	}

	// Check for initializer
	// _assign_right_expr

	right := node.ChildByFieldName("right")
	if right != nil {
		variable.Initializer = convert_expression(right, source)
	}

	return variable
}

func convert_enum_declaration(node *sitter.Node, sourceCode []byte) Expression {
	enumDecl := EnumDecl{
		Name: node.ChildByFieldName("name").Content(sourceCode),
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "enum_spec":
			enumDecl.BaseType = convert_type(n.Child(1), sourceCode).(TypeInfo)
			if n.ChildCount() >= 3 {
				param_list := n.Child(2)
				for p := 0; p < int(param_list.ChildCount()); p++ {
					paramNode := param_list.Child(p)
					if paramNode.Type() == "enum_param_declaration" {
						enumDecl.Properties = append(
							enumDecl.Properties,
							EnumProperty{
								ASTBaseNode: NewBaseNodeBuilder().
									WithSitterPosRange(paramNode.StartPoint(), paramNode.EndPoint()).
									Build(),
								Name: Identifier{
									Name: paramNode.Child(1).Content(sourceCode),
									ASTBaseNode: NewBaseNodeBuilder().
										WithSitterPosRange(paramNode.Child(1).StartPoint(), paramNode.Child(1).EndPoint()).
										Build(),
								},
								Type: convert_type(paramNode.Child(0), sourceCode).(TypeInfo),
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
							ASTBaseNode: NewBaseNodeBuilder().
								WithSitterPosRange(name.StartPoint(), name.EndPoint()).
								Build(),
						},
						Value: compositeLiteral,
						ASTBaseNode: NewBaseNodeBuilder().
							WithSitterPosRange(enumeratorNode.StartPoint(), enumeratorNode.EndPoint()).
							Build(),
					},
				)

			}
		}
	}

	return enumDecl
}

func convert_struct_declaration(node *sitter.Node, sourceCode []byte) Expression {
	structDecl := StructDecl{
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
		StructType: StructTypeNormal,
	}

	structDecl.Name = node.ChildByFieldName("name").Content(sourceCode)

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

		//fmt.Println("body child:", memberNode.Type())
		if memberNode.Type() != "struct_member_declaration" {
			continue
		}
		fmt.Printf("%d - %s\n", i, memberNode.Content(sourceCode))

		fieldType := TypeInfo{}
		member := StructMemberDecl{
			ASTBaseNode: NewBaseNodeBuilder().
				WithSitterPosRange(memberNode.StartPoint(), memberNode.EndPoint()).
				Build(),
		}

		for x := 0; x < int(memberNode.ChildCount()); x++ {
			n := memberNode.Child(x)
			//debugNode(n, sourceCode)

			switch n.Type() {
			case "type":
				fieldType = convert_type(n, sourceCode).(TypeInfo)
				member.Type = fieldType
			case "identifier_list":
				for j := 0; j < int(n.ChildCount()); j++ {
					member.Names = append(
						member.Names,
						convert_ident(n.Child(j), sourceCode).(Identifier),
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
					convert_ident(n, sourceCode).(Identifier),
				)
			}
		}

		if len(member.Names) > 0 {
			structDecl.Members = append(structDecl.Members, member)
		}
	}

	return structDecl
}

func convert_bitstruct_declaration(node *sitter.Node, sourceCode []byte) Expression {
	structDecl := StructDecl{
		ASTBaseNode: NewBaseNodeBuilder().WithSitterPosRange(node.StartPoint(), node.EndPoint()).Build(),
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
			structDecl.BackingType = option.Some(convert_type(child, sourceCode).(TypeInfo))

		case "bitstruct_body":
			structDecl.Members = convert_bitstruct_members(child, sourceCode)
		}
	}

	return structDecl
}

func convert_bitstruct_members(node *sitter.Node, source []byte) []StructMemberDecl {
	members := []StructMemberDecl{}
	for i := 0; i < int(node.ChildCount()); i++ {
		bdefnode := node.Child(i)
		//debugNode(bdefnode, source)
		bType := bdefnode.Type()
		member := StructMemberDecl{
			ASTBaseNode: NewBaseNodeBuilder().
				WithSitterPosRange(bdefnode.StartPoint(), bdefnode.EndPoint()).
				Build(),
		}

		if bType == "bitstruct_member_declaration" {
			for x := 0; x < int(bdefnode.ChildCount()); x++ {
				xNode := bdefnode.Child(x)
				//fmt.Println(xNode.Type())
				switch xNode.Type() {
				case "base_type":
					// Note: here we consciously pass bdefnode because typeNodeToType expects a child node of base_type. If we send xNode it will not find it.
					member.Type = convert_type(bdefnode, source).(TypeInfo)
				case "ident":
					member.Names = append(
						member.Names,
						convert_ident(xNode, source).(Identifier),
					)
				}
			}

			bitRanges := [2]uint{}
			lowBit, _ := strconv.ParseInt(bdefnode.Child(3).Content(source), 10, 32)
			bitRanges[0] = uint(lowBit)

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
							ASTBaseNode: NewBaseNodeBuilder().
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
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	return fault
}

func convert_const_declaration(node *sitter.Node, source []byte) Expression {
	constant := ConstDecl{
		Names: []Identifier{},
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	var idNode *sitter.Node

	//fmt.Println(node.ChildCount())
	//fmt.Println(node)
	//fmt.Println(node.Content(sourceCode))

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type":
			constant.Type = option.Some(convert_type(n, source).(TypeInfo))

		case "const_ident":
			idNode = n

		case "attributes":
			// TODO
		}
	}

	constant.Names = append(constant.Names,
		NewIdentifierBuilder().
			WithName(idNode.Content(source)).
			WithSitterPos(idNode).
			Build(),
	)

	right := node.ChildByFieldName("right")
	if right != nil {
		constant.Initializer = convert_expression(right, source)
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
func convert_def_declaration(node *sitter.Node, sourceCode []byte) Expression {
	defBuilder := NewDefDeclBuilder().
		WithSitterPos(node)

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type_ident", "define_ident":
			defBuilder.WithName(n.Content(sourceCode)).
				WithIdentifierSitterPos(n)

		case "typedef_type":
			var _type TypeInfo
			if n.Child(0).Type() == "type" {
				// Might contain module path
				_type = convert_type(n.Child(0), sourceCode).(TypeInfo)
				defBuilder.WithResolvesToType(_type)
			} else if n.Child(0).Type() == "func_typedef" {
				// TODO Parse full info of this func typedefinition
				defBuilder.WithResolvesTo(n.Content(sourceCode))
			}
		}
	}

	return defBuilder.Build()
}

func convert_interface_declaration(node *sitter.Node, sourceCode []byte) Expression {
	// TODO parse attributes
	methods := []FunctionSignature{}
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
	_interface := InterfaceDecl{
		ASTBaseNode: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		Name:        NewIdentifierBuilder().WithName(nameNode.Content(sourceCode)).WithSitterPos(nameNode).Build(),
		Methods:     methods,
	}

	return _interface
}

func convert_macro_declaration(node *sitter.Node, sourceCode []byte) Expression {
	var nameNode *sitter.Node

	parameters := []FunctionParameter{}
	nodeParameters := node.Child(2)
	if nodeParameters.ChildCount() > 2 {
		for i := uint32(0); i < nodeParameters.ChildCount(); i++ {
			argNode := nodeParameters.Child(int(i))
			if argNode.Type() != "parameter" {
				continue
			}

			parameters = append(
				parameters,
				convert_function_parameter(argNode, option.None[Identifier](), sourceCode),
			)
		}
	}

	nameNode = node.Child(1).ChildByFieldName("name")
	macro := MacroDecl{
		ASTBaseNode: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		Signature: MacroSignature{
			Name: NewIdentifierBuilder().
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

func convert_ident(node *sitter.Node, source []byte) Expression {
	return Identifier{
		Name: node.Content(source),
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}
}

func convert_type(node *sitter.Node, sourceCode []byte) Expression {
	return extTypeNodeToType(node, sourceCode)
}

func convert_var_decl(node *sitter.Node, source []byte) Expression {
	//for i := 0; i < int(node.ChildCount()); i++ {

	//}
	return ASTBaseNode{}
}

func extTypeNodeToType(
	node *sitter.Node,
	sourceCode []byte,
) TypeInfo {
	typeInfo := TypeInfo{
		Optional: false,
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
		Pointer: uint(0),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//fmt.Println(n.Type(), n.Content(sourceCode))
		switch n.Type() {
		case "base_type":
			typeInfo.ASTBaseNode = NewBaseNodeBuilder().
				WithSitterPosRange(n.StartPoint(), n.EndPoint()).
				Build()

			for b := 0; b < int(n.ChildCount()); b++ {
				bn := n.Child(b)
				//fmt.Println("---"+bn.Type(), bn.Content(sourceCode))

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
							gType := convert_type(gn, sourceCode).(TypeInfo)
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
		case "!":
			typeInfo.Optional = true
		}
	}

	return typeInfo
}

func inArray(needle string, haystack []string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}

	return false
}

func convert_function_declaration(node *sitter.Node, source []byte) Expression {
	var typeIdentifier option.Option[Identifier]
	funcHeader := node.Child(1)
	//debugNode(funcHeader, source)

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = option.Some(NewIdentifierBuilder().
			WithName(funcHeader.ChildByFieldName("method_type").Content(source)).
			WithSitterPos(funcHeader.ChildByFieldName("method_type")).
			Build())
	}
	signature := convert_function_signature(node, source)

	bodyNode := node.ChildByFieldName("body")
	var body Expression
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

	funcDecl := FunctionDecl{
		ASTBaseNode:  NewBaseNodeBuilder().WithSitterPos(node).Build(),
		ParentTypeId: typeIdentifier,
		Signature:    signature,
		Body:         body,
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

func convert_function_signature(node *sitter.Node, sourceCode []byte) FunctionSignature {
	var typeIdentifier option.Option[Identifier]
	funcHeader := node.Child(1)
	nameNode := funcHeader.ChildByFieldName("name")

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = option.Some(NewIdentifierBuilder().
			WithName(funcHeader.ChildByFieldName("method_type").Content(sourceCode)).
			WithSitterPos(funcHeader.ChildByFieldName("method_type")).
			Build())
	}

	signatureDecl := FunctionSignature{
		Name: NewIdentifierBuilder().
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		ReturnType: convert_type(funcHeader.ChildByFieldName("return_type"), sourceCode).(TypeInfo),
		Parameters: convert_function_parameter_list(node.Child(2), typeIdentifier, sourceCode),
		ASTBaseNode: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	return signatureDecl
}

func convert_function_parameter_list(node *sitter.Node, typeIdentifier option.Option[Identifier], source []byte) []FunctionParameter {
	if node.Type() != "fn_parameter_list" {
		panic(
			fmt.Sprintf("Wrong node provided: Expected fn_parameter_list, provided %s", node.Type()),
		)
	}

	parameters := []FunctionParameter{}
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
func convert_function_parameter(argNode *sitter.Node, methodIdentifier option.Option[Identifier], sourceCode []byte) FunctionParameter {
	var identifier Identifier
	var argType TypeInfo
	ampersandFound := false

	for i := 0; i < int(argNode.ChildCount()); i++ {
		n := argNode.Child(int(i))

		switch n.Type() {
		case "&":
			ampersandFound = true

		case "type":
			argType = convert_type(n, sourceCode).(TypeInfo)
		case "ident":
			identifier = NewIdentifierBuilder().
				WithName(n.Content(sourceCode)).
				WithSitterPos(n).
				Build()

			// When detecting a self, the type is the Struct type
			if identifier.Name == "self" && methodIdentifier.IsSome() {
				pointer := uint(0)
				if ampersandFound {
					pointer = 1
				}

				argType = TypeInfo{
					Identifier: NewIdentifierBuilder().
						WithName(methodIdentifier.Get().Name).
						WithSitterPos(n).
						Build(),
					Pointer:     pointer,
					ASTBaseNode: NewBaseNodeBuilder().WithSitterPos(argNode).Build(),
				}
			}
		}
	}

	variable := FunctionParameter{
		Name:        identifier,
		Type:        argType,
		ASTBaseNode: NewBaseNodeBuilder().WithSitterPos(argNode).Build(),
	}

	return variable
}

func convert_lambda_declaration(node *sitter.Node, source []byte) Expression {
	rType := option.None[TypeInfo]()
	parameters := []FunctionParameter{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type", "optional_type":
			r := convert_type(n, source).(TypeInfo)
			rType = option.Some[TypeInfo](r)
		case "fn_parameter_list":
			parameters = convert_function_parameter_list(n, option.None[Identifier](), source)
		case "attributes":
			// TODO
		}
	}

	return LambdaDeclaration{
		ASTBaseNode: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		ReturnType:  rType,
		Parameters:  parameters,
	}
}

func convert_lambda_declaration_with_body(node *sitter.Node, source []byte) Expression {
	expression := convert_lambda_declaration(node, source)

	lambda := expression.(LambdaDeclaration)
	lambda.Body = convert_compound_stmt(node.NextSibling(), source).(CompoundStatement)

	return lambda
}

func convert_lambda_expr(node *sitter.Node, source []byte) Expression {
	expr := convert_lambda_declaration(node.Child(0), source)

	lambda := expr.(LambdaDeclaration)

	bodyNode := node.Child(1).ChildByFieldName("body")

	lambda.ASTBaseNode.EndPos.Column = uint(bodyNode.EndPoint().Column)
	lambda.ASTBaseNode.EndPos.Line = uint(bodyNode.EndPoint().Row)
	lambda.Body = ReturnStatement{
		ASTBaseNode: NewBaseNodeBuilder().WithSitterPos(bodyNode).Build(),
		Return:      option.Some(convert_expression(bodyNode, source)),
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
func convert_expression(node *sitter.Node, source []byte) Expression {
	//fmt.Print("convert_expression:\n")
	//debugNode(node, source)
	return anyOf("_expr", []NodeRule{
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
	}, node, source, false)
}

func convert_base_expression(node *sitter.Node, source []byte) Expression {
	var expression Expression
	//debugNode(node, source)

	return anyOf("_base_expr", []NodeRule{
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
		NodeOfType("paren_expr"),       // TODO
		NodeOfType("expr_block"),       // TODO

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

	return expression
}

func convert_assignment_expr(node *sitter.Node, source []byte) Expression {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	var left Expression
	var right Expression
	operator := ""
	if leftNode.Type() == "ct_type_ident" {
		left = convert_ct_type_ident(leftNode, source)
		right = convert_type(rightNode, source)
		operator = "="
	} else {
		left = convert_expression(leftNode, source)
		right = convert_expression(rightNode, source)
		operator = node.ChildByFieldName("operator").Content(source)
	}

	return AssignmentStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Left:        left,
		Right:       right,
		Operator:    operator,
	}
}

func convert_binary_expr(node *sitter.Node, source []byte) Expression {
	left := convert_expression(node.ChildByFieldName("left"), source)
	operator := node.ChildByFieldName("operator").Content(source)
	right := convert_expression(node.ChildByFieldName("right"), source)

	return BinaryExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Left:        left,
		Operator:    operator,
		Right:       right,
	}
}

func convert_bytes_expr(node *sitter.Node, source []byte) Expression {
	return convert_literal(node.Child(0), source)
}

func convert_ternary_expr(node *sitter.Node, source []byte) Expression {
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

	return TernaryExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Condition:   condition,
		Consequence: convert_expression(node.ChildByFieldName("consequence"), source),
		Alternative: convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_elvis_orelse_expr(node *sitter.Node, source []byte) Expression {
	conditionNode := node.ChildByFieldName("condition")

	return TernaryExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Condition:   convert_expression(conditionNode, source),
		Consequence: convert_ident(conditionNode, source),
		Alternative: convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_optional_expr(node *sitter.Node, source []byte) Expression {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(source)
	if operatorNode.NextSibling() != nil && operatorNode.NextSibling().Type() == "!" {
		operator += "!"
	}

	argumentNode := node.ChildByFieldName("argument")
	return OptionalExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    operator,
		Argument:    convert_expression(argumentNode, source),
	}
}

func convert_unary_expr(node *sitter.Node, source []byte) Expression {
	return UnaryExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    node.ChildByFieldName("operator").Content(source),
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_update_expr(node *sitter.Node, source []byte) Expression {
	return UpdateExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    node.ChildByFieldName("operator").Content(source),
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_subscript_expr(node *sitter.Node, source []byte) Expression {
	var index Expression
	indexNode := node.ChildByFieldName("index")
	if indexNode != nil {
		index = convert_expression(indexNode, source)
	} else {
		rangeNode := node.ChildByFieldName("range")
		if rangeNode != nil {
			index = convert_range_expr(rangeNode, source)
		}
	}

	return SubscriptExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Index:       index,
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_cast_expr(node *sitter.Node, source []byte) Expression {
	return CastExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Type:        convert_type(node.ChildByFieldName("type"), source).(TypeInfo),
		Argument:    convert_expression(node.ChildByFieldName("value"), source),
	}
}

func convert_rethrow_expr(node *sitter.Node, source []byte) Expression {
	return RethrowExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    node.ChildByFieldName("operator").Content(source),
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_call_expr(node *sitter.Node, source []byte) Expression {

	invocationNode := node.ChildByFieldName("arguments")
	args := []Arg{}
	for i := 0; i < int(invocationNode.ChildCount()); i++ {
		n := invocationNode.Child(i)
		if n.Type() == "arg" {
			args = append(args, convert_arg(n, source))
		}
	}

	trailingNode := node.ChildByFieldName("trailing")
	compoundStmt := option.None[CompoundStatement]()
	if trailingNode != nil {
		compoundStmt = option.Some(convert_compound_stmt(trailingNode, source).(CompoundStatement))
	}

	expr := convert_expression(node.ChildByFieldName("function"), source)
	var identifier Expression
	genericArguments := option.None[[]Expression]()
	switch expr.(type) {
	case Identifier:
		identifier = expr
	case TrailingGenericsExpr:
		identifier = expr.(TrailingGenericsExpr).Identifier
		ga := expr.(TrailingGenericsExpr).GenericArguments
		genericArguments = option.Some(ga)
	}

	return FunctionCall{
		ASTBaseNode:      NewBaseNodeFromSitterNode(node),
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
func convert_trailing_generic_expr(node *sitter.Node, source []byte) Expression {
	argNode := node.ChildByFieldName("argument")
	expr := convert_expression(argNode, source)

	operator := convert_generic_arguments(node.ChildByFieldName("operator"), source)

	return TrailingGenericsExpr{
		ASTBaseNode:      NewBaseNodeFromSitterNode(node),
		Identifier:       expr.(Identifier),
		GenericArguments: operator,
	}
}

func convert_generic_arguments(node *sitter.Node, source []byte) []Expression {
	args := []Expression{}
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

func convert_type_with_initializer_list(node *sitter.Node, source []byte) Expression {
	baseExpr := convert_base_expression(node.NextNamedSibling(), source)
	initList, ok := baseExpr.(InitializerList)
	if !ok {
		initList = InitializerList{}
	}

	expression := InlineTypeWithInitizlization{
		ASTBaseNode: NewBaseNodeBuilder().
			WithStartEnd(
				uint(node.StartPoint().Row),
				uint(node.StartPoint().Column),
				initList.ASTBaseNode.EndPos.Line,
				initList.ASTBaseNode.EndPos.Column,
			).Build(),
		Type:            convert_type(node, source).(TypeInfo),
		InitializerList: initList,
	}

	return expression
}

func convert_module_ident_expr(node *sitter.Node, source []byte) Expression {
	return NewIdentifierBuilder().
		WithName(node.ChildByFieldName("ident").Content(source)).
		WithPath(node.Child(0).Child(0).Content(source)).
		WithSitterPos(node).Build()
}

func convert_literal(node *sitter.Node, sourceCode []byte) Expression {
	var literal Expression
	//fmt.Printf("Converting literal %s\n", node.Type())
	switch node.Type() {
	case "string_literal", "char_literal", "raw_string_literal", "bytes_literal":
		literal = Literal{Value: node.Content(sourceCode)}
	case "integer_literal":
		literal = IntegerLiteral{Value: node.Content(sourceCode)}
	case "real_literal":
		literal = RealLiteral{Value: node.Content(sourceCode)}
	case "false":
		literal = BoolLiteral{Value: false}
	case "true":
		literal = BoolLiteral{Value: true}
	case "null":
		literal = Literal{Value: "null"}
	default:
		panic(fmt.Sprintf("Literal type not supported: %s\n", node.Type()))
	}

	return literal
}

func convert_as_literal(node *sitter.Node, source []byte) Expression {
	return Literal{Value: node.Content(source)}
}

func convert_initializer_list(node *sitter.Node, source []byte) Expression {
	initList := InitializerList{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "arg" {
			initList.Args = append(initList.Args, convert_arg(n, source))
		}
	}

	return initList
}

func convert_arg(node *sitter.Node, source []byte) Arg {
	childCount := int(node.ChildCount())

	if is_literal(node.Child(0)) {
		return convert_literal(node.Child(0), source)
	}

	switch node.Child(0).Type() {
	case "param_path":
		param_path := node.Child(0)
		var arg Arg
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
			arg = ArgFieldSet{
				FieldName: param_path_element.Child(1).Content(source),
			}
		} else {
			arg = ArgParamPathSet{
				Path: node.Child(0).Content(source),
			}
		}

		for j := 1; j < childCount; j++ {
			fmt.Print("\t       ")
			n := node.Child(j)
			var expr Expression
			if n.Type() == "type" {
				expr = convert_type(n, source)
			} else if n.Type() != "=" {
				expr = convert_expression(n, source)
			}

			switch v := arg.(type) {
			case ArgParamPathSet:
				v.Expr = expr
				arg = v
			case ArgFieldSet:
				v.Expr = expr
				arg = v
			}
		}
		return arg

	case "type":
		return Expression(convert_type(node.Child(0), source))
	case "$vasplat":
		return Literal{Value: node.Content(source)}
	case "...":
		return Expression(convert_expression(node.Child(1), source))
	default:

		// try expr
		expr := convert_expression(node.Child(0), source)
		if expr != nil {
			return expr
		}
	}

	return nil
}

const (
	PathIdent = iota
	PathField
)

func convert_param_path(param_path *sitter.Node, source []byte) Path {
	var path Path
	param_path_element := param_path.Child(0)

	pathType := PathTypeIndexed
	for p := 0; p < int(param_path_element.ChildCount()); p++ {
		pnode := param_path_element.Child(p)
		if pnode.IsNamed() {
			if pnode.Type() == "ident" {
				pathType = PathTypeField
			}
		} else if pnode.Type() == ".." {
			pathType = PathTypeRange
		}
	}

	path = Path{
		PathType: pathType,
	}
	if pathType == PathTypeField {
		path.FieldName = param_path_element.Child(1).Content(source)
	} else if pathType == PathTypeRange {
		path.PathStart = param_path_element.Child(1).Content(source)
		path.PathEnd = param_path_element.Child(3).Content(source)

	} else {
		path.Path = param_path.Child(0).Content(source)
	}

	return path
}

func convert_flat_path(node *sitter.Node, source []byte) Expression {
	node = node.Child(0)

	if node.Type() == "type" {
		return convert_type(node, source)
	}

	base_expr := convert_base_expression(node, source)

	next := node.NextSibling()
	if next != nil {
		// base_expr + param_path
		//base_expr := convert_base_expression(node, source)
		//param_path := convert_param_path(node.NextSibling(), source)
		path := convert_param_path(next, source)
		switch path.PathType {
		case PathTypeIndexed:
			return IndexAccess{
				Array: base_expr,
				Index: path.Path,
			}
		case PathTypeField:
			return FieldAccess{
				Object: base_expr,
				Field:  path,
			}
		case PathTypeRange:
			return RangeAccess{
				Array:      base_expr,
				RangeStart: utils.StringToUint(path.PathStart),
				RangeEnd:   utils.StringToUint(path.PathEnd),
			}
		}
	}

	return base_expr
}

func convert_range_expr(node *sitter.Node, source []byte) Expression {
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

	return RangeIndex{
		Start: left,
		End:   right,
	}
}

func cast_expressions_to_args(expressions []Expression) []Arg {
	var args []Arg

	for _, expr := range expressions {
		// Realiza una conversión de tipo de Expression a Arg
		if arg, ok := expr.(Arg); ok {
			args = append(args, arg)
		} else {
			// Si algún elemento no puede convertirse, retornamos un error
			panic(fmt.Sprintf("no se pudo convertir %v a Arg", expr))
		}
	}

	return args
}

type NodeConverterSeparated func(node *sitter.Node, source []byte) (Expression, int)

func convert_token_separated(node *sitter.Node, separator string, source []byte, convert_func NodeConverter) []Expression {
	expressions := []Expression{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == separator {
			continue
		}
		expr := convert_func(n, source)

		if expr != nil {
			expressions = append(expressions, expr)
		}
		//i += advance
	}

	return expressions
}

func convert_dummy(node *sitter.Node, source []byte) Expression {
	return nil
}

func debugNode(node *sitter.Node, source []byte, tag string) {
	if node == nil {
		fmt.Printf("Node is nil\n")
		return
	}

	fmt.Printf("%s: %s: %s\n----- %s\n", tag, node.Type(), node.Content(source), node)
}
