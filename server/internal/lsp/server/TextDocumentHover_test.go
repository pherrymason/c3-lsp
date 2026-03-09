package server

import (
	"strings"
	"testing"

	lspcontext "github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type nilSymbolSearchMock struct{}

type fixedSymbolSearchMock struct {
	symbol symbols.Indexable
}

func (nilSymbolSearchMock) FindSymbolDeclarationInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) option.Option[symbols.Indexable] {
	var variable *symbols.Variable
	var indexable symbols.Indexable = variable
	return option.Some(indexable)
}

func (m fixedSymbolSearchMock) FindSymbolDeclarationInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) option.Option[symbols.Indexable] {
	return option.Some(m.symbol)
}

func (m fixedSymbolSearchMock) FindImplementationsInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) []symbols.Indexable {
	return nil
}

func (m fixedSymbolSearchMock) FindReferencesInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState, _ bool) []protocol.Location {
	return nil
}

func (m fixedSymbolSearchMock) BuildCompletionList(_ lspcontext.CursorContext, _ *project_state.ProjectState) []protocol.CompletionItem {
	return nil
}

func (nilSymbolSearchMock) FindImplementationsInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) []symbols.Indexable {
	return nil
}

func (nilSymbolSearchMock) FindReferencesInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState, _ bool) []protocol.Location {
	return nil
}

func (nilSymbolSearchMock) BuildCompletionList(_ lspcontext.CursorContext, _ *project_state.ProjectState) []protocol.CompletionItem {
	return nil
}

func TestGenericTypeSuffixAtPosition(t *testing.T) {
	t.Run("simple generic", func(t *testing.T) {
		source := "List{int} l;"
		suffix, ok := genericTypeSuffixAtPosition(source, symbols.NewPositionFromLSPPosition(protocol.Position{Line: 0, Character: 1}))
		if !ok {
			t.Fatalf("expected generic suffix to be found")
		}
		if suffix != "{int}" {
			t.Fatalf("unexpected suffix: got %q", suffix)
		}
	})

	t.Run("nested generic with spacing", func(t *testing.T) {
		source := "HashMap { String, List{int} } m;"
		suffix, ok := genericTypeSuffixAtPosition(source, symbols.NewPositionFromLSPPosition(protocol.Position{Line: 0, Character: 2}))
		if !ok {
			t.Fatalf("expected generic suffix to be found")
		}
		if suffix != "{ String, List{int} }" {
			t.Fatalf("unexpected suffix: got %q", suffix)
		}
	})

	t.Run("not a generic type usage", func(t *testing.T) {
		source := "List l;"
		_, ok := genericTypeSuffixAtPosition(source, symbols.NewPositionFromLSPPosition(protocol.Position{Line: 0, Character: 1}))
		if ok {
			t.Fatalf("expected no generic suffix")
		}
	})
}

