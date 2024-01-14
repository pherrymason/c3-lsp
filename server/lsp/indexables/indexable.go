package indexables

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tliron/glsp/protocol_3_16"
)

type Indexable interface {
	GetName() string
	GetKind() protocol.CompletionItemKind
	GetDocumentURI() string
	GetDeclarationRange() Range
	GetDocumentRange() Range
	GetModule() string

	GetHoverInfo() string
}

type IndexableCollection []Indexable

type BaseIndexable struct {
	module          string
	documentURI     string
	identifierRange Range
	documentRange   Range
	Kind            protocol.CompletionItemKind
}

func NewBaseIndexable(docId protocol.DocumentUri, idRange Range, docRange Range, kind protocol.CompletionItemKind) BaseIndexable {
	return BaseIndexable{
		documentURI:     docId,
		identifierRange: idRange,
		documentRange:   docRange,
		Kind:            kind,
	}
}

type Range struct {
	Start Position
	End   Position
}

func NewRange(startLine uint, startChar uint, endLine uint, endChar uint) Range {
	return Range{
		Start: NewPosition(startLine, startChar),
		End:   NewPosition(endLine, endChar),
	}
}

func NewRangeFromSitterPositions(start sitter.Point, end sitter.Point) Range {
	return Range{
		Start: NewPositionFromSitterPoint(start),
		End:   NewPositionFromSitterPoint(end),
	}
}

func (r Range) HasPosition(position protocol.Position) bool {
	line := uint(position.Line)
	ch := uint(position.Character)

	if line >= r.Start.Line && ch <= r.End.Line {
		// Exactly same line
		if line == r.Start.Line && ch == r.End.Line {
			// Must be inside character ranges
			if ch >= r.Start.Character && ch <= r.End.Character {
				return true
			}
		} else {
			return true
		}
	}

	return false
}

type Position struct {
	Line      uint
	Character uint
}

func NewPosition(line uint, char uint) Position {
	return Position{Line: line, Character: char}
}

func NewPositionFromSitterPoint(position sitter.Point) Position {
	return NewPosition(uint(position.Row), uint(position.Column))
}
