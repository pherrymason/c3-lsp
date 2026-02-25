package server

import (
	"encoding/json"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestBuildCallableSnippet(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		detail   string
		expected string
		ok       bool
	}{
		{
			name:     "function without args",
			label:    "run",
			detail:   "fn void()",
			expected: "run()",
			ok:       true,
		},
		{
			name:     "function with required args",
			label:    "test",
			detail:   "fn int(int hola, float world)",
			expected: "test(${1:hola}, ${2:world})",
			ok:       true,
		},
		{
			name:     "skips default args",
			label:    "test",
			detail:   "fn int(int hola, int world = 2)",
			expected: "test(${1:hola})",
			ok:       true,
		},
		{
			name:     "skips varargs",
			label:    "test",
			detail:   "fn void(int a, int... rest)",
			expected: "test(${1:a})",
			ok:       true,
		},
		{
			name:     "skips implicit self in methods",
			label:    "transparentize",
			detail:   "fn Color(Color self)",
			expected: "transparentize()",
			ok:       true,
		},
		{
			name:     "strips method type qualifier",
			label:    "Tile.print_tile",
			detail:   "fn void(Tile* self)",
			expected: "print_tile()",
			ok:       true,
		},
		{
			name:     "macro with trailing body",
			label:    "transform",
			detail:   "macro int(int x; @body)",
			expected: "transform(${1:x})",
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, ok := buildCallableSnippet(tt.label, tt.detail)
			if ok != tt.ok {
				t.Fatalf("unexpected ok: got %v, want %v", ok, tt.ok)
			}

			if actual != tt.expected {
				t.Fatalf("unexpected snippet: got %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestSnippetToPlainInsertText(t *testing.T) {
	actual := snippetToPlainInsertText("test(${1:hola}, ${2:world})")
	if actual != "test(hola, world)" {
		t.Fatalf("unexpected plain insert text: got %q", actual)
	}
}

func TestClientSupportsCompletionSnippets(t *testing.T) {
	var withSnippet protocol.ClientCapabilities
	err := json.Unmarshal(
		[]byte(`{"textDocument":{"completion":{"completionItem":{"snippetSupport":true}}}}`),
		&withSnippet,
	)
	if err != nil {
		t.Fatalf("failed to unmarshal capabilities with snippet support: %v", err)
	}

	var withoutSnippet protocol.ClientCapabilities
	err = json.Unmarshal(
		[]byte(`{"textDocument":{"completion":{"completionItem":{"snippetSupport":false}}}}`),
		&withoutSnippet,
	)
	if err != nil {
		t.Fatalf("failed to unmarshal capabilities without snippet support: %v", err)
	}

	if !clientSupportsCompletionSnippets(withSnippet) {
		t.Fatalf("expected snippet support to be true")
	}

	if clientSupportsCompletionSnippets(withoutSnippet) {
		t.Fatalf("expected snippet support to be false")
	}

	if clientSupportsCompletionSnippets(protocol.ClientCapabilities{}) {
		t.Fatalf("expected snippet support to be false when capability is missing")
	}
}

func TestBuildStructSnippetDeclaration(t *testing.T) {
	snippet, ok := buildStructSnippet(structCompletionDeclaration, "Thing", []string{"a"})
	if !ok {
		t.Fatalf("expected struct snippet to be built")
	}

	expected := "Thing ${1:thing} = { .a = ${2:a} };"
	if snippet != expected {
		t.Fatalf("unexpected struct snippet: got %q, want %q", snippet, expected)
	}
}

func TestBuildStructSnippetValue(t *testing.T) {
	snippet, ok := buildStructSnippet(structCompletionValue, "Thing", []string{"a", "b"})
	if !ok {
		t.Fatalf("expected struct value snippet to be built")
	}

	expected := "{\n\t.a = ${1:a},\n\t.b = ${2:b},\n}"
	if snippet != expected {
		t.Fatalf("unexpected struct value snippet: got %q, want %q", snippet, expected)
	}
}

func TestToLowerCamelName(t *testing.T) {
	if actual := toLowerCamelName("Thing"); actual != "thing" {
		t.Fatalf("unexpected lower camel result: got %q", actual)
	}

	if actual := toLowerCamelName("URLParser"); actual != "urlParser" {
		t.Fatalf("unexpected acronym lower camel result: got %q", actual)
	}
}

func TestStructCompletionContext(t *testing.T) {
	if actual := structCompletionContext("example::Thing", 0, len("example::Thing")); actual != structCompletionDeclaration {
		t.Fatalf("unexpected declaration context: got %d", actual)
	}

	bodyDeclaration := "fn void main() {\n\texample::Thing\n}"
	if actual := structCompletionContext(bodyDeclaration, len("fn void main() {\n\t"), len("fn void main() {\n\texample::Thing")); actual != structCompletionDeclaration {
		t.Fatalf("unexpected function-body declaration context: got %d", actual)
	}

	if actual := structCompletionContext("value = example::Thing", len("value = "), len("value = example::Thing")); actual != structCompletionValue {
		t.Fatalf("unexpected assignment context: got %d", actual)
	}

	if actual := structCompletionContext("call(example::Thing", len("call("), len("call(example::Thing")); actual != structCompletionValue {
		t.Fatalf("unexpected call argument context: got %d", actual)
	}

	if actual := structCompletionContext("fn void f(example::Thing", len("fn void f("), len("fn void f(example::Thing")); actual != structCompletionNone {
		t.Fatalf("unexpected function signature context: got %d", actual)
	}

	bodyCallArg := "fn void main() {\n\texample::do_thing(example::Thing);\n}"
	if actual := structCompletionContext(bodyCallArg, len("fn void main() {\n\texample::do_thing("), len("fn void main() {\n\texample::do_thing(example::Thing")); actual != structCompletionValue {
		t.Fatalf("unexpected function call argument context: got %d", actual)
	}

	if actual := structCompletionContext("HashMap { Tile", len("HashMap { "), len("HashMap { Tile")); actual != structCompletionNone {
		t.Fatalf("unexpected generic key type context: got %d", actual)
	}

	if actual := structCompletionContext("HashMap { int, Tile", len("HashMap { int, "), len("HashMap { int, Tile")); actual != structCompletionNone {
		t.Fatalf("unexpected generic value type context: got %d", actual)
	}

	if actual := structCompletionContext("HashMap{String, List{Tile", len("HashMap{String, List{"), len("HashMap{String, List{Tile")); actual != structCompletionNone {
		t.Fatalf("unexpected nested generic type context: got %d", actual)
	}
}

func TestCompletedStructTypeName(t *testing.T) {
	if actual := completedStructTypeName("example::Th", "Thing"); actual != "example::Thing" {
		t.Fatalf("unexpected module-qualified type: got %q", actual)
	}

	if actual := completedStructTypeName("Thi", "Thing"); actual != "Thing" {
		t.Fatalf("unexpected plain type: got %q", actual)
	}
}

func TestChooseTrailingToken(t *testing.T) {
	if actual := chooseTrailingToken("call(abc", len("call(abc")); actual != "," {
		t.Fatalf("unexpected token in list context: got %q", actual)
	}

	if actual := chooseTrailingToken("for (abc", len("for (abc")); actual != "" {
		t.Fatalf("unexpected token in control header: got %q", actual)
	}

	if actual := chooseTrailingToken("abc", len("abc")); actual != ";" {
		t.Fatalf("unexpected token in statement context: got %q", actual)
	}

	if actual := chooseTrailingToken("abc;", len("abc")); actual != "" {
		t.Fatalf("unexpected token before existing semicolon: got %q", actual)
	}
}
