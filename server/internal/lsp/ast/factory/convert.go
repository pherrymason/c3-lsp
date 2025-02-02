package factory

import "C"

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/dedent"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"log"
	"strconv"
	"strings"
)

var ConvertDebug bool = false

// statements not implemented yet and we want to ignore
var ignoreStatements = [2]string{
	"line_comment",
	"block_comment",
}

type IDGenerator interface {
	GenerateID() ast.NodeId
}

type AutoIncrementGenerator struct {
	nextID ast.NodeId
}

func (g *AutoIncrementGenerator) GenerateID() ast.NodeId {
	nextID := g.nextID
	g.nextID++
	return nextID
}

func GetCST(sourceCode string) *sitter.Tree {
	return cst.GetParsedTreeFromString(sourceCode)
}

type ASTConverter struct {
	rules       map[string]conversionInfo
	idGenerator IDGenerator
	debug       bool
}

func NewASTConverter() *ASTConverter {
	c := &ASTConverter{
		idGenerator: &AutoIncrementGenerator{nextID: 0},
	}
	c.generateConversionInfo()
	return c
}

func (c *ASTConverter) getNextID() ast.NodeId {
	return c.idGenerator.GenerateID()
}

func (c *ASTConverter) ConvertToAST(cstNode *sitter.Node, sourceCode string, fileName string) *ast.File {
	c.generateConversionInfo()
	source := []byte(sourceCode)

	var prg *ast.File
	if cstNode.Type() == "source_file" {
		prg = ast.NewFile(c.getNextID(), fileName, lsp.NewRangeFromSitterNode(cstNode), []*ast.Module{})
	}

	anonymousModule := false
	var lastMod *ast.Module
	var lastNode *sitter.Node
	var lastDocComment *ast.DocComment

	var node *sitter.Node
	for i := 0; i < int(cstNode.ChildCount()); i++ {
		node = cstNode.Child(i)
		parsedModulesCount := len(prg.Modules)
		if parsedModulesCount == 0 && node.Type() != "module" && node.Type() != "doc_comment" {
			anonymousModule = true
			prg.AddModule(
				ast.NewModule(0, symbols.NormalizeModuleName(fileName), lsp.NewRangeFromSitterNode(node), prg),
			)
			parsedModulesCount = len(prg.Modules)
		}

		if parsedModulesCount > 0 {
			lastMod = prg.Modules[len(prg.Modules)-1]
			if anonymousModule && node.Type() != "module" {
				// Update end position
				lastMod.NodeAttributes.Range.End = lsp.Position{Line: uint(node.EndPoint().Row), Column: uint(node.EndPoint().Column)}
			}
		}

		switch node.Type() {
		case "doc_comment":
			lastDocComment = c.convert_doc_comment(node, source)
		case "module":
			if anonymousModule {
				anonymousModule = false
				lastMod.NodeAttributes.Range.End = lsp.Position{Line: uint(node.StartPoint().Row), Column: uint(node.StartPoint().Column)}
			}

			module := convert_module(node, source)
			if lastDocComment != nil {
				module.SetDocComment(lastDocComment)
			}

			prg.AddModule(module)
			if lastMod != nil && lastNode != nil {
				// Update range end of module
				lastMod.NodeAttributes.Range.End = lsp.Position{
					Line:   uint(lastNode.EndPoint().Row),
					Column: uint(lastNode.EndPoint().Column),
				}
			}
			lastMod = prg.Modules[len(prg.Modules)-1]

		case "import_declaration":
			anImport := c.convert_imports(node, source).(*ast.Import)
			lastMod.Imports = append(lastMod.Imports, anImport)

		case "global_declaration":
			declaration := c.convert_global_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, &declaration)

		case "enum_declaration":
			declaration := c.convert_enum_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "struct_declaration":
			declaration := c.convert_struct_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "bitstruct_declaration":
			declaration := c.convert_bitstruct_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "fault_declaration":
			declaration := c.convert_fault_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "const_declaration":
			declaration := c.convert_const_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "define_declaration":
			declaration := c.convert_def_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "func_definition", "func_declaration":
			declaration := c.convert_function_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "interface_declaration":
			declaration := c.convert_interface_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)

		case "macro_declaration":
			declaration := c.convert_macro_declaration(node, source)
			if lastDocComment != nil {
				declaration.SetDocComment(lastDocComment)
			}
			lastMod.Declarations = append(lastMod.Declarations, declaration)
		}

		lastNode = node
	}

	if node != nil {
		lastMod.NodeAttributes.Range.End =
			lsp.Position{Line: uint(node.EndPoint().Row), Column: uint(node.EndPoint().Column)}

		if node.Type() != "doc_comment" {
			// Ensure the next node won't receive the same doc comment
			lastDocComment = nil
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

func convert_module(node *sitter.Node, source []byte) *ast.Module {
	module := ast.NewModule(0, node.ChildByFieldName("path").Content(source), lsp.NewRangeFromSitterNode(node), nil)

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

func (c *ASTConverter) convert_imports(node *sitter.Node, source []byte) ast.Statement {
	imports := &ast.Import{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Path: ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(node.ChildByFieldName("path").Content(source)).
			Build(),
	}

	return imports
}

func (c *ASTConverter) convert_global_declaration(node *sitter.Node, source []byte) ast.GenDecl {
	variable := ast.GenDecl{
		Token:          ast.VAR,
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	valueSpec := &ast.ValueSpec{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//debugNode(n, source)
		switch n.Type() {
		case "type":
			valueSpec.Type = c.convert_type(n, source)

		case "ident":
			valueSpec.Names = append(valueSpec.Names, c.convert_ident(n, source).(*ast.Ident))

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
		expr := c.convert_expression(right, source).(ast.Expression)
		variable.Spec.(*ast.ValueSpec).Value = expr

		//variable.Initializer = convert_expression(right, source).(Expression)
	}

	return variable
}

func (c *ASTConverter) convert_enum_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {

	nodeId := c.getNextID()
	enumType := &ast.EnumType{
		StaticValues: []ast.Expression{},
	}
	spec := &ast.TypeSpec{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(nodeId, node),
		Name: ast.NewIdentifierBuilder().
			WithId(nodeId).
			WithName(node.ChildByFieldName("name").Content(sourceCode)).
			Build(),
		TypeDescription: enumType,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "enum_spec":
			enumType.BaseType = option.Some(c.convert_type(n.Child(1), sourceCode))
			if n.ChildCount() >= 3 {
				paramList := n.Child(2)
				for p := 0; p < int(paramList.ChildCount()); p++ {
					paramNode := paramList.Child(p)
					if paramNode.Type() == "enum_param_declaration" {
						convertType := c.convert_type(paramNode.Child(0), sourceCode)
						enumType.StaticValues = append(
							enumType.StaticValues,
							&ast.Field{
								NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), paramNode),
								Name: ast.NewIdentifierBuilder().
									WithId(c.getNextID()).
									WithName(paramNode.Child(1).Content(sourceCode)).
									WithSitterPos(paramNode.Child(1)).
									Build(),
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

				compositeLiteral := &ast.CompositeLiteral{
					NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), n),
				}
				args := enumeratorNode.ChildByFieldName("args")
				if args != nil && args.ChildCount() > 0 {
					lastChild := args.Child(int(args.ChildCount()) - 1)
					if is_literal(lastChild) {
						compositeLiteral.Elements = append(compositeLiteral.Elements,
							c.convert_literal(lastChild, sourceCode),
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
								compositeLiteral.Elements = append(compositeLiteral.Elements,
									c.convert_literal(arg.Child(0), sourceCode),
								)
							}
						}
					}
				}

				nameNode := enumeratorNode.ChildByFieldName("name")
				enumType.Values = append(enumType.Values,
					&ast.EnumValue{
						NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), nameNode),
						Name: ast.NewIdentifierBuilder().
							WithId(c.getNextID()).
							WithName(nameNode.Content(sourceCode)).
							WithSitterRange(nameNode).
							Build(),
						Value: compositeLiteral,
					},
				)

			}
		}
	}

	return &ast.GenDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Token:          ast.Token(ast.ENUM),
		Spec:           spec,
	}
}

