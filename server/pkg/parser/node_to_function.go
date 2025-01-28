package parser

import (
	"errors"
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/option"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

/*
		func_definition: $ => seq(
	      'fn',
	      $.func_header,
	      $.fn_parameter_list,
	      optional($.attributes),
	      field('body', $.macro_func_body),
	    ),
		func_header: $ => seq(
			field('return_type', $._type_or_optional_type),
			optional(seq(field('method_type', $.type), '.')),
			field('name', $._func_macro_name),
		),
*/
func (p *Parser) nodeToFunction(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) (idx.Function, error) {
	var typeIdentifier string
	funcHeader := node.Child(1)

	if funcHeader == nil {
		return idx.Function{}, errors.New("child node not found")
	}

	nameNode := funcHeader.ChildByFieldName("name")

	if nameNode == nil {
		return idx.Function{}, errors.New("child node not found")
	}

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = funcHeader.ChildByFieldName("method_type").Content(sourceCode)
	}

	functionName := nameNode.Content(sourceCode)

	var argumentIds []string
	var arguments []*idx.Variable
	parameters := node.Child(2)
	parameterIndex := 0

	if parameters.ChildCount() > 2 {
		for i := uint32(0); i < parameters.ChildCount(); i++ {
			argNode := parameters.Child(int(i))
			if argNode.Type() != "parameter" {
				continue
			}

			argument := p.nodeToArgument(argNode, typeIdentifier, currentModule, docId, sourceCode, parameterIndex)
			arguments = append(
				arguments,
				argument,
			)

			argumentIds = append(argumentIds, argument.GetName())
			parameterIndex += 1
		}
	}

	var symbol idx.Function
	if typeIdentifier != "" {
		symbol = idx.NewTypeFunction(
			typeIdentifier,
			functionName,
			p.typeNodeToType(funcHeader.ChildByFieldName("return_type"), currentModule, sourceCode),
			//funcHeader.ChildByFieldName("return_type").Content(sourceCode),
			argumentIds,
			currentModule.GetModuleString(),
			*docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
				nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
				node.EndPoint()),
			protocol.CompletionItemKindFunction,
		)
	} else {
		symbol = idx.NewFunction(
			functionName,
			p.typeNodeToType(funcHeader.ChildByFieldName("return_type"), currentModule, sourceCode),
			argumentIds,
			currentModule.GetModuleString(),
			*docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
				nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
				node.EndPoint()),
		)
	}

	var variables []*idx.Variable
	if node.ChildByFieldName("body") != nil {
		variables = p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
	}

	variables = append(variables, arguments...)

	symbol.AddVariables(variables)

	return symbol, nil
}

// nodeToArgument Very similar to nodeToVariable, but arguments have optional identifiers (for example when using `self` for struct methods)
/*
	_assign_right_expr: $ => seq('=', field('right', $._expr)),
	parameter_default: $ => $._assign_right_expr,
	parameter: $ => seq($._parameter, optional($.parameter_default))
    _parameter: $ => choice(
        // Typed parameters
        seq(
	        field('type', $.type),  // 1
	        optional(choice(
	            '...',  															   // 2
	            seq(optional('...'), field('name', $.ident), optional($.attributes)),  // 2/3/4
	            // Macro parameters
	            seq(field('name', $.ct_ident), optional($.attributes)),				   // 2/3
	            seq(field('name', $.hash_ident), optional($.attributes)),			   // 2/3
	            seq('&', field('name', $.ident), optional($.attributes)), 			   // 3/4
	        ))
        ),

        // Untyped parameters
        '...',																			// 1
        seq(field('name', $.ident), optional('...'), optional($.attributes)),           // 2/3/4
        // Macro parameters
        seq(field('name', $.ct_ident), optional($.attributes)),                         // 1/2
        seq(field('name', $.hash_ident), optional($.attributes)),                       // 1/2
        seq('&', field('name', $.ident), optional($.attributes)),                       // 2/3
    ),
*/
func (p *Parser) nodeToArgument(argNode *sitter.Node, methodIdentifier string, currentModule *idx.Module, docId *string, sourceCode []byte, parameterIndex int) *idx.Variable {
	var identifier = ""
	var idRange idx.Range
	var argType idx.Type
	foundType := false
	varArg := false
	ref := ""
	paramDefault := option.None[string]()

	for i := uint32(0); i < argNode.ChildCount(); i++ {
		n := argNode.Child(int(i))

		switch n.Type() {
		case "type":
			argType = p.typeNodeToType(n, currentModule, sourceCode)
			foundType = true
		case "...":
			varArg = true
			if foundType {
				// int.. args. -> int[] args
				argType = argType.UnsizedCollectionOf()
			} else {
				// args... -> any*... args -> any*[] args
				argType = idx.
					NewTypeFromString("any*", currentModule.GetModuleString()).
					UnsizedCollectionOf()
			}
		case "&":
			ref = "*"
		case "ident":
			identifier = n.Content(sourceCode)
			idRange = idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint())
			// When detecting a self, the type is the Struct type, plus '*' for '&self'
			if identifier == "self" && methodIdentifier != "" {
				argType = idx.NewTypeFromString(methodIdentifier+ref, currentModule.GetModuleString())
			}

		// $arg (macro)
		case "ct_ident":
			identifier = n.Content(sourceCode)
			idRange = idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint())

		// #arg (macro)
		case "hash_ident":
			identifier = n.Content(sourceCode)
			idRange = idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint())

		// = default
		case "parameter_default":
			assigned := n.ChildByFieldName("right")
			if assigned != nil {
				paramDefault = option.Some(assigned.Content(sourceCode))
			}
		}
	}

	// if identifier is empty (unnamed argument), then use generic $arg{parameterIndex} name
	if len(identifier) == 0 {
		identifier = fmt.Sprintf("$arg%d", parameterIndex)
	}

	variable := idx.NewVariable(
		identifier,
		argType,
		currentModule.GetModuleString(),
		*docId,
		idRange,
		idx.NewRangeFromTreeSitterPositions(argNode.StartPoint(),
			argNode.EndPoint()),
	)

	variable.Arg.VarArg = varArg
	variable.Arg.Default = paramDefault

	return &variable
}

