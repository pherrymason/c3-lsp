package server

import (
	"sort"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
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

	if target.name != "old::name" {
		t.Fatalf("unexpected target name: got %q", target.name)
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
