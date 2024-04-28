package document

import (
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/lsp/cst"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Document struct {
	ContextSyntaxTree       *sitter.Tree
	ModuleName              string
	URI                     protocol.DocumentUri
	NeedsRefreshDiagnostics bool
	Content                 string
	lines                   []string
	imports                 []string
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

func (d *Document) SymbolInPosition(position protocol.Position) (string, error) {
	index := position.IndexIn(d.Content)
	return d.symbolInIndex(index)
}

func (d *Document) ParentSymbolInPosition(position protocol.Position) (string, error) {
	if !d.HasPointInFrontSymbol(position) {
		return "", errors.New("No previous '.' found.")
	}

	index := position.IndexIn(d.Content)
	start, _, errRange := d.getSymbolStartEndAtIndex(index)
	if errRange != nil {
		return "", errRange
	}

	index = start - 2
	foundPreviousChar := false
	for {
		if index == 0 {
			break
		}
		r := rune(d.Content[index])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			foundPreviousChar = true
			break
		}
		index -= 1
	}

	if foundPreviousChar {
		parentSymbol, errSymbol := d.symbolInIndex(index)

		return parentSymbol, errSymbol
	}

	return "", errors.New("No previous symbol found")
}

func (d *Document) symbolInIndex(index int) (string, error) {

	start, end, err := d.getSymbolStartEndAtIndex(index)

	if err != nil {
		return "", err
	}

	return d.Content[start : end+1], nil
}

func (d *Document) HasPointInFrontSymbol(position protocol.Position) bool {
	index := position.IndexIn(d.Content)
	start, _, _ := d.getSymbolStartEndAtIndex(index)

	if start == 0 {
		return false
	}

	if rune(d.Content[start-1]) == '.' {
		return true
	}

	return false
}

func (d *Document) GetSymbolPositionAtPosition(position protocol.Position) (protocol.Position, error) {
	index := position.IndexIn(d.Content)
	startIndex, _, _error := d.getSymbolStartEndAtIndex(index)

	symbolStartPosition := d.IndexToPosition(startIndex)

	return symbolStartPosition, _error
}

func (d *Document) IndexToPosition(index int) protocol.Position {
	character := 0
	line := 0

	for i := 0; i < len(d.Content); {
		r, size := utf8.DecodeRuneInString(d.Content[i:])
		if i == index {
			// We've reached the wanted position skip and build position
			break
		}

		if r == '\n' {
			// We've found a new line
			line++
			character = 0
		} else {
			character++
		}

		// Advance the correct number of bytes
		i += size
	}

	return protocol.Position{
		Line:      protocol.UInteger(line),
		Character: protocol.UInteger(character),
	}
}

func (d *Document) getSymbolStartEndAtIndex(index int) (int, int, error) {
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
		return 0, 0, errors.New("wordStart out of bounds")
	} else if wordEnd > len(text) {
		return 0, 0, errors.New("wordEnd out of bounds")
	} else if wordStart > wordEnd {
		return 0, 0, errors.New("wordStart > wordEnd!")
	}

	return wordStart, wordEnd, nil
}

func (d *Document) AddImport(module string) {
	d.imports = append(d.imports, module)
}

func (d Document) GetImports() []string {
	return d.imports
}
