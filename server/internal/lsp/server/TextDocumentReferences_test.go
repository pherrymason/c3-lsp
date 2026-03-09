package server

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentReferences_function_usage_excludes_declaration_when_requested(t *testing.T) {
	source := `module app;

fn void parse_port() {
}

fn void run() {
	parse_port();
}`

	uri := protocol.DocumentUri("file:///tmp/references_function_exclude_decl_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port()"))
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}
	if len(locations) != 1 {
		t.Fatalf("expected one usage location without declaration, got: %d", len(locations))
	}
	if locations[0].Range.Start.Line != 6 {
		t.Fatalf("expected usage location on line 6, got line %d", locations[0].Range.Start.Line)
	}
}

func TestTextDocumentReferences_function_usage_includes_declaration_when_requested(t *testing.T) {
	source := `module app;

fn void parse_port() {
}

fn void run() {
	parse_port();
}`

	uri := protocol.DocumentUri("file:///tmp/references_function_include_decl_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port()"))
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("expected declaration and usage locations, got: %d", len(locations))
	}
}

func TestTextDocumentReferences_function_parity_with_rename_edits_across_files(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/references_parity_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/references_parity_use_test.c3")

	declSource := `module app;

fn void parse_port() {
}`

	useSource := `module app;

fn void run() {
	parse_port();
	parse_port();
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "parse_port"))
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}

	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
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

	refKeys := make([]string, 0, len(locations))
	for _, loc := range locations {
		refKeys = append(refKeys, locationKey(loc.URI, loc.Range))
	}
	sort.Strings(refKeys)

	renameKeys := make([]string, 0)
	for uri, edits := range edit.Changes {
		for _, textEdit := range edits {
			renameKeys = append(renameKeys, locationKey(uri, textEdit.Range))
		}
	}
	sort.Strings(renameKeys)

	if len(refKeys) == 0 {
		t.Fatalf("expected references for function symbol")
	}
	if strings.Join(refKeys, "|") != strings.Join(renameKeys, "|") {
		t.Fatalf("expected references and rename edits parity\nrefs=%v\nrename=%v", refKeys, renameKeys)
	}
}

func TestTextDocumentReferences_local_variable_parity_with_rename_edits(t *testing.T) {
	source := `module app;

fn void a() {
	int value = 1;
	value = value + 1;
}

fn void b() {
	int value = 2;
	value = value + 1;
}`

	uri := protocol.DocumentUri("file:///tmp/references_local_variable_parity_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "value = 1"))
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}

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

	refKeys := make([]string, 0, len(locations))
	for _, loc := range locations {
		refKeys = append(refKeys, locationKey(loc.URI, loc.Range))
	}
	sort.Strings(refKeys)

	renameKeys := make([]string, 0)
	for uriKey, edits := range edit.Changes {
		for _, textEdit := range edits {
			renameKeys = append(renameKeys, locationKey(uriKey, textEdit.Range))
		}
	}
	sort.Strings(renameKeys)

	if strings.Join(refKeys, "|") != strings.Join(renameKeys, "|") {
		t.Fatalf("expected local variable references and rename edits parity\nrefs=%v\nrename=%v", refKeys, renameKeys)
	}
}

func TestTextDocumentReferences_parameter_parity_with_rename_edits(t *testing.T) {
	source := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator) {
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
	blem::Fiber* f2 = blem::create(64 * 1024, null, allocator);
	allocator::free(f1.allocator, f1.stack);
}`

	uri := protocol.DocumentUri("file:///tmp/references_parameter_parity_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "allocator)"))
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}

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

	refKeys := make([]string, 0, len(locations))
	for _, loc := range locations {
		refKeys = append(refKeys, locationKey(loc.URI, loc.Range))
	}
	sort.Strings(refKeys)

	renameKeys := make([]string, 0)
	for uriKey, edits := range edit.Changes {
		for _, textEdit := range edits {
			renameKeys = append(renameKeys, locationKey(uriKey, textEdit.Range))
		}
	}
	sort.Strings(renameKeys)

	if strings.Join(refKeys, "|") != strings.Join(renameKeys, "|") {
		t.Fatalf("expected parameter references and rename edits parity\nrefs=%v\nrename=%v", refKeys, renameKeys)
	}
}

