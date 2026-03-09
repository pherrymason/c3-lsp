package parser

import (
	"strings"
	"unicode"

	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) unwrapBindingNodeToVariable(bindingNode *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) *idx.Variable {
	if bindingNode == nil {
		return nil
	}

	var idNode *sitter.Node
	var vType idx.Type
	typedBinding := false
	hasAssignment := false

	for i := uint32(0); i < bindingNode.ChildCount(); i++ {
		child := bindingNode.Child(int(i))
		switch child.Type() {
		case "type":
			vType = p.typeNodeToType(child, currentModule, sourceCode)
			typedBinding = true
		case "ident":
			if idNode == nil {
				idNode = child
			}
		case "=":
			hasAssignment = true
		}
	}

	if !hasAssignment || idNode == nil {
		return nil
	}

	if !typedBinding {
		if bindingNode.Type() == "catch_unwrap" {
			vType = idx.NewTypeFromString("fault", currentModule.GetModuleString())
		} else if inferredType, ok := inferTypeFromUnwrapBindingSource(bindingNode.Content(sourceCode), currentModule.GetModuleString()); ok {
			vType = inferredType
		} else {
			vType = idx.NewTypeFromString("", currentModule.GetModuleString())
		}
	}

	variable := idx.NewVariable(
		idNode.Content(sourceCode),
		vType,
		currentModule.GetModuleString(),
		*docId,
		idx.NewRangeFromTreeSitterPositions(idNode.StartPoint(), idNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(bindingNode.StartPoint(), bindingNode.EndPoint()),
	)

	return &variable
}

func inferTypeFromUnwrapBindingSource(bindingSource string, module string) (idx.Type, bool) {
	eqIndex := strings.Index(bindingSource, "=")
	if eqIndex < 0 {
		return idx.Type{}, false
	}

	rhs := strings.TrimSpace(bindingSource[eqIndex+1:])
	openParen := strings.Index(rhs, "(")
	if openParen < 0 {
		return idx.Type{}, false
	}

	callee := strings.TrimSpace(rhs[:openParen])
	methodName := trailingIdentifier(callee)
	if !supportsUnwrapInferenceForMethod(methodName) {
		return idx.Type{}, false
	}

	firstArg, ok := firstCallArgument(rhs, openParen)
	if !ok || !looksLikeTypeToken(firstArg) {
		return idx.Type{}, false
	}

	return idx.NewTypeFromString(firstArg, module), true
}

func trailingIdentifier(input string) string {
	end := len(input) - 1
	for end >= 0 && !isIdentChar(rune(input[end])) {
		end--
	}
	if end < 0 {
		return ""
	}

	start := end
	for start >= 0 && isIdentChar(rune(input[start])) {
		start--
	}

	return input[start+1 : end+1]
}

func supportsUnwrapInferenceForMethod(name string) bool {
	switch name {
	case "to_integer", "to_int", "to_long", "to_short", "to_ichar", "to_uint", "to_ulong", "to_ushort", "to_uchar":
		return true
	default:
		return false
	}
}

func firstCallArgument(rhs string, openParen int) (string, bool) {
	argStart := openParen + 1
	if argStart >= len(rhs) {
		return "", false
	}

	depth := 0
	for i := argStart; i < len(rhs); i++ {
		switch rhs[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				arg := strings.TrimSpace(rhs[argStart:i])
				if arg == "" {
					return "", false
				}
				return arg, true
			}
			depth--
		case ',':
			if depth == 0 {
				arg := strings.TrimSpace(rhs[argStart:i])
				if arg == "" {
					return "", false
				}
				return arg, true
			}
		}
	}

	return "", false
}

