package server

import (
	"testing"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDiagnosticsRelevantFiles_filters_by_impacted_modules(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	docA := document.NewDocumentFromString("a.c3", "module app::a; import app::b; fn void a() {}")
	docB := document.NewDocumentFromString("b.c3", "module app::b; fn void b() {}")
	state.RefreshDocumentIdentifiers(&docA, &p)
	state.RefreshDocumentIdentifiers(&docB, &p)

	changeB := document.NewDocumentFromString("b.c3", "module app::b; fn int b(int x) { return x; }")
	state.RefreshDocumentIdentifiers(&changeB, &p)

	uri := protocol.DocumentUri("file:///tmp/b.c3")
	relevant := diagnosticsRelevantFiles(&state, &uri, state.GetLastInvalidationScope())

	assert.True(t, relevant["a.c3"])
	assert.True(t, relevant["b.c3"])
}

func TestDiagnosticsRelevantFiles_without_trigger_returns_all(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)

	relevant := diagnosticsRelevantFiles(&state, nil, project_state.InvalidationScope{ImpactedModules: []string{"app::x"}})
	assert.Nil(t, relevant)
}

func TestClearDiagnosticsForFiles_publishes_synchronously(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)

	state.SetDocumentDiagnostics("a.c3", []protocol.Diagnostic{{Message: "boom"}})

	srv := &Server{options: ServerOpts{C3: c3c.C3Opts{StdlibPath: option.None[string]()}}}
	notifyCalls := 0
	notify := func(method string, params any) {
		if method == protocol.ServerTextDocumentPublishDiagnostics {
			notifyCalls++
		}
	}

	cleared := srv.clearDiagnosticsForFiles(&state, notify, nil)

	assert.Equal(t, 1, cleared)
	assert.Equal(t, 1, notifyCalls)
	assert.Empty(t, state.GetDocumentDiagnostics())
}
