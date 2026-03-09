package server

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentDocumentLink_import_module_and_qualified_usage(t *testing.T) {
	appURI := protocol.DocumentUri("file:///tmp/document_link_app_test.c3")
	utilURI := protocol.DocumentUri("file:///tmp/document_link_util_test.c3")

	appSource := `module app;
import util::math;

fn void run() {
	util::math::add(1, 2);
}`

	utilSource := `module util::math;
fn int add(int a, int b) {
	return a + b;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: appURI, source: appSource},
		{uri: utilURI, source: utilSource},
	})

	links, err := srv.TextDocumentDocumentLink(nil, &protocol.DocumentLinkParams{TextDocument: protocol.TextDocumentIdentifier{URI: appURI}})
	if err != nil {
		t.Fatalf("unexpected documentLink error: %v", err)
	}
	if len(links) < 3 {
		t.Fatalf("expected links for module/import/qualified usage, got: %#v", links)
	}

	hasImport := false
	hasQualified := false
	for _, link := range links {
		if link.Target == nil {
			t.Fatalf("expected link target to be set")
		}
		if *link.Target != utilURI && *link.Target != appURI {
			t.Fatalf("unexpected link target: %s", *link.Target)
		}

		start := symbols.NewPositionFromLSPPosition(link.Range.Start).IndexIn(appSource)
		end := symbols.NewPositionFromLSPPosition(link.Range.End).IndexIn(appSource)
		text := appSource[start:end]
		if text == "util::math" {
			if link.Range.Start.Line == 1 {
				hasImport = true
			}
			if link.Range.Start.Line == 4 {
				hasQualified = true
			}
		}
	}

	if !hasImport || !hasQualified {
		t.Fatalf("expected both import and qualified usage links, got: %#v", links)
	}
}

func TestDocumentLinkResolve_sets_target_from_data(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/document_link_resolve_test.c3"), "module app;")
	link := &protocol.DocumentLink{
		Range: protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 3}},
		Data: map[string]any{
			"target": "file:///tmp/target.c3",
		},
	}

	resolved, err := srv.DocumentLinkResolve(nil, link)
	if err != nil {
		t.Fatalf("unexpected documentLink/resolve error: %v", err)
	}
	if resolved == nil || resolved.Target == nil {
		t.Fatalf("expected resolved target")
	}
	if *resolved.Target != protocol.DocumentUri("file:///tmp/target.c3") {
		t.Fatalf("unexpected resolved target: %s", *resolved.Target)
	}
}

func TestTextDocumentDocumentLink_stdlike_imports_module_attributes_and_generics(t *testing.T) {
	appURI := protocol.DocumentUri("file:///tmp/document_link_stdlike_app_test.c3")
	tmpMapURI := protocol.DocumentUri("file:///tmp/document_link_stdlike_tmp_map_test.c3")
	distURI := protocol.DocumentUri("file:///tmp/document_link_stdlike_distributions_test.c3")
	ioURI := protocol.DocumentUri("file:///tmp/document_link_stdlike_io_test.c3")
	posixURI := protocol.DocumentUri("file:///tmp/document_link_stdlike_posix_test.c3")
	threadURI := protocol.DocumentUri("file:///tmp/document_link_stdlike_thread_test.c3")

	appSource := `module app;
import libc, std::io, std::os::posix;
import std::math::distributions @public;
module std::collections::tmp_map <Key, Value> @private;

fn void run() {
	std::thread::sleep_ms(1);
	// import std::io;
	const String s = "std::io::printn";
}`

	tmpMapSource := `module std::collections::tmp_map <Key, Value>;
struct Entry { int value; }`
	distSource := `module std::math::distributions;
fn void sample() {}`
	ioSource := `module std::io;
fn void printn(String s) {}`
	posixSource := `module std::os::posix;
fn void noop() {}`
	threadSource := `module std::thread;
fn void sleep_ms(int ms) {}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: appURI, source: appSource},
		{uri: tmpMapURI, source: tmpMapSource},
		{uri: distURI, source: distSource},
		{uri: ioURI, source: ioSource},
		{uri: posixURI, source: posixSource},
		{uri: threadURI, source: threadSource},
	})

	links, err := srv.TextDocumentDocumentLink(nil, &protocol.DocumentLinkParams{TextDocument: protocol.TextDocumentIdentifier{URI: appURI}})
	if err != nil {
		t.Fatalf("unexpected documentLink error: %v", err)
	}

	targetsByText := map[string]protocol.DocumentUri{}
	stdIOLinkCount := 0
	stdIOLinkOnImportLine := false
	for _, link := range links {
		if link.Target == nil {
			t.Fatalf("expected target for link: %#v", link)
		}
		text := textAtLSPRange(appSource, link.Range)
		if text == "std::io" {
			stdIOLinkCount++
			if link.Range.Start.Line == 1 {
				stdIOLinkOnImportLine = true
			}
		}
		targetsByText[text] = *link.Target
	}
	if stdIOLinkCount != 1 || !stdIOLinkOnImportLine {
		t.Fatalf("expected exactly one std::io link from import line, got count=%d links=%#v", stdIOLinkCount, links)
	}

	expected := map[string]protocol.DocumentUri{
		"std::io":                   ioURI,
		"std::os::posix":            posixURI,
		"std::math::distributions":  distURI,
		"std::collections::tmp_map": appURI,
		"std::thread":               threadURI,
	}

	for text, target := range expected {
		got, ok := targetsByText[text]
		if !ok {
			t.Fatalf("expected link for %q, got %#v", text, targetsByText)
		}
		if got != target {
			t.Fatalf("expected %q target %s, got %s", text, target, got)
		}
	}
}

func textAtLSPRange(source string, r protocol.Range) string {
	start := symbols.NewPositionFromLSPPosition(r.Start).IndexIn(source)
	end := symbols.NewPositionFromLSPPosition(r.End).IndexIn(source)
	if start < 0 || end < 0 || end < start || end > len(source) {
		return ""
	}

	return source[start:end]
}
