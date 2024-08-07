package ast

import "C"

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func GetCST(sourceCode string) *sitter.Node {
	return cst.GetParsedTreeFromString(sourceCode).RootNode()
}

func ConvertToAST(cstNode *sitter.Node, sourceCode string, fileName string) File {
	source := []byte(sourceCode)

	var prg File

	if cstNode.Type() == "source_file" {
		prg = File{
			Name:        fileName,
			ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(cstNode).Build(),
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
					ASTNodeBase: NewBaseNodeBuilder().WithStartEnd(uint(node.StartPoint().Row), uint(node.StartPoint().Column), 0, 0).Build(),
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
				lastMod.ASTNodeBase.EndPos = Position{uint(node.StartPoint().Row), uint(node.StartPoint().Column)}
			}

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
		//fmt.Println(i, ":", n.Type(), ":: ", n.Content(source), ":: has errors: ", n.HasError())
		switch n.Type() {
		case "type":
			variable.Type = convert_type(n, source)

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
		}
	}

	// Check for initializer
	// _assign_right_expr
	right := node.ChildByFieldName("right")
	if right != nil {
		variable.Initializer = convert_expression(right, source)
		//variable.Initializer = convert_base_expression(right, source)
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
			enumDecl.BaseType = convert_type(n.Child(1), sourceCode)
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
								Type: convert_type(paramNode.Child(0), sourceCode),
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
				fieldType = convert_type(n, sourceCode)
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
			structDecl.BackingType = option.Some(convert_type(child, sourceCode))

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
					member.Type = convert_type(bdefnode, sourceCode)
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

func convert_const_declaration(node *sitter.Node, sourceCode []byte) Expression {
	constant := ConstDecl{
		Names: []Identifier{},
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	var idNode *sitter.Node

	//fmt.Println(node.ChildCount())
	//fmt.Println(node)
	//fmt.Println(node.Content(sourceCode))

	for i := uint32(0); i < node.ChildCount(); i++ {
		n := node.Child(int(i))
		switch n.Type() {
		case "type":
			constant.Type = convert_type(n, sourceCode)

		case "const_ident":
			idNode = n
		}
	}

	constant.Names = append(constant.Names,
		NewIdentifierBuilder().
			WithName(idNode.Content(sourceCode)).
			WithSitterPos(idNode).
			Build(),
	)

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
				_type = convert_type(n.Child(0), sourceCode)
				defBuilder.WithResolvesToType(_type)
			} else if n.Child(0).Type() == "func_typedef" {
				// TODO Parse full info of this func typedefinition
				defBuilder.WithResolvesTo(n.Content(sourceCode))
			}
		}
	}

	return defBuilder.Build()
}

func convert_function_declaration(node *sitter.Node, sourceCode []byte) Expression {
	var typeIdentifier option.Option[Identifier]
	funcHeader := node.Child(1)

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = option.Some(NewIdentifierBuilder().
			WithName(funcHeader.ChildByFieldName("method_type").Content(sourceCode)).
			WithSitterPos(funcHeader.ChildByFieldName("method_type")).
			Build())
	}
	signature := convert_function_signature(node, sourceCode)

	funcDecl := FunctionDecl{
		ASTNodeBase:  NewBaseNodeBuilder().WithSitterPos(node).Build(),
		ParentTypeId: typeIdentifier,
		Signature:    signature,
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
				convert_function_parameter(argNode, typeIdentifier, sourceCode),
			)
		}
	}

	signatureDecl := FunctionSignature{
		Name: NewIdentifierBuilder().
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		ReturnType: convert_type(funcHeader.ChildByFieldName("return_type"), sourceCode),
		Parameters: parameters,
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), node.EndPoint()).
			Build(),
	}

	return signatureDecl
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
			argType = convert_type(n, sourceCode)
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
					ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(argNode).Build(),
				}
			}
		}
	}

	if ampersandFound {
		fmt.Println("poiner")
	}

	variable := FunctionParameter{
		Name:        identifier,
		Type:        argType,
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(argNode).Build(),
	}

	return variable
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
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
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
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
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

func convert_type(node *sitter.Node, sourceCode []byte) TypeInfo {
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
							gType := convert_type(gn, sourceCode)
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