func (c *ASTConverter) convert_struct_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	specId := c.getNextID()
	typeId := c.getNextID()
	structType := &ast.StructType{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(typeId, node),
		Type:           ast.StructTypeNormal,
		Fields:         []*ast.StructField{},
	}
	spec := &ast.TypeSpec{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(specId, node),
		Name: ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithSitterPos(node.ChildByFieldName("name")).
			WithName(node.ChildByFieldName("name").Content(sourceCode)).
			Build(),
		TypeParams:      nil, // Generic parameters
		TypeDescription: structType,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "union":
			structType.Type = ast.StructTypeUnion
		case "interface_impl":
			for x := 0; x < int(child.ChildCount()); x++ {
				n := child.Child(x)
				if n.Type() == "interface" {
					structType.Implements = append(
						structType.Implements,
						ast.NewIdentifierBuilder().
							WithId(c.getNextID()).
							WithName(n.Content(sourceCode)).
							WithSitterPos(n).
							Build(),
					)
				}
			}
		case "attributes":
			// TODO attributes
		}
	}

	bodyNode := node.ChildByFieldName("body")
	structType.Fields = c.convert_struct_members(bodyNode, sourceCode)

	return &ast.GenDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Token:          ast.Token(ast.STRUCT),
		Spec:           spec,
	}
}

func (c *ASTConverter) convert_struct_members(bodyNode *sitter.Node, sourceCode []byte) []*ast.StructField {
	fields := []*ast.StructField{}

	// Search Struct members
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		memberNode := bodyNode.Child(i)
		if memberNode.Type() != "struct_member_declaration" {
			continue
		}

		fieldType := &ast.TypeInfo{}
		structField := &ast.StructField{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), memberNode),
		}

		for x := 0; x < int(memberNode.ChildCount()); x++ {
			n := memberNode.Child(x)
			//debugNode(n, sourceCode)

			switch n.Type() {
			case "type":
				fieldType = c.convert_type(n, sourceCode)
				structField.Type = fieldType
			case "identifier_list":
				for j := 0; j < int(n.ChildCount()); j++ {
					structField.Names = append(
						structField.Names,
						c.convert_ident(n.Child(j), sourceCode).(*ast.Ident),
					)
				}
			case "attributes":
				// TODO

			case "bitstruct_body":
				// is an anonymous bitstruct
				bitStructsMembers := c.convert_bitstruct_members(n, sourceCode)
				fields = append(fields, bitStructsMembers...)

			case "struct_body":
				// Is an anonymous struct
				subStructFieldType := ast.NewNodeAttributesBuilder().
					WithRange(structField.Range).
					Build()
				structField.Type = &ast.StructType{
					NodeAttributes: subStructFieldType,
					Type:           ast.StructTypeNormal,
					Fields:         c.convert_struct_members(n, sourceCode),
				}

			case "inline":
				structField.Inlined = true

			case "ident":
				structField.Names = append(
					structField.Names,
					c.convert_ident(n, sourceCode).(*ast.Ident),
				)
			}
		}

		if len(structField.Names) > 0 {
			fields = append(fields, structField)
		}
	}

	return fields
}

