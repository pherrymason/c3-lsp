package sourcecode

import (
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
)

// A Word is a symbol together with all its previous accessors
type Word struct {
	text      string
	textRange symbols.Range

	//
	parentAccessPath   []Word
	modulePath         []Word   // Specified module path
	resolvedModulePath []string // Real full module path. This is to support cases like
	// import std::io; io::printf();
	// Where only the last part of the module path is used in the document.
}

func NewWord(text string, positionRange symbols.Range) Word {
	return Word{
		text:      text,
		textRange: positionRange,
	}
}

func NewWordWithModulePath(text string, positionRange symbols.Range, modulePath []Word) Word {
	return Word{
		text:       text,
		textRange:  positionRange,
		modulePath: modulePath,
	}
}

func (w Word) Text() string {
	return w.text
}

func (w Word) TextRange() symbols.Range {
	return w.textRange
}

func (w Word) AccessPath() []Word {
	return w.parentAccessPath
}

func (w Word) FullTextRange() symbols.Range {
	var startLine, startCharacter uint
	if len(w.modulePath) > 0 {
		// TODO: store ranges of modulePath too so this is just a copy instead of a calculation
		startLine = w.modulePath[0].textRange.Start.Line
		startCharacter = w.modulePath[0].textRange.Start.Character
	} else if w.HasAccessPath() {
		startLine = w.parentAccessPath[0].textRange.Start.Line
		startCharacter = w.parentAccessPath[0].textRange.Start.Character
	} else {
		startLine = w.textRange.Start.Line
		startCharacter = w.textRange.Start.Character
	}

	return symbols.NewRange(startLine, startCharacter, w.textRange.End.Line, w.textRange.End.Character)
}

func (w Word) GetFullAccessPath() []Word {
	tokens := append(
		w.parentAccessPath,
		NewWord(w.text, w.textRange),
	)

	return tokens
}

func (w Word) GetFullQualifiedName() string {
	modulePaths := []string{}
	for _, m := range w.ModulePath() {
		modulePaths = append(modulePaths, m.text)
	}

	if len(modulePaths) > 0 {
		modulePaths = append(modulePaths, w.text)
		return strings.Join(modulePaths, "::")
	}

	return w.text
}

func (w Word) ModulePath() []Word {
	return w.modulePath
}

func (w Word) ResolvedModulePath() []string {
	return w.resolvedModulePath
}

func (w Word) HasModulePath() bool {
	return len(w.modulePath) > 0
}

func (w *Word) AdvanceEndCharacter() {
	w.textRange.End.Character = +1
}

func (w Word) IsSeparator() bool {
	return w.text == "." || w.text == ":"
}

func (w Word) HasAccessPath() bool {
	return len(w.parentAccessPath) > 0
}

func (w Word) PrevAccessPath() Word {
	n := len(w.parentAccessPath)
	return w.parentAccessPath[n-1]
}

type WordBuilder struct {
	word Word
}

func NewWordBuilder(text string, textRange symbols.Range) *WordBuilder {
	return &WordBuilder{
		word: Word{
			text:      text,
			textRange: textRange,
		},
	}
}
func NewWordBuilderE() *WordBuilder {
	return &WordBuilder{
		word: Word{},
	}
}

func (wb *WordBuilder) WithText(text string, textRange symbols.Range) *WordBuilder {
	wb.word.text = text
	wb.word.textRange = textRange

	return wb
}

func (wb *WordBuilder) WithModule(modulePath []Word) *WordBuilder {
	wb.word.modulePath = modulePath

	return wb
}

func (wb *WordBuilder) WithResolvedModulePath(modulePath string) *WordBuilder {
	wb.word.resolvedModulePath = strings.Split(modulePath, "::")

	return wb
}

func (wb *WordBuilder) WithAccessPath(accessPath []Word) *WordBuilder {
	wb.word.parentAccessPath = accessPath

	return wb
}

func (wb WordBuilder) Build() Word {
	return wb.word
}
