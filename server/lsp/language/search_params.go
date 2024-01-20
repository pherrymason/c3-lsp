package language

type SearchParams struct {
	symbol       string
	parentSymbol string
}

func NewSearch(symbol string) SearchParams {
	return SearchParams{
		symbol: symbol,
	}
}

func (s *SearchParams) HasParentSymbol() bool {
	return s.parentSymbol != ""
}
