package ast

import "C"

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
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
				if enumeratorNode.Type() == "enum_constant" {
					name := enumeratorNode.ChildByFieldName("name")
					enumDecl.Members = append(enumDecl.Members,
						EnumMember{
							Name: Identifier{
								Name: name.Content(sourceCode),
								ASTNodeBase: NewBaseNodeBuilder().
									WithSitterPosRange(name.StartPoint(), name.EndPoint()).
									Build(),
							},
							ASTNodeBase: NewBaseNodeBuilder().
								WithSitterPosRange(enumeratorNode.StartPoint(), enumeratorNode.EndPoint()).
								Build(),
						},
					)
				}
			}
		}
	}

	return enumDecl
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
			for b := 0; b < int(n.ChildCount()); b++ {
				bn := n.Child(b)
				//fmt.Println("---"+bn.Type(), bn.Content(sourceCode))
				switch bn.Type() {
				case "base_type_name":
					typeInfo.Name = bn.Content(sourceCode)
					typeInfo.BuiltIn = true
				case "type_ident":
					typeInfo.Name = bn.Content(sourceCode)
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
					//modulePath = strings.Trim(bn.Child(0).Content(sourceCode), ":")
					//baseType = bn.Child(1).Content(sourceCode)
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
