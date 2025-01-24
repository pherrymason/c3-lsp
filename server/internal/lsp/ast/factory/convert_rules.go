package factory

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

// Here lays methods to help define expected CST nodes
type NodeRule interface {
	Validate(node *sitter.Node, source []byte, c *ASTConverter) bool
	Type() string
}

// -----------------------------------
// OfType
// -----------------------------------

type OfType struct {
	Name string
}

func (o OfType) Validate(node *sitter.Node, source []byte, c *ASTConverter) bool {
	return node.Type() == o.Name
}

func (o OfType) Type() string {
	return o.Name
}

func NodeOfType(name string) OfType {
	return OfType{Name: name}
}

// -----------------------------------
// SequenceOf
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

func (r SequenceOf) Validate(node *sitter.Node, source []byte, c *ASTConverter) bool {

	if r.SequenceType == SequenceTypeChild {
		childCount := node.ChildCount()
		if int(childCount) != len(r.ExpectedSequence) {
			return false
		}

		for i, rule := range r.ExpectedSequence {
			child := node.Child(i)
			if child == nil || !rule.Validate(child, source, c) {
				return false
			}
		}
	}

	if r.SequenceType == SequenceTypeSiblings {
		next := node
		siblings := 0
		for {
			if next == nil {
				break
			}
			next = next.NextNamedSibling()

			siblings++
		}

		if siblings < len(r.ExpectedSequence) {
			return false
		}

		next = node
		for _, rule := range r.ExpectedSequence {
			if next == nil || !rule.Validate(next, source, c) {
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
// AnonNode
// This rule is for usage on anonymous nodes that cannot be detected by its type, but
// one needs to try to convert them, and if does not return nil, it succeeds
type AnonNode struct {
	FuncName string
}

func (a AnonNode) Validate(node *sitter.Node, source []byte, c *ASTConverter) bool {
	return true
}
func (a AnonNode) Type() string {
	return a.FuncName
}
func NodeAnonymous(conversionRule string) AnonNode {
	return AnonNode{FuncName: conversionRule}
}

// -----------------------------------
// TryConversionFunc
// -----------------------------------
type TryConversionFunc struct {
	FuncName string
}

func (t TryConversionFunc) Validate(node *sitter.Node, source []byte, c *ASTConverter) bool {
	conversion, err := c.nodeTypeConverterMap(t.FuncName)
	if err != nil {
		return false
	}

	var expr ast.Expression

	func() {
		defer func() {
			if r := recover(); r != nil {
				//		log.Printf("Error validating node: %s\nNode type: %s. Content: %s. Cv: %s", r, node.Type(), node.Content(source), t.FuncName)
				expr = nil
			}
		}()
		expr = conversion.convert(node, source, false).(ast.Expression)
	}()

	return expr != nil
}
func (o TryConversionFunc) Type() string {
	return o.FuncName
}

func NodeTryConversionFunc(name string) TryConversionFunc {
	return TryConversionFunc{FuncName: name}
}
