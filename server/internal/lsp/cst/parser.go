package cst

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	"context"
	"fmt"
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

var Language *sitter.Language

func init() {
	languagePtr := unsafe.Pointer(C.tree_sitter_c3())
	if languagePtr == nil {
		panic("Couldnt get c3 tree sitter language")
	}
	Language = sitter.NewLanguage(languagePtr)
}

func NewSitterParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(Language)

	return parser
}

func GetParsedTreeFromString(source string) *sitter.Tree {
	sourceCode := []byte(source)
	parser := NewSitterParser()
	n, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		panic(fmt.Errorf("failed parsing tree: %v", err))
	}

	return n
}

func RunQuery(query *sitter.Query, node *sitter.Node) *sitter.QueryCursor {
	qc := sitter.NewQueryCursor()
	qc.Exec(query, node)

	return qc
}
