package server

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestModuleRenameEdits(t *testing.T) {
	source := "module old::name;\nimport old::name;\nold::name::Thing value;\nother_old::name;\n"
	edits := moduleRenameEdits(source, "old::name", "new::name")

	if len(edits) != 3 {
		t.Fatalf("unexpected edit count: got %d", len(edits))
	}

	applied := applyTextEdits(source, edits)
	expected := "module new::name;\nimport new::name;\nnew::name::Thing value;\nother_old::name;\n"
	if applied != expected {
		t.Fatalf("unexpected renamed output:\n%s", applied)
	}
}

func TestModuleRenameTargetOnDeclaration(t *testing.T) {
	docID := "test.c3"
	unitModules := symbols_table.NewParsedModules(&docID)

	source := "module old::name;\n"
	position := protocol.Position{Line: 0, Character: 12} // on 'n' from 'name'

	target, ok := moduleRenameTarget(source, position, &unitModules)
	if !ok {
		t.Fatalf("expected module rename target")
	}

	if target.name != "name" {
		t.Fatalf("unexpected target name: got %q", target.name)
	}
	if target.moduleFullName != "old::name" {
		t.Fatalf("unexpected full module target name: got %q", target.moduleFullName)
	}
}

func TestModuleRenameTargetRejectsNonModuleSymbol(t *testing.T) {
	docID := "test.c3"
	unitModules := symbols_table.NewParsedModules(&docID)

	source := "fn void test() {\n\tint value;\n}\n"
	position := protocol.Position{Line: 1, Character: 6} // on 'v' from value

	_, ok := moduleRenameTarget(source, position, &unitModules)
	if ok {
		t.Fatalf("expected non-module symbol to be rejected")
	}
}

func TestTextDocumentRename_includes_document_changes_when_client_supports_it(t *testing.T) {
	source := `module app;

fn void parse_port() {
}

fn void run() {
	parse_port();
}`

	uri := protocol.DocumentUri("file:///tmp/rename_document_changes_supported_test.c3")
	srv := buildRenameTestServer(uri, source)

	var caps protocol.ClientCapabilities
	if err := json.Unmarshal([]byte(`{"workspace":{"workspaceEdit":{"documentChanges":true}}}`), &caps); err != nil {
		t.Fatalf("failed to build client capabilities: %v", err)
	}
	srv.clientCapabilities = caps

	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "parse_socket_port",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}
	if len(edit.Changes) == 0 {
		t.Fatalf("expected changes map to remain populated")
	}
	if len(edit.DocumentChanges) == 0 {
		t.Fatalf("expected documentChanges when client supports it")
	}
}

func TestTextDocumentRename_keeps_changes_only_when_document_changes_not_supported(t *testing.T) {
	source := `module app;

fn void parse_port() {
}

fn void run() {
	parse_port();
}`

	uri := protocol.DocumentUri("file:///tmp/rename_document_changes_unsupported_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "parse_socket_port",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}
	if len(edit.Changes) == 0 {
		t.Fatalf("expected changes map to be populated")
	}
	if len(edit.DocumentChanges) != 0 {
		t.Fatalf("expected no documentChanges without support")
	}
}

func TestTextDocumentRename_cross_client_smoke_vscode_capabilities(t *testing.T) {
	runRenameCrossClientSmoke(t, `{"workspace":{"workspaceEdit":{"documentChanges":true}}}`)
}

func TestTextDocumentRename_cross_client_smoke_zed_capabilities(t *testing.T) {
	runRenameCrossClientSmoke(t, `{"workspace":{"workspaceEdit":{"documentChanges":false}}}`)
}

func TestTextDocumentRename_cross_client_smoke_neovim_minimal_capabilities(t *testing.T) {
	runRenameCrossClientSmoke(t, `{}`)
}

func runRenameCrossClientSmoke(t *testing.T, capabilitiesJSON string) {
	t.Helper()

	source := `module app;

fn void parse_port() {
}

fn void run() {
	parse_port();
}`

	uri := protocol.DocumentUri("file:///tmp/rename_cross_client_smoke_test.c3")
	srv := buildRenameTestServer(uri, source)

	if capabilitiesJSON != "" {
		var caps protocol.ClientCapabilities
		if err := json.Unmarshal([]byte(capabilitiesJSON), &caps); err != nil {
			t.Fatalf("failed to build client capabilities: %v", err)
		}
		srv.clientCapabilities = caps
	}

	preparePos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port();")+len("parse_port"))
	prepareResult, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     preparePos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if prepareResult == nil {
		t.Fatalf("expected prepare rename result")
	}

	preparedRange, ok := prepareResult.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if preparedRange.Placeholder != "parse_port" {
		t.Fatalf("expected prepare rename placeholder parse_port, got: %q", preparedRange.Placeholder)
	}

	renamePos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port()"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     renamePos,
		},
		NewName: "parse_socket_port",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}
	if len(edit.Changes) == 0 {
		t.Fatalf("expected changes map to be populated")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "fn void parse_socket_port()") || !strings.Contains(updated, "parse_socket_port();") {
		t.Fatalf("expected declaration and callsite rename, got: %s", updated)
	}
}

