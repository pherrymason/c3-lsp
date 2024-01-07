package lsp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWordInPosition(t *testing.T) {
	cases := []struct {
		name     string
		expected string
		position int
	}{
		{"start of doc", "hello", 1},
		{"word", "expected", 14},
		{"word with underscore", "bye_bye", 24},
	}

	source := "hello this is expected bye_bye"
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			word := wordInPosition(source, tt.position)

			assert.Equal(t, tt.expected, word)
		})
	}
}
