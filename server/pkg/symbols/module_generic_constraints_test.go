package symbols

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModuleGenericConstraintLines(t *testing.T) {
	mod := NewModule("std::collections::map", "doc", NewRange(0, 0, 0, 0), NewRange(0, 0, 0, 0))
	mod.SetGenericParameters(map[string]*GenericParameter{
		"Key":   NewGenericParameter("Key", mod.GetName(), "doc", NewRange(0, 0, 0, 0), NewRange(0, 0, 0, 0)),
		"Value": NewGenericParameter("Value", mod.GetName(), "doc", NewRange(0, 0, 0, 0), NewRange(0, 0, 0, 0)),
	})
	mod.SetGenericParameterOrder([]string{"Key", "Value"})

	doc := NewDocCommentBuilder("").
		WithContract("@require", "$defined((Key){}.hash()) : `No .hash function found on the key`").
		Build()
	mod.SetDocComment(&doc)

	lines := ModuleGenericConstraintLines(mod)
	assert.Equal(t, []string{
		"**@require** $defined((Key){}.hash()) : `No .hash function found on the key`",
	}, lines)
}
