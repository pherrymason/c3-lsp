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

type SequenceOf struct {
	ExpectedSequence []NodeRule
	AsType           string
}

func (r SequenceOf) Validate(node *sitter.Node, source []byte) bool {
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
	return true
}
func (s SequenceOf) Type() string {
	return s.AsType
}
func NodeSequenceOf(rules []NodeRule, asType string) SequenceOf {
	return SequenceOf{ExpectedSequence: rules, AsType: asType}
}

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

func nodeTypeConverterMap(nodeType string) NodeConverter {
	funcMap := map[string]NodeConverter{
		"assignment_expr": convert_assignment_expr,
		"binary_expr":     convert_binary_expr,
		"bytes_literal":   convert_literal,
		//"call_expr":             convert_dummy,
		//"cast_expr":             convert_dummy,
		"char_literal":      convert_literal,
		"elvis_orelse_expr": convert_elvis_orelse_expr,
		"ident":             convert_ident,
		"integer_literal":   convert_literal,
		//"initializer_list":      convert_dummy,
		"lambda_expr":        convert_lambda_expr,
		"raw_string_literal": convert_literal,
		"real_literal":       convert_literal,
		//"rethrow_expr":          convert_dummy,
		//"suffix_expr":           convert_dummy,
		//"subscript_expr":        convert_dummy,
		"string_literal": convert_literal,
		"ternary_expr":   convert_ternary_expr,
		//"trailing_generic_expr": convert_dummy,
		//"unary_expr":            convert_dummy,
		//"update_expr":           convert_dummy,
		"_expr":      convert_expression,
		"_base_expr": convert_base_expression,
	}

	if function, exists := funcMap[nodeType]; exists {
		return function
	}

	return nil
	//panic(fmt.Sprintf("La función %s no existe\n", nodeType))
}

func anyOf(rules []NodeRule, node *sitter.Node, source []byte) Expression {
	var converter NodeConverter
	//debugNode(node, source)

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

	panic(fmt.Sprintf("Could not find method to convert %s node type.\n", node.Type()))
}
