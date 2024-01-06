package c3

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	sitter "github.com/smacker/go-tree-sitter"
	"unsafe"
)

func getParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage())

	return parser
}

func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_c3())
	return sitter.NewLanguage(ptr)
}

func FindIdentifiers(source string, debug bool) []string {
	parser := getParser()

	// Query with predicates
	/*query := `[
			(var_declaration (identifier) @variable_name)
	        (_var_declaration (identifier) @variable_name2)
			(function_declaration (name: (identifier) @function_name))
		] @bla`*/

	sourceCode := []byte(source)
	n := parser.Parse(nil, sourceCode)
	/*q, err := sitter.NewQuery([]byte(query), GetLanguage())
	if err != nil {
		panic(err)
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, n.RootNode())
	if debug {
		fmt.Print(n.RootNode())
	}
	*/
	// Iterate over query results
	//var identifiers []string
	variable_identifiers := FindVariableDeclarations(sourceCode, n)
	function_identifiers := FindFunctionDeclarations(sourceCode, n)

	identifiers := append(variable_identifiers, function_identifiers...)
	/*
		found := make(map[string]bool)
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
					identifiers = append(identifiers, content)
				}
			}
		}
	*/
	return identifiers
}

func FindVariableDeclarations(sourceCode []byte, n *sitter.Tree) []string {
	query := `(var_declaration (identifier) @variable_name)`
	q, err := sitter.NewQuery([]byte(query), GetLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, n.RootNode())

	var identifiers []string
	found := make(map[string]bool)
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
				identifiers = append(identifiers, content)
			}
		}
	}

	return identifiers
}

func FindFunctionDeclarations(sourceCode []byte, n *sitter.Tree) []string {
	query := `(_function_signature (name: (identifier) @function_name))`
	q, err := sitter.NewQuery([]byte(query), GetLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, n.RootNode())

	var identifiers []string
	found := make(map[string]bool)
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
				identifiers = append(identifiers, content)
			}
		}
	}

	return identifiers
}
