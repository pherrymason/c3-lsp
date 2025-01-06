package factory

import (
	"errors"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

type nodeConverter func(node *sitter.Node, source []byte) ast.Node
type conversionInfo struct {
	method  nodeConverter
	goChild bool
}

func (c conversionInfo) convert(node *sitter.Node, source []byte, debug bool) ast.Node {
	n := node
	if c.goChild {
		n = node.Child(0)
	}

	if debug {
		//debugNode()
	}

	return c.method(n, source)
}

type StatementConverter func(node *sitter.Node, source []byte) ast.Statement
type ExpressionConverter func(node *sitter.Node, source []byte) ast.Expression
type DeclarationConverter func(node *sitter.Node, source []byte) ast.Declaration

func cv_stmt_fn(fn StatementConverter) nodeConverter {
	return func(node *sitter.Node, source []byte) ast.Node {
		return fn(node, source).(ast.Node)
	}
}

func cv_expr_fn(fn ExpressionConverter) nodeConverter {
	return func(node *sitter.Node, source []byte) ast.Node {
		return fn(node, source).(ast.Node)
	}
}

func cv_decl_fn(fn DeclarationConverter) nodeConverter {
	return func(node *sitter.Node, source []byte) ast.Node {
		return fn(node, source).(ast.Node)
	}
}

func (c *ASTConverter) generateConversionInfo() {
	c.rules = map[string]conversionInfo{
		// Expressions
		"paren_expr": {method: cv_expr_fn(c.convert_paren_expr)},

		// Statements

		"assignment_expr":        {method: cv_expr_fn(c.convert_assignment_expr)},
		"assert_stmt":            {method: cv_stmt_fn(c.convert_assert_stmt)},
		"at_ident":               {method: cv_expr_fn(c.convert_ident)},
		"binary_expr":            {method: cv_expr_fn(c.convert_binary_expr)},
		"break_stmt":             {method: cv_stmt_fn(c.convert_break_stmt)},
		"bytes_expr":             {method: cv_expr_fn(c.convert_bytes_expr)},
		"builtin":                {method: cv_expr_fn(c.convert_as_literal)},
		"call_expr":              {method: cv_expr_fn(c.convert_call_expr)},
		"continue_stmt":          {method: cv_stmt_fn(c.convert_continue_stmt)},
		"cast_expr":              {method: cv_expr_fn(c.convert_cast_expr)},
		"const_ident":            {method: cv_expr_fn(c.convert_ident)},
		"compound_stmt":          {method: cv_stmt_fn(c.convert_compound_stmt)},
		"ct_ident":               {method: cv_expr_fn(c.convert_ident)},
		"declaration_stmt":       {method: cv_stmt_fn(c.convert_declaration_stmt)},
		"defer_stmt":             {method: cv_stmt_fn(c.convert_defer_stmt)},
		"do_stmt":                {method: cv_stmt_fn(c.convert_do_stmt)},
		"split_declaration_stmt": {method: cv_stmt_fn(c.convert_split_declaration_stmt)},
		"elvis_orelse_expr":      {method: cv_expr_fn(c.convert_elvis_orelse_expr)},
		"expr_stmt":              {method: cv_stmt_fn(c.convert_expr_stmt), goChild: true},
		"for_stmt":               {method: cv_stmt_fn(c.convert_for_stmt)},
		"foreach_stmt":           {method: cv_stmt_fn(c.convert_foreach_stmt)},
		"hash_ident":             {method: cv_expr_fn(c.convert_ident)},
		"ident":                  {method: cv_expr_fn(c.convert_ident)},
		"if_stmt":                {method: cv_stmt_fn(c.convert_if_stmt)},
		"initializer_list":       {method: cv_expr_fn(c.convert_initializer_list)},
		"type_access_expr":       {method: cv_expr_fn(c.convert_type_access_expr)},

		"lambda_declaration":    {method: cv_expr_fn(c.convert_lambda_declaration)},
		"lambda_expr":           {method: cv_expr_fn(c.convert_lambda_expr)},
		"local_decl_after_type": {method: cv_decl_fn(c.convert_local_declaration_after_type)},
		"module_ident_expr":     {method: cv_expr_fn(c.convert_module_ident_expr)},
		"nextcase_stmt":         {method: cv_stmt_fn(c.convert_nextcase_stmt)},
		"optional_expr":         {method: cv_expr_fn(c.convert_optional_expr)},
		"rethrow_expr":          {method: cv_expr_fn(c.convert_rethrow_expr)},
		"return_stmt":           {method: cv_stmt_fn(c.convert_return_stmt)},
		//"suffix_expr":           convert_dummy,
		"subscript_expr":        {method: cv_expr_fn(c.convert_subscript_expr)},
		"switch_stmt":           {method: cv_stmt_fn(c.convert_switch_stmt)},
		"ternary_expr":          {method: cv_expr_fn(c.convert_ternary_expr)},
		"trailing_generic_expr": {method: cv_expr_fn(c.convert_trailing_generic_expr)},
		"type": {method: func(node *sitter.Node, source []byte) ast.Node {
			n := c.convert_type(node, source)
			return n
		}},
		"unary_expr":  {method: cv_expr_fn(c.convert_unary_expr)},
		"update_expr": {method: cv_expr_fn(c.convert_update_expr)},
		"var_stmt":    {method: cv_decl_fn(c.convert_var_decl), goChild: true},
		"while_stmt":  {method: cv_stmt_fn(c.convert_while_stmt)},
		"field_expr":  {method: cv_expr_fn(c.convert_field_expr)},

		// Builtins ----------------
		"$vacount": {method: cv_expr_fn(c.convert_as_literal)},
		"$feature": {method: cv_expr_fn(c.convert_feature)},

		"$alignof":   {method: cv_expr_fn(c.convert_compile_time_call)},
		"$extnameof": {method: cv_expr_fn(c.convert_compile_time_call)},
		"$nameof":    {method: cv_expr_fn(c.convert_compile_time_call)},
		"$offsetof":  {method: cv_expr_fn(c.convert_compile_time_call)},
		"$qnameof":   {method: cv_expr_fn(c.convert_compile_time_call)},

		"$vaconst": {method: cv_expr_fn(c.convert_compile_time_arg)},
		"$vaarg":   {method: cv_expr_fn(c.convert_compile_time_arg)},
		"$varef":   {method: cv_expr_fn(c.convert_compile_time_arg)},
		"$vaexpr":  {method: cv_expr_fn(c.convert_compile_time_arg)},

		"$eval":      {method: cv_expr_fn(c.convert_compile_time_analyse)},
		"$is_const":  {method: cv_expr_fn(c.convert_compile_time_analyse)},
		"$sizeof":    {method: cv_expr_fn(c.convert_compile_time_analyse)},
		"$stringify": {method: cv_expr_fn(c.convert_compile_time_analyse)},

		"$and":     {method: cv_expr_fn(c.convert_compile_time_call_unk)},
		"$append":  {method: cv_expr_fn(c.convert_compile_time_call_unk)},
		"$concat":  {method: cv_expr_fn(c.convert_compile_time_call_unk)},
		"$defined": {method: cv_expr_fn(c.convert_compile_time_call_unk)},
		"$embed":   {method: cv_expr_fn(c.convert_compile_time_call_unk)},
		"$or":      {method: cv_expr_fn(c.convert_compile_time_call_unk)},

		"_expr":      {method: cv_expr_fn(c.convert_expression)},
		"_base_expr": {method: cv_expr_fn(c.convert_base_expression)},
		"_statement": {method: cv_stmt_fn(c.convert_statement)},

		// Literals
		"string_literal":     {method: cv_expr_fn(c.convert_literal)},
		"char_literal":       {method: cv_expr_fn(c.convert_literal)},
		"raw_string_literal": {method: cv_expr_fn(c.convert_literal)},
		"integer_literal":    {method: cv_expr_fn(c.convert_literal)},
		"real_literal":       {method: cv_expr_fn(c.convert_literal)},
		"bytes_literal":      {method: cv_expr_fn(c.convert_literal)},
		"true":               {method: cv_expr_fn(c.convert_literal)},
		"false":              {method: cv_expr_fn(c.convert_literal)},
		"null":               {method: cv_expr_fn(c.convert_literal)},

		// Custom ones ----------------
		"..type_with_initializer_list..":   {method: cv_expr_fn(c.convert_type_with_initializer_list)},
		"..lambda_declaration_with_body..": {method: cv_expr_fn(c.convert_lambda_declaration_with_body)},
	}
}

func (c *ASTConverter) nodeTypeConverterMap(nodeType string) (conversionInfo, error) {

	if function, exists := c.rules[nodeType]; exists {
		return function, nil
	}

	return conversionInfo{}, errors.New("conversion not found")
	//panic(fmt.Sprintf("La función %s no existe\n", nodeType))
}

func (c *ASTConverter) choice(types []string, node *sitter.Node, source []byte, debug bool) ast.Node {
	rules := []NodeRule{}
	for _, typ := range types {
		rules = append(rules, NodeOfType(typ))
	}

	return c.anyOf("x", rules, node, source, debug)
}

func (c *ASTConverter) anyOf(name string, rules []NodeRule, node *sitter.Node, source []byte, debug bool) ast.Node {
	//fmt.Printf("anyOf: ")
	if debug {
		debugNode(node, source, "AnyOf["+name+"]")
	}
	if node == nil {
		panic("Nil node supplied!")
	}

	for _, rule := range rules {
		if rule.Validate(node, source, c) {
			if debug {
				log.Printf("Converted selected %s\n", rule.Type())
			}
			conversion, err := c.nodeTypeConverterMap(rule.Type())
			if err != nil {
				continue
			}

			expr := conversion.convert(node, source, debug)
			if expr != nil {
				return expr
			}
		}
	}

	return nil
}

func commaSep(convert nodeConverter, node *sitter.Node, source []byte) []ast.Node {
	var nodes []ast.Node
	for {
		//debugNode(node, source, "commaSep")
		condition := convert(node, source)

		if condition != nil {
			nodes = append(nodes, condition)
		} else {
			break
		}

		// Search next ','
		for {
			if node == nil {
				break
			} else if node.Type() != "," {
				node = node.NextSibling()
			} else if node.Type() == "," {
				node = node.NextSibling()
				break
			}
		}

		if node == nil {
			break
		}
	}

	return nodes
}
