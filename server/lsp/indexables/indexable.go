package indexables

import (
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Indexable interface {
	GetName() string
	GetKind() protocol.CompletionItemKind
	GetDocumentURI() string
	GetIdRange() Range
	GetDocumentRange() Range
	GetModuleString() string
	GetModule() ModulePath
	IsSubModuleOf(parentModule ModulePath) bool

	GetHoverInfo() string
}

type IndexableCollection []Indexable

type BaseIndexable struct {
	moduleString string
	module       ModulePath
	documentURI  string
	idRange      Range
	docRange     Range
	Kind         protocol.CompletionItemKind
}

func NewBaseIndexable(docId protocol.DocumentUri, idRange Range, docRange Range, kind protocol.CompletionItemKind) BaseIndexable {
	return BaseIndexable{
		documentURI: docId,
		idRange:     idRange,
		docRange:    docRange,
		Kind:        kind,
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
		Start: NewPositionFromTreeSitterPoint(start),
		End:   NewPositionFromTreeSitterPoint(end),
	}
}

func (r Range) HasPosition(position protocol.Position) bool {
	line := uint(position.Line)
	ch := uint(position.Character)

	if line >= r.Start.Line && line <= r.End.Line {
		// Exactly same line
		if line == r.Start.Line && line == r.End.Line {
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

func NewPositionFromTreeSitterPoint(position sitter.Point) Position {
	return NewPosition(uint(position.Row), uint(position.Column))
}

func (p Position) ToLSPPosition() protocol.Position {
	return protocol.Position{
		Line:      uint32(p.Line),
		Character: uint32(p.Character),
	}
}