func looksLikeTypeToken(token string) bool {
	if token == "" {
		return false
	}

	for _, r := range token {
		if isIdentChar(r) || r == ':' || r == '[' || r == ']' || r == '*' || r == '?' {
			continue
		}
		return false
	}

	return true
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (p *Parser) variableDeclarationNodeToVariable(declarationNode *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) []*idx.Variable {
	var variables []*idx.Variable
	//var typeNodeContent string
	var vType idx.Type

	//fmt.Println(declarationNode.ChildCount())
	//fmt.Println(declarationNode)
	//fmt.Println(declarationNode.Content(sourceCode))
	//fmt.Println("----")

	for i := uint32(0); i < declarationNode.ChildCount(); i++ {
		n := declarationNode.Child(int(i))
		//fmt.Println(i, ":", n.Type(), ":: ", n.Content(sourceCode), ":: has errors: ", n.HasError())
		switch n.Type() {
		case "type":
			//typeNodeContent = n.Content(sourceCode)
			vType = p.typeNodeToType(n, currentModule, sourceCode)
		case "ident":
			variable := idx.NewVariable(
				n.Content(sourceCode),
				vType,
				//idx.NewTypeFromString(typeNodeContent, moduleName), // <-- moduleName is potentially wrong
				currentModule.GetModuleString(),
				*docId,
				idx.NewRangeFromTreeSitterPositions(
					n.StartPoint(),
					n.EndPoint(),
				),
				idx.NewRangeFromTreeSitterPositions(
					declarationNode.StartPoint(),
					declarationNode.EndPoint()),
			)
			variables = append(variables, &variable)
		case "identifier_list":
			for j := 0; j < int(n.ChildCount()); j++ {

				bn := n.Child(j)
				if bn.Type() != "ident" {
					continue
				}
				variable := idx.NewVariable(
					bn.Content(sourceCode),
					vType,
					//idx.NewTypeFromString(typeNodeContent, moduleName), // <-- moduleName is potentially wrong
					currentModule.GetModuleString(),
					*docId,
					idx.NewRangeFromTreeSitterPositions(
						bn.StartPoint(),
						bn.EndPoint(),
					),
					idx.NewRangeFromTreeSitterPositions(
						declarationNode.StartPoint(),
						declarationNode.EndPoint()),
				)
				variables = append(variables, &variable)
			}
		case ";":
			if n.HasError() && len(variables) > 0 {
				// Last variable is incomplete, remove it
				variables = variables[:len(variables)-1]
			}

		}

	}

	return variables
}

/*
		const_declaration: $ => seq(
	      'const',
	      field('type', optional($.type)),
	      $.const_ident,
	      optional($.attributes),
	      optional($._assign_right_expr),
	      ';'
	    )
*/
func (p *Parser) nodeToConstant(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Variable {
	var constant idx.Variable
	var typeNodeContent string
	var idNode *sitter.Node
	constantValue := ""

	//fmt.Println(node.ChildCount())
	//fmt.Println(node)
	//fmt.Println(node.Content(sourceCode))

	for i := uint32(0); i < node.ChildCount(); i++ {
		n := node.Child(int(i))
		switch n.Type() {
		case "type":
			typeNodeContent = n.Content(sourceCode)

		case "const_ident":
			idNode = n
		case "_assign_right_expr":
			value := strings.TrimSpace(n.Content(sourceCode))
			value = strings.TrimSpace(strings.TrimPrefix(value, "="))
			constantValue = strings.TrimSuffix(value, ";")
		}
	}

	if constantValue == "" {
		constantValue = parseConstantValueFromDeclaration(node.Content(sourceCode))
	}

	constant = idx.NewConstant(
		idNode.Content(sourceCode),
		idx.NewTypeFromString(typeNodeContent, currentModule.GetModuleString()), // <-- moduleName is potentially wrong
		currentModule.GetModuleString(),
		*docId,
		idx.NewRangeFromTreeSitterPositions(
			idNode.StartPoint(),
			idNode.EndPoint(),
		),
		idx.NewRangeFromTreeSitterPositions(
			node.StartPoint(),
			node.EndPoint()),
	)
	constant.SetConstantValue(constantValue)

	return constant
}

func parseConstantValueFromDeclaration(content string) string {
	eqIndex := strings.Index(content, "=")
	if eqIndex < 0 {
		return ""
	}

	value := strings.TrimSpace(content[eqIndex+1:])
	value = strings.TrimSuffix(value, ";")
	return strings.TrimSpace(value)
}
