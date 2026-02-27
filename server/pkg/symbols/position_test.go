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