/*
		trailing_block_param: $ => seq(
	      $.at_ident,
	      optional($.fn_parameter_list),
	    ),
		macro_parameter_list: $ => seq(
		  '(',
		  optional(
		    choice(
		      $._parameters,
		      seq(
		        optional($._parameters),
		        ';',
		        $.trailing_block_param,
		      ),
		    ),
		  ),
		  ')',
		),
		macro_declaration: $ => seq(
		  'macro',
		  $.macro_header,
		  $.macro_parameter_list,
		  optional($.attributes),
		  field('body', $.macro_func_body),
		),

	    macro_header: $ => seq(
	      optional(field('return_type', $._type_optional)), // Return type is optional for macros
	      optional(seq(field('method_type', $.type), '.')),
	      field('name', $._func_macro_name),
	    ),
*/
func (p *Parser) nodeToMacro(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) (idx.Function, error) {
	var nameNode *sitter.Node
	macroHeader := node.Child(1)

	if macroHeader == nil {
		return idx.Function{}, errors.New("child node not found")
	}

	nameNode = macroHeader.ChildByFieldName("name")

	if nameNode == nil {
		return idx.Function{}, errors.New("child node not found")
	}

	var typeIdentifier string = ""
	var returnType *idx.Type = nil

	if macroHeader.Type() == "macro_header" {
		methodTypeNode := macroHeader.ChildByFieldName("method_type")
		if methodTypeNode != nil {
			typeIdentifier = methodTypeNode.Content(sourceCode)
		}

		returnTypeNode := macroHeader.ChildByFieldName("return_type")
		if returnTypeNode != nil {
			returnType = cast.ToPtr(p.typeNodeToType(returnTypeNode, currentModule, sourceCode))
		}
	}

	var argumentIds []string
	arguments := []*idx.Variable{}
	parameters := node.Child(2)
	parameterIndex := 0

	if parameters.ChildCount() > 2 {
		for i := uint32(0); i < parameters.ChildCount(); i++ {
			var argument *idx.Variable
			argNode := parameters.Child(int(i))

			// '@body' in macro name(args; @body) { ... }
			if argNode.Type() == "trailing_block_param" {
				identNode := argNode.Child(0)
				identifier := identNode.Content(sourceCode)
				idRange := idx.NewRangeFromTreeSitterPositions(identNode.StartPoint(), identNode.EndPoint())

				// Get body function signature
				// If it's missing, it's just empty args
				bodyParams := "()"
				if argNode.ChildCount() >= 2 && argNode.Child(1).Type() == "fn_parameter_list" {
					// TODO: Maybe we should properly parse the parameters at some point
					// For now, simple string manipulation suffices
					bodyParams = argNode.Child(1).Content(sourceCode)
				}

				// '@body' is equivalent to a function
				// Use a callback type
				argType := idx.NewTypeFromString("fn void"+bodyParams, currentModule.GetModuleString())

				variable := idx.NewVariable(
					identifier,
					argType,
					currentModule.GetModuleString(),
					*docId,
					idRange,
					idx.NewRangeFromTreeSitterPositions(argNode.StartPoint(),
						argNode.EndPoint()),
				)

				argument = &variable
			} else if argNode.Type() == "parameter" {
				argument = p.nodeToArgument(argNode, typeIdentifier, currentModule, docId, sourceCode, parameterIndex)
			} else {
				continue
			}

			arguments = append(
				arguments,
				argument,
			)
			argumentIds = append(argumentIds, argument.GetName())
			parameterIndex += 1
		}
	}

	macroName := nameNode.Content(sourceCode)

	var symbol idx.Function
	if typeIdentifier != "" {
		symbol = idx.NewTypeMacro(
			typeIdentifier,
			macroName,
			argumentIds,
			returnType,
			currentModule.GetModuleString(),
			*docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
				nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
				node.EndPoint()),
			protocol.CompletionItemKindFunction,
		)
	} else {
		symbol = idx.NewMacro(
			macroName,
			argumentIds,
			returnType,
			currentModule.GetModuleString(),
			*docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
				nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
				node.EndPoint()),
		)
	}

	if node.ChildByFieldName("body") != nil {
		variables := p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
		variables = append(arguments, variables...)
		symbol.AddVariables(variables)
	}

	return symbol, nil
}