func TestTextDocumentRename_module_name_updates_cross_file_qualified_usages(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_module_blem_net_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_module_blem_net_use_test.c3")

	declSource := `module blem::net;

faultdef TIMEOUT;

fn int read_timeout() {
	return TIMEOUT!;
}`

	useSource := `module blem::http;

import blem::net;

fn bool timed_out(fault err) {
	if (err == blem::net::TIMEOUT) return true;
	(void)blem::net::read_timeout();
	return false;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "blem::net")+len("blem::net")-1)
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		NewName: "blem::network",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "module blem::network;") {
		t.Fatalf("expected declaration module rename, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if !strings.Contains(updatedUse, "import blem::network;") {
		t.Fatalf("expected import module rename, got: %s", updatedUse)
	}
	if !strings.Contains(updatedUse, "blem::network::TIMEOUT") {
		t.Fatalf("expected qualified fault usage module rename, got: %s", updatedUse)
	}
	if !strings.Contains(updatedUse, "blem::network::read_timeout") {
		t.Fatalf("expected qualified function usage module rename, got: %s", updatedUse)
	}
}

func TestTextDocumentRename_module_leaf_segment_updates_cross_file_qualified_usages(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_module_segment_blem_net_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_module_segment_blem_net_use_test.c3")

	declSource := `module blem::net;

faultdef TIMEOUT;

fn int read_timeout() {
	return TIMEOUT!;
}`

	useSource := `module blem::http;

import blem::net;

fn bool timed_out(fault err) {
	if (err == blem::net::TIMEOUT) return true;
	(void)blem::net::read_timeout();
	return false;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "::net")+len("::n"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		NewName: "network",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "module blem::network;") {
		t.Fatalf("expected declaration module leaf rename, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if !strings.Contains(updatedUse, "import blem::network;") {
		t.Fatalf("expected import module leaf rename, got: %s", updatedUse)
	}
	if !strings.Contains(updatedUse, "blem::network::TIMEOUT") {
		t.Fatalf("expected qualified fault usage module leaf rename, got: %s", updatedUse)
	}
	if !strings.Contains(updatedUse, "blem::network::read_timeout") {
		t.Fatalf("expected qualified function usage module leaf rename, got: %s", updatedUse)
	}
}

func TestTextDocumentPrepareRename_module_leaf_segment_returns_leaf_placeholder(t *testing.T) {
	source := `module blem::net;

fn void test() {
	(void)0;
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_module_leaf_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "::net")+len("::n"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename result for module leaf segment")
	}

	withPlaceholder, ok := result.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if withPlaceholder.Placeholder != "net" {
		t.Fatalf("expected module leaf placeholder net, got: %q", withPlaceholder.Placeholder)
	}
	rangeText := source[symbols.NewPositionFromLSPPosition(withPlaceholder.Range.Start).IndexIn(source):symbols.NewPositionFromLSPPosition(withPlaceholder.Range.End).IndexIn(source)]
	if rangeText != "net" {
		t.Fatalf("expected module leaf range text net, got: %q", rangeText)
	}
}

func TestTextDocumentRename_module_root_segment_updates_only_root_part(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_module_root_segment_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_module_root_segment_use_test.c3")

	declSource := `module blem::net;

fn int read_timeout() {
	return 0;
}`

	useSource := `module blem::http;

import blem::net;

fn void use_net() {
	(void)blem::net::read_timeout();
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "blem::")+1)
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		NewName: "core",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "module core::net;") {
		t.Fatalf("expected declaration module root rename, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if !strings.Contains(updatedUse, "import core::net;") {
		t.Fatalf("expected import module root rename, got: %s", updatedUse)
	}
	if !strings.Contains(updatedUse, "core::net::read_timeout") {
		t.Fatalf("expected qualified function usage module root rename, got: %s", updatedUse)
	}
}

func applyTextEdits(source string, edits []protocol.TextEdit) string {
	sorted := make([]protocol.TextEdit, len(edits))
	copy(sorted, edits)

	sort.Slice(sorted, func(i, j int) bool {
		left := symbols.NewPositionFromLSPPosition(sorted[i].Range.Start).IndexIn(source)
		right := symbols.NewPositionFromLSPPosition(sorted[j].Range.Start).IndexIn(source)
		return left > right
	})

	out := source
	for _, edit := range sorted {
		start := symbols.NewPositionFromLSPPosition(edit.Range.Start).IndexIn(out)
		end := symbols.NewPositionFromLSPPosition(edit.Range.End).IndexIn(out)
		out = out[:start] + edit.NewText + out[end:]
	}

	return out
}

func TestTextDocumentRename_local_variable_is_scope_safe(t *testing.T) {
	source := `module app;
fn void a() {
	int value = 1;
	value = value + 1;
}
fn void b() {
	int value = 2;
	value = value + 1;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_local_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "value = 1"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "localValue",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Count(updated, "localValue") != 3 {
		t.Fatalf("expected exactly 3 renamed occurrences in first scope, got: %s", updated)
	}
	if !strings.Contains(updated, "int value = 2") {
		t.Fatalf("expected second scope variable to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_try_bound_variable_is_scope_safe(t *testing.T) {
	source := `module app;
fn uint parse_clients(String[] args, uint default_value = 32) {
	if (args.len < 2) return default_value;
	if (try try_n_bind = args[1].to_integer(uint, 10)) {
		if (try_n_bind > 0) return try_n_bind;
	}
	return default_value;
}
fn uint parse_port(String[] args, uint default_value = 19080) {
	if (try try_n_bind = args[1].to_integer(uint, 10)) {
		if (try_n_bind > 0) return try_n_bind;
	}
	return default_value;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_try_bound_variable_scope_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "try_n_bind = args[1]"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "parsed_clients",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Count(updated, "parsed_clients") != 3 {
		t.Fatalf("expected exactly 3 renamed try-bound occurrences in first function, got: %s", updated)
	}
	if strings.Count(updated, "try_n_bind") != 3 {
		t.Fatalf("expected second function try binding to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_catch_bound_variable_is_scope_safe(t *testing.T) {
	source := `module app;
fn bool read_http() {
	if (catch catch_err = maybe_client()) {
		io::printfn("err=%s", catch_err);
		return false;
	}
	return true;
}
fn bool read_tcp() {
	if (catch catch_err = maybe_socket()) {
		io::printfn("err=%s", catch_err);
		return false;
	}
	return true;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_catch_bound_variable_scope_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "catch_err = maybe_client()"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "client_err",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Count(updated, "client_err") != 2 {
		t.Fatalf("expected declaration + usage rename for first catch binding, got: %s", updated)
	}
	if strings.Count(updated, "catch_err") != 2 {
		t.Fatalf("expected second function catch binding to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_function_conflict_returns_error(t *testing.T) {
	source := `module app;

fn void parse_port() {
}

fn void parse_socket_port() {
}`

	uri := protocol.DocumentUri("file:///tmp/rename_function_conflict_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port"))
	_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "parse_socket_port",
	})
	if err == nil {
		t.Fatalf("expected rename conflict error")
	}
	if !strings.Contains(err.Error(), "rename conflict") {
		t.Fatalf("expected conflict error, got: %v", err)
	}
}

func TestTextDocumentRename_struct_member_conflict_returns_error(t *testing.T) {
	source := `module app;

struct Fiber {
	int done;
	int closed;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_struct_member_conflict_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "done;"))
	_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "closed",
	})
	if err == nil {
		t.Fatalf("expected rename conflict error")
	}
	if !strings.Contains(err.Error(), "rename conflict") {
		t.Fatalf("expected conflict error, got: %v", err)
	}
}

