package ast

import (
	"errors"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

type NodeConverter func(node *sitter.Node, source []byte) Node
type ConversionInfo struct {
	method  NodeConverter
	goChild bool
}

func (c ConversionInfo) convert(node *sitter.Node, source []byte) Node {
	n := node
	if c.goChild {
		n = node.Child(0)
	}

	return c.method(n, source)
}

type StatementConverter func(node *sitter.Node, source []byte) Statement
type ExpressionConverter func(node *sitter.Node, source []byte) Expression
type DeclarationConverter func(node *sitter.Node, source []byte) Declaration

func cv_stmt_fn(fn StatementConverter) NodeConverter {
	return func(node *sitter.Node, source []byte) Node {
		return fn(node, source).(Node)
	}
}

func cv_expr_fn(fn ExpressionConverter) NodeConverter {
	return func(node *sitter.Node, source []byte) Node {
		return fn(node, source).(Node)
	}
}

func cv_decl_fn(fn DeclarationConverter) NodeConverter {
	return func(node *sitter.Node, source []byte) Node {
		return fn(node, source).(Node)
	}
}

func nodeTypeConverterMap(nodeType string) (ConversionInfo, error) {
	funcMap := map[string]ConversionInfo{
		"assignment_expr":        {method: cv_stmt_fn(convert_assignment_expr)},
		"assert_stmt":            {method: cv_stmt_fn(convert_assert_stmt)},
		"at_ident":               {method: cv_expr_fn(convert_ident)},
		"binary_expr":            {method: cv_expr_fn(convert_binary_expr)},
		"break_stmt":             {method: cv_stmt_fn(convert_break_stmt)},
		"bytes_expr":             {method: cv_expr_fn(convert_bytes_expr)},
		"builtin":                {method: cv_expr_fn(convert_as_literal)},
		"call_expr":              {method: cv_expr_fn(convert_call_expr)},
		"continue_stmt":          {method: cv_stmt_fn(convert_continue_stmt)},
		"cast_expr":              {method: cv_expr_fn(convert_cast_expr)},
		"const_ident":            {method: cv_expr_fn(convert_ident)},
		"compound_stmt":          {method: cv_stmt_fn(convert_compound_stmt)},
		"ct_ident":               {method: cv_expr_fn(convert_ident)},
		"declaration_stmt":       {method: cv_decl_fn(convert_declaration_stmt)},
		"defer_stmt":             {method: cv_stmt_fn(convert_defer_stmt)},
		"do_stmt":                {method: cv_stmt_fn(convert_do_stmt)},
		"split_declaration_stmt": {method: cv_decl_fn(convert_split_declaration_stmt)},
		"elvis_orelse_expr":      {method: cv_expr_fn(convert_elvis_orelse_expr)},
		"expr_stmt":              {method: cv_stmt_fn(convert_expr_stmt), goChild: true},
		"for_stmt":               {method: cv_stmt_fn(convert_for_stmt)},
		"foreach_stmt":           {method: cv_stmt_fn(convert_foreach_stmt)},
		"hash_ident":             {method: cv_expr_fn(convert_ident)},
		"ident":                  {method: cv_expr_fn(convert_ident)},
		"if_stmt":                {method: cv_stmt_fn(convert_if_stmt)},
		"initializer_list":       {method: cv_expr_fn(convert_initializer_list)},

		"lambda_declaration":    {method: cv_expr_fn(convert_lambda_declaration)},
		"lambda_expr":           {method: cv_expr_fn(convert_lambda_expr)},
		"local_decl_after_type": {method: cv_decl_fn(convert_local_declaration_after_type)},
		"module_ident_expr":     {method: cv_expr_fn(convert_module_ident_expr)},
		"nextcase_stmt":         {method: cv_stmt_fn(convert_nextcase_stmt)},
		"optional_expr":         {method: cv_expr_fn(convert_optional_expr)},
		"rethrow_expr":          {method: cv_expr_fn(convert_rethrow_expr)},
		"return_stmt":           {method: cv_stmt_fn(convert_return_stmt)},
		//"suffix_expr":           convert_dummy,
		"subscript_expr":        {method: cv_expr_fn(convert_subscript_expr)},
		"switch_stmt":           {method: cv_stmt_fn(convert_switch_stmt)},
		"ternary_expr":          {method: cv_expr_fn(convert_ternary_expr)},
		"trailing_generic_expr": {method: cv_expr_fn(convert_trailing_generic_expr)},
		"type": {method: func(node *sitter.Node, source []byte) Node {
			n := convert_type(node, source)
			return n
		}},
		"unary_expr":  {method: cv_expr_fn(convert_unary_expr)},
		"update_expr": {method: cv_expr_fn(convert_update_expr)},
		"var_stmt":    {method: cv_decl_fn(convert_var_decl), goChild: true},
		"while_stmt":  {method: cv_stmt_fn(convert_while_stmt)},
		"field_expr":  {method: cv_expr_fn(convert_field_expr)},

		// Builtins ----------------
		"$vacount": {method: cv_expr_fn(convert_as_literal)},
		"$feature": {method: cv_expr_fn(convert_feature)},

		"$alignof":   {method: cv_expr_fn(convert_compile_time_call)},
		"$extnameof": {method: cv_expr_fn(convert_compile_time_call)},
		"$nameof":    {method: cv_expr_fn(convert_compile_time_call)},
		"$offsetof":  {method: cv_expr_fn(convert_compile_time_call)},
		"$qnameof":   {method: cv_expr_fn(convert_compile_time_call)},

		"$vaconst": {method: cv_expr_fn(convert_compile_time_arg)},
		"$vaarg":   {method: cv_expr_fn(convert_compile_time_arg)},
		"$varef":   {method: cv_expr_fn(convert_compile_time_arg)},
		"$vaexpr":  {method: cv_expr_fn(convert_compile_time_arg)},

		"$eval":      {method: cv_expr_fn(convert_compile_time_analyse)},
		"$is_const":  {method: cv_expr_fn(convert_compile_time_analyse)},
		"$sizeof":    {method: cv_expr_fn(convert_compile_time_analyse)},
		"$stringify": {method: cv_expr_fn(convert_compile_time_analyse)},

		"$and":     {method: cv_expr_fn(convert_compile_time_call_unk)},
		"$append":  {method: cv_expr_fn(convert_compile_time_call_unk)},
		"$concat":  {method: cv_expr_fn(convert_compile_time_call_unk)},
		"$defined": {method: cv_expr_fn(convert_compile_time_call_unk)},
		"$embed":   {method: cv_expr_fn(convert_compile_time_call_unk)},
		"$or":      {method: cv_expr_fn(convert_compile_time_call_unk)},

		"_expr":      {method: cv_expr_fn(convert_expression)},
		"_base_expr": {method: cv_expr_fn(convert_base_expression)},
		"_statement": {method: cv_stmt_fn(convert_statement)},

		// Literals
		"string_literal":     {method: cv_expr_fn(convert_literal)},
		"char_literal":       {method: cv_expr_fn(convert_literal)},
		"raw_string_literal": {method: cv_expr_fn(convert_literal)},
		"integer_literal":    {method: cv_expr_fn(convert_literal)},
		"real_literal":       {method: cv_expr_fn(convert_literal)},
		"bytes_literal":      {method: cv_expr_fn(convert_literal)},
		"true":               {method: cv_expr_fn(convert_literal)},
		"false":              {method: cv_expr_fn(convert_literal)},
		"null":               {method: cv_expr_fn(convert_literal)},

		// Custom ones ----------------
		"..type_with_initializer_list..":   {method: cv_expr_fn(convert_type_with_initializer_list)},
		"..lambda_declaration_with_body..": {method: cv_expr_fn(convert_lambda_declaration_with_body)},
	}

	if function, exists := funcMap[nodeType]; exists {
		return function, nil
	}

	return ConversionInfo{}, errors.New("conversion not found")
	//panic(fmt.Sprintf("La función %s no existe\n", nodeType))
}

func anyOf(name string, rules []NodeRule, node *sitter.Node, source []byte, debug bool) Node {
	//fmt.Printf("anyOf: ")
	if debug {
		debugNode(node, source, "AnyOf["+name+"]")
	}
	if node == nil {
		panic("Nil node supplied!")
	}

	for _, rule := range rules {
		if rule.Validate(node, source) {
			if debug {
				fmt.Printf("Converted selected %s\n", rule.Type())
			}
			conversion, err := nodeTypeConverterMap(rule.Type())
			if err != nil {
				continue
			}

			expr := conversion.convert(node, source)
			if expr != nil {
				return expr
			}
		}
	}

	return nil
}

func commaSep(convert NodeConverter, node *sitter.Node, source []byte) []Node {
	var nodes []Node
	for {
		debugNode(node, source, "commaSep")
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
