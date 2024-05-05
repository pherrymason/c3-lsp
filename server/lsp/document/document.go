package document

import (
	"errors"

	"github.com/pherrymason/c3-lsp/lsp/cst"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/utils"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const STRUCT_SEPARATOR = '.'
const MODULE_SEPARATOR = ':'

type Document struct {
	ContextSyntaxTree       *sitter.Tree
	URI                     protocol.DocumentUri
	NeedsRefreshDiagnostics bool
	Content                 string
	lines                   []string
}

func NewDocument(docId protocol.DocumentUri, documentContent string) Document {
	return Document{
		ContextSyntaxTree:       cst.GetParsedTreeFromString(documentContent),
		URI:                     docId,
		NeedsRefreshDiagnostics: false,
		Content:                 documentContent,
	}
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

func (d *Document) HasPointInFrontSymbol(position protocol.Position) bool {
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

func (d *Document) HasModuleSeparatorInFrontSymbol(position protocol.Position) bool {
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

func (d *Document) GetSymbolPositionAtPosition(position protocol.Position) (indexables.Position, error) {
	index := position.IndexIn(d.Content)
	startIndex, _, _error := d.getSymbolRangeIndexesAtIndex(index)

	symbolStartPosition := d.indexToPosition(startIndex)

	return symbolStartPosition, _error
}

// Returns start and end index of symbol present in index.
// If no symbol is found in index, error will be returned
func (d *Document) getSymbolRangeIndexesAtIndex(index int) (int, int, error) {
	if !utils.IsAZ09_(rune(d.Content[index])) {
		return 0, 0, errors.New("No symbol at position")
	}

	symbolStart := 0
	for i := index; i >= 0; i-- {
		r := rune(d.Content[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i + 1
			break
		}
	}

	symbolEnd := len(d.Content) - 1
	for i := index; i < len(d.Content); i++ {
		r := rune(d.Content[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained last character of symbol
			symbolEnd = i - 1
			break
		}
	}

	if symbolStart > len(d.Content) {
		return 0, 0, errors.New("wordStart out of bounds")
	} else if symbolEnd > len(d.Content) {
		return 0, 0, errors.New("wordEnd out of bounds")
	} else if symbolStart > symbolEnd {
		return 0, 0, errors.New("wordStart > wordEnd!")
	}

	return symbolStart, symbolEnd, nil
}
