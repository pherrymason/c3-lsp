package language

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
	l := NewLanguage(logger, option.Some("dummy"))
	p := parser.NewParser(logger)
	doc := document.NewDocumentFromString(
		"doc-id",
		`module app::something_app;
		fn void main() {}
		`)
	l.RefreshDocumentIdentifiers(&doc, &p)
	result := l.indexByFQN.SearchByFQN("app::something_app.main")
	assert.Equal(t, 1, len(result))

	// Force a modification
	doc = document.NewDocumentFromString(
		"doc-id",
		`module app::something_new;
		fn void main() {}
		`)
	l.RefreshDocumentIdentifiers(&doc, &p)

	result = l.indexByFQN.SearchByFQN("app::something_app.main")
	assert.Equal(t, 0, len(result))

	result = l.indexByFQN.SearchByFQN("app::something_new.main")
	assert.Equal(t, 1, len(result))
}
