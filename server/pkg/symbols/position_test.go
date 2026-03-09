package symbols

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Position_IndexIn_clamps_when_line_past_content(t *testing.T) {
	content := "abc\n"
	pos := NewPosition(10, 0)

	assert.Equal(t, len(content), pos.IndexIn(content))
}

func Test_Position_IndexIn_clamps_when_character_past_line_length(t *testing.T) {
	content := "abc\nxyz"
	pos := NewPosition(0, 100)

	assert.Equal(t, 3, pos.IndexIn(content))
}

func Test_Position_IndexIn_handles_utf16_surrogate_pairs(t *testing.T) {
	content := "😀x"

	assert.Equal(t, 0, NewPosition(0, 1).IndexIn(content))
	assert.Equal(t, 4, NewPosition(0, 2).IndexIn(content))
}

func Test_Position_IndexIn_clamps_before_crlf_newline(t *testing.T) {
	content := "a\r\nb"

	assert.Equal(t, 1, NewPosition(0, 100).IndexIn(content))
}

func Test_Position_IndexIn_table_unicode_crlf_tabs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		pos     Position
		index   int
	}{
		{
			name:    "tab character counts as one",
			content: "\tfoo",
			pos:     NewPosition(0, 1),
			index:   1,
		},
		{
			name:    "crlf second line start",
			content: "ab\r\ncd",
			pos:     NewPosition(1, 0),
			index:   4,
		},
		{
			name:    "unicode character after ascii",
			content: "aá",
			pos:     NewPosition(0, 2),
			index:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.index, tt.pos.IndexIn(tt.content))
		})
	}
}
