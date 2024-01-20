package language

type SearchParams struct {
	selectedSymbol string
	parentSymbol   string
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
