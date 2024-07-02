package search

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/tliron/commonlog"
)

type TestState struct {
	state  project_state.ProjectState
	docs   map[string]document.Document
	parser p.Parser
}

func NewSearchWithoutLog() Search {
	logger := &MockLogger{
		tracker: make(map[string][]string),
	}
	search := NewSearch(logger, true)

	return search
}

func (t TestState) GetDoc(docId string) document.Document {
	return t.docs[docId]
}

func NewTestState(loggers ...commonlog.Logger) TestState {
	var logger commonlog.Logger

	if len(loggers) == 0 {
		logger = commonlog.MockLogger{}
	} else {
		logger = loggers[0]
	}

	l := project_state.NewProjectState(logger, option.Some("dummy"), false)

	s := TestState{
		state:  l,
		docs:   make(map[string]document.Document, 0),
		parser: p.NewParser(logger),
	}
	return s
}

func NewTestStateWithStdLibVersion(version string, loggers ...commonlog.Logger) TestState {
	var logger commonlog.Logger

	if len(loggers) == 0 {
		logger = commonlog.MockLogger{}
	} else {
		logger = loggers[0]
	}

	l := project_state.NewProjectState(logger, option.Some(version), false)

	s := TestState{
		state:  l,
		docs:   make(map[string]document.Document, 0),
		parser: p.NewParser(logger),
	}
	return s
}

func (s *TestState) clearDocs() {
	s.docs = make(map[string]document.Document, 0)
}

func (s *TestState) registerDoc(docId string, source string) {
	s.docs[docId] = document.NewDocument(docId, source)
	doc := s.docs[docId]
	s.state.RefreshDocumentIdentifiers(&doc, &s.parser)
}

func createParser() p.Parser {
	logger := &commonlog.MockLogger{}
	return p.NewParser(logger)
}