func TestTextDocumentHover_includes_generic_arguments_for_hovered_type_identifier(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
	struct List {
		int value;
	}

	fn void main() {
		List{int} list;
	}`

	doc := document.NewDocumentFromDocURI("file:///tmp/hover_generic_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/hover_generic_test.c3"},
			Position: protocol.Position{
				Line:      6,
				Character: 4,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "List{int}") {
		t.Fatalf("expected hover to include generic type arguments, got: %s", content.Value)
	}
}

func TestTextDocumentHover_returns_nil_inside_string_literal(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
alias s = int;
fn void main() {
	io::printfn("%s", 1);
}`

	doc := document.NewDocumentFromDocURI("file:///tmp/hover_literal_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/hover_literal_test.c3"},
			Position: protocol.Position{
				Line:      3,
				Character: 15,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover != nil {
		t.Fatalf("expected nil hover inside string literal, got: %#v", hover)
	}
}

func TestTextDocumentHover_displays_friendly_anonymous_module_name(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	uri := protocol.DocumentUri("file:///tmp/vulkan.c3")
	source := "int value = 1;\n"
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position: protocol.Position{
				Line:      0,
				Character: 5,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "In module **[vulkan (anon#") {
		t.Fatalf("expected friendly anonymous module label, got: %s", content.Value)
	}
}

func TestTextDocumentHover_returns_nil_when_search_returns_typed_nil_symbol(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: nilSymbolSearchMock{},
	}

	source := `module app;
fn void main() {
	int thread;
}`

	doc := document.NewDocumentFromDocURI("file:///tmp/hover_nil_symbol_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/hover_nil_symbol_test.c3"},
			Position: protocol.Position{
				Line:      2,
				Character: 5,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover != nil {
		t.Fatalf("expected nil hover for typed nil search result, got: %#v", hover)
	}
}

func TestTextDocumentHover_resolves_method_hover_from_nested_block_local_variable(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Value {
	int inner;
}

fn bool Value.to_bool(self) {
	return true;
}

fn void main() {
	if (true) {
		Value val = {};
		val.to_bool();
	}
}`

	doc := document.NewDocumentFromDocURI("file:///tmp/hover_nested_local_method_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/hover_nested_local_method_test.c3"},
			Position: protocol.Position{
				Line:      12,
				Character: 8,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Value.to_bool") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_method_hover_from_indexed_receiver(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Value {
	int inner;
}

fn bool Value.to_bool(self) {
	return true;
}

fn void main() {
	Value[4] values;
	values[0].to_bool();
}`

	doc := document.NewDocumentFromDocURI("file:///tmp/hover_indexed_receiver_method_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/hover_indexed_receiver_method_test.c3"},
			Position: protocol.Position{
				Line:      11,
				Character: 13,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Value.to_bool") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_member_hover_with_non_self_receiver_name(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Route {
	int id;
}

struct Router {
	List{Route} routes;
}

fn void Router.free(&router) {
	router.routes.free();
}`

	uri := protocol.DocumentUri("file:///tmp/hover_non_self_receiver_member_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "router.routes") + len("router.")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "routes") {
		t.Fatalf("expected hover to resolve routes member, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_foreach_iterator_variable(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Route {
	String method;
}

struct Router {
	List{Route} routes;
}

fn Handler Router.match(&router, String method, String path)
{
	foreach (route : router.routes)
	{
		if (route.method == method) return null;
	}
	return null;
}`

	uri := protocol.DocumentUri("file:///tmp/hover_foreach_iterator_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "route :") + 1
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for foreach iterator")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "route") {
		t.Fatalf("expected hover to resolve foreach iterator variable, got: %s", content.Value)
	}

	memberIdx := strings.Index(source, "route.method") + len("route.")
	memberPos := byteIndexToLSPPosition(source, memberIdx)

	memberHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     memberPos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected member hover error: %v", err)
	}
	if memberHover == nil {
		t.Fatalf("expected hover response for foreach member access")
	}

	memberContent, ok := memberHover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in member hover")
	}

	if !strings.Contains(memberContent.Value, "method") {
		t.Fatalf("expected hover to resolve foreach member access, got: %s", memberContent.Value)
	}
}

func TestTextDocumentHover_resolves_foreach_index_variable(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Route { String method; }

fn void run(Route[] routes)
{
	foreach (i, route : routes)
	{
		if (i > 0 && route.method.len > 0) return;
	}
}`

	uri := protocol.DocumentUri("file:///tmp/hover_foreach_index_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "i, route")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for foreach index variable")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "usz i") {
		t.Fatalf("expected hover to resolve foreach index as usz, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_method_hover_through_void_cast_expression(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Socket {}

fn bool Socket.close(self) {
	return true;
}
struct Client {
	Socket socket;
}

fn void cleanup(Client* client)
{
	(void)client.socket.close();
}`

	uri := protocol.DocumentUri("file:///tmp/hover_void_cast_method_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "client.socket.close") + len("client.socket.")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}

	middlePos := byteIndexToLSPPosition(source, idx+3)
	middleHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     middlePos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error on middle method char: %v", err)
	}
	if middleHover == nil {
		t.Fatalf("expected hover response on middle method char")
	}
}

