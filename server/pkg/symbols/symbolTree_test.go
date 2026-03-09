package symbols

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNode_Insert_adds_input_node_as_child(t *testing.T) {
	parent := NewNode(nil)
	child := NewNode(nil)

	parent.Insert(&child)

	if assert.Len(t, parent.children, 1) {
		assert.Same(t, &child, parent.children[0])
	}
}
