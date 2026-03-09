package server

import (
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type lspSpecBenchFixture struct {
	server         *Server
	uri            protocol.DocumentUri
	hoverPos       protocol.Position
	callPos        protocol.Position
	varPos         protocol.Position
	signaturePos   protocol.Position
	docLinkResolve protocol.DocumentLink
}

func setupLspSpecBenchFixture() lspSpecBenchFixture {
	source := `module app;
import std::io;

struct Todo {
	int id;
}

fn int add(int a, int b) {
	return a + b;
}

fn void use() {
	Todo t = {.id = 1};
	int x = add(1, 2);
	io::printfn("%d", x);
	t.id = x;
}`

	uri := protocol.DocumentUri("file:///tmp/lsp_spec_perf_fixture.c3")
	srv := buildRenameTestServer(uri, source)
	srv.initialized.Store(true)

	hoverPos := byteIndexToLSPPosition(source, indexOrPanic(source, "printfn")+2)
	callPos := byteIndexToLSPPosition(source, indexOrPanic(source, "add(1, 2)")+1)
	varPos := byteIndexToLSPPosition(source, indexOrPanic(source, "x = add")+1)
	signaturePos := byteIndexToLSPPosition(source, indexOrPanic(source, "add(1, 2)")+len("add(1"))

	resolveTarget := protocol.DocumentUri("file:///tmp/lsp_spec_perf_target.c3")
	link := protocol.DocumentLink{Data: map[string]any{"target": string(resolveTarget)}}

	return lspSpecBenchFixture{
		server:         srv,
		uri:            uri,
		hoverPos:       hoverPos,
		callPos:        callPos,
		varPos:         varPos,
		signaturePos:   signaturePos,
		docLinkResolve: link,
	}
}

func BenchmarkLspSpec_TextDocumentHover(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.hoverPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentHover(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentDefinition(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DefinitionParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.callPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentDefinition(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentDeclaration(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DeclarationParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.callPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentDeclaration(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentImplementation(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.ImplementationParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.callPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentImplementation(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentTypeDefinition(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.TypeDefinitionParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.varPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentTypeDefinition(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentSignatureHelp(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.SignatureHelpParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.signaturePos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentSignatureHelp(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentDocumentHighlight(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DocumentHighlightParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.varPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentDocumentHighlight(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentDocumentSymbol(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DocumentSymbolParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentDocumentSymbol(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentCodeAction_Empty(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.CodeActionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentCodeAction(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentFoldingRange(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.FoldingRangeParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentFoldingRange(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentSelectionRange(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.SelectionRangeParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Positions: []protocol.Position{f.varPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentSelectionRange(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentLinkedEditingRange(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.LinkedEditingRangeParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.varPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentLinkedEditingRange(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentMoniker(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.MonikerParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.varPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentMoniker(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentPrepareCallHierarchy(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.CallHierarchyPrepareParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.callPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentPrepareCallHierarchy(nil, params)
	}
}

func BenchmarkLspSpec_DocumentLinkResolve(b *testing.B) {
	f := setupLspSpecBenchFixture()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		link := f.docLinkResolve
		_, _ = f.server.DocumentLinkResolve(nil, &link)
	}
}

func BenchmarkLspSpec_WorkspaceSymbol(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.WorkspaceSymbolParams{Query: "add"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.WorkspaceSymbol(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentPrepareRename(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.PrepareRenameParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.varPos}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentPrepareRename(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentFormatting_ErrorPath(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DocumentFormattingParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentFormatting(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentRangeFormatting_ErrorPath(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DocumentRangeFormattingParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Range: protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 1, Character: 0}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentRangeFormatting(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentOnTypeFormatting_ErrorPath(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DocumentOnTypeFormattingParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}, Position: f.varPos}, Ch: ";"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentOnTypeFormatting(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentDidOpen(b *testing.B) {
	f := setupLspSpecBenchFixture()
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{URI: f.uri, LanguageID: "c3", Version: 2, Text: "module app;\nfn void x() {}"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.server.TextDocumentDidOpen(ctx, params)
	}
}

func BenchmarkLspSpec_TextDocumentDidClose(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.DidCloseTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.server.TextDocumentDidClose(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentWillSave(b *testing.B) {
	f := setupLspSpecBenchFixture()
	params := &protocol.WillSaveTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.server.TextDocumentWillSave(nil, params)
	}
}

func BenchmarkLspSpec_TextDocumentWillSaveWaitUntil_DefaultNoEdits(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.options.Formatting.WillSaveWaitUntil = false
	params := &protocol.WillSaveTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: f.uri}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.TextDocumentWillSaveWaitUntil(nil, params)
	}
}

func BenchmarkLspSpec_WorkspaceDidDeleteFiles(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.initialized.Store(true)
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.DeleteFilesParams{Files: []protocol.FileDelete{{URI: f.uri}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.server.WorkspaceDidDeleteFiles(ctx, params)
	}
}

func BenchmarkLspSpec_WorkspaceDidRenameFiles(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.initialized.Store(true)
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.RenameFilesParams{Files: []protocol.FileRename{{OldURI: f.uri, NewURI: protocol.DocumentUri("file:///tmp/lsp_spec_perf_fixture_renamed.c3")}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.server.WorkspaceDidRenameFiles(ctx, params)
	}
}

func BenchmarkLspSpec_WorkspaceDidCreateFiles(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.initialized.Store(true)
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.CreateFilesParams{Files: []protocol.FileCreate{{URI: protocol.DocumentUri("file:///tmp/lsp_spec_perf_created.c3")}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.server.WorkspaceDidCreateFiles(ctx, params)
	}
}

func BenchmarkLspSpec_WorkspaceWillCreateFiles(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.initialized.Store(true)
	params := &protocol.CreateFilesParams{Files: []protocol.FileCreate{{URI: protocol.DocumentUri("file:///tmp/lsp_spec_perf_created.c3")}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.WorkspaceWillCreateFiles(nil, params)
	}
}

func BenchmarkLspSpec_WorkspaceWillRenameFiles(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.initialized.Store(true)
	params := &protocol.RenameFilesParams{Files: []protocol.FileRename{{OldURI: f.uri, NewURI: protocol.DocumentUri("file:///tmp/lsp_spec_perf_will_renamed.c3")}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.WorkspaceWillRenameFiles(nil, params)
	}
}

func BenchmarkLspSpec_WorkspaceWillDeleteFiles(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.initialized.Store(true)
	params := &protocol.DeleteFilesParams{Files: []protocol.FileDelete{{URI: f.uri}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.WorkspaceWillDeleteFiles(nil, params)
	}
}

func BenchmarkLspSpec_WorkspaceDidChangeConfiguration(b *testing.B) {
	f := setupLspSpecBenchFixture()
	f.server.initialized.Store(true)
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.DidChangeConfigurationParams{Settings: map[string]any{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.server.WorkspaceDidChangeConfiguration(ctx, params)
	}
}

func BenchmarkLspSpec_WorkspaceExecuteCommand_ClearDiagnosticsCache(b *testing.B) {
	f := setupLspSpecBenchFixture()
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.ExecuteCommandParams{Command: workspaceCommandClearDiagnostics}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.server.WorkspaceExecuteCommand(ctx, params)
	}
}

func indexOrPanic(source string, needle string) int {
	idx := -1
	for i := 0; i+len(needle) <= len(source); i++ {
		if source[i:i+len(needle)] == needle {
			idx = i
			break
		}
	}
	if idx < 0 {
		panic("needle not found: " + needle)
	}
	return idx
}
