package language

type SearchParams struct {
	selectedSymbol string
	parentSymbol   string // Limit search to symbols that has are child from parentSymbol
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
