package document

import (
	"github.com/pherrymason/c3-lsp/lsp/cst"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const STRUCT_SEPARATOR = '.'
const MODULE_SEPARATOR = ':'

type Document struct {
	ContextSyntaxTree       *sitter.Tree
	URI                     string
	NeedsRefreshDiagnostics bool
	Content                 string
}

func NewDocument(docId string, documentContent string) Document {
	return Document{
		ContextSyntaxTree:       cst.GetParsedTreeFromString(documentContent),
		URI:                     docId,
		NeedsRefreshDiagnostics: false,
		Content:                 documentContent,
	}
}

func NewDocumentFromString(docId string, documentContent string) Document {
	return NewDocument(docId, documentContent)
}

// ApplyChanges updates the content of the Document from LSP textDocument/didChange events.
func (d *Document) ApplyChanges(changes []interface{}) {
	for _, change := range changes {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEvent:
			startIndex, endIndex := c.Range.IndexesIn(d.Content)
			d.Content = d.Content[:startIndex] + c.Text + d.Content[endIndex:]
		case protocol.TextDocumentContentChangeEventWhole:
			d.Content = c.Text
		}
	}

	//d.lines = nil
	d.updateParsedTree()
	d.ContextSyntaxTree = cst.GetParsedTreeFromString(d.Content)
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
	index := position.IndexIn(d.Content)
	start, _, _ := d.getSymbolRangeIndexesAtIndex(index)

	if start == 0 {
		return false
	}

	if rune(d.Content[start-1]) == STRUCT_SEPARATOR {
		return true
	}

	return false
}

func (d *Document) HasModuleSeparatorInFrontSymbol(position symbols.Position) bool {
	index := position.IndexIn(d.Content)
	start, _, _ := d.getSymbolRangeIndexesAtIndex(index)

	if start == 0 {
		return false
	}

	if rune(d.Content[start-1]) == MODULE_SEPARATOR && rune(d.Content[start-2]) == MODULE_SEPARATOR {
		return true
	}

	return false
}

func (d *Document) GetSymbolPositionAtPosition(position symbols.Position) (symbols.Position, error) {
	index := position.IndexIn(d.Content)
	startIndex, _, _error := d.getSymbolRangeIndexesAtIndex(index)

	symbolStartPosition := d.indexToPosition(startIndex)

	return symbolStartPosition, _error
}
