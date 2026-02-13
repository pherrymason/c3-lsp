package server

import "testing"

func TestNormalizeStdlibRootPath(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{input: "/Users/f00lg/github/c3/c3c/lib", expected: "/Users/f00lg/github/c3/c3c/lib"},
		{input: "/Users/f00lg/github/c3/c3c/lib/std", expected: "/Users/f00lg/github/c3/c3c/lib"},
		{input: "/Users/f00lg/github/c3/c3c/lib/std/", expected: "/Users/f00lg/github/c3/c3c/lib"},
	}

	for _, tt := range cases {
		if got := normalizeStdlibRootPath(tt.input); got != tt.expected {
			t.Fatalf("normalizeStdlibRootPath(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