func (c *ASTConverter) convert_bitstruct_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	structType := &ast.StructType{
		Type:   ast.StructTypeBitStruct,
		Fields: []*ast.StructField{},
	}
	spec := &ast.TypeSpec{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Name: ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithSitterPos(node).
			WithName(node.ChildByFieldName("name").Content(sourceCode)).
			Build(),
		TypeParams:      nil, // Generic parameters
		TypeDescription: structType,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "interface_impl":
			for x := 0; x < int(child.ChildCount()); x++ {
				n := child.Child(x)
				if n.Type() == "interface" {
					structType.Implements = append(
						structType.Implements,
						ast.NewIdentifierBuilder().
							WithId(c.getNextID()).
							WithName(n.Content(sourceCode)).
							WithSitterPos(n).
							Build(),
					)
				}
			}

		case "attributes":
			// TODO attributes

		case "type":
			structType.BackingType = option.Some(c.convert_type(child, sourceCode))

		case "bitstruct_body":
			structType.Fields = c.convert_bitstruct_members(child, sourceCode)
		}
	}

	return &ast.GenDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Token:          ast.Token(ast.STRUCT),
		Spec:           spec,
	}
}

func (c *ASTConverter) convert_bitstruct_members(node *sitter.Node, source []byte) []*ast.StructField {
	members := []*ast.StructField{}
	for i := 0; i < int(node.ChildCount()); i++ {
		bDefNode := node.Child(i)
		bType := bDefNode.Type()
		member := &ast.StructField{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), bDefNode),
		}

		if bType == "bitstruct_member_declaration" {
			for x := 0; x < int(bDefNode.ChildCount()); x++ {
				xNode := bDefNode.Child(x)
				//fmt.Println(xNode.TypeDescription())
				switch xNode.Type() {
				case "base_type":
					// Note: here we consciously pass bDefNode because typeNodeToType expects a child node of base_type. If we send xNode it will not find it.
					member.Type = c.convert_type(bDefNode, source)
				case "ident":
					member.Names = append(
						member.Names,
						c.convert_ident(xNode, source).(*ast.Ident),
					)
				}
			}

			bitRanges := [2]uint{}
			rangeDefined := false
			if bDefNode.ChildCount() >= 4 {
				lowBit, _ := strconv.ParseInt(bDefNode.Child(3).Content(source), 10, 32)
				bitRanges[0] = uint(lowBit)
				rangeDefined = true
			}

			if bDefNode.ChildCount() >= 6 {
				highBit, _ := strconv.ParseInt(bDefNode.Child(5).Content(source), 10, 32)
				bitRanges[1] = uint(highBit)
				rangeDefined = true
			}

			if rangeDefined {
				// TODO: Must store range positions so cursor can be located over this.
				member.BitRange = option.Some(bitRanges)
			} else {
				member.BitRange = option.None[[2]uint]()
			}

			/*member := idx.NewStructMember(
				identity,
				memberType,
				option.Some(bitRanges),
				currentModule.GetModuleString(),
				docId,
				idx.NewRangeFromTreeSitterPositions(bDefNode.Child(1).StartPoint(), bDefNode.Child(1).EndPoint()),
			)*/
			members = append(members, member)
		} else if bType == "_bitstruct_simple_defs" {
			// TODO: Could not make examples with these to parse.
		}
	}

	return members
}

func (c *ASTConverter) convert_fault_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	// TODO parse attributes

	baseType := option.None[*ast.TypeInfo]() // TODO Parse type!
	var constants []*ast.FaultMember

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "fault_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				constantNode := n.Child(i)

				if constantNode.Type() == "const_ident" {
					constants = append(constants,
						&ast.FaultMember{
							Name: ast.NewIdentifierBuilder().
								WithId(c.getNextID()).
								WithName(constantNode.Content(sourceCode)).
								WithSitterPos(constantNode).
								Build(),
							NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), constantNode),
						},
					)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	fault := &ast.FaultDecl{
		Name: ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		BackingType:    baseType,
		Members:        constants,
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	return fault
}

func (c *ASTConverter) convert_const_declaration(node *sitter.Node, source []byte) ast.Declaration {
	constant := &ast.GenDecl{
		Token:          ast.Token(ast.CONST),
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}
	valueSpec := &ast.ValueSpec{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	var identNode *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type":
			valueSpec.Type = c.convert_type(n, source)

		case "const_ident":
			identNode = n

		case "attributes":
			// TODO
		}
	}

	valueSpec.Names = append(valueSpec.Names,
		ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(identNode.Content(source)).
			WithSitterPos(identNode).
			Build(),
	)

	right := node.ChildByFieldName("right")
	if right != nil {
		expr := c.convert_expression(right, source).(ast.Expression)
		valueSpec.Value = expr
	}
	constant.Spec = valueSpec

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
func (c *ASTConverter) convert_def_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	defBuilder := ast.NewDefDeclBuilder(c.getNextID()).
		WithSitterPos(node)

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type_ident", "define_ident":
			defBuilder.WithName(n.Content(sourceCode)).
				WithIdentifierSitterPos(n)

		case "typedef_type":
			var _type *ast.TypeInfo
			if n.Child(0).Type() == "type" {
				// Might contain module path
				_type = c.convert_type(n.Child(0), sourceCode)
				defBuilder.WithExpression(_type)
			} else if n.Child(0).Type() == "func_typedef" {
				funcTypeDefNode := n.Child(0)
				name := funcTypeDefNode.Child(2)
				funcType := &ast.FuncType{
					NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), funcTypeDefNode),
					ReturnType:     c.convert_type(funcTypeDefNode.ChildByFieldName("return_type"), sourceCode),
					Params:         c.convert_function_parameter_list(name, option.None[*ast.Ident](), sourceCode),
				}

				defBuilder.WithExpression(funcType)
			}
		}
	}

	def := defBuilder.Build()
	return &def
}

