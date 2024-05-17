package lsp

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/parser"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
)

func assertSameRange(t *testing.T, expected idx.Range, actual idx.Range, msg string) {
	assert.Equal(t, expected.Start, actual.Start, fmt.Sprint(msg, " start"))
	assert.Equal(t, expected.Start, actual.Start, fmt.Sprint(msg, " end"))
}

func createParser() parser.Parser {
	return parser.Parser{
		Logger: commonlog.MockLogger{},
	}
}