func TestTextDocumentReferences_parameter_excludes_module_scope_token(t *testing.T) {
	source := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator) {
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
	allocator::free(f1.allocator, f1.stack);
}`

	uri := protocol.DocumentUri("file:///tmp/references_parameter_exclude_scope_token_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "allocator)"))
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}

	for _, loc := range locations {
		if loc.Range.Start.Line == 4 && loc.Range.Start.Character == 1 {
			t.Fatalf("unexpected module scope token reference on allocator::free")
		}
	}
}

func TestTextDocumentReferences_enum_constant_parity_with_rename_edits_across_files(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/references_enum_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/references_enum_use_test.c3")

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
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}

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

	refKeys := make([]string, 0, len(locations))
	for _, loc := range locations {
		refKeys = append(refKeys, locationKey(loc.URI, loc.Range))
	}
	sort.Strings(refKeys)

	renameKeys := make([]string, 0)
	for uriKey, edits := range edit.Changes {
		for _, textEdit := range edits {
			renameKeys = append(renameKeys, locationKey(uriKey, textEdit.Range))
		}
	}
	sort.Strings(renameKeys)

	if strings.Join(refKeys, "|") != strings.Join(renameKeys, "|") {
		t.Fatalf("expected enum references and rename edits parity\nrefs=%v\nrename=%v", refKeys, renameKeys)
	}
}

func TestTextDocumentReferences_fault_constant_parity_with_rename_edits_across_files_without_import(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/references_fault_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/references_fault_use_test.c3")

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
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}

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

	refKeys := make([]string, 0, len(locations))
	for _, loc := range locations {
		refKeys = append(refKeys, locationKey(loc.URI, loc.Range))
	}
	sort.Strings(refKeys)

	renameKeys := make([]string, 0)
	for uriKey, edits := range edit.Changes {
		for _, textEdit := range edits {
			renameKeys = append(renameKeys, locationKey(uriKey, textEdit.Range))
		}
	}
	sort.Strings(renameKeys)

	if strings.Join(refKeys, "|") != strings.Join(renameKeys, "|") {
		t.Fatalf("expected fault references and rename edits parity\nrefs=%v\nrename=%v", refKeys, renameKeys)
	}
}

func TestTextDocumentReferences_struct_member_parity_with_rename_edits(t *testing.T) {
	source := `module blem;

struct Fiber {
	Allocator allocator;
	void* stack;
}

fn void destroy(Fiber* fiber) {
	allocator::free(fiber.allocator, fiber.stack);
	allocator::free(fiber.allocator, fiber);
}`

	uri := protocol.DocumentUri("file:///tmp/references_struct_member_parity_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "allocator;"))
	locations, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	if err != nil {
		t.Fatalf("unexpected references error: %v", err)
	}

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

	refKeys := make([]string, 0, len(locations))
	for _, loc := range locations {
		refKeys = append(refKeys, locationKey(loc.URI, loc.Range))
	}
	sort.Strings(refKeys)

	renameKeys := make([]string, 0)
	for uriKey, edits := range edit.Changes {
		for _, textEdit := range edits {
			renameKeys = append(renameKeys, locationKey(uriKey, textEdit.Range))
		}
	}
	sort.Strings(renameKeys)

	if strings.Join(refKeys, "|") != strings.Join(renameKeys, "|") {
		t.Fatalf("expected struct member references and rename edits parity\nrefs=%v\nrename=%v", refKeys, renameKeys)
	}
}

func locationKey(uri protocol.DocumentUri, rng protocol.Range) string {
	return fmt.Sprintf("%s:%d:%d-%d:%d", uri, rng.Start.Line, rng.Start.Character, rng.End.Line, rng.End.Character)
}