func TestTextDocumentRename_function_renames_declaration_and_callsite(t *testing.T) {
	source := `module app;
fn int bump(int n) {
	return n + 1;
}

fn void main() {
	bump(1);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_function_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bump(int"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "grow",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "fn int grow(int n)") || !strings.Contains(updated, "grow(1)") {
		t.Fatalf("expected declaration and callsite rename, got: %s", updated)
	}
}

func TestTextDocumentRename_function_from_callsite_cursor_on_open_paren(t *testing.T) {
	source := `module app;
fn int bump(int n) {
	return n + 1;
}

fn void main() {
	bump(1);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_function_callsite_open_paren_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bump(1)")+len("bump"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "grow",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "fn int grow(int n)") || !strings.Contains(updated, "grow(1)") {
		t.Fatalf("expected declaration and callsite rename from open paren cursor, got: %s", updated)
	}
}

func TestTextDocumentRename_rejects_invalid_identifier_name(t *testing.T) {
	source := `module app;
fn int bump(int n) {
	return n + 1;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_invalid_identifier_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bump(int"))
	_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "123bad",
	})

	if err == nil {
		t.Fatalf("expected invalid identifier rename to fail")
	}
}

func TestTextDocumentRename_rejects_invalid_function_name_with_kind_specific_error(t *testing.T) {
	source := `module app;
fn int bump(int n) {
	return n + 1;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_invalid_function_name_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bump(int"))
	_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "123bad",
	})

	if err == nil {
		t.Fatalf("expected invalid function rename to fail")
	}
	if !strings.Contains(err.Error(), "invalid function name") {
		t.Fatalf("expected kind-specific function validation error, got: %v", err)
	}
}