func (c *ASTConverter) convert_interface_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	// TODO parse attributes
	var methods []*ast.FunctionSignature
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "interface_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				m := n.Child(i)
				if m.Type() == "func_declaration" {
					fun := c.convert_function_signature(m, sourceCode)
					methods = append(methods, fun)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	_interface := &ast.InterfaceDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Name: ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		Methods: methods,
	}

	return _interface
}

func (c *ASTConverter) convert_macro_declaration(node *sitter.Node, sourceCode []byte) ast.Declaration {
	var nameNode *sitter.Node

	var parameters []*ast.FunctionParameter
	nodeParameters := node.Child(2)
	if nodeParameters.ChildCount() > 2 {
		for i := uint32(0); i < nodeParameters.ChildCount(); i++ {
			argNode := nodeParameters.Child(int(i))
			if argNode.Type() != "parameter" {
				continue
			}

			parameters = append(
				parameters,
				c.convert_function_parameter(argNode, option.None[*ast.Ident](), sourceCode),
			)
		}
	}

	nameNode = node.Child(1).ChildByFieldName("name")
	macro := &ast.MacroDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Signature: &ast.MacroSignature{
			Name: ast.NewIdentifierBuilder().
				WithId(c.getNextID()).
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

func (c *ASTConverter) convert_ident(node *sitter.Node, source []byte) ast.Expression {
	return ast.NewIdentifierBuilder().
		WithId(c.getNextID()).
		WithName(node.Content(source)).
		IsCompileTime(node.Type() == "ct_ident").
		WithSitterRange(node).
		Build()
}

func (c *ASTConverter) convert_module_ident_expr(node *sitter.Node, source []byte) ast.Expression {
	return c.convert_module_type_ident(node, source)
}

func (c *ASTConverter) convert_module_type_ident(node *sitter.Node, sourceCode []byte) *ast.Ident {
	rootId := c.getNextID()
	var ident *ast.Ident
	pathTokens := []string{}
	modulePathRange := lsp.Range{}
	firstModuleFound := false

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		nodeType := n.Type()
		switch nodeType {
		case "module_type_resolution", "module_resolution":
			pathTokens = append(pathTokens, n.Child(0).Content(sourceCode))
			if !firstModuleFound {
				firstModuleFound = true
				modulePathRange.Start = lsp.Position{Line: uint(n.StartPoint().Row), Column: uint(n.StartPoint().Column)}
			}
			modulePathRange.End = lsp.Position{Line: uint(n.EndPoint().Row), Column: uint(n.EndPoint().Column)}

		case "type_ident", "ident":
			ident = ast.NewIdentifierBuilder().
				WithId(rootId).
				WithSitterPos(node).
				WithName(n.Content(sourceCode)).
				Build()
		}
	}

	if len(pathTokens) > 0 {
		ident.ModulePath = ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(strings.Join(pathTokens, "::")).
			WithRange(modulePathRange).
			Build()
	}

	return ident
}

func (c *ASTConverter) convert_var_decl(node *sitter.Node, source []byte) ast.Declaration {
	//for i := 0; i < int(node.ChildCount()); i++ {

	//}
	decl := &ast.GenDecl{
		Token:          ast.VAR,
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	return decl
}

func (c *ASTConverter) convert_type(node *sitter.Node, source []byte) *ast.TypeInfo {
	typeInfo := &ast.TypeInfo{
		Optional:       false,
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Pointer:        uint(0),
	}

	correctRootRange := true
	if node.Type() == "type" {
		correctRootRange = false
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "base_type":
			if correctRootRange {
				typeInfo.NodeAttributes.Range = lsp.NewRangeFromSitterNode(n)
			}

			for b := 0; b < int(n.ChildCount()); b++ {
				bn := n.Child(b)
				//fmt.Println("---"+bn.TypeDescription(), bn.Content(sourceCode))

				switch bn.Type() {
				case "base_type_name":
					typeInfo.Identifier = ast.NewIdentifierBuilder().
						WithId(c.getNextID()).
						WithName(bn.Content(source)).
						WithSitterPos(n).
						WithSitterRange(n).
						Build()

					typeInfo.BuiltIn = true
				case "type_ident":
					typeInfo.Identifier = ast.NewIdentifierBuilder().
						WithId(c.getNextID()).
						WithName(bn.Content(source)).
						WithSitterPos(n).
						WithSitterRange(n).
						Build()

				case "generic_arguments":
					for g := 0; g < int(bn.ChildCount()); g++ {
						gn := bn.Child(g)
						if gn.Type() == "type" {
							gType := c.convert_type(gn, source)
							typeInfo.Generics = append(typeInfo.Generics, gType)
						}
					}

				case "module_type_ident":
					identifier := c.convert_module_type_ident(bn, source)
					typeInfo.Identifier = identifier
				}
			}

		case "type_suffix":
			suffix := n.Content(source)
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

func (c *ASTConverter) convert_function_declaration(node *sitter.Node, source []byte) ast.Declaration {
	var typeIdentifier option.Option[*ast.Ident]
	funcHeader := node.Child(1)

	if funcHeader.ChildByFieldName("method_type") != nil {
		var ident *ast.Ident
		methodTypeNode := funcHeader.ChildByFieldName("method_type")
		//debugNode(methodTypeNode.Child(0).Child(0), source, "method")
		if methodTypeNode.Child(0).Child(0).Type() == "module_type_ident" {
			ident = c.convert_module_type_ident(methodTypeNode.Child(0).Child(0), source)
		} else {
			ident = ast.NewIdentifierBuilder().
				WithId(c.getNextID()).
				WithName(methodTypeNode.Content(source)).
				WithSitterPos(methodTypeNode).
				Build()
		}

		typeIdentifier = option.Some(ident)
	}
	signature := c.convert_function_signature(node, source)

	bodyNode := node.ChildByFieldName("body")
	var body ast.Node
	if bodyNode != nil {
		n := bodyNode.Child(0)
		// options
		// compound_stmt
		// => expr;
		if n.Type() == "compound_stmt" {
			body = c.convert_compound_stmt(n, source)
		} else {
			body = c.convert_expression(n.NextSibling(), source)
		}
	}

	funcDecl := &ast.FunctionDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
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

func (c *ASTConverter) convert_function_signature(node *sitter.Node, sourceCode []byte) *ast.FunctionSignature {
	var typeIdentifier option.Option[*ast.Ident]
	funcHeader := node.Child(1)
	nameNode := funcHeader.ChildByFieldName("name")

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = option.Some(
			ast.NewIdentifierBuilder().
				WithId(c.getNextID()).
				WithName(funcHeader.ChildByFieldName("method_type").Content(sourceCode)).
				WithSitterPos(funcHeader.ChildByFieldName("method_type")).
				Build(),
		)
	}

	signatureDecl := &ast.FunctionSignature{
		Name: ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(nameNode.Content(sourceCode)).
			WithSitterPos(nameNode).
			Build(),
		ReturnType:     c.convert_type(funcHeader.ChildByFieldName("return_type"), sourceCode),
		Parameters:     c.convert_function_parameter_list(node.Child(2), typeIdentifier, sourceCode),
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	return signatureDecl
}

func (c *ASTConverter) convert_function_parameter_list(node *sitter.Node, typeIdentifier option.Option[*ast.Ident], source []byte) []*ast.FunctionParameter {
	if node.Type() != "fn_parameter_list" {
		debugNode(node, source, "fn_parameter_list")
		panic(
			fmt.Sprintf("Wrong node provided: Expected fn_parameter_list, provided %s", node.Type()),
		)
	}

	var parameters []*ast.FunctionParameter
	if node.ChildCount() > 2 {
		for i := 0; i < int(node.ChildCount()); i++ {
			argNode := node.Child(i)
			if argNode.Type() != "parameter" {
				continue
			}

			parameters = append(
				parameters,
				c.convert_function_parameter(argNode, typeIdentifier, source),
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
func (c *ASTConverter) convert_function_parameter(argNode *sitter.Node, methodIdentifier option.Option[*ast.Ident], sourceCode []byte) *ast.FunctionParameter {
	var identifier *ast.Ident
	var argType *ast.TypeInfo
	ampersandFound := false

	for i := 0; i < int(argNode.ChildCount()); i++ {
		n := argNode.Child(i)

		switch n.Type() {
		case "&":
			ampersandFound = true

		case "type":
			argType = c.convert_type(n, sourceCode)
		case "ident":
			identifier = ast.NewIdentifierBuilder().
				WithId(c.getNextID()).
				WithName(n.Content(sourceCode)).
				WithSitterPos(n).
				Build()

			// When detecting a self, the type is the Struct type
			if identifier.Name == "self" && methodIdentifier.IsSome() {
				pointer := uint(0)
				if ampersandFound {
					pointer = 1
				}

				argType = &ast.TypeInfo{
					Identifier: //&ast.PathIdent{
					//NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), n),
					ast.NewIdentifierBuilder().
						WithId(c.getNextID()).
						WithName(methodIdentifier.Get().Name).
						WithSitterPos(n).
						Build(),
					Pointer: pointer,
					NodeAttributes: ast.NewNodeAttributesBuilder().
						WithId(c.getNextID()).
						WithSitterPos(argNode).
						Build(),
				}
			}
		}
	}

	variable := &ast.FunctionParameter{
		Name: identifier,
		Type: argType,
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithId(c.getNextID()).
			WithSitterPos(argNode).
			Build(),
	}

	return variable
}

func (c *ASTConverter) convert_lambda_declaration(node *sitter.Node, source []byte) ast.Expression {
	rType := option.None[*ast.TypeInfo]()
	var parameters []*ast.FunctionParameter

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type", "optional_type":
			r := c.convert_type(n, source)
			rType = option.Some[*ast.TypeInfo](r)
		case "fn_parameter_list":
			parameters = c.convert_function_parameter_list(n, option.None[*ast.Ident](), source)
		case "attributes":
			// TODO
		}
	}

	return &ast.LambdaDeclarationExpr{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithId(c.getNextID()).WithSitterPos(node).Build(),
		ReturnType:     rType,
		Parameters:     parameters,
	}
}

func (c *ASTConverter) convert_lambda_declaration_with_body(node *sitter.Node, source []byte) ast.Expression {
	expression := c.convert_lambda_declaration(node, source)

	lambda := expression.(*ast.LambdaDeclarationExpr)
	lambda.Body = c.convert_compound_stmt(node.NextSibling(), source).(*ast.CompoundStmt)

	return lambda
}

func (c *ASTConverter) convert_lambda_expr(node *sitter.Node, source []byte) ast.Expression {
	expr := c.convert_lambda_declaration(node.Child(0), source)

	lambda := expr.(*ast.LambdaDeclarationExpr)

	bodyNode := node.Child(1).ChildByFieldName("body")

	lambda.NodeAttributes.Range.End.Column = uint(bodyNode.EndPoint().Column)
	lambda.NodeAttributes.Range.End.Line = uint(bodyNode.EndPoint().Row)
	expression := c.convert_expression(bodyNode, source).(ast.Expression)
	lambda.Body = &ast.ReturnStatement{
		NodeAttributes: ast.NewNodeAttributesBuilder().WithId(c.getNextID()).WithSitterPos(bodyNode).Build(),
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
func (c *ASTConverter) convert_expression(node *sitter.Node, source []byte) ast.Expression {
	if c.debug {
		log.Print("[trying to convert_expression]")
		debugNode(node, source, "convert_expression")
	}

	rules := []NodeRule{
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
	}
	converted, _ := c.anyOf("_expr", rules, node, source, c.debug)

	if converted == nil {
		return nil
	}
	return converted.(ast.Expression)
}

func (c *ASTConverter) convert_expr_block(node *sitter.Node, source []byte) ast.Expression {
	var statements []ast.Statement
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() != "{|" && n.Type() != "|}" {
			statements = append(statements, c.convert_statement(n, source))
		}
	}

	return &ast.BlockExpr{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		List:           statements,
	}
}

func (c *ASTConverter) convert_expr_stmt(node *sitter.Node, source []byte) ast.Statement {
	expr := c.convert_expression(node, source)

	return &ast.ExpressionStmt{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Expr:           expr,
	}
}

func (c *ASTConverter) convert_base_expression(node *sitter.Node, source []byte) ast.Expression {
	//debugNode(node, source)

	rules := []NodeRule{
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

		NodeOfType("field_expr"),
		NodeOfType("type_access_expr"), // TODO
		NodeOfType("paren_expr"),
		NodeOfType("expr_block"),

		NodeOfType("$vacount"),

		NodeOfType("$alignof"),
		NodeOfType("$assignable"),
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
	}
	found, _ := c.anyOf("_base_expr", rules, node, source, c.debug)

	return found.(ast.Expression)
}

// convert_assignment_expr
// TODO Naming does not match with return type
func (c *ASTConverter) convert_assignment_expr(node *sitter.Node, source []byte) ast.Expression {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	var left ast.Expression
	var right ast.Expression
	operator := ""
	if leftNode.Type() == "ct_type_ident" {
		left = c.convert_ct_type_ident(leftNode, source)
		right = c.convert_type(rightNode, source)
		operator = "="
	} else {
		ConvertDebug = true
		left = c.convert_expression(leftNode, source)
		right = c.convert_expression(rightNode, source)
		ConvertDebug = false
		operator = node.ChildByFieldName("operator").Content(source)
	}

	return &ast.AssignmentExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Left:           left,
		Right:          right,
		Operator:       operator,
	}
}

func (c *ASTConverter) convert_binary_expr(node *sitter.Node, source []byte) ast.Expression {
	left := c.convert_expression(node.ChildByFieldName("left"), source)
	operator := node.ChildByFieldName("operator").Content(source)
	right := c.convert_expression(node.ChildByFieldName("right"), source)

	return &ast.BinaryExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Left:           left,
		Operator:       operator,
		Right:          right,
	}
}

func (c *ASTConverter) convert_bytes_expr(node *sitter.Node, source []byte) ast.Expression {
	return c.convert_literal(node.Child(0), source)
}

func (c *ASTConverter) convert_ternary_expr(node *sitter.Node, source []byte) ast.Expression {
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
	condition, _ := c.anyOf("ternary_expr", expected, node.ChildByFieldName("condition"), source, c.debug)

	return &ast.TernaryExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Condition:      condition.(ast.Expression),
		Consequence:    c.convert_expression(node.ChildByFieldName("consequence"), source),
		Alternative:    c.convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func (c *ASTConverter) convert_type_access_expr(node *sitter.Node, source []byte) ast.Expression {
	var x ast.Expression
	var y *ast.Ident

	idNode := c.getNextID()
	argumentNode := node.ChildByFieldName("argument")
	x = c.convert_type(argumentNode, source)
	y = c.choice([]string{"access_ident", "const_ident"}, node.ChildByFieldName("field"), source, false).(*ast.Ident)

	return &ast.SelectorExpr{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(idNode, node),
		X:              x,
		Sel:            y,
	}
}

func (c *ASTConverter) convert_field_expr(node *sitter.Node, source []byte) ast.Expression {
	//debugNode(node, source, "????")
	argument := node.ChildByFieldName("argument")
	var argumentNode ast.Expression

	if argument.Type() == "ident" {
		argumentNode = ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(argument.Content(source)).
			WithSitterPos(argument).
			Build()
	} else {
		//debugNode(argument, source, "field_expr/expr")
		argumentNode = c.convert_expression(argument, source)
	}
	start := argumentNode.StartPosition()
	field := node.ChildByFieldName("field")

	selExpr := &ast.SelectorExpr{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithId(c.getNextID()).
			WithRangePositions(
				start.Line,
				start.Column,
				uint(field.EndPoint().Row),
				uint(field.EndPoint().Column),
			).
			Build(),
		X: argumentNode,
		Sel: ast.NewIdentifierBuilder().
			WithId(c.getNextID()).
			WithName(field.Content(source)).
			WithSitterPos(field).
			Build(),
	}
	//fmt.Printf("X:%s", selExpr.X)
	return selExpr
}

func (c *ASTConverter) convert_elvis_orelse_expr(node *sitter.Node, source []byte) ast.Expression {
	conditionNode := node.ChildByFieldName("condition")

	return &ast.TernaryExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Condition:      c.convert_expression(conditionNode, source),
		Consequence:    c.convert_ident(conditionNode, source),
		Alternative:    c.convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func (c *ASTConverter) convert_optional_expr(node *sitter.Node, source []byte) ast.Expression {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(source)
	if operatorNode.NextSibling() != nil && operatorNode.NextSibling().Type() == "!" {
		operator += "!"
	}

	argumentNode := node.ChildByFieldName("argument")
	return &ast.OptionalExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Operator:       operator,
		Argument:       c.convert_expression(argumentNode, source),
	}
}

func (c *ASTConverter) convert_unary_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.UnaryExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Operator:       node.ChildByFieldName("operator").Content(source),
		Argument:       c.convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func (c *ASTConverter) convert_update_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.UpdateExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Operator:       node.ChildByFieldName("operator").Content(source),
		Argument:       c.convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func (c *ASTConverter) convert_subscript_expr(node *sitter.Node, source []byte) ast.Expression {
	var index ast.Expression
	indexNode := node.ChildByFieldName("index")
	if indexNode != nil {
		index = c.convert_expression(indexNode, source)
	} else {
		rangeNode := node.ChildByFieldName("range")
		if rangeNode != nil {
			index = c.convert_range_expr(rangeNode, source)
		}
	}

	return &ast.SubscriptExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Index:          index,
		Argument:       c.convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func (c *ASTConverter) convert_cast_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.CastExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Type:           c.convert_type(node.ChildByFieldName("type"), source),
		Argument:       c.convert_expression(node.ChildByFieldName("value"), source),
	}
}

func (c *ASTConverter) convert_rethrow_expr(node *sitter.Node, source []byte) ast.Expression {
	return &ast.RethrowExpression{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Operator:       node.ChildByFieldName("operator").Content(source),
		Argument:       c.convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func (c *ASTConverter) convert_call_expr(node *sitter.Node, source []byte) ast.Expression {

	invocationNode := node.ChildByFieldName("arguments")
	args := []ast.Expression{}
	Lparen := uint32(0)
	Rparen := uint32(0)
	for i := 0; i < int(invocationNode.ChildCount()); i++ {
		n := invocationNode.Child(i)
		if n.Type() == "(" {
			Lparen = n.StartPoint().Column
		} else if n.Type() == "call_arg" {
			args = append(args, c.convert_call_arg(n, source))
		} else if n.Type() == ")" {
			Rparen = n.StartPoint().Column
		}
	}

	trailingNode := node.ChildByFieldName("trailing")
	compoundStmt := option.None[*ast.CompoundStmt]()
	if trailingNode != nil {
		compoundStmt = option.Some(c.convert_compound_stmt(trailingNode, source).(*ast.CompoundStmt))
	}

	expr := c.convert_expression(node.ChildByFieldName("function"), source)
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

	fnCall := &ast.CallExpr{
		NodeAttributes:   ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Identifier:       identifier,
		GenericArguments: genericArguments,
		Lparen:           uint(Lparen),
		Arguments:        args,
		Rparen:           uint(Rparen),
		TrailingBlock:    compoundStmt,
	}
	return fnCall
}

/*
trailing_generic_expr: $ => prec.right(PREC.TRAILING, seq(
field('argument', $._expr),
field('operator', $.generic_arguments),
)),
*/
func (c *ASTConverter) convert_trailing_generic_expr(node *sitter.Node, source []byte) ast.Expression {
	argNode := node.ChildByFieldName("argument")
	expr := c.convert_expression(argNode, source)

	operator := c.convert_generic_arguments(node.ChildByFieldName("operator"), source)

	return &ast.TrailingGenericsExpr{
		NodeAttributes:   ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Identifier:       expr.(*ast.Ident),
		GenericArguments: operator,
	}
}

func (c *ASTConverter) convert_generic_arguments(node *sitter.Node, source []byte) []ast.Expression {
	var args []ast.Expression
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)

		switch n.Type() {
		case "(<", ">)", ",":
			//ignore
		case "type":
			args = append(args, c.convert_type(n, source))

		default:
			args = append(args, c.convert_expression(n, source))
		}
	}

	return args
}

func (c *ASTConverter) convert_type_with_initializer_list(node *sitter.Node, source []byte) ast.Expression {
	baseExpr := c.convert_base_expression(node.NextNamedSibling(), source)
	initList, ok := baseExpr.(*ast.InitializerList)
	if !ok {
		initList = &ast.InitializerList{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		}
	}

	expression := &ast.InlineTypeWithInitialization{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithId(c.getNextID()).
			WithRangePositions(
				uint(node.StartPoint().Row),
				uint(node.StartPoint().Column),
				initList.NodeAttributes.Range.End.Line,
				initList.NodeAttributes.Range.End.Column,
			).Build(),
		Type:            c.convert_type(node, source),
		InitializerList: initList,
	}

	return expression
}

func (c *ASTConverter) convert_literal(node *sitter.Node, sourceCode []byte) ast.Expression {
	basicLiteral := ast.BasicLit{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
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

func (c *ASTConverter) convert_as_literal(node *sitter.Node, source []byte) ast.Expression {
	return &ast.BasicLit{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Kind:           ast.STRING,
		Value:          node.Content(source),
	}
}

func (c *ASTConverter) convert_initializer_list(node *sitter.Node, source []byte) ast.Expression {
	initList := &ast.InitializerList{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "arg" {
			initList.Args = append(initList.Args, c.convert_arg(n, source))
		}
	}

	return initList
}

func (c *ASTConverter) convert_call_arg(node *sitter.Node, source []byte) ast.Expression {
	child := node.Child(0)

	if child.Type() == "type" {
		return c.convert_type(child, source)
	}

	// Try expression
	expr := c.convert_expression(child, source)
	if expr != nil {
		return expr
	}

	if child.Type() == "$vasplat" {
		// TODO
		return nil
	}

	// Named arguments
	if node.ChildByFieldName("name") != nil {

	}

	return nil
}

/*
TODO Update with following definition:

		arg: $ => choice(
	      seq($.param_path),
	      seq($.param_path, '=', $._expr),
	      $._expr,
	    ),
*/
func (c *ASTConverter) convert_arg(node *sitter.Node, source []byte) ast.Expression {
	childCount := int(node.ChildCount())

	if is_literal(node.Child(0)) {
		return c.convert_literal(node.Child(0), source)
	}

	switch node.Child(0).Type() {
	case "param_path":
		param_path := node.Child(0)
		var arg ast.Expression
		param_path_element := param_path.Child(0)

		argType := 0
		for p := 0; p < int(param_path_element.ChildCount()); p++ {
			pNode := param_path_element.Child(p)
			if pNode.IsNamed() {
				if pNode.Type() == "ident" {
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
			n := node.Child(j)
			var expr ast.Expression
			if n.Type() == "type" {
				expr = c.convert_type(n, source)
			} else if n.Type() != "=" {
				expr = c.convert_expression(n, source)
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
		return ast.Expression(c.convert_type(node.Child(0), source))
	case "$vasplat":
		return &ast.BasicLit{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
			Kind:           ast.STRING,
			Value:          node.Content(source),
		}
	case "...":
		return c.convert_expression(node.Child(1), source)
	default:
		// try expr
		expr := c.convert_expression(node.Child(0), source)
		if expr != nil {
			return expr
		}
	}

	return nil
}

func (c *ASTConverter) convert_param_path(param_path *sitter.Node, source []byte) ast.Path {
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
		// TODO: Does Path need a NodeAttributes field??
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), param_path),
		PathType:       pathType,
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

func (c *ASTConverter) convert_flat_path(node *sitter.Node, source []byte) ast.Expression {
	node = node.Child(0)

	if node.Type() == "type" {
		return c.convert_type(node, source)
	}

	baseExpr := c.convert_base_expression(node, source)

	next := node.NextSibling()
	if next != nil {
		path := c.convert_param_path(next, source)
		switch path.PathType {
		case ast.PathTypeIndexed:
			return &ast.IndexAccessExpr{
				NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
				Array:          baseExpr,
				Index:          path.Path,
			}
		case ast.PathTypeField:
			return &ast.FieldAccessExpr{
				NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
				Object:         baseExpr,
				Field:          &path,
			}
		case ast.PathTypeRange:
			return &ast.RangeAccessExpr{
				NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
				Array:          baseExpr,
				RangeStart:     utils.StringToUint(path.PathStart),
				RangeEnd:       utils.StringToUint(path.PathEnd),
			}
		}
	}

	return baseExpr
}

func (c *ASTConverter) convert_range_expr(node *sitter.Node, source []byte) ast.Expression {
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
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Start:          left,
		End:            right,
	}
}

type NodeConverterSeparated func(node *sitter.Node, source []byte) (ast.Expression, int)

func (c *ASTConverter) convert_token_separated(node *sitter.Node, separator string, source []byte, convert_func nodeConverter) []ast.Node {
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

func (c *ASTConverter) convert_paren_expr(node *sitter.Node, source []byte) ast.Expression {
	child := node.Child(0)
	if child.Type() != "(" {
		panic(
			fmt.Sprintf("convert_paren_expr: Incorrect type. Expected \"(\": %s", node.Type()),
		)
	}

	next := child.NextSibling()
	return &ast.ParenExpr{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		X:              c.convert_expression(next, source),
	}
}

/*
// From grammar.js in tree-sitter-c3 v0.2.3:

// Doc comments and contracts
// -------------------------
// NOTE parsed by scanner.c (scan_doc_comment_contract_text)
doc_comment_contract: $ => seq(

	field('name', $.at_ident),
	optional($.doc_comment_contract_text)

),
doc_comment: $ => seq(

	'<*',
	optional($.doc_comment_text), // NOTE parsed by scanner.c (scan_doc_comment_text)
	repeat($.doc_comment_contract),
	'*>',

),

// (...)

at_ident: _ => token(seq('@', IDENT)),
*/
func (c *ASTConverter) convert_doc_comment(node *sitter.Node, sourceCode []byte) *ast.DocComment {
	body := ""
	bodyNode := node.Child(1)
	if bodyNode.Type() == "doc_comment_text" {
		body = dedent.Dedent(bodyNode.Content(sourceCode))
	}

	docComment := ast.NewDocComment(body)

	if node.ChildCount() >= 4 {
		for i := 2; i <= int(node.ChildCount())-2; i++ {
			contractNode := node.Child(i)
			if contractNode.Type() == "doc_comment_contract" {
				name := contractNode.ChildByFieldName("name").Content(sourceCode)
				body := ""
				if contractNode.ChildCount() >= 2 {
					// Right now, contracts can only have a single line, so we don't dedent.
					// They can also be arbitrary expressions, so it's best to not modify them
					// at the moment.
					body = contractNode.Child(1).Content(sourceCode)
				}

				contract := ast.NewDocCommentContract(name, body)

				docComment.AddContracts([]*ast.DocCommentContract{contract})
			}
		}
	}

	return docComment
}

func debugNode(node *sitter.Node, source []byte, tag string) {
	if node == nil {
		log.Printf("[tag: %s] Node is nil\n", tag)
		return
	}

	log.Printf("[tag: %s]: type: %s: content: %s ----> node: %s\n", tag, node.Type(), node.Content(source), node)
}
