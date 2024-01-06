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

func FindIdentifiers(source string) []string {
	parser := getParser()

	// Query with predicates
	query := `(
		(identifier) @constant
		
	)`

	sourceCode := []byte(source)
	n := parser.Parse(nil, sourceCode)
	q, _ := sitter.NewQuery([]byte(query), GetLanguage())
	qc := sitter.NewQueryCursor()
	qc.Exec(q, n.RootNode())

	//parsed := fmt.Sprint(n.RootNode())
	//fmt.Print(parsed)

	// Iterate over query results
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
			//fmt.Println(c.Node.Content(sourceCode))
			content := c.Node.Content(sourceCode)
			if _, exists := found[content]; !exists {
				found[content] = true
				identifiers = append(identifiers, content)
			}
		}
	}

	// Remove duplicates

	return identifiers
}
