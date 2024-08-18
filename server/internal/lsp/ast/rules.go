package ast

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// Here lays methods to help define expected CST nodes
type NodeRule interface {
	Validate(node *sitter.Node, source []byte) bool
	Type() string
}

// -----------------------------------
const (
	SequenceTypeChild = iota
	SequenceTypeSiblings
)

type SequenceOf struct {
	SequenceType     int
	ExpectedSequence []NodeRule
	AsType           string
}

func (r SequenceOf) Validate(node *sitter.Node, source []byte) bool {

	if r.SequenceType == SequenceTypeChild {
		childCount := node.ChildCount()
		if int(childCount) != len(r.ExpectedSequence) {
			return false
		}

		for i, rule := range r.ExpectedSequence {
			child := node.Child(i)
			if child == nil || !rule.Validate(child, source) {
				return false
			}
		}
	}

	if r.SequenceType == SequenceTypeSiblings {
		next := node
		siblings := 0
		for {
			debugNode(next, source)
			if next == nil {
				break
			}
			next = next.NextNamedSibling()

			siblings++
		}

		if siblings != len(r.ExpectedSequence) {
			return false
		}

		next = node
		for _, rule := range r.ExpectedSequence {
			if next == nil || !rule.Validate(next, source) {
				return false
			}
			next = next.NextSibling()
		}
	}

	return true
}
func (s SequenceOf) Type() string {
	return s.AsType
}
func NodeChildWithSequenceOf(rules []NodeRule, asType string) SequenceOf {
	return SequenceOf{SequenceType: SequenceTypeChild, ExpectedSequence: rules, AsType: asType}
}
func NodeSiblingsWithSequenceOf(rules []NodeRule, asType string) SequenceOf {
	return SequenceOf{SequenceType: SequenceTypeSiblings, ExpectedSequence: rules, AsType: asType}
}

// -----------------------------------

type OfType struct {
	Name string
}

func (o OfType) Validate(node *sitter.Node, source []byte) bool {
	return node.Type() == o.Name
}

func (o OfType) Type() string {
	return o.Name
}

func NodeOfType(name string) OfType {
	return OfType{Name: name}
}

type TryConversionFunc struct {
	FuncName string
}

func (t TryConversionFunc) Validate(node *sitter.Node, source []byte) bool {
	debugNode(node, source)
	converter := nodeTypeConverterMap(t.FuncName)
	if converter == nil {
		return false
	}

	expr := converter(node, source)

	return expr != nil
}
func (o TryConversionFunc) Type() string {
	return o.FuncName
}

func NodeTryConversionFunc(name string) TryConversionFunc {
	return TryConversionFunc{FuncName: name}
}

type NodeConverter func(node *sitter.Node, source []byte) Expression

func nodeTypeConverterMap(nodeType string) NodeConverter {
	funcMap := map[string]NodeConverter{
		"assignment_expr":                  convert_assignment_expr,
		"at_ident":                         convert_ident,
		"binary_expr":                      convert_binary_expr,
		"bytes_expr":                       convert_bytes_expr,
		"builtin":                          convert_as_literal,
		"call_expr":                        convert_call_expr,
		"cast_expr":                        convert_cast_expr,
		"const_ident":                      convert_ident,
		"ct_ident":                         convert_ident,
		"declaration_stmt":                 convert_declaration_stmt,
		"elvis_orelse_expr":                convert_elvis_orelse_expr,
		"hash_ident":                       convert_ident,
		"ident":                            convert_ident,
		"initializer_list":                 convert_initializer_list,
		"..type_with_initializer_list..":   convert_type_with_initializer_list,
		"..lambda_declaration_with_body..": convert_lambda_declaration_with_body,

		"lambda_declaration": convert_lambda_declaration,
		"lambda_expr":        convert_lambda_expr,
		"module_ident_expr":  convert_module_ident_expr,
		"optional_expr":      convert_optional_expr,
		"rethrow_expr":       convert_rethrow_expr,
		"return_stmt":        convert_return_stmt,
		//"suffix_expr":           convert_dummy,
		"subscript_expr":        convert_subscript_expr,
		"ternary_expr":          convert_ternary_expr,
		"trailing_generic_expr": convert_trailing_generic_expr,
		"type":                  convert_type,
		"unary_expr":            convert_unary_expr,
		"update_expr":           convert_update_expr,

		"$vacount": convert_as_literal,
		"$feature": convert_feature,

		"$alignof":   convert_compile_time_call,
		"$extnameof": convert_compile_time_call,
		"$nameof":    convert_compile_time_call,
		"$offsetof":  convert_compile_time_call,
		"$qnameof":   convert_compile_time_call,

		"$vaconst": convert_compile_time_arg,
		"$vaarg":   convert_compile_time_arg,
		"$varef":   convert_compile_time_arg,
		"$vaexpr":  convert_compile_time_arg,

		"$eval":      convert_compile_time_analyse,
		"$is_const":  convert_compile_time_analyse,
		"$sizeof":    convert_compile_time_analyse,
		"$stringify": convert_compile_time_analyse,

		"$and":     convert_compile_time_call_unk,
		"$append":  convert_compile_time_call_unk,
		"$concat":  convert_compile_time_call_unk,
		"$defined": convert_compile_time_call_unk,
		"$embed":   convert_compile_time_call_unk,
		"$or":      convert_compile_time_call_unk,

		"_expr":      convert_expression,
		"_base_expr": convert_base_expression,
		"_statement": convert_statement,

		// Literals
		"string_literal":     convert_literal,
		"char_literal":       convert_literal,
		"raw_string_literal": convert_literal,
		"integer_literal":    convert_literal,
		"real_literal":       convert_literal,
		"bytes_literal":      convert_literal,
		"true":               convert_literal,
		"false":              convert_literal,
		"null":               convert_literal,
	}

	if function, exists := funcMap[nodeType]; exists {
		return function
	}

	return nil
	//panic(fmt.Sprintf("La función %s no existe\n", nodeType))
}

func anyOf(rules []NodeRule, node *sitter.Node, source []byte) Expression {
	var converter NodeConverter
	fmt.Printf("anyOf: ")
	debugNode(node, source)
	if node == nil {
		panic("Nil node supplied!")
	}

	for _, rule := range rules {
		if rule.Validate(node, source) {
			converter = nodeTypeConverterMap(rule.Type())
			if converter != nil {
				expr := converter(node, source)
				if expr != nil {
					return expr
				}
			} else {
				// Continue
			}
		}
	}

	panic(fmt.Sprintf("Unexpected node found: \"%s\" node type.\n", node.Type()))
}
