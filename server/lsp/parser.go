package lsp

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
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

func FindIdentifiers(doc *Document) []Identifier {
	// Iterate over query results
	variableIdentifiers := FindVariableDeclarations(doc)
	functionIdentifiers := FindFunctionDeclarations(doc)

	identifiers := append(variableIdentifiers, functionIdentifiers...)

	return identifiers
}

func FindVariableDeclarations(doc *Document) []Identifier {
	query := `(var_declaration (identifier) @variable_name)`
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())

	var identifiers []Identifier
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
				identifier := Identifier{
					name:                content,
					kind:                protocol.CompletionItemKindVariable,
					declarationPosition: protocol.Position{c.Node.StartPoint().Row, c.Node.StartPoint().Column},
					documentURI:         doc.URI,
				}

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}

func FindFunctionDeclarations(doc *Document) []Identifier {
	query := `(function_declaration name: (identifier) @function_name)`
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())

	var identifiers []Identifier
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
				identifier := Identifier{
					name:                content,
					kind:                protocol.CompletionItemKindFunction,
					declarationPosition: protocol.Position{c.Node.StartPoint().Row, c.Node.StartPoint().Column},
					documentURI:         doc.URI,
				}

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}
