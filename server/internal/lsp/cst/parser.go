package cst

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	"context"
	"fmt"
	"log"
	"strings"
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
	sourceCode := []byte(normalizeLegacyConstdef(source))
	parser := NewSitterParser()
	n, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		log.Printf("failed parsing tree, falling back to empty tree: %v", err)
		n, err = parser.ParseCtx(context.Background(), nil, []byte(""))
		if err != nil {
			panic(fmt.Errorf("failed parsing fallback empty tree: %v", err))
		}
	}

	return n
}

// NormalizeSource applies any source-level transformations needed before
// passing code to the tree-sitter parser (e.g. rewriting legacy keywords).
// It is byte-offset-preserving: every replacement has the same byte length as
// the original token, so CST node positions remain valid against the original
// source text.
func NormalizeSource(source string) string {
	return normalizeLegacyConstdef(source)
}

func normalizeLegacyConstdef(source string) string {
	const keyword = "constdef"
	if !strings.Contains(source, keyword) {
		return source
	}

	out := []byte(source)
	for i := 0; i+len(keyword) <= len(out); i++ {
		if string(out[i:i+len(keyword)]) != keyword {
			continue
		}

		beforeIsIdent := i > 0 && isIdentChar(out[i-1])
		afterIdx := i + len(keyword)
		afterIsIdent := afterIdx < len(out) && isIdentChar(out[afterIdx])
		if beforeIsIdent || afterIsIdent {
			continue
		}

		copy(out[i:i+len(keyword)], []byte("cenum   "))
		i += len(keyword) - 1
	}

	return string(out)
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func RunQuery(query *sitter.Query, node *sitter.Node) *sitter.QueryCursor {
	qc := sitter.NewQueryCursor()
	qc.Exec(query, node)

	return qc
}
