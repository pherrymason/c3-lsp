package server

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentRename_e2e_fixture_function_across_files(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/rename_e2e_fixture_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/rename_e2e_fixture_use_test.c3")

	declSource := `module blem::example;

fn void run_demo_with_allocator(Allocator allocator)
{
	blem::Fiber* f1 = blem::create(64 * 1024, null, allocator);
	blem::Fiber* f2 = blem::create(64 * 1024, null, allocator);
}`

	useSource := `module blem::sample;

import blem::example;

fn void run_demo()
{
	blem::example::run_demo_with_allocator(mem);
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "run_demo_with_allocator(Allocator"))
	edit, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
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

	updatedDecl := applyTextEdits(declSource, edit.Changes[declURI])
	if !strings.Contains(updatedDecl, "fn void run_demo_with_alloc(Allocator allocator)") {
		t.Fatalf("expected declaration rename in fixture decl file, got: %s", updatedDecl)
	}

	updatedUse := applyTextEdits(useSource, edit.Changes[useURI])
	if !strings.Contains(updatedUse, "blem::example::run_demo_with_alloc(mem)") {
		t.Fatalf("expected qualified usage rename in fixture use file, got: %s", updatedUse)
	}
}

func TestTextDocumentRename_e2e_fixture_struct_member_with_done_function_collision(t *testing.T) {
	source := `module blem;

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
}`

	uri := protocol.DocumentUri("file:///tmp/rename_e2e_fixture_member_collision_test.c3")
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
		t.Fatalf("expected member declaration rename in e2e fixture, got: %s", updated)
	}
	if strings.Count(updated, ".completed") != 2 {
		t.Fatalf("expected member usages rename in e2e fixture, got: %s", updated)
	}
	if strings.Contains(updated, "fn void completed()") {
		t.Fatalf("expected function done() to remain unchanged in e2e fixture, got: %s", updated)
	}
}