func TestTextDocumentHover_resolves_method_hover_through_defer_catch_void_cast_expression(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Socket {}

fn bool Socket.close(self) {
	return true;
}

struct Client {
	Socket socket;
}

fn void cleanup(Client* client)
{
	defer catch (void)client.socket.close();
}`

	uri := protocol.DocumentUri("file:///tmp/hover_defer_catch_void_cast_method_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "client.socket.close") + len("client.socket.")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}

	lastCharIdx := strings.Index(source, "client.socket.close") + len("client.socket.close") - 1
	lastCharPos := byteIndexToLSPPosition(source, lastCharIdx)

	lastCharHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     lastCharPos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error on last method char: %v", err)
	}
	if lastCharHover == nil {
		t.Fatalf("expected hover response on last method char")
	}
}

func TestTextDocumentHover_resolves_method_hover_on_inline_distinct_member(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
struct Socket {}
typedef TcpSocket = inline Socket;

fn bool Socket.close(self) {
	return true;
}

struct Client {
	TcpSocket socket;
}

fn void cleanup(Client* client)
{
	(void)client.socket.close();
}`

	uri := protocol.DocumentUri("file:///tmp/hover_inline_distinct_member_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "client.socket.close") + len("client.socket.")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_method_hover_on_inline_distinct_member_across_imported_module(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	netSource := `module std::net;
struct Socket {}

fn bool Socket.close(self) {
	return true;
}`
	tcpSource := `module std::net::tcp;
import std::net;
typedef TcpSocket = inline Socket;`
	appSource := `module app;
import std::net::tcp;

struct Client {
	tcp::TcpSocket socket;
}

fn void cleanup(Client* client)
{
	(void)client.socket.close();
}`

	netURI := protocol.DocumentUri("file:///tmp/hover_inline_distinct_import_net.c3")
	tcpURI := protocol.DocumentUri("file:///tmp/hover_inline_distinct_import_tcp.c3")
	appURI := protocol.DocumentUri("file:///tmp/hover_inline_distinct_import_app.c3")

	netDoc := document.NewDocumentFromDocURI(netURI, netSource, 1)
	tcpDoc := document.NewDocumentFromDocURI(tcpURI, tcpSource, 1)
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)

	state.RefreshDocumentIdentifiers(netDoc, &prs)
	state.RefreshDocumentIdentifiers(tcpDoc, &prs)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "client.socket.close") + len("client.socket.")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_method_hover_on_inline_distinct_member_across_imported_module_with_defer_catch(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	netSource := `module std::net;
struct Socket {}

fn void? Socket.close(&self) {
	return;
}`
	tcpSource := `module std::net::tcp;
import std::net;
typedef TcpSocket = inline Socket;`
	appSource := `module app;
import std::net::tcp;

struct Client {
	tcp::TcpSocket socket;
}

fn void cleanup(Client* client)
{
	defer catch (void)client.socket.close();
}`

	netURI := protocol.DocumentUri("file:///tmp/hover_defer_import_net.c3")
	tcpURI := protocol.DocumentUri("file:///tmp/hover_defer_import_tcp.c3")
	appURI := protocol.DocumentUri("file:///tmp/hover_defer_import_app.c3")

	netDoc := document.NewDocumentFromDocURI(netURI, netSource, 1)
	tcpDoc := document.NewDocumentFromDocURI(tcpURI, tcpSource, 1)
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)

	state.RefreshDocumentIdentifiers(netDoc, &prs)
	state.RefreshDocumentIdentifiers(tcpDoc, &prs)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "client.socket.close") + len("client.socket.")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}

	middlePos := byteIndexToLSPPosition(appSource, idx+3)
	middleHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     middlePos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error on middle method char: %v", err)
	}
	if middleHover == nil {
		t.Fatalf("expected hover response on middle method char")
	}
}

func TestTextDocumentHover_resolves_method_hover_on_inline_distinct_local_variable_with_void_cast(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	netSource := `module std::net;
struct Socket {}

fn void? Socket.close(&self) {
	return;
}`
	tcpSource := `module std::net::tcp;
import std::net;
typedef TcpSocket = inline Socket;`
	appSource := `module app;
import std::net::tcp;

fn void cleanup()
{
	tcp::TcpSocket socket;
	(void)socket.close();
}`

	netURI := protocol.DocumentUri("file:///tmp/hover_local_var_net.c3")
	tcpURI := protocol.DocumentUri("file:///tmp/hover_local_var_tcp.c3")
	appURI := protocol.DocumentUri("file:///tmp/hover_local_var_app.c3")

	netDoc := document.NewDocumentFromDocURI(netURI, netSource, 1)
	tcpDoc := document.NewDocumentFromDocURI(tcpURI, tcpSource, 1)
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)

	state.RefreshDocumentIdentifiers(netDoc, &prs)
	state.RefreshDocumentIdentifiers(tcpDoc, &prs)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "socket.close") + len("socket.")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_method_hover_on_inline_distinct_struct_member_with_void_cast(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	netSource := `module std::net;
struct Socket {}

fn void? Socket.close(&self) {
	return;
}`
	tcpSource := `module std::net::tcp;
import std::net;
typedef TcpSocket = inline Socket;`
	appSource := `module app;
import std::net::tcp;

struct ClientState {
	tcp::TcpSocket socket;
}

fn void client_free(ClientState* client)
{
	(void)client.socket.close();
}`

	netURI := protocol.DocumentUri("file:///tmp/hover_struct_member_net.c3")
	tcpURI := protocol.DocumentUri("file:///tmp/hover_struct_member_tcp.c3")
	appURI := protocol.DocumentUri("file:///tmp/hover_struct_member_app.c3")

	netDoc := document.NewDocumentFromDocURI(netURI, netSource, 1)
	tcpDoc := document.NewDocumentFromDocURI(tcpURI, tcpSource, 1)
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)

	state.RefreshDocumentIdentifiers(netDoc, &prs)
	state.RefreshDocumentIdentifiers(tcpDoc, &prs)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "client.socket.close") + len("client.socket.")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_method_hover_on_inline_distinct_local_struct_value_with_defer_catch(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	netSource := `module std::net;
struct Socket {}

fn void? Socket.close(&self) {
	return;
}`
	tcpSource := `module std::net::tcp;
import std::net;
typedef TcpSocket = inline Socket;`
	appSource := `module app;
import std::net::tcp;

struct ClientState {
	tcp::TcpSocket socket;
}

fn void run()
{
	ClientState client;
	defer catch (void)client.socket.close();
}`

	netURI := protocol.DocumentUri("file:///tmp/hover_local_struct_value_net.c3")
	tcpURI := protocol.DocumentUri("file:///tmp/hover_local_struct_value_tcp.c3")
	appURI := protocol.DocumentUri("file:///tmp/hover_local_struct_value_app.c3")

	netDoc := document.NewDocumentFromDocURI(netURI, netSource, 1)
	tcpDoc := document.NewDocumentFromDocURI(tcpURI, tcpSource, 1)
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)

	state.RefreshDocumentIdentifiers(netDoc, &prs)
	state.RefreshDocumentIdentifiers(tcpDoc, &prs)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "client.socket.close") + len("client.socket.") + 3
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "Socket.close") {
		t.Fatalf("expected hover to contain method signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_returns_builtin_collection_len_hover(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
fn uint parse_clients(String[] args, uint default_value = 32)
{
	if (args.len < 2) return default_value;
	return default_value;
}`

	uri := protocol.DocumentUri("file:///tmp/hover_collection_len_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "args.len") + len("args.")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for collection len")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "usz len") {
		t.Fatalf("expected len hover signature, got: %s", content.Value)
	}
}

