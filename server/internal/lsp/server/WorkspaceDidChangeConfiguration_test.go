package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseRuntimeSettings_UppercaseKeys(t *testing.T) {
	settings := map[string]any{
		"C3": map[string]any{
			"version":      "0.7.10",
			"path":         "c3c",
			"stdlib-path":  "/tmp/lib",
			"compile-args": []any{"--test", "foo"},
		},
		"Diagnostics": map[string]any{
			"enabled": true,
			"delay":   float64(1500),
		},
	}

	runtime, ok := parseRuntimeSettings(settings)
	assert.True(t, ok)
	assert.Equal(t, "0.7.10", *runtime.C3.Version)
	assert.Equal(t, "c3c", *runtime.C3.Path)
	assert.Equal(t, "/tmp/lib", *runtime.C3.StdlibPath)
	assert.Equal(t, []string{"--test", "foo"}, runtime.C3.CompileArgs)
	assert.True(t, *runtime.Diagnostics.Enabled)
	assert.Equal(t, 1500*time.Nanosecond, *runtime.Diagnostics.Delay)
}

func TestParseRuntimeSettings_LowercaseKeys(t *testing.T) {
	settings := map[string]any{
		"c3": map[string]any{
			"version":     "0.7.10",
			"stdlib-path": "/tmp/lib/std",
		},
		"diagnostics": map[string]any{
			"enabled": false,
		},
	}

	runtime, ok := parseRuntimeSettings(settings)
	assert.True(t, ok)
	assert.Equal(t, "0.7.10", *runtime.C3.Version)
	assert.Equal(t, "/tmp/lib/std", *runtime.C3.StdlibPath)
	assert.False(t, *runtime.Diagnostics.Enabled)
}

func TestParseRuntimeSettings_UnsupportedPayload(t *testing.T) {
	runtime, ok := parseRuntimeSettings(map[string]any{"foo": "bar"})
	assert.False(t, ok)
	assert.False(t, hasRuntimeSettings(runtime))
}
