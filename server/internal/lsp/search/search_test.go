package search

import (
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
)

type TestState struct {
	state  *project_state.ProjectState
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
		state:  &l,
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
		state:  &l,
		docs:   make(map[string]document.Document, 0),
		parser: p.NewParser(logger),
	}
	return s
}

func (s *TestState) registerDoc(docId string, source string) {
	s.docs[docId] = document.NewDocument(docId, source)
	doc := s.docs[docId]
	s.state.RefreshDocumentIdentifiers(&doc, &s.parser)
}

func buildPosition(line uint, character uint) symbols.Position {
	return symbols.Position{Line: line - 1, Character: character}
}

// Parses a test body with a '|||' cursor, returning the body without
// the cursor and the position of that cursor.
//
// Useful for tests where we check what the language server responds if the
// user cursor is at a certain position.
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

func createParser() p.Parser {
	logger := &commonlog.MockLogger{}
	return p.NewParser(logger)
}

// Helper functions for auto-calculating ranges in tests

// findNthRange searches for the nth occurrence of text in source (1-indexed).
// Returns symbols.Range{} if not found or n is invalid.
func findNthRange(source string, text string, n int) symbols.Range {
	if n < 1 {
		return symbols.Range{}
	}

	lines := splitLines(source)
	occurrences := 0

	for lineIdx, line := range lines {
		col := 0
		for {
			foundIdx := findInString(line[col:], text)
			if foundIdx == -1 {
				break
			}
			occurrences++
			if occurrences == n {
				startCol := col + foundIdx
				endCol := startCol + len(text)
				return symbols.NewRange(uint(lineIdx), uint(startCol), uint(lineIdx), uint(endCol))
			}
			col += foundIdx + 1
		}
	}

	return symbols.Range{}
}

func splitLines(s string) []string {
	lines := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" || len(s) > 0 {
		lines = append(lines, current)
	}
	return lines
}

func findInString(s string, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
