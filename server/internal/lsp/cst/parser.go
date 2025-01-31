package cst

// #cgo CFLAGS: -std=c11 -fPIC
// #include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

func NewSitterParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage())

	return parser
}

func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_c3())
	return sitter.NewLanguage(ptr)
}

func GetParsedTreeFromString(source string) *sitter.Tree {
	sourceCode := []byte(source)
	parser := NewSitterParser()
	n := parser.Parse(nil, sourceCode)

	return n
}

func RunQuery(query string, node *sitter.Node) *sitter.QueryCursor {
	q, err := sitter.NewQuery([]byte(query), GetLanguage())
	if err != nil {
		panic(err)
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, node)

	return qc
}
