package search_v2

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
	"strings"
)

// TestState wraps the project state for testing
type TestState struct {
	State  project_state.ProjectState
	Docs   map[string]document.Document
	Parser p.Parser
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
		State:  l,
		Docs:   make(map[string]document.Document, 0),
		Parser: p.NewParser(logger),
	}
	return s
}

func (t TestState) GetDoc(docId string) document.Document {
	return t.Docs[docId]
}

func (s *TestState) RegisterDoc(docId string, source string) {
	s.Docs[docId] = document.NewDocument(docId, source)
	doc := s.Docs[docId]
	s.State.RefreshDocumentIdentifiers(&doc, &s.Parser)
}

func buildPosition(line uint, character uint) symbols.Position {
	return symbols.Position{Line: line - 1, Character: character}
}

// Parses a test body with a '|||' cursor, returning the body without
// the cursor and the position of that cursor.
func parseBodyWithCursor(body string) (string, symbols.Position) {
	cursorLine, cursorCol := utils.FindLineColOfSubstring(body, "|||")
	if cursorLine == 0 {
		panic("Please add the cursor position to the test body with '|||'")
	}
	if strings.Count(body, "|||") > 1 {
		panic("There are multiple '|||' cursors in the test body, please add only one")
	}

	cursorlessBody := strings.ReplaceAll(body, "|||", "")
	position := buildPosition(cursorLine, cursorCol)

	return cursorlessBody, position
}

// NewSearchV2WithoutLog creates a SearchV2 instance for testing
func NewSearchV2WithoutLog() *SearchV2 {
	logger := commonlog.MockLogger{}
	return NewSearchV2(logger, false)
}

// NewOldSearchWithoutLog creates the old search implementation for comparison tests
func NewOldSearchWithoutLog() search.Search {
	return search.NewSearch(commonlog.MockLogger{}, false)
}
