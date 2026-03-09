package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
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
			"enabled":              true,
			"delay":                float64(1500),
			"save-full-idle-ms":    float64(10000),
			"full-min-interval-ms": float64(30000),
		},
		"Formatting": map[string]any{
			"c3fmt":                "/tmp/c3fmt",
			"config":               ":default:",
			"will-save-wait-until": true,
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
	assert.Equal(t, 10000*time.Nanosecond, *runtime.Diagnostics.SaveFullIdle)
	assert.Equal(t, 30000*time.Nanosecond, *runtime.Diagnostics.FullMinInterval)
	assert.Equal(t, "/tmp/c3fmt", *runtime.Formatting.C3Fmt)
	assert.Equal(t, ":default:", *runtime.Formatting.Config)
	assert.True(t, *runtime.Formatting.WillSaveWaitUntil)
}

func TestParseRuntimeSettings_LowercaseKeys(t *testing.T) {
	settings := map[string]any{
		"c3": map[string]any{
			"version":     "0.7.10",
			"stdlib-path": "/tmp/lib/std",
		},
		"diagnostics": map[string]any{
			"enabled":              false,
			"save-full-idle-ms":    float64(8000),
			"full-min-interval-ms": float64(20000),
		},
		"formatting": map[string]any{
			"c3fmt":                "/tmp/c3fmt",
			"config":               "/tmp/.c3fmt",
			"will-save-wait-until": false,
		},
	}

	runtime, ok := parseRuntimeSettings(settings)
	assert.True(t, ok)
	assert.Equal(t, "0.7.10", *runtime.C3.Version)
	assert.Equal(t, "/tmp/lib/std", *runtime.C3.StdlibPath)
	assert.False(t, *runtime.Diagnostics.Enabled)
	assert.Equal(t, 8000*time.Nanosecond, *runtime.Diagnostics.SaveFullIdle)
	assert.Equal(t, 20000*time.Nanosecond, *runtime.Diagnostics.FullMinInterval)
	assert.Equal(t, "/tmp/c3fmt", *runtime.Formatting.C3Fmt)
	assert.Equal(t, "/tmp/.c3fmt", *runtime.Formatting.Config)
	assert.False(t, *runtime.Formatting.WillSaveWaitUntil)
}

func TestParseRuntimeSettings_UnsupportedPayload(t *testing.T) {
	runtime, ok := parseRuntimeSettings(map[string]any{"foo": "bar"})
	assert.False(t, ok)
	assert.False(t, hasRuntimeSettings(runtime))
}

func TestWorkspaceDidChangeConfiguration_cancels_active_indexing(t *testing.T) {
	root := t.TempDir()
	projectFile := filepath.Join(root, "project.json")
	if err := os.WriteFile(projectFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project file: %v", err)
	}

	srv := NewServer(ServerOpts{
		C3: c3c.C3Opts{
			Version:     option.None[string](),
			Path:        option.None[string](),
			StdlibPath:  option.None[string](),
			CompileArgs: []string{},
		},
		Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 1},
		Formatting: FormattingOpts{
			C3FmtPath: option.None[string](),
			Config:    option.None[string](),
		},
		LogFilepath: option.None[string](),
	}, "test-server", "test")

	srv.initialized.Store(true)
	srv.state.SetProjectRootURI(root)

	started := make(chan struct{}, 1)
	cancelled := make(chan struct{}, 1)
	srv.workspaceIndexer = func(ctx context.Context, path string) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		select {
		case cancelled <- struct{}{}:
		default:
		}
	}

	srv.indexWorkspaceAtAsync(root)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected indexing to start")
	}

	glspContext := &glsp.Context{Notify: func(string, any) {}}
	err := srv.WorkspaceDidChangeConfiguration(glspContext, &protocol.DidChangeConfigurationParams{Settings: map[string]any{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected active indexing to be cancelled on config change")
	}
}
