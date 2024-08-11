package ast

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

func convert_function_declaration(node *sitter.Node, source []byte) Expression {
	var typeIdentifier option.Option[Identifier]
	funcHeader := node.Child(1)

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
		debugNode(bodyNode, source)
		n := bodyNode.Child(0)
		// options
		// compound_stmt
		// => expr;
		if n.Type() == "compound_stmt" {
			body = convert_compound_stmt(n, source)
		} else {
			debugNode(n.NextSibling(), source)
			body = convert_expression(n.NextSibling(), source)
		}
	}

	funcDecl := FunctionDecl{
		ASTNodeBase:  NewBaseNodeBuilder().WithSitterPos(node).Build(),
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
		ReturnType: convert_type(funcHeader.ChildByFieldName("return_type"), sourceCode),
		Parameters: convert_function_parameter_list(node.Child(2), typeIdentifier, sourceCode),
		ASTNodeBase: NewBaseNodeBuilder().
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

	variable := FunctionParameter{
		Name:        identifier,
		Type:        argType,
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(argNode).Build(),
	}

	return variable
}
