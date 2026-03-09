package server

import (
	"strconv"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func BenchmarkTextDocumentRename_Function(b *testing.B) {
	source := `module app;

fn void parse_port() {
}

fn void run() {
	parse_port();
	parse_port();
	parse_port();
}`

	uri := protocol.DocumentUri("file:///tmp/bench_rename_function_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     pos,
			},
			NewName: "parse_socket_port",
		})
		if err != nil {
			b.Fatalf("rename failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentRename_LocalVariable(b *testing.B) {
	source := `module app;

fn void run() {
	int value = 1;
	value = value + 1;
	value = value + 2;
}`

	uri := protocol.DocumentUri("file:///tmp/bench_rename_local_variable_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "value = 1"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     pos,
			},
			NewName: "localValue",
		})
		if err != nil {
			b.Fatalf("rename failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentRename_Parameter(b *testing.B) {
	source := `module app;

fn void run_with_allocator(Allocator allocator) {
	(void)allocator;
	(void)allocator;
	allocator::free(allocator, null);
}`

	uri := protocol.DocumentUri("file:///tmp/bench_rename_parameter_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "allocator)"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     pos,
			},
			NewName: "alloc",
		})
		if err != nil {
			b.Fatalf("rename failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentReferences_Function(b *testing.B) {
	source := `module app;

fn void parse_port() {
}

fn void run() {
	parse_port();
	parse_port();
	parse_port();
}`

	uri := protocol.DocumentUri("file:///tmp/bench_references_function_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "parse_port"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.TextDocumentReferences(nil, &protocol.ReferenceParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     pos,
			},
			Context: protocol.ReferenceContext{IncludeDeclaration: true},
		})
		if err != nil {
			b.Fatalf("references failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentRename_FunctionLargeFileHighMatch(b *testing.B) {
	var sourceBuilder strings.Builder
	sourceBuilder.WriteString("module app;\n\n")
	sourceBuilder.WriteString("fn int hot_symbol(int v) {\n\treturn v + 1;\n}\n\n")

	for i := 0; i < 800; i++ {
		sourceBuilder.WriteString("fn int caller_")
		sourceBuilder.WriteString(strconv.Itoa(i))
		sourceBuilder.WriteString("(int input) {\n")
		sourceBuilder.WriteString("\treturn hot_symbol(input) + hot_symbol(input + 1);\n")
		sourceBuilder.WriteString("}\n\n")
	}

	source := sourceBuilder.String()
	uri := protocol.DocumentUri("file:///tmp/bench_rename_function_large_high_match_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "hot_symbol(int"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     pos,
			},
			NewName: "hot_symbol_renamed",
		})
		if err != nil {
			b.Fatalf("rename failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentRename_StructMemberLargeFileHighMatch(b *testing.B) {
	var sourceBuilder strings.Builder
	sourceBuilder.WriteString("module app;\n\n")
	sourceBuilder.WriteString("struct Fiber {\n\tint entry;\n\tint done;\n}\n\n")
	sourceBuilder.WriteString("fn void run(Fiber* fiber) {\n")

	for i := 0; i < 1200; i++ {
		sourceBuilder.WriteString("\tfiber.entry = fiber.entry + ")
		sourceBuilder.WriteString("1")
		sourceBuilder.WriteString(";\n")
		if i%3 == 0 {
			sourceBuilder.WriteString("\tif (fiber.done > 0) fiber.done = fiber.done - 1;\n")
		}
	}

	sourceBuilder.WriteString("}\n")

	source := sourceBuilder.String()
	uri := protocol.DocumentUri("file:///tmp/bench_rename_struct_member_large_high_match_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "entry;"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.TextDocumentRename(nil, &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     pos,
			},
			NewName: "callback",
		})
		if err != nil {
			b.Fatalf("rename failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentCompletion_InScope(b *testing.B) {
	source := `module app;

fn void run() {
	int local_value = 1;
	int local_counter = 2;
	loc
}`

	uri := protocol.DocumentUri("file:///tmp/bench_completion_in_scope_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "loc")+3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := srv.TextDocumentCompletion(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     pos,
			},
		})
		if err != nil {
			b.Fatalf("completion failed: %v", err)
		}
	}
}
