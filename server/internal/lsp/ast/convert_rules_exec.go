package ast

import (
	"errors"

	sitter "github.com/smacker/go-tree-sitter"
)

type NodeConverter func(node *sitter.Node, source []byte) Expression
type ConversionInfo struct {
	method  NodeConverter
	goChild bool
}

func (c ConversionInfo) convert(node *sitter.Node, source []byte) Expression {
	n := node
	if c.goChild {
		n = node.Child(0)
	}

	return c.method(n, source)
}

func nodeTypeConverterMap(nodeType string) (ConversionInfo, error) {
	funcMap := map[string]ConversionInfo{
		"assignment_expr":        {method: convert_assignment_expr},
		"at_ident":               {method: convert_ident},
		"binary_expr":            {method: convert_binary_expr},
		"break_stmt":             {method: convert_break_stmt},
		"bytes_expr":             {method: convert_bytes_expr},
		"builtin":                {method: convert_as_literal},
		"call_expr":              {method: convert_call_expr},
		"continue_stmt":          {method: convert_continue_stmt},
		"cast_expr":              {method: convert_cast_expr},
		"const_ident":            {method: convert_ident},
		"compound_stmt":          {method: convert_compound_stmt},
		"ct_ident":               {method: convert_ident},
		"declaration_stmt":       {method: convert_declaration_stmt},
		"defer_stmt":             {method: convert_defer_stmt},
		"do_stmt":                {method: convert_do_stmt},
		"split_declaration_stmt": {method: convert_split_declaration_stmt},
		"elvis_orelse_expr":      {method: convert_elvis_orelse_expr},
		"expr_stmt":              {method: convert_expression, goChild: true},
		"for_stmt":               {method: convert_for_stmt},
		"foreach_stmt":           {method: convert_foreach_stmt},
		"hash_ident":             {method: convert_ident},
		"ident":                  {method: convert_ident},
		"if_stmt":                {method: convert_if_stmt},
		"initializer_list":       {method: convert_initializer_list},

		"lambda_declaration":    {method: convert_lambda_declaration},
		"lambda_expr":           {method: convert_lambda_expr},
		"local_decl_after_type": {method: convert_local_declaration_after_type},
		"module_ident_expr":     {method: convert_module_ident_expr},
		"nextcase_stmt":         {method: convert_nextcase_stmt},
		"optional_expr":         {method: convert_optional_expr},
		"rethrow_expr":          {method: convert_rethrow_expr},
		"return_stmt":           {method: convert_return_stmt},
		//"suffix_expr":           convert_dummy,
		"subscript_expr":        {method: convert_subscript_expr},
		"switch_stmt":           {method: convert_switch_stmt},
		"ternary_expr":          {method: convert_ternary_expr},
		"trailing_generic_expr": {method: convert_trailing_generic_expr},
		"type":                  {method: convert_type},
		"unary_expr":            {method: convert_unary_expr},
		"update_expr":           {method: convert_update_expr},
		"var_stmt":              {method: convert_var_decl, goChild: true},
		"while_stmt":            {method: convert_while_stmt},

		// Builtins ----------------
		"$vacount": {method: convert_as_literal},
		"$feature": {method: convert_feature},

		"$alignof":   {method: convert_compile_time_call},
		"$extnameof": {method: convert_compile_time_call},
		"$nameof":    {method: convert_compile_time_call},
		"$offsetof":  {method: convert_compile_time_call},
		"$qnameof":   {method: convert_compile_time_call},

		"$vaconst": {method: convert_compile_time_arg},
		"$vaarg":   {method: convert_compile_time_arg},
		"$varef":   {method: convert_compile_time_arg},
		"$vaexpr":  {method: convert_compile_time_arg},

		"$eval":      {method: convert_compile_time_analyse},
		"$is_const":  {method: convert_compile_time_analyse},
		"$sizeof":    {method: convert_compile_time_analyse},
		"$stringify": {method: convert_compile_time_analyse},

		"$and":     {method: convert_compile_time_call_unk},
		"$append":  {method: convert_compile_time_call_unk},
		"$concat":  {method: convert_compile_time_call_unk},
		"$defined": {method: convert_compile_time_call_unk},
		"$embed":   {method: convert_compile_time_call_unk},
		"$or":      {method: convert_compile_time_call_unk},

		"_expr":      {method: convert_expression},
		"_base_expr": {method: convert_base_expression},
		"_statement": {method: convert_statement},

		// Literals
		"string_literal":     {method: convert_literal},
		"char_literal":       {method: convert_literal},
		"raw_string_literal": {method: convert_literal},
		"integer_literal":    {method: convert_literal},
		"real_literal":       {method: convert_literal},
		"bytes_literal":      {method: convert_literal},
		"true":               {method: convert_literal},
		"false":              {method: convert_literal},
		"null":               {method: convert_literal},

		// Custom ones ----------------
		"..type_with_initializer_list..":   {method: convert_type_with_initializer_list},
		"..lambda_declaration_with_body..": {method: convert_lambda_declaration_with_body},
	}

	if function, exists := funcMap[nodeType]; exists {
		return function, nil
	}

	return ConversionInfo{}, errors.New("conversion not found")
	//panic(fmt.Sprintf("La función %s no existe\n", nodeType))
}

func anyOf(name string, rules []NodeRule, node *sitter.Node, source []byte, debug bool) Expression {
	//fmt.Printf("anyOf: ")
	if debug {
		debugNode(node, source, "AnyOf["+name+"]")
	}
	if node == nil {
		panic("Nil node supplied!")
	}

	for _, rule := range rules {
		if rule.Validate(node, source) {
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

func commaSep(convert NodeConverter, node *sitter.Node, source []byte) []Expression {
	expressions := []Expression{}
	for {
		debugNode(node, source, "commaSep")
		condition := convert(node, source)

		if condition != nil {
			expressions = append(expressions, condition)
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

	return expressions
}
