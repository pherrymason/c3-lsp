package parser

import (
	"errors"
	"fmt"
	"strings"

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
	attributes := parseNodeAttributes(node, sourceCode)
	funcHeader := firstChildOfType(node, "func_header")

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
	if typeIdentifier == "" {
		typeIdentifier = inferMethodTypeFromHeader(funcHeader, nameNode, sourceCode)
	}

	var argumentIds []string
	var arguments []*idx.Variable
	parameters := firstChildOfType(node, "func_param_list")
	if parameters == nil {
		return idx.Function{}, errors.New("func parameters not found")
	}
	parameterIndex := 0

	if parameters.ChildCount() > 2 {
		for i := uint32(0); i < parameters.ChildCount(); i++ {
			argNode := parameters.Child(int(i))
			if argNode.Type() != "param" {
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
			idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node),
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
			idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node),
				node.EndPoint()),
		)
	}

	var variables []*idx.Variable
	if node.ChildByFieldName("body") != nil {
		variables = p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
	}

	variables = append(variables, arguments...)
	p.inferForeachIteratorVariableTypes(node, currentModule, variables, sourceCode)

	symbol.AddVariables(variables)
	symbol.SetAttributes(attributes)

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
			if identifier == "self" && methodIdentifier != "" && !foundType {
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
		case "param_default":
			assigned := n.ChildByFieldName("right")
			if assigned != nil {
				paramDefault = option.Some(assigned.Content(sourceCode))
			}
		}
	}

	if methodIdentifier != "" && parameterIndex == 0 && !foundType && ref == "*" {
		argType = idx.NewTypeFromString(methodIdentifier+ref, currentModule.GetModuleString())
	}

	// if identifier is empty (unnamed argument), then use generic $arg{parameterIndex} name
	if len(identifier) == 0 {
		identifier = fmt.Sprintf("$arg#%d", parameterIndex)
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
	if hasImmediateErrorNode(node) {
		return idx.Function{}, errors.New("invalid macro declaration")
	}

	attributes := parseNodeAttributes(node, sourceCode)

	var nameNode *sitter.Node
	macroHeader := firstChildOfType(node, "macro_header")

	if macroHeader == nil {
		return idx.Function{}, errors.New("child node not found")
	}

	nameNode = macroHeader.ChildByFieldName("name")

	if nameNode == nil {
		return idx.Function{}, errors.New("child node not found")
	}

	var typeIdentifier = ""
	var returnType *idx.Type = nil

	if macroHeader.Type() == "macro_header" {
		methodTypeNode := macroHeader.ChildByFieldName("method_type")
		if methodTypeNode != nil {
			typeIdentifier = methodTypeNode.Content(sourceCode)
		}

		returnTypeNode := macroHeader.ChildByFieldName("return_type")
		if returnTypeNode != nil {
			if returnTypeNode.Type() == "func_signature" {
				return idx.Function{}, errors.New("invalid macro return type")
			}
			returnType = cast.ToPtr(p.typeNodeToType(returnTypeNode, currentModule, sourceCode))
		}
	}

	var argumentIds []string
	arguments := []*idx.Variable{}
	parameters := firstChildOfType(node, "macro_param_list")
	if parameters == nil {
		return idx.Function{}, errors.New("macro parameters not found")
	}
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
				if argNode.ChildCount() >= 2 && argNode.Child(1).Type() == "func_param_list" {
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
			} else if argNode.Type() == "param" {
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
			idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node),
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
			idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node),
				node.EndPoint()),
		)
	}

	if node.ChildByFieldName("body") != nil {
		variables := p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
		variables = append(arguments, variables...)
		symbol.AddVariables(variables)
	}

	symbol.SetAttributes(attributes)

	return symbol, nil
}

func hasImmediateErrorNode(node *sitter.Node) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "ERROR" {
			return true
		}
	}

	return false
}

func (p *Parser) inferForeachIteratorVariableTypes(node *sitter.Node, currentModule *idx.Module, variables []*idx.Variable, sourceCode []byte) {
	if node == nil || currentModule == nil {
		return
	}

	byKey := map[string]*idx.Variable{}
	for _, variable := range variables {
		if variable == nil {
			continue
		}
		byKey[variableLookupKey(variable.GetName(), variable.GetIdRange())] = variable
	}

	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		if n.Type() == "foreach_stmt" {
			p.inferForeachIteratorVariableTypesInStmt(n, currentModule, byKey, sourceCode)
		}

		for i := uint32(0); i < n.ChildCount(); i++ {
			walk(n.Child(int(i)))
		}
	}

	walk(node)
}

