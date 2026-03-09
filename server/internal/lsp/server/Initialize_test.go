package server

import (
	"testing"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestInitializeWorkspaceURI_PrefersRootURI(t *testing.T) {
	rootURI := protocol.DocumentUri("file:///tmp/root")
	folderURI := protocol.DocumentUri("file:///tmp/folder")

	params := &protocol.InitializeParams{
		RootURI: &rootURI,
		WorkspaceFolders: []protocol.WorkspaceFolder{{
			URI:  folderURI,
			Name: "folder",
		}},
	}

	got := initializeWorkspaceURI(params)
	if got == nil || *got != rootURI {
		t.Fatalf("initializeWorkspaceURI should prefer RootURI, got %v", got)
	}
}

func TestInitializeWorkspaceURI_UsesFirstWorkspaceFolder(t *testing.T) {
	folderURI := protocol.DocumentUri("file:///tmp/workspace")

	params := &protocol.InitializeParams{
		WorkspaceFolders: []protocol.WorkspaceFolder{{
			URI:  folderURI,
			Name: "workspace",
		}},
	}

	got := initializeWorkspaceURI(params)
	if got == nil || *got != folderURI {
		t.Fatalf("initializeWorkspaceURI should use first workspace folder, got %v", got)
	}
}

func TestInitialize_setsWillSaveCapabilities(t *testing.T) {
	srv := NewServer(ServerOpts{
		C3: c3c.C3Opts{
			Version:     option.None[string](),
			Path:        option.None[string](),
			StdlibPath:  option.None[string](),
			CompileArgs: []string{},
		},
		Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 1},
		Formatting:  FormattingOpts{},
		LogFilepath: option.None[string](),
	}, "test-server", "test")

	params := &protocol.InitializeParams{}
	capabilities := protocol.ServerCapabilities{}
	result, err := srv.Initialize("test-server", "test", capabilities, &glsp.Context{Notify: func(string, any) {}}, params)
	if err != nil {
		t.Fatalf("expected initialize to succeed, got: %v", err)
	}

	initResult, ok := result.(protocol.InitializeResult)
	if !ok {
		t.Fatalf("expected initialize result type, got %T", result)
	}

	if initResult.Capabilities.TextDocumentSync == nil {
		t.Fatalf("expected textDocumentSync capability to be set")
	}

	syncOpts, ok := initResult.Capabilities.TextDocumentSync.(protocol.TextDocumentSyncOptions)
	if !ok {
		t.Fatalf("expected textDocumentSync options capabilities")
	}
	if syncOpts.WillSave == nil || !*syncOpts.WillSave {
		t.Fatalf("expected WillSave capability enabled")
	}
	if syncOpts.WillSaveWaitUntil == nil || !*syncOpts.WillSaveWaitUntil {
		t.Fatalf("expected WillSaveWaitUntil capability enabled")
	}

	if initResult.Capabilities.ExecuteCommandProvider == nil {
		t.Fatalf("expected executeCommand capability to be set")
	}

	if len(initResult.Capabilities.ExecuteCommandProvider.Commands) != 3 {
		t.Fatalf("expected three workspace commands, got %d", len(initResult.Capabilities.ExecuteCommandProvider.Commands))
	}

	linkedEditingEnabled, ok := initResult.Capabilities.LinkedEditingRangeProvider.(bool)
	if !ok || !linkedEditingEnabled {
		t.Fatalf("expected linkedEditingRange capability enabled")
	}

	monikerEnabled, ok := initResult.Capabilities.MonikerProvider.(bool)
	if !ok || !monikerEnabled {
		t.Fatalf("expected moniker capability enabled")
	}

	callHierarchyEnabled, ok := initResult.Capabilities.CallHierarchyProvider.(bool)
	if !ok || !callHierarchyEnabled {
		t.Fatalf("expected callHierarchy capability enabled")
	}

	if initResult.Capabilities.CodeActionProvider == nil {
		t.Fatalf("expected codeAction capability enabled")
	}

	var codeActionOpts protocol.CodeActionOptions
	switch value := initResult.Capabilities.CodeActionProvider.(type) {
	case *protocol.CodeActionOptions:
		codeActionOpts = *value
	case protocol.CodeActionOptions:
		codeActionOpts = value
	default:
		t.Fatalf("expected CodeActionOptions capability, got %T", initResult.Capabilities.CodeActionProvider)
	}
	if codeActionOpts.ResolveProvider == nil || !*codeActionOpts.ResolveProvider {
		t.Fatalf("expected codeAction resolve provider enabled")
	}
}
