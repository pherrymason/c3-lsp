package analysis

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// Parses a test body with a '|||' cursor, returning the body without
// the cursor and the position of that cursor.
//
// Useful for tests where we check what the language server responds if the
// user cursor is at a certain position.
func parseBodyWithCursor(body string) (string, lsp.Position) {
	cursorLine, cursorCol := utils.FindLineColOfSubstring(body, "|||")
	if cursorLine == 0 {
		panic("Please add the cursor position to the test body with '|||'")
	}
	if strings.Count(body, "|||") > 1 {
		panic("There are multiple '|||' cursors in the test body, please add only one")
	}

	cursorlessBody := strings.ReplaceAll(body, "|||", "")
	position := lsp.NewPosition(cursorLine, cursorCol)

	return cursorlessBody, position
}

func getTree(source string, fileName string) *ast.File {
	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source).RootNode(), source, fileName)

	return tree
}

func TestFindSymbol_ignores_language_keywords(t *testing.T) {
	t.Skip("Need analyzer to know context to complement it with a list of blacklist")
	cases := []struct {
		source string
	}{
		{"void"}, {"bool"}, {"char"}, {"double"},
		{"float"}, {"float16"}, {"int128"}, {"ichar"},
		{"int"}, {"iptr"}, {"isz"}, {"long"},
		{"short"}, {"uint128"}, {"uint"}, {"ulong"},
		{"uptr"}, {"ushort"}, {"usz"}, {"float128"},
		{"any"}, {"anyfault"}, {"typeid"}, {"assert"},
		{"asm"}, {"bitstruct"}, {"break"}, {"case"},
		{"catch"}, {"const"}, {"continue"}, {"def"},
		{"default"}, {"defer"}, {"distinct"}, {"do"},
		{"else"}, {"enum"}, {"extern"}, {"false"},
		{"fault"}, {"for"}, {"foreach"}, {"foreach_r"},
		{"fn"}, {"tlocal"}, {"if"}, {"inline"},
		{"import"}, {"macro"}, {"module"}, {"nextcase"},
		{"null"}, {"return"}, {"static"}, {"struct"},
		{"switch"}, {"true"}, {"try"}, {"union"},
		{"var"}, {"while"},
		{"$alignof"}, {"$assert"}, {"$case"}, {"$default"},
		{"$defined"}, {"$echo"}, {"$embed"}, {"$exec"},
		{"$else"}, {"$endfor"}, {"$endforeach"}, {"$endif"},
		{"$endswitch"}, {"$eval"}, {"$evaltype"}, {"$error"},
		{"$extnameof"}, {"$for"}, {"$foreach"}, {"$if"},
		{"$include"}, {"$nameof"}, {"$offsetof"}, {"$qnameof"},
		{"$sizeof"}, {"$stringify"}, {"$switch"}, {"$typefrom"},
		{"$typeof"}, {"$vacount"}, {"$vatype"}, {"$vaconst"},
		{"$varef"}, {"$vaarg"}, {"$vaexpr"}, {"$vasplat"},
	}

	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			fileName := tt.source
			tree := getTree("module foo;"+tt.source, fileName)
			symbolTable := BuildSymbolTable(tree, "")

			cursorPosition := lsp.Position{Line: 0, Column: 12}
			symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)

			assert.True(t, symbolOpt.IsNone(), fmt.Sprintf("Found symbol for keyword %s", tt.source))
		})
	}
}
