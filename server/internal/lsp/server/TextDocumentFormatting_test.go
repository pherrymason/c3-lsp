package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestFormatterCommandArgs_DefaultDiscovery(t *testing.T) {
	args := formatterCommandArgs(option.None[string]())
	if len(args) != 2 || args[0] != "--stdin" || args[1] != "--stdout" {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestFormatterCommandArgs_DefaultConfigSentinel(t *testing.T) {
	args := formatterCommandArgs(option.Some(":default:"))
	if len(args) != 3 || args[2] != "--default" {
		t.Fatalf("unexpected args for default config: %v", args)
	}
}

func TestFormatterCommandArgs_ExplicitConfig(t *testing.T) {
	args := formatterCommandArgs(option.Some("/tmp/.c3fmt"))
	if len(args) != 3 || args[2] != "--config=/tmp/.c3fmt" {
		t.Fatalf("unexpected args for explicit config: %v", args)
	}
}

func TestResolveFormatterBinary_FromDirectory(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}

	binaryPath := filepath.Join(buildDir, "c3fmt")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to create formatter binary: %v", err)
	}

	resolved, err := resolveFormatterBinary(root)
	if err != nil {
		t.Fatalf("expected resolveFormatterBinary to succeed: %v", err)
	}

	if resolved != binaryPath {
		t.Fatalf("unexpected resolved path: got %q, want %q", resolved, binaryPath)
	}
}

func TestFullDocumentTextEdit(t *testing.T) {
	original := "module app;\nfn void main(){}\n"
	formatted := "module app;\nfn void main()\n{\n}\n"

	edit := fullDocumentTextEdit(original, formatted)
	if edit.Range.Start.Line != 0 || edit.Range.Start.Character != 0 {
		t.Fatalf("unexpected start range: %+v", edit.Range.Start)
	}

	end := byteIndexToLSPPosition(original, len(original))
	if edit.Range.End != end {
		t.Fatalf("unexpected end range: got %+v, want %+v", edit.Range.End, end)
	}

	if edit.NewText != formatted {
		t.Fatalf("unexpected new text: got %q", edit.NewText)
	}
}

func TestMinimalTextEdit(t *testing.T) {
	original := "module app;\nfn void main(){}\n"
	formatted := "module app;\nfn void main()\n{\n}\n"

	edit, ok := minimalTextEdit(original, formatted)
	if !ok {
		t.Fatalf("expected minimal edit")
	}
	if edit.NewText == formatted {
		t.Fatalf("expected partial replacement, got full document replacement")
	}
}

func TestRangesOverlap(t *testing.T) {
	a := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 0},
		End:   protocol.Position{Line: 6, Character: 0},
	}
	b := protocol.Range{
		Start: protocol.Position{Line: 5, Character: 0},
		End:   protocol.Position{Line: 7, Character: 0},
	}
	c := protocol.Range{
		Start: protocol.Position{Line: 7, Character: 1},
		End:   protocol.Position{Line: 9, Character: 0},
	}

	if !rangesOverlap(a, b) {
		t.Fatalf("expected overlapping ranges")
	}
	if rangesOverlap(a, c) {
		t.Fatalf("expected non-overlapping ranges")
	}
}

func TestSupportedOnTypeFormattingTriggers(t *testing.T) {
	if !isSupportedOnTypeFormattingTrigger("}") {
		t.Fatalf("expected } to trigger on-type formatting")
	}
	if !isSupportedOnTypeFormattingTrigger(";") {
		t.Fatalf("expected ; to trigger on-type formatting")
	}
	if isSupportedOnTypeFormattingTrigger("\n") {
		t.Fatalf("did not expect newline to trigger on-type formatting")
	}
}

func TestOnTypeFormattingAdditionalTriggerCharacters_ExcludesNewline(t *testing.T) {
	for _, ch := range onTypeFormattingAdditionalTriggerCharacters() {
		if ch == "\n" {
			t.Fatalf("unexpected newline in additional on-type triggers")
		}
	}
}

func TestTextDocumentOnTypeFormatting_IgnoresNewlineTrigger(t *testing.T) {
	srv := &Server{}

	edits, err := srv.TextDocumentOnTypeFormatting(nil, &protocol.DocumentOnTypeFormattingParams{
		Ch: "\n",
	})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(edits) != 0 {
		t.Fatalf("expected no edits for newline trigger, got: %#v", edits)
	}
}