func TestTextDocumentRename_rejects_invalid_struct_member_name_with_kind_specific_error(t *testing.T) {
	source := `module blem;

struct Fiber {
	int entry;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_invalid_struct_member_name_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "entry;"))
	_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "not-valid!",
	})

	if err == nil {
		t.Fatalf("expected invalid struct member rename to fail")
	}
	if !strings.Contains(err.Error(), "invalid struct member name") {
		t.Fatalf("expected kind-specific struct member validation error, got: %v", err)
	}
}

func TestTextDocumentRename_function_does_not_touch_comments_or_strings(t *testing.T) {
	source := `module app;
fn int bump(int n) {
	// bump should stay in comment
	io::printf("bump should stay in string");
	return bump(n - 1);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_function_comment_string_safety_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bump(int"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "grow",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "fn int grow(int n)") || !strings.Contains(updated, "return grow(n - 1)") {
		t.Fatalf("expected declaration and callsite rename, got: %s", updated)
	}
	if !strings.Contains(updated, "// bump should stay in comment") {
		t.Fatalf("expected comment content to remain unchanged, got: %s", updated)
	}
	if !strings.Contains(updated, "\"bump should stay in string\"") {
		t.Fatalf("expected string literal content to remain unchanged, got: %s", updated)
	}
	if strings.Contains(updated, "grow should stay") {
		t.Fatalf("expected no comment/string literal rewrite, got: %s", updated)
	}
}

func TestTextDocumentRename_module_does_not_touch_comments_or_strings(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_module_comment_string_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_module_comment_string_use_test.c3")

	declSource := `module blem::net;
// keep blem::net in comment
const String MOD = "blem::net";
faultdef TIMEOUT;`

	useSource := `module blem::http;
import blem::net;

fn bool timed_out(fault err) {
	// keep blem::net in comment
	const String MOD = "blem::net";
	return err == blem::net::TIMEOUT;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "blem::net")+len("blem::"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		NewName: "network",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "module blem::network;") {
		t.Fatalf("expected module declaration rename, got: %s", updatedDecl)
	}
	if !strings.Contains(updatedDecl, "// keep blem::net in comment") || !strings.Contains(updatedDecl, "\"blem::net\"") {
		t.Fatalf("expected declaration comments/strings unchanged, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if !strings.Contains(updatedUse, "import blem::network;") || !strings.Contains(updatedUse, "blem::network::TIMEOUT") {
		t.Fatalf("expected import and qualified usage rename, got: %s", updatedUse)
	}
	if !strings.Contains(updatedUse, "// keep blem::net in comment") || !strings.Contains(updatedUse, "\"blem::net\"") {
		t.Fatalf("expected usage comments/strings unchanged, got: %s", updatedUse)
	}
}

func TestTextDocumentRename_cursor_inside_comment_returns_empty_edit(t *testing.T) {
	source := `module app;
fn int bump(int n) {
	// bump is mentioned in comment
	return bump(n);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_cursor_inside_comment_noop_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bump is mentioned"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "grow",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}
	if len(edit.Changes) != 0 {
		t.Fatalf("expected no edits when cursor is inside comment, got: %#v", edit.Changes)
	}
}

func TestTextDocumentRename_constdef_alias_renames_declaration_and_usages(t *testing.T) {
	source := `module blem;

constdef Sig : inline CInt {
	BLOCK = 1,
	UNBLOCK = 2,
	SETMASK = 3,
}

fn void main() {
	(void)sigprocmask(Sig.UNBLOCK, null, null);
	(void)sigprocmask(Sig.SETMASK, null, null);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_constdef_sig_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "Sig :"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "SignalMask",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Count(updated, "SignalMask") != 3 {
		t.Fatalf("expected declaration + 2 usages to rename, got: %s", updated)
	}
}

func TestTextDocumentPrepareRename_constdef_alias_returns_range(t *testing.T) {
	source := `module blem;
constdef Sig : inline CInt { BLOCK = 1 }`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_constdef_sig_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "Sig :"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename range")
	}
}

func TestTextDocumentRename_constdef_member_renames_declaration_and_accesses(t *testing.T) {
	source := `module blem;

constdef Sig : inline CInt {
	BLOCK = 1,
	UNBLOCK = 2,
	SETMASK = 3,
}

fn void main() {
	(void)sigprocmask(Sig.SETMASK, null, null);
	(void)sigprocmask(Sig.UNBLOCK, null, null);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_constdef_member_setmask_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "SETMASK ="))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "RESETMASK",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "RESETMASK = 3") {
		t.Fatalf("expected constdef member declaration rename, got: %s", updated)
	}
	if !strings.Contains(updated, "Sig.RESETMASK") {
		t.Fatalf("expected constdef member access rename, got: %s", updated)
	}
	if strings.Contains(updated, "Sig.UNBLOCK") == false {
		t.Fatalf("expected unrelated member to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_constdef_member_middle_entry_renames_all_references(t *testing.T) {
	source := `module blem;

constdef Sig : inline CInt {
	BLOCK = 1,
	UNBLOCK = 2,
	SETMASK = 3,
}

fn void main() {
	(void)sigprocmask(Sig.SETMASK, null, null);
	(void)sigprocmask(Sig.UNBLOCK, null, null);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_constdef_member_unblock_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "UNBLOCK ="))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "ALLOW",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "ALLOW = 2") {
		t.Fatalf("expected middle constdef member declaration rename, got: %s", updated)
	}
	if !strings.Contains(updated, "Sig.ALLOW") {
		t.Fatalf("expected middle constdef member access rename, got: %s", updated)
	}
	if strings.Contains(updated, "Sig.SETMASK") == false {
		t.Fatalf("expected unrelated member to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_struct_member_from_access_usage(t *testing.T) {
	source := `module blem;

alias Coroutine = fn void();

struct Fiber {
	Coroutine entry;
}

Fiber* running;

fn void invoke() {
	running.entry();
}`

	uri := protocol.DocumentUri("file:///tmp/rename_fiber_entry_member_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "entry()"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "callback",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Coroutine callback;") || !strings.Contains(updated, "running.callback();") {
		t.Fatalf("expected member declaration and access usage to be renamed, got: %s", updated)
	}
}

func TestTextDocumentRename_struct_member_from_declaration_with_shadowed_param(t *testing.T) {
	source := `module blem;

alias Coroutine = fn void();

struct Fiber {
	Coroutine entry;
}

fn void init_common(Fiber* fiber, Coroutine entry) {
	fiber.entry = entry;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_fiber_entry_decl_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "entry;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "callback",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Coroutine callback;") {
		t.Fatalf("expected member declaration rename, got: %s", updated)
	}
	if !strings.Contains(updated, "fiber.callback = entry;") {
		t.Fatalf("expected member access rename but keep param reference, got: %s", updated)
	}
	if strings.Contains(updated, "Coroutine callback)") {
		t.Fatalf("expected function parameter named entry to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_struct_member_propagates_across_multiple_access_sites(t *testing.T) {
	source := `module blem;

alias Coroutine = fn void();

struct Fiber {
	Coroutine entry;
	bool done;
}

Fiber primary;
Fiber* running = null;
Fiber* creating = null;

fn void springboard() {
	if (creating == null) return;
	running.entry();
	running.done = true;
}

fn void init_common(Fiber* fiber, Coroutine entry) {
	if (fiber.done) fiber.entry = entry;
	if (fiber.entry == null) {
		fiber.entry = entry;
	}
}

fn void resume(Fiber* fiber) {
	if (fiber == null || fiber.done || fiber == running) return;
	if (fiber.entry != null) {
		fiber.entry();
	}
}
`

	uri := protocol.DocumentUri("file:///tmp/rename_fiber_entry_propagation_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "entry;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "callback",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Count(updated, ".callback") != 6 {
		t.Fatalf("expected all member access sites to rename, got: %s", updated)
	}
	if !strings.Contains(updated, "Coroutine callback;") {
		t.Fatalf("expected member declaration to rename, got: %s", updated)
	}
	if strings.Contains(updated, "Coroutine callback)") {
		t.Fatalf("expected parameter named entry to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_struct_member_does_not_rename_module_path_token(t *testing.T) {
	source := `module blem;

struct Fiber {
	Allocator allocator;
	void* stack;
}

fn void destroy(Fiber* fiber) {
	allocator::free(fiber.allocator, fiber.stack);
	allocator::free(fiber.allocator, fiber);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_fiber_allocator_member_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "allocator;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "my_alloc",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Contains(updated, "my_alloc::free") {
		t.Fatalf("module path token should not be renamed for member rename, got: %s", updated)
	}
	if strings.Count(updated, "fiber.my_alloc") != 2 {
		t.Fatalf("expected member accesses to rename, got: %s", updated)
	}
}

func TestTextDocumentRename_function_parameter_renames_in_scope_only(t *testing.T) {
	source := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator) {
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
	blem::Fiber* f2 = blem::create(64 * 1024, null, allocator);
	allocator::free(f1.allocator, f1.stack);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_param_allocator_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "allocator)"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "alloc",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Allocator alloc)") {
		t.Fatalf("expected parameter declaration rename, got: %s", updated)
	}
	if strings.Count(updated, ", alloc)") != 2 {
		t.Fatalf("expected in-scope parameter usage renames, got: %s", updated)
	}
	if strings.Contains(updated, "alloc::free") {
		t.Fatalf("module path should not be renamed for parameter rename, got: %s", updated)
	}
}

func TestTextDocumentRename_example_allocator_parameter_from_realistic_snippet(t *testing.T) {
	source := `module blem::example;

import std::io;
import blem;

tlocal int counter = 0;

fn void worker_a()
{
    counter += 1;
    blem::yield();
}

fn void worker_b()
{
    counter += 10;
    blem::yield();
}

fn void run_demo()
{
    run_demo_with_allocator(mem);
}

fn void run_demo_with_allocator(Allocator allocator)
{
    counter = 0;
    blem::Fiber* f1 = blem::create(64 * 1024, &worker_a, allocator);
    blem::Fiber* f2 = blem::create(64 * 1024, &worker_b, allocator);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_example_allocator_param_test.c3")
	srv := buildRenameTestServer(uri, source)

	paramIdx := strings.Index(source, "Allocator allocator") + len("Allocator ")
	pos := byteIndexToLSPPosition(source, paramIdx)
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "alloc",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Allocator alloc)") {
		t.Fatalf("expected parameter declaration rename, got: %s", updated)
	}
	if strings.Count(updated, ", alloc)") != 2 {
		t.Fatalf("expected both create-call arguments to rename, got: %s", updated)
	}
}

func TestTextDocumentRename_parameter_when_cursor_on_type_token(t *testing.T) {
	source := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator)
{
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
	blem::Fiber* f2 = blem::create(64 * 1024, null, allocator);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_param_cursor_on_type_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "Allocator allocator"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "alloc",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Allocator alloc)") {
		t.Fatalf("expected parameter declaration rename with cursor on type token, got: %s", updated)
	}
	if strings.Count(updated, ", alloc)") != 2 {
		t.Fatalf("expected parameter usages to rename with cursor on type token, got: %s", updated)
	}
}

func TestTextDocumentRename_parameter_updates_param_and_require_doc_contracts(t *testing.T) {
	source := `module app;

<*
 @param a : "Left operand."
 @param b : "Right operand."
 @require a >= 0 && b >= 0 : "Operands are non-negative in this test."
*>
fn int add(int a, int b) {
	return a + b;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_param_doc_contracts_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "int a, int b")+len("int "))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "left",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "@param left : \"Left operand.\"") {
		t.Fatalf("expected @param contract name update, got: %s", updated)
	}
	if !strings.Contains(updated, "@require left >= 0 && b >= 0") {
		t.Fatalf("expected @require expression update, got: %s", updated)
	}
	if !strings.Contains(updated, "fn int add(int left, int b)") {
		t.Fatalf("expected signature parameter rename, got: %s", updated)
	}
	if !strings.Contains(updated, "return left + b;") {
		t.Fatalf("expected function body parameter usage rename, got: %s", updated)
	}
}

func TestTextDocumentRename_parameter_updates_param_contract_with_qualifier(t *testing.T) {
	source := `module app;

<*
 @param [in] a : "Left operand."
 @param [in] b : "Right operand."
 @require a >= 0 && b >= 0 : "Operands are non-negative in this test."
*>
fn int add(int a, int b) {
	return a + b;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_param_doc_qualifier_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "int a, int b")+len("int "))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "left",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "@param [in] left : \"Left operand.\"") {
		t.Fatalf("expected @param qualified contract name update, got: %s", updated)
	}
	if !strings.Contains(updated, "@require left >= 0 && b >= 0") {
		t.Fatalf("expected @require expression update, got: %s", updated)
	}
}

func TestTextDocumentRename_try_bound_variable_in_condition_chain_is_scope_safe(t *testing.T) {
	source := `module app;
fn bool parse_a(String[] args) {
	if (try n = args[0].to_integer(int, 10) && n > 0) return true;
	return false;
}
fn bool parse_b(String[] args) {
	if (try n = args[0].to_integer(int, 10) && n > 0) return true;
	return false;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_try_chain_scope_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "n = args[0]"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "parsed_n",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Count(updated, "parsed_n") != 2 {
		t.Fatalf("expected declaration + chained condition use to rename in first scope, got: %s", updated)
	}
	if strings.Count(updated, "if (try n = args[0].to_integer(int, 10) && n > 0)") != 1 {
		t.Fatalf("expected second function chain binding to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_catch_bound_variable_in_condition_chain_is_scope_safe(t *testing.T) {
	source := `module app;
fn bool read_a() {
	if (catch err = maybe_a() && err != IO_ERROR) return false;
	return true;
}
fn bool read_b() {
	if (catch err = maybe_b() && err != IO_ERROR) return false;
	return true;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_catch_chain_scope_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "err = maybe_a()"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "read_err",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if strings.Count(updated, "read_err") != 2 {
		t.Fatalf("expected declaration + chained condition use to rename in first scope, got: %s", updated)
	}
	if strings.Count(updated, "if (catch err = maybe_b() && err != IO_ERROR)") != 1 {
		t.Fatalf("expected second function chain binding to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_function_from_real_example_snippet(t *testing.T) {
	source := `module blem::example;

fn void run_demo()
{
	run_demo_with_allocator(mem);
}

fn void run_demo_with_allocator(Allocator allocator)
{
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
	blem::Fiber* f2 = blem::create(64 * 1024, null, allocator);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_run_demo_with_allocator_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "run_demo_with_allocator(Allocator"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "run_demo_with_alloc",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "fn void run_demo_with_alloc(Allocator allocator)") {
		t.Fatalf("expected function declaration rename, got: %s", updated)
	}
	if !strings.Contains(updated, "run_demo_with_alloc(mem);") {
		t.Fatalf("expected function call rename, got: %s", updated)
	}
}

func TestTextDocumentPrepareRename_parameter_when_cursor_on_type_token_returns_containing_range(t *testing.T) {
	source := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator)
{
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_param_cursor_on_type_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "Allocator allocator"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename range")
	}

	var rangeResult protocol.Range
	switch typed := result.(type) {
	case protocol.Range:
		rangeResult = typed
	case protocol.RangeWithPlaceholder:
		rangeResult = typed.Range
	default:
		t.Fatalf("expected prepare rename result range")
	}
	if rangeResult.Start.Line > pos.Line || rangeResult.End.Line < pos.Line {
		t.Fatalf("expected prepare range to include cursor line")
	}
}

func TestTextDocumentPrepareRename_function_name_from_realistic_signature(t *testing.T) {
	source := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator)
{
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_function_realistic_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "run_demo_with_allocator"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename range for function")
	}

	withPlaceholder, ok := result.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if withPlaceholder.Placeholder != "run_demo_with_allocator" {
		t.Fatalf("expected function placeholder to contain symbol name, got: %q", withPlaceholder.Placeholder)
	}
}

func TestTextDocumentPrepareRename_function_usage_returns_non_empty_placeholder(t *testing.T) {
	source := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator)
{
}

fn void run_demo()
{
	run_demo_with_allocator(mem);
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_function_usage_placeholder_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "run_demo_with_allocator(mem)")+len("run_"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename result for function usage")
	}

	withPlaceholder, ok := result.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if withPlaceholder.Placeholder != "run_demo_with_allocator" {
		t.Fatalf("expected function usage placeholder to contain symbol name, got: %q", withPlaceholder.Placeholder)
	}
}

func TestTextDocumentPrepareRename_function_declaration_cursor_on_open_paren_uses_function_name_placeholder(t *testing.T) {
	source := `module blem::example;

fn uint parse_port(String s)
{
	return 0;
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_function_decl_open_paren_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port(")+len("parse_port"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename result for function declaration open paren cursor")
	}

	withPlaceholder, ok := result.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if withPlaceholder.Placeholder != "parse_port" {
		t.Fatalf("expected placeholder parse_port when cursor is on open paren, got: %q", withPlaceholder.Placeholder)
	}
}

func TestTextDocumentPrepareRename_parameter_when_cursor_on_closing_paren_prefers_parameter_placeholder(t *testing.T) {
	source := `module blem::example;

fn void main(String[] args)
{
	uint clients = parse_clients(args, 32);
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_param_closing_paren_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "args)")+len("args"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename result for parameter via closing paren cursor")
	}

	withPlaceholder, ok := result.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if withPlaceholder.Placeholder != "args" {
		t.Fatalf("expected parameter placeholder args, got: %q", withPlaceholder.Placeholder)
	}
}

func TestTextDocumentPrepareRename_thread_main_signature_closing_paren_returns_args(t *testing.T) {
	source := `module app;

fn uint parse_clients(String[] args, uint default_value = 32)
{
	if (args.len < 2) return default_value;
	return default_value;
}

fn uint parse_port(String[] args, uint default_value = 19080)
{
	if (args.len < 3) return default_value;
	return default_value;
}

fn void main(String[] args)
{
	uint clients = parse_clients(args, 32);
	uint port = parse_port(args, 19080);
	io::printfn("clients=%d, port=%d", clients, port);
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_thread_main_closing_paren_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "main(String[] args)")+len("main(String[] args"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename result for main signature closing paren")
	}

	withPlaceholder, ok := result.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if withPlaceholder.Placeholder != "args" {
		t.Fatalf("expected parameter placeholder args, got: %q", withPlaceholder.Placeholder)
	}
}

func TestTextDocumentPrepareRename_parameter_when_cursor_on_comma_prefers_left_parameter_placeholder(t *testing.T) {
	source := `module app;
fn uint parse_clients(String[] args, uint default_value = 32)
{
	if (args.len < 2) return default_value;
	return default_value;
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_param_comma_test.c3")
	srv := buildRenameTestServer(uri, source)

	commaPos := byteIndexToLSPPosition(source, strings.Index(source, "args, uint")+len("args"))
	result, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     commaPos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepare rename error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected prepare rename result for comma cursor")
	}

	withPlaceholder, ok := result.(protocol.RangeWithPlaceholder)
	if !ok {
		t.Fatalf("expected prepare rename range with placeholder")
	}
	if withPlaceholder.Placeholder != "args" {
		t.Fatalf("expected parameter placeholder args when cursor is on comma, got: %q", withPlaceholder.Placeholder)
	}
}

func TestTextDocumentPrepareRename_parameter_when_cursor_on_comma_returns_without_hang(t *testing.T) {
	source := `module app;
fn uint parse_clients(String[] args, uint default_value = 32)
{
	if (args.len < 2) return default_value;
	return default_value;
}`

	uri := protocol.DocumentUri("file:///tmp/prepare_rename_param_comma_no_hang_test.c3")
	srv := buildRenameTestServer(uri, source)

	commaPos := byteIndexToLSPPosition(source, strings.Index(source, "args, uint")+len("args"))
	done := make(chan error, 1)

	go func() {
		_, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     commaPos,
			},
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected prepare rename error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("prepare rename appears to hang for comma cursor")
	}
}

func TestTextDocumentRename_parameter_when_cursor_on_comma_renames_left_parameter(t *testing.T) {
	source := `module app;
fn uint parse_clients(String[] args, uint default_value = 32)
{
	if (args.len < 2) return default_value;
	if (try n = args[1].to_integer(uint, 10))
	{
		if (n > 0) return n;
	}
	return default_value;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_param_comma_left_preference_test.c3")
	srv := buildRenameTestServer(uri, source)

	commaPos := byteIndexToLSPPosition(source, strings.Index(source, "args, uint")+len("args"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     commaPos,
		},
		NewName: "argv",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "fn uint parse_clients(String[] argv, uint default_value = 32)") {
		t.Fatalf("expected left parameter declaration rename, got: %s", updated)
	}
	if !strings.Contains(updated, "if (argv.len < 2) return default_value;") {
		t.Fatalf("expected left parameter usages rename, got: %s", updated)
	}
	if strings.Contains(updated, "uint argv = 32") {
		t.Fatalf("did not expect right parameter to be renamed, got: %s", updated)
	}
}

func TestTextDocumentRename_fault_constant_renames_qualified_usage_in_same_file(t *testing.T) {
	source := `module blem::net;

faultdef TIMEOUT;

fn bool timed_out(fault err) {
	return err == blem::net::TIMEOUT;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_fault_timeout_same_file_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "TIMEOUT;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "DEADLINE",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "faultdef DEADLINE;") {
		t.Fatalf("expected fault constant declaration rename, got: %s", updated)
	}
	if !strings.Contains(updated, "blem::net::DEADLINE") {
		t.Fatalf("expected qualified fault usage rename, got: %s", updated)
	}
}

func TestTextDocumentRename_enum_constant_renames_qualified_usage_across_files(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_enum_state_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_enum_state_use_test.c3")

	declSource := `module blem::status;

enum State {
	OPEN,
	CLOSED,
}`

	useSource := `module blem::http;

fn bool is_closed(blem::status::State state) {
	return state == blem::status::State.CLOSED;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "CLOSED,"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		NewName: "DONE",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "DONE,") {
		t.Fatalf("expected enum declaration rename, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if !strings.Contains(updatedUse, "blem::status::State.DONE") {
		t.Fatalf("expected qualified enum usage rename, got: %s", updatedUse)
	}
}

func TestTextDocumentRename_fault_constant_renames_qualified_usage_across_files(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_fault_timeout_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_fault_timeout_use_test.c3")

	declSource := `module blem::net;

faultdef TIMEOUT;`

	useSource := `module blem::http;

import blem::net;

fn bool timed_out(fault err) {
	return err == blem::net::TIMEOUT;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "TIMEOUT;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		NewName: "DEADLINE",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "faultdef DEADLINE;") {
		t.Fatalf("expected declaration rename in declaration file, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if !strings.Contains(updatedUse, "blem::net::DEADLINE") {
		t.Fatalf("expected qualified usage rename in usage file, got: %s", updatedUse)
	}
}

func TestTextDocumentRename_fault_constant_renames_qualified_usage_across_files_without_import(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_fault_timeout_decl_no_import_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_fault_timeout_use_no_import_test.c3")

	declSource := `module blem::net;

faultdef TIMEOUT;`

	useSource := `module blem::http;

fn bool timed_out(fault err) {
	if (err == blem::net::TIMEOUT) return true;
	if (err == blem::net::TIMEOUT) return true;
	return false;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "TIMEOUT;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		NewName: "DEADLINE",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "faultdef DEADLINE;") {
		t.Fatalf("expected declaration rename in declaration file, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if strings.Count(updatedUse, "blem::net::DEADLINE") != 2 {
		t.Fatalf("expected both qualified usages rename in usage file, got: %s", updatedUse)
	}
}

func TestTextDocumentRename_struct_member_accepts_qualified_new_name(t *testing.T) {
	source := `module blem;

alias Coroutine = fn void();

struct Fiber {
	Coroutine entry;
}

fn void invoke(Fiber* fiber) {
	fiber.entry();
}`

	uri := protocol.DocumentUri("file:///tmp/rename_member_qualified_new_name_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "entry;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "blem::nettted",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Coroutine nettted;") {
		t.Fatalf("expected member declaration rename using tail token, got: %s", updated)
	}
	if !strings.Contains(updated, "fiber.nettted();") {
		t.Fatalf("expected member usage rename using tail token, got: %s", updated)
	}
}

func TestTextDocumentRename_rejects_qualified_new_name_with_invalid_tail(t *testing.T) {
	source := `module blem;

alias Coroutine = fn void();

struct Fiber {
	Coroutine entry;
}`

	uri := protocol.DocumentUri("file:///tmp/rename_member_qualified_invalid_tail_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "entry;"))
	_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "blem::123bad",
	})

	if err == nil {
		t.Fatalf("expected invalid qualified identifier rename to fail")
	}
}

func TestTextDocumentRename_allocator_member_in_conditional_fiber_variants(t *testing.T) {
	source := `module blem @if(env::POSIX);

alias Coroutine = fn void();

struct Fiber
{
	Coroutine entry;
	void* stack;
	Allocator allocator;
	bool done;
}

fn void delete(Fiber* fiber)
{
	if (fiber == null) return;
	allocator::free(fiber.allocator, fiber.stack);
	allocator::free(fiber.allocator, fiber);
}

module blem @if(env::WIN32);

alias WinFiber = void*;
alias Coroutine = fn void();

struct Fiber
{
	WinFiber wfiber;
	Allocator allocator;
	bool done;
}

fn void delete(Fiber* fiber)
{
	if (fiber == null) return;
	allocator::free(fiber.allocator, fiber);
}

module blem @if(!env::POSIX && !env::WIN32);

alias Coroutine = fn void();

struct Fiber
{
	Allocator allocator;
	bool done;
}
`

	uri := protocol.DocumentUri("file:///tmp/rename_allocator_member_conditional_fiber_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "Allocator allocator;"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "allocx",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Allocator allocx;") {
		t.Fatalf("expected allocator declaration rename at target site, got: %s", updated)
	}
	if strings.Count(updated, "fiber.allocx") != 3 {
		t.Fatalf("expected allocator member accesses to rename, got: %s", updated)
	}
	if strings.Contains(updated, "allocx::free") {
		t.Fatalf("module path token should not be renamed, got: %s", updated)
	}
}

func TestTextDocumentRename_fiber_allocator_member_from_usage_with_param_name_collision(t *testing.T) {
	source := `module blem;

alias Coroutine = fn void();

struct Fiber
{
	Coroutine entry;
	Allocator allocator;
	bool done;
}

fn Fiber* create(Coroutine entry, Allocator allocator = mem)
{
	Fiber* fiber = (Fiber*)allocator::malloc(allocator, Fiber.sizeof);
	if (fiber == null) return null;
	fiber.entry = entry;
	fiber.allocator = allocator;
	return fiber;
}

fn void delete(Fiber* fiber)
{
	if (fiber == null) return;
	allocator::free(fiber.allocator, fiber);
}`

	uri := protocol.DocumentUri("file:///tmp/rename_fiber_allocator_usage_collision_test.c3")
	srv := buildRenameTestServer(uri, source)

	usageIdx := strings.Index(source, "fiber.allocator =") + len("fiber.")
	pos := byteIndexToLSPPosition(source, usageIdx)
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "allocx",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "Allocator allocx;") {
		t.Fatalf("expected member declaration rename, got: %s", updated)
	}
	if strings.Count(updated, "fiber.allocx") != 2 {
		t.Fatalf("expected member usages to rename, got: %s", updated)
	}
	if strings.Contains(updated, "Allocator allocx = mem") {
		t.Fatalf("expected function parameter to remain unchanged, got: %s", updated)
	}
	if strings.Contains(updated, "allocx::free") {
		t.Fatalf("expected module path token to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_fiber_done_member_does_not_conflict_with_done_function(t *testing.T) {
	source := `module blem;

alias Coroutine = fn void();

struct Fiber
{
	Coroutine entry;
	bool done;
}

tlocal Fiber primary;
tlocal Fiber* running = null;

fn void switch_to(Fiber* fiber)
{
	if (fiber == null || fiber.done || fiber == running) return;
	running = fiber;
}

fn void done()
{
	running.done = true;
}
`

	uri := protocol.DocumentUri("file:///tmp/rename_fiber_done_member_collision_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bool done;")+len("bool "))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "completed",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "bool completed;") {
		t.Fatalf("expected member declaration rename, got: %s", updated)
	}
	if strings.Count(updated, ".completed") != 2 {
		t.Fatalf("expected member usages to rename, got: %s", updated)
	}
	if strings.Contains(updated, "fn void completed()") {
		t.Fatalf("expected function done() to remain unchanged, got: %s", updated)
	}
}

func TestTextDocumentRename_fiber_done_member_across_conditional_modules_keeps_done_functions(t *testing.T) {
	source := `module blem @if(env::POSIX);

struct Fiber
{
	bool done;
}

tlocal Fiber* running = null;

fn void switch_to(Fiber* fiber)
{
	if (fiber == null || fiber.done || fiber == running) return;
	running = fiber;
}

fn void done()
{
	running.done = true;
}

module blem @if(env::WIN32);

struct Fiber
{
	bool done;
}

tlocal Fiber* running = null;

fn void switch_to(Fiber* fiber)
{
	if (fiber == null || fiber.done || fiber == running) return;
	running = fiber;
}

fn void done()
{
	running.done = true;
}

module blem @if(!env::POSIX && !env::WIN32);

struct Fiber
{
	bool done;
}

tlocal Fiber* running = null;

fn void switch_to(Fiber* fiber)
{
	if (fiber == null || fiber.done || fiber == running) return;
	running = fiber;
}

fn void done()
{
	running.done = true;
}
`

	uri := protocol.DocumentUri("file:///tmp/rename_fiber_done_member_conditional_collision_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bool done;")+len("bool "))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: "completed",
	})
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if edit == nil {
		t.Fatalf("expected workspace edit")
	}

	updated := applyTextEdits(source, edit.Changes[uri])
	if !strings.Contains(updated, "bool completed;") {
		t.Fatalf("expected member declaration at target site to rename, got: %s", updated)
	}
	if strings.Count(updated, ".completed") != 6 {
		t.Fatalf("expected all Fiber.done usages to rename, got: %s", updated)
	}
	if strings.Contains(updated, "fn void completed()") {
		t.Fatalf("expected done() functions to remain unchanged, got: %s", updated)
	}
}

func buildRenameTestServer(uri protocol.DocumentUri, source string) *Server {
	return buildRenameTestServerWithDocuments([]renameTestDocument{{uri: uri, source: source}})
}

type renameTestDocument struct {
	uri    protocol.DocumentUri
	source string
}

func buildRenameTestServerWithDocuments(docs []renameTestDocument) *Server {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	for _, docInput := range docs {
		doc := document.NewDocumentFromDocURI(docInput.uri, docInput.source, 1)
		state.RefreshDocumentIdentifiers(doc, &prs)
	}

	return &Server{state: &state, parser: &prs, search: &searchImpl}
}
