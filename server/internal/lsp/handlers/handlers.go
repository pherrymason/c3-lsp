package handlers

import (
	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	s "github.com/pherrymason/c3-lsp/internal/lsp/search"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
)

type Handlers struct {
	state  *l.ProjectState
	parser *p.Parser
	search s.Search
}

func NewHandlers(state *l.ProjectState, parser *p.Parser, search s.Search) Handlers {
	return Handlers{
		state:  state,
		parser: parser,
		search: search,
	}
}
