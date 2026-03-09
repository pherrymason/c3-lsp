package server

import (
	"encoding/json"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
)

func TestNormalizeStdlibRootPath(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{input: "/Users/f00lg/github/c3/c3c/lib", expected: "/Users/f00lg/github/c3/c3c/lib"},
		{input: "/Users/f00lg/github/c3/c3c/lib/std", expected: "/Users/f00lg/github/c3/c3c/lib"},
		{input: "/Users/f00lg/github/c3/c3c/lib/std/", expected: "/Users/f00lg/github/c3/c3c/lib"},
	}

	for _, tt := range cases {
		if got := normalizeStdlibRootPath(tt.input); got != tt.expected {
			t.Fatalf("normalizeStdlibRootPath(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestServerOptsJson_unmarshal_env_flags(t *testing.T) {
	payload := []byte(`{"Env":{"PERF_TRACE":true,"RENAME_DEBUG":true,"USE_SEARCH_V2":false},"log-filepath":"/tmp/c3lsp.log","debug":true,"Diagnostics":{"save-full-idle-ms":10000,"full-min-interval-ms":30000},"Formatting":{"will-save-wait-until":true}}`)

	var opts ServerOptsJson
	if err := json.Unmarshal(payload, &opts); err != nil {
		t.Fatalf("failed to unmarshal opts: %v", err)
	}

	if !opts.Env["PERF_TRACE"] {
		t.Fatalf("expected PERF_TRACE=true in Env")
	}
	if !opts.Env["RENAME_DEBUG"] {
		t.Fatalf("expected RENAME_DEBUG=true in Env")
	}
	if opts.Env["USE_SEARCH_V2"] {
		t.Fatalf("expected USE_SEARCH_V2=false in Env")
	}
	if opts.LogFilepath == nil || *opts.LogFilepath != "/tmp/c3lsp.log" {
		t.Fatalf("expected log-filepath to be parsed")
	}
	if opts.Debug == nil || !*opts.Debug {
		t.Fatalf("expected debug=true to be parsed")
	}
	if opts.Diagnostics.SaveFullIdle != 10000 {
		t.Fatalf("expected save-full-idle-ms to be parsed, got %v", opts.Diagnostics.SaveFullIdle)
	}
	if opts.Diagnostics.FullMinInterval != 30000 {
		t.Fatalf("expected full-min-interval-ms to be parsed, got %v", opts.Diagnostics.FullMinInterval)
	}
	if opts.Formatting.WillSaveWaitUntil == nil || !*opts.Formatting.WillSaveWaitUntil {
		t.Fatalf("expected formatting.will-save-wait-until=true to be parsed")
	}
}

func TestLogPathChanged(t *testing.T) {
	tests := []struct {
		name    string
		current option.Option[string]
		next    option.Option[string]
		want    bool
	}{
		{name: "none to none", current: option.None[string](), next: option.None[string](), want: false},
		{name: "none to some", current: option.None[string](), next: option.Some("/tmp/a.log"), want: true},
		{name: "some to none", current: option.Some("/tmp/a.log"), next: option.None[string](), want: true},
		{name: "same path", current: option.Some("/tmp/a.log"), next: option.Some("/tmp/a.log"), want: false},
		{name: "different path", current: option.Some("/tmp/a.log"), next: option.Some("/tmp/b.log"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logPathChanged(tt.current, tt.next); got != tt.want {
				t.Fatalf("logPathChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResetDiagnosticsSchedulers_clampsSaveDiagnosticsWindows(t *testing.T) {
	srv := &Server{options: ServerOpts{Diagnostics: DiagnosticsOpts{
		Enabled:         true,
		Delay:           2000,
		SaveFullIdle:    100,
		FullMinInterval: 900000,
	}}}

	srv.resetDiagnosticsSchedulers()

	if srv.options.Diagnostics.SaveFullIdle != 500 {
		t.Fatalf("expected save-full-idle-ms to clamp to 500, got %d", srv.options.Diagnostics.SaveFullIdle)
	}
	if srv.options.Diagnostics.FullMinInterval != 600000 {
		t.Fatalf("expected full-min-interval-ms to clamp to 600000, got %d", srv.options.Diagnostics.FullMinInterval)
	}
}
