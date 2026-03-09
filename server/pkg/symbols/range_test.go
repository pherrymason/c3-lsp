package symbols

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRange_HasPosition_respects_start_and_end_characters_on_multiline_ranges(t *testing.T) {
	r := NewRange(2, 3, 4, 5)

	assert.False(t, r.HasPosition(NewPosition(2, 2)))
	assert.True(t, r.HasPosition(NewPosition(2, 3)))
	assert.True(t, r.HasPosition(NewPosition(3, 999)))
	assert.True(t, r.HasPosition(NewPosition(4, 5)))
	assert.False(t, r.HasPosition(NewPosition(4, 6)))
}
