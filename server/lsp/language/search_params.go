package language

import protocol "github.com/tliron/glsp/protocol_3_16"

type SearchParams struct {
	selectedSymbol string
	parentSymbol   string // Limit search to symbols that has are child from parentSymbol
	position       protocol.Position
	docId          string
}

func NewSearchParams(selectedSymbol string, docId string) SearchParams {
	return SearchParams{
		selectedSymbol: selectedSymbol,
		docId:          docId,
	}
}

func (s *SearchParams) HasParentSymbol() bool {
	return s.parentSymbol != ""
}