func (p *Parser) inferForeachIteratorVariableTypesInStmt(stmt *sitter.Node, currentModule *idx.Module, byKey map[string]*idx.Variable, sourceCode []byte) {
	foreachVar := firstChildOfType(stmt, "foreach_var")
	foreachCond := firstChildOfType(stmt, "foreach_cond")
	if foreachVar == nil && foreachCond != nil {
		foreachVar = firstChildOfType(foreachCond, "foreach_var")
	}
	if foreachVar == nil || foreachCond == nil {
		return
	}

	iterType, ok := p.resolveForeachIterableType(strings.TrimSpace(foreachCond.Content(sourceCode)), currentModule, byKey)
	if !ok {
		return
	}

	elementType, ok := foreachElementType(iterType)
	if !ok {
		return
	}

	identNames := []string{}
	identRanges := []idx.Range{}
	foreachVarNodes := []*sitter.Node{}
	for i := 0; i < int(foreachCond.NamedChildCount()); i++ {
		child := foreachCond.NamedChild(i)
		if child != nil && child.Type() == "foreach_var" {
			foreachVarNodes = append(foreachVarNodes, child)
		}
	}
	if len(foreachVarNodes) == 0 {
		foreachVarNodes = append(foreachVarNodes, foreachVar)
	}

	for _, foreachVarNode := range foreachVarNodes {
		for i := 0; i < int(foreachVarNode.NamedChildCount()); i++ {
			child := foreachVarNode.NamedChild(i)
			if child == nil || child.Type() != "ident" {
				continue
			}

			name := child.Content(sourceCode)
			if name == "" {
				continue
			}

			identNames = append(identNames, name)
			identRanges = append(identRanges, idx.NewRangeFromTreeSitterPositions(child.StartPoint(), child.EndPoint()))
		}
	}

	for i, name := range identNames {
		idRange := identRanges[i]
		inferredType := elementType
		if len(identNames) > 1 && i == 0 {
			inferredType = idx.NewTypeFromString("usz", currentModule.GetModuleString())
		}

		variable, found := byKey[variableLookupKey(name, idRange)]
		if found {
			if variable.GetType().GetName() != "" {
				continue
			}
			variable.Type = inferredType
			continue
		}

		for _, candidate := range byKey {
			if candidate.GetName() == name && candidate.GetType().GetName() == "" {
				candidate.Type = inferredType
				break
			}
		}
	}
}

func (p *Parser) resolveForeachIterableType(foreachCond string, currentModule *idx.Module, byKey map[string]*idx.Variable) (idx.Type, bool) {
	expr := strings.TrimSpace(foreachCond)
	if separator := strings.LastIndex(expr, ":"); separator >= 0 && separator+1 < len(expr) {
		expr = strings.TrimSpace(expr[separator+1:])
	}
	expr = strings.TrimSpace(strings.Trim(expr, "()"))
	if expr == "" {
		return idx.Type{}, false
	}

	segments := strings.Split(expr, ".")
	for i := range segments {
		segments[i] = strings.TrimSpace(segments[i])
	}
	if len(segments) == 0 || !isSimpleIdentifier(segments[0]) {
		return idx.Type{}, false
	}

	baseType, ok := p.resolveForeachBaseIdentifierType(segments[0], currentModule, byKey)
	if !ok {
		return idx.Type{}, false
	}

	currentType := baseType
	for i := 1; i < len(segments); i++ {
		segment := segments[i]
		if !isSimpleIdentifier(segment) {
			return idx.Type{}, false
		}

		strukt, found := currentModule.Structs[currentType.GetName()]
		if !found {
			return idx.Type{}, false
		}

		memberType, found := foreachStructMemberType(strukt, segment)
		if !found {
			return idx.Type{}, false
		}

		currentType = memberType
	}

	return currentType, true
}

func (p *Parser) resolveForeachBaseIdentifierType(name string, currentModule *idx.Module, byKey map[string]*idx.Variable) (idx.Type, bool) {
	for _, variable := range byKey {
		if variable != nil && variable.GetName() == name && variable.GetType().GetName() != "" {
			return *variable.GetType(), true
		}
	}

	if moduleVar, found := currentModule.Variables[name]; found && moduleVar.GetType().GetName() != "" {
		return *moduleVar.GetType(), true
	}

	return idx.Type{}, false
}

func foreachStructMemberType(strukt *idx.Struct, memberName string) (idx.Type, bool) {
	for _, member := range strukt.GetMembers() {
		if member.GetName() == memberName {
			return *member.GetType(), true
		}
	}

	return idx.Type{}, false
}

func foreachElementType(iterType idx.Type) (idx.Type, bool) {
	if iterType.HasGenericArguments() {
		return iterType.GetGenericArgument(0), true
	}

	if !iterType.IsCollection() {
		return idx.Type{}, false
	}

	module := iterType.GetModule()

	return idx.NewType(
		iterType.IsBaseTypeLanguage(),
		iterType.GetName(),
		iterType.GetPointerCount(),
		iterType.IsGenericArgument(),
		false,
		option.None[int](),
		module,
	), true
}

func variableLookupKey(name string, idRange idx.Range) string {
	return fmt.Sprintf("%s:%d:%d:%d:%d", name, idRange.Start.Line, idRange.Start.Character, idRange.End.Line, idRange.End.Character)
}

func isSimpleIdentifier(value string) bool {
	if value == "" {
		return false
	}

	for i, r := range value {
		if r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (i <= 0 || r < '0' || r > '9') {
			return false
		}
	}

	return true
}

func inferMethodTypeFromHeader(funcHeader *sitter.Node, nameNode *sitter.Node, sourceCode []byte) string {
	if funcHeader == nil || nameNode == nil {
		return ""
	}

	start := int(nameNode.StartByte())
	if start <= 0 || start > len(sourceCode) {
		return ""
	}

	i := start - 1
	for i >= 0 && (sourceCode[i] == ' ' || sourceCode[i] == '\t' || sourceCode[i] == '\n' || sourceCode[i] == '\r') {
		i--
	}
	if i < 0 || sourceCode[i] != '.' {
		return ""
	}

	i--
	for i >= 0 && (sourceCode[i] == ' ' || sourceCode[i] == '\t' || sourceCode[i] == '\n' || sourceCode[i] == '\r') {
		i--
	}
	if i < 0 {
		return ""
	}

	end := i
	for i >= 0 {
		c := sourceCode[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == ':' {
			i--
			continue
		}
		break
	}

	if i == end {
		return ""
	}

	typeIdentifier := strings.TrimSpace(string(sourceCode[i+1 : end+1]))
	if !strings.Contains(typeIdentifier, ":") && !isSimpleIdentifier(typeIdentifier) {
		return ""
	}

	return typeIdentifier
}
