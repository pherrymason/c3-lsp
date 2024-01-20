package document

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWordInIndex(t *testing.T) {
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
	doc := NewDocument("x", "x", source)
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			word, _ := doc.WordInIndex(tt.position)

			assert.Equal(t, tt.expected, word)
		})
	}
}
