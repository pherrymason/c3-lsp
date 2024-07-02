package project_state

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
)

func TestRefreshDocumentIdentifiers_should_clear_cached_stuff_test(t *testing.T) {

	var logger commonlog.Logger
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)
	doc := document.NewDocumentFromString(
		"doc-id",
		`module app::something_app;
		fn void main() {}
		`)
	s.RefreshDocumentIdentifiers(&doc, &p)
	result := s.indexByFQN.SearchByFQN("app::something_app.main")
	assert.Equal(t, 1, len(result))

	// Force a modification
	doc = document.NewDocumentFromString(
		"doc-id",
		`module app::something_new;
		fn void main() {}
		`)
	s.RefreshDocumentIdentifiers(&doc, &p)

	result = s.indexByFQN.SearchByFQN("app::something_app.main")
	assert.Equal(t, 0, len(result))

	result = s.indexByFQN.SearchByFQN("app::something_new.main")
	assert.Equal(t, 1, len(result))
}
