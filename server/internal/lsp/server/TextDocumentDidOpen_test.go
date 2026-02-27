package server

import "testing"

func TestIsC3LanguageID(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected bool
	}{
		{name: "lowercase", language: "c3", expected: true},
		{name: "uppercase", language: "C3", expected: true},
		{name: "mixed case", language: "C3", expected: true},
		{name: "other", language: "rust", expected: false},
		{name: "empty", language: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isC3LanguageID(tt.language); got != tt.expected {
				t.Fatalf("isC3LanguageID(%q) = %v, expected %v", tt.language, got, tt.expected)
			}
		})
	}
}

func TestIsC3Document(t *testing.T) {
	tests := []struct {
		name     string
		language string
		uri      string
		expected bool
	}{
		{name: "language id lowercase", language: "c3", uri: "file:///tmp/test.txt", expected: true},
		{name: "language id uppercase", language: "C3", uri: "file:///tmp/test.txt", expected: true},
		{name: "extension c3", language: "plaintext", uri: "file:///tmp/main.c3", expected: true},
		{name: "extension c3i", language: "plaintext", uri: "file:///tmp/main.c3i", expected: true},
		{name: "extension c3t", language: "plaintext", uri: "file:///tmp/main.c3t", expected: true},
		{name: "unsupported", language: "plaintext", uri: "file:///tmp/main.rs", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isC3Document(tt.language, tt.uri); got != tt.expected {
				t.Fatalf("isC3Document(%q, %q) = %v, expected %v", tt.language, tt.uri, got, tt.expected)
			}
		})
	}
}
