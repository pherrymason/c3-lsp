package lsp

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"unsafe"
)

func getParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(getLanguage())

	return parser
}

func getLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_c3())
	return sitter.NewLanguage(ptr)
}

func GetParsedTree(source []byte) *sitter.Tree {
	parser := getParser()
	n := parser.Parse(nil, source)

	return n
}

func GetParsedTreeFromString(source string) *sitter.Tree {
	sourceCode := []byte(source)
	parser := getParser()
	n := parser.Parse(nil, sourceCode)

	return n
}

func FindIdentifiers(doc *Document) []Indexable {
	variableIdentifiers := FindVariableDeclarations(doc)
	functionIdentifiers := FindFunctionDeclarations(doc)

	var elements []Indexable
	elements = append(elements, variableIdentifiers...)
	elements = append(elements, functionIdentifiers...)

	return elements
}

func FindVariableDeclarations(doc *Document) []Indexable {
	query := `(var_declaration
		name: (identifier) @variable_name
	)`
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())
	doc.parsedTree.RootNode().FieldNameForChild(1)

	var identifiers []Indexable
	found := make(map[string]bool)
	sourceCode := []byte(doc.Content)
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		// Apply predicates filtering
		m = qc.FilterPredicates(m, sourceCode)
		for _, c := range m.Captures {
			content := c.Node.Content(sourceCode)

			if _, exists := found[content]; !exists {
				// Find type
				typeNode := c.Node.PrevSibling()
				typeNodeContent := typeNode.Content(sourceCode)

				found[content] = true
				identifier := indexables.NewVariableIndexable(content, typeNodeContent, doc.URI, protocol.Position{Line: c.Node.StartPoint().Row, Character: c.Node.StartPoint().Column}, protocol.CompletionItemKindVariable)

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}

func FindFunctionDeclarations(doc *Document) []Indexable {
	query := `(function_declaration name: (identifier) @function_name)`
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())

	var identifiers []Indexable
	found := make(map[string]bool)
	sourceCode := []byte(doc.Content)
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		// Apply predicates filtering
		m = qc.FilterPredicates(m, sourceCode)
		for _, c := range m.Captures {
			content := c.Node.Content(sourceCode)
			c.Node.Parent().Type()
			if _, exists := found[content]; !exists {
				found[content] = true
				identifier := indexables.NewFunctionIndexable(
					content,
					doc.URI,
					protocol.Position{c.Node.StartPoint().Row, c.Node.StartPoint().Column},
					protocol.CompletionItemKindFunction,
				)

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}