func TestTextDocumentHover_len_on_non_collection_returns_nil(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
fn void main()
{
	int value = 1;
	value.len;
}`

	uri := protocol.DocumentUri("file:///tmp/hover_non_collection_len_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "value.len") + len("value.")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover != nil {
		t.Fatalf("expected nil hover for non-collection len, got: %#v", hover)
	}
}

func TestTextDocumentHover_try_binding_infers_type_in_hover(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
fn uint parse_clients(String[] args, uint default_value = 32)
{
	if (try n = args[1].to_integer(uint, 10))
	{
		if (n > 0) return n;
	}
	return default_value;
}`

	uri := protocol.DocumentUri("file:///tmp/hover_try_binding_type_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "try n =") + len("try ")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for try binding")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}

	if !strings.Contains(content.Value, "uint n") {
		t.Fatalf("expected inferred try-binding type in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_try_binding_from_optional_variable_infers_unwrapped_type(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
fn void test()
{
	int? maybe_value = 1;
	if (try accepted = maybe_value)
	{
		(void)accepted;
	}
}`

	uri := protocol.DocumentUri("file:///tmp/hover_try_binding_optional_value_type_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "try accepted =") + len("try ")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for try binding")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}

	if !strings.Contains(content.Value, "int accepted") {
		t.Fatalf("expected inferred try-binding type from optional variable in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_catch_binding_infers_type_in_hover(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
fn uint parse_port(String[] args, uint default_value = 19080)
{
	if (catch reason = args[1].to_integer(uint, 10))
	{
		return default_value;
	}
	return default_value;
}`

	uri := protocol.DocumentUri("file:///tmp/hover_catch_binding_type_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "catch reason =") + len("catch ")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for catch binding")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}

	if !strings.Contains(content.Value, "fault reason") {
		t.Fatalf("expected catch-binding fault type in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_catch_binding_from_optional_variable_is_fault_typed(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
fn void run()
{
	ulong? poll_result = 0;
	if (catch reason = poll_result)
	{
		(void)reason;
	}
}`

	uri := protocol.DocumentUri("file:///tmp/hover_catch_binding_optional_value_type_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "catch reason =") + len("catch ")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for catch binding")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}

	if !strings.Contains(content.Value, "fault reason") {
		t.Fatalf("expected catch-binding fault type in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_module_separator_prefers_symbol_after_separator(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	mathSource := `module std::math;
macro log(x, base) {}`
	mathURI := protocol.DocumentUri("file:///tmp/hover_module_sep_math.c3")
	mathDoc := document.NewDocumentFromDocURI(mathURI, mathSource, 1)
	state.RefreshDocumentIdentifiers(mathDoc, &prs)

	logSource := `module std::core::log;
macro void error(String fmt, ...) {}`
	logURI := protocol.DocumentUri("file:///tmp/hover_module_sep_log.c3")
	logDoc := document.NewDocumentFromDocURI(logURI, logSource, 1)
	state.RefreshDocumentIdentifiers(logDoc, &prs)

	appSource := `module app;
import std::core::log;
fn void main() {
	log::error("oops");
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_module_sep_app.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "log::error") + len("log")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover on module separator")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}

	if !strings.Contains(content.Value, "error(") {
		t.Fatalf("expected hover to resolve symbol after module separator, got: %s", content.Value)
	}
	if !strings.Contains(content.Value, "In module **[std::core::log]**") {
		t.Fatalf("expected hover module std::core::log, got: %s", content.Value)
	}
}

func TestTextDocumentHover_constant_includes_const_keyword_and_value(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
const String SQL_CREATE_TABLE = "CREATE TABLE todos";

fn void main() {
	io::printfn("%s", SQL_CREATE_TABLE);
}`

	uri := protocol.DocumentUri("file:///tmp/hover_const_keyword_value_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "SQL_CREATE_TABLE);") + 2
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for constant")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}

	if !strings.Contains(content.Value, "const String SQL_CREATE_TABLE = \"CREATE TABLE todos\"") {
		t.Fatalf("expected const keyword/type/value in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_constant_fallback_extracts_value_from_source_when_missing_on_symbol(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	source := `module app;
const String SQL_CREATE_TABLE = "CREATE TABLE todos";
fn void main() {}`
	uri := protocol.DocumentUri("file:///tmp/hover_const_fallback_value_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	declStart := strings.Index(source, "const String SQL_CREATE_TABLE")
	declEnd := strings.Index(source[declStart:], ";") + declStart + 1
	nameStart := strings.Index(source, "SQL_CREATE_TABLE")
	nameEnd := nameStart + len("SQL_CREATE_TABLE")
	nameStartPos := byteIndexToLSPPosition(source, nameStart)
	nameEndPos := byteIndexToLSPPosition(source, nameEnd)
	declStartPos := byteIndexToLSPPosition(source, declStart)
	declEndPos := byteIndexToLSPPosition(source, declEnd)

	constant := symbols.NewConstant(
		"SQL_CREATE_TABLE",
		symbols.NewTypeFromString("String", "app"),
		"app",
		string(uri),
		symbols.NewRange(
			uint(nameStartPos.Line),
			uint(nameStartPos.Character),
			uint(nameEndPos.Line),
			uint(nameEndPos.Character),
		),
		symbols.NewRange(
			uint(declStartPos.Line),
			uint(declStartPos.Character),
			uint(declEndPos.Line),
			uint(declEndPos.Character),
		),
	)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: fixedSymbolSearchMock{symbol: &constant},
	}

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position: protocol.Position{
				Line:      1,
				Character: 14,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}

	if !strings.Contains(content.Value, "const String SQL_CREATE_TABLE = \"CREATE TABLE todos\"") {
		t.Fatalf("expected fallback const value extraction in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_constdef_member_includes_assigned_value(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
constdef SqliteResult : int {
	ROW = 100,
	DONE = 101,
}

fn void main() {
	if (SqliteResult.ROW == SqliteResult.DONE) {}
}`

	uri := protocol.DocumentUri("file:///tmp/hover_constdef_member_value_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.LastIndex(source, "ROW")
	pos := byteIndexToLSPPosition(source, idx+1)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "enum SqliteResult.ROW = 100") {
		t.Fatalf("expected constdef member hover value, got: %s", content.Value)
	}
}

func TestTextDocumentHover_enum_associated_member_includes_member_values(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
enum State : int (String state_desc, bool active)
{
	PENDING = {"pending start", false},
	RUNNING = {"running", true},
}

fn void main() {
	State value = State.PENDING;
}`

	uri := protocol.DocumentUri("file:///tmp/hover_enum_associated_values_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.LastIndex(source, "PENDING")
	pos := byteIndexToLSPPosition(source, idx+2)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "enum State.PENDING {String state_desc: \"pending start\", bool active: false}") {
		t.Fatalf("expected enum associated member hover details, got: %s", content.Value)
	}
}

func TestTextDocumentHover_try_binding_infers_type_from_function_return(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
fn sqlite3::SqliteStmt? prepare_stmt() {
	return null;
}
fn void main() {
	if (try s = prepare_stmt()) {
		(void)s;
	}
}`

	uri := protocol.DocumentUri("file:///tmp/hover_try_function_return_type_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "try s =") + len("try ")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for try-bound variable")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "sqlite3::SqliteStmt s") {
		t.Fatalf("expected inferred type from function return, got: %s", content.Value)
	}
}

func TestTextDocumentHover_try_binding_infers_type_from_method_return(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
struct Object {}

fn bool? Object.get_bool(Object* self, String key) {
	return true;
}

fn void main() {
	Object body;
	if (try value = body.get_bool("done")) {
		(void)value;
	}
}`

	uri := protocol.DocumentUri("file:///tmp/hover_try_method_return_type_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "try value =") + len("try ")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for try-bound method return variable")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "bool value") {
		t.Fatalf("expected inferred type from method return, got: %s", content.Value)
	}
}

func TestTextDocumentHover_optional_function_includes_doc_faults_section(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
struct Object {}
faultdef TYPE_MISMATCH;
<*
 @return ? TYPE_MISMATCH : "type mismatch"
*>
fn String? Object.get_string(self, String key) {
	return TYPE_MISMATCH~;
}

fn void main() {
	Object body = {};
	body.get_string("task");
}`

	uri := protocol.DocumentUri("file:///tmp/hover_optional_doc_faults_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "body.get_string") + len("body.") + 2
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "@faults: app::TYPE_MISMATCH") {
		t.Fatalf("expected doc fault list in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_optional_function_includes_inferred_faults_section(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
struct Object {}
faultdef TYPE_MISMATCH;

fn String? Object.get_string(self, String key) {
	return TYPE_MISMATCH~;
}

fn void main() {
	Object body = {};
	body.get_string("task");
}`

	uri := protocol.DocumentUri("file:///tmp/hover_optional_inferred_faults_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "body.get_string") + len("body.") + 2
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "@faults (inferred): app::TYPE_MISMATCH") {
		t.Fatalf("expected inferred fault list in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_optional_function_supports_return_question_contract_name(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
struct Object {}
faultdef TYPE_MISMATCH;
<*
 @return? TYPE_MISMATCH : "type mismatch"
*>
fn String? Object.get_string(self, String key) {
	return TYPE_MISMATCH~;
}

fn void main() {
	Object body = {};
	body.get_string("task");
}`

	uri := protocol.DocumentUri("file:///tmp/hover_optional_returnq_contract_faults_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "body.get_string") + len("body.") + 2
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "@faults: app::TYPE_MISMATCH") {
		t.Fatalf("expected @return? contract fault list in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_optional_function_inferred_faults_support_builtin_qualification(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	builtinSource := `module std::core::builtin;
faultdef TYPE_MISMATCH;`
	builtinURI := protocol.DocumentUri("file:///tmp/hover_builtin_fault_module_test.c3")
	builtinDoc := document.NewDocumentFromDocURI(builtinURI, builtinSource, 1)
	state.RefreshDocumentIdentifiers(builtinDoc, &prs)

	objectSource := `module std::collections::object;
struct Object {}
fn String? Object.get_string(self, String key) {
	return TYPE_MISMATCH~;
}`
	objectURI := protocol.DocumentUri("file:///tmp/hover_object_inferred_builtin_fault_test.c3")
	objectDoc := document.NewDocumentFromDocURI(objectURI, objectSource, 1)
	state.RefreshDocumentIdentifiers(objectDoc, &prs)

	appSource := `module app;
import std::collections::object;
fn void main() {
	Object body = {};
	body.get_string("task");
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_builtin_fault_usage_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "body.get_string") + len("body.") + 2
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "@faults (inferred): @builtin::TYPE_MISMATCH") {
		t.Fatalf("expected builtin-qualified inferred fault list in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_qualified_fault_constant_resolves_from_imported_short_module(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	netSource := `module blem::net;
faultdef ACCEPT_FAILED;`
	netURI := protocol.DocumentUri("file:///tmp/hover_fault_net_module_test.c3")
	netDoc := document.NewDocumentFromDocURI(netURI, netSource, 1)
	state.RefreshDocumentIdentifiers(netDoc, &prs)

	appSource := `module app;
import blem::net;

fn String? read_data() {
	return net::ACCEPT_FAILED~;
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_fault_qualified_usage_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "ACCEPT_FAILED") + len("ACCEPT_")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "ACCEPT_FAILED") {
		t.Fatalf("expected qualified fault hover content, got: %s", content.Value)
	}
}

func TestTextDocumentHover_fault_constant_includes_faultdef_doc_comment(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;
<*
 MY TIMEOUT FAULT
*>
faultdef TIMEOUT;

fn String? read_data() {
	return TIMEOUT~;
}`

	uri := protocol.DocumentUri("file:///tmp/hover_faultdef_doc_for_constant_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.Index(source, "TIMEOUT~") + len("TIME")
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "```c3\nTIMEOUT\n```") {
		t.Fatalf("expected fault constant header in hover, got: %s", content.Value)
	}
	if !strings.Contains(content.Value, "MY TIMEOUT FAULT") {
		t.Fatalf("expected faultdef doc comment in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_inline_lambda_parameter_hover(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	source := `module app;

fn void main() {
	BGTransCallbacks tcb = {
		.constant = fn String(String str, Allocator alloc) =>
			str,
	};
}`

	uri := protocol.DocumentUri("file:///tmp/hover_inline_lambda_param_hover_test.c3")
	doc := document.NewDocumentFromDocURI(uri, source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	idx := strings.LastIndex(source, "alloc") + 1
	pos := byteIndexToLSPPosition(source, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for inline lambda parameter")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "alloc") {
		t.Fatalf("expected lambda parameter hover content, got: %s", content.Value)
	}
}
