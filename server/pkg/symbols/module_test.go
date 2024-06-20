package symbols

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModule_is_child_of(t *testing.T) {
	cases := []struct {
		module  string
		childOf string
	}{
		{"foo::bar", "foo"},
		{"foo::bar::dashed", "foo"},
		{"foo::bar::dashed", "foo::bar"},
	}

	for _, tt := range cases {
		t.Run(fmt.Sprintf("Test %s is child of %s", tt.module, tt.childOf), func(t *testing.T) {
			moduleA := NewModulePathFromString(tt.module)
			moduleB := NewModulePathFromString(tt.childOf)

			assert.True(t, moduleA.IsSubModuleOf(moduleB))
		})
	}
}

func TestModule_is_not_child_of(t *testing.T) {
	cases := []struct {
		module  string
		childOf string
	}{
		{"bar", "foo"},
		{"foo", "foo::bar::dashed"},
		{"foo::circle", "foo::bar"},
		{"foo::circle::dashed", "foo::bar"},
	}

	for _, tt := range cases {
		t.Run(fmt.Sprintf("Test %s is not child of %s", tt.module, tt.childOf), func(t *testing.T) {
			moduleA := NewModulePathFromString(tt.module)
			moduleB := NewModulePathFromString(tt.childOf)

			assert.False(t, moduleA.IsSubModuleOf(moduleB))
		})
	}
}
