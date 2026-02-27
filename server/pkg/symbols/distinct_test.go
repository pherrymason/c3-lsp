package symbols

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistinct_GetHoverInfo_handles_nil_base_type(t *testing.T) {
	d := NewDistinctBuilder("Thread", "std::threads", "thread.c3").Build()

	assert.NotPanics(t, func() {
		hover := d.GetHoverInfo()
		assert.Contains(t, hover, "distinct Thread")
	})
}
