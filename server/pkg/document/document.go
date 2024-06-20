package document

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const STRUCT_SEPARATOR = '.'
const MODULE_SEPARATOR = ':'

type Document struct {
	URI string
	//NeedsRefreshDiagnostics bool
	ContextSyntaxTree *sitter.Tree
	SourceCode        code.SourceCode
}

func NewDocument(docId string, sourceCode string) Document {
	return Document{
		URI:               docId,
		ContextSyntaxTree: cst.GetParsedTreeFromString(sourceCode),
		SourceCode:        code.NewSourceCode(sourceCode),
	}
}

func NewDocumentFromString(docId string, sourceCode string) Document {
	return NewDocument(docId, sourceCode)
}

// ApplyChanges updates the content of the Document from LSP textDocument/didChange events.
func (d *Document) ApplyChanges(changes []interface{}) {
	for _, change := range changes {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEvent:
			startIndex, endIndex := c.Range.IndexesIn(d.SourceCode.Text)
			d.SourceCode.Text = d.SourceCode.Text[:startIndex] + c.Text + d.SourceCode.Text[endIndex:]
		case protocol.TextDocumentContentChangeEventWhole:
			d.SourceCode.Text = c.Text
		}
	}

	//d.lines = nil
	d.updateParsedTree()
	d.ContextSyntaxTree = cst.GetParsedTreeFromString(d.SourceCode.Text)
}
func (d *Document) updateParsedTree() {
	// TODO
	// should the Document store the parsed CTS?
	// would allow parsing incrementally and be faster
	// // change 1 -> true
	//		newText := []byte("let a = true")
	//		tree.Edit(sitter.EditInput{
	//    		StartIndex:  8,
	//    		OldEndIndex: 9,
	//    		NewEndIndex: 12,
	//    		StartPoint: sitter.Point{
	//    		    Row:    0,
	//    		    Column: 8,
	//    		},
	//    		OldEndPoint: sitter.Point{
	//    		    Row:    0,
	//    		    Column: 9,
	//    		},
	//    		NewEndPoint: sitter.Point{
	//    		    Row:    0,
	//    		    Column: 12,
	//    		},
	//		})
}

func (d *Document) HasPointInFrontSymbol(position symbols.Position) bool {
	index := position.IndexIn(d.SourceCode.Text)
	start, _, _ := d.getWordIndexLimits(index)

	if start == 0 {
		return false
	}

	if rune(d.SourceCode.Text[start-1]) == STRUCT_SEPARATOR {
		return true
	}

	return false
}

func (d *Document) HasModuleSeparatorInFrontSymbol(position symbols.Position) bool {
	index := position.IndexIn(d.SourceCode.Text)
	start, _, _ := d.getWordIndexLimits(index)

	if start == 0 {
		return false
	}

	if rune(d.SourceCode.Text[start-1]) == MODULE_SEPARATOR && rune(d.SourceCode.Text[start-2]) == MODULE_SEPARATOR {
		return true
	}

	return false
}

func (d *Document) GetSymbolPositionAtPosition(position symbols.Position) (symbols.Position, error) {
	index := position.IndexIn(d.SourceCode.Text)
	startIndex, _, _error := d.getWordIndexLimits(index)

	symbolStartPosition := d.indexToPosition(startIndex)

	return symbolStartPosition, _error
}
