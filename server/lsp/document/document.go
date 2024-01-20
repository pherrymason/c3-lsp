package document

import (
	"errors"
	"github.com/pherrymason/c3-lsp/lsp/cst"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"unicode"
)

type Document struct {
	ContextSyntaxTree       *sitter.Tree
	ModuleName              string
	URI                     protocol.DocumentUri
	NeedsRefreshDiagnostics bool
	Content                 string
	lines                   []string
}

func NewDocument(docId protocol.DocumentUri, moduleName string, documentContent string) Document {
	return Document{
		ContextSyntaxTree:       cst.GetParsedTreeFromString(documentContent),
		ModuleName:              moduleName,
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

func (d *Document) WordInPosition(position protocol.Position) (string, error) {
	index := position.IndexIn(d.Content)
	return d.WordInIndex(index)
}

func (d *Document) WordInIndex(index int) (string, error) {
	text := d.Content

	wordStart := 0
	for i := index; i >= 0; i-- {
		if !(unicode.IsLetter(rune(text[i])) || unicode.IsDigit(rune(text[i])) || text[i] == '_') {
			wordStart = i + 1
			break
		}
	}

	wordEnd := len(text) - 1
	for i := index; i < len(text); i++ {
		if !(unicode.IsLetter(rune(text[i])) || unicode.IsDigit(rune(text[i])) || text[i] == '_') {
			wordEnd = i - 1
			break
		}
	}

	if wordStart > len(text) {
		return "", errors.New("wordStart out of bounds")
	} else if wordEnd > len(text) {
		return "", errors.New("wordEnd out of bounds")
	} else if wordStart > wordEnd {
		return "", errors.New("wordStart > wordEnd!")
	}

	word := text[wordStart : wordEnd+1]
	return word, nil
}
