package server

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/utils"
)

func TestLoadRuntimeOpts_requestTimeout_fromFeatureFlag(t *testing.T) {
	_ = os.Unsetenv("C3LSP_REQUEST_TIMEOUT_MS")
	utils.SetRuntimeFeatureFlags(map[string]bool{"REQUEST_TIMEOUT": true})
	t.Cleanup(func() { utils.SetRuntimeFeatureFlags(nil) })

	r := loadRuntimeOpts()
	if r.RequestTimeout != 2*time.Second {
		t.Fatalf("expected 2s timeout from feature flag, got %s", r.RequestTimeout)
	}
}

func TestLoadRuntimeOpts_slowRequest_fromFeatureFlag(t *testing.T) {
	_ = os.Unsetenv("C3LSP_SLOW_REQUEST_MS")
	utils.SetRuntimeFeatureFlags(map[string]bool{"REQUEST_SLOW_LOG": true})
	t.Cleanup(func() { utils.SetRuntimeFeatureFlags(nil) })

	r := loadRuntimeOpts()
	if r.SlowRequestThreshold != 400*time.Millisecond {
		t.Fatalf("expected 400ms slow threshold from feature flag, got %s", r.SlowRequestThreshold)
	}
}

func TestLoadRuntimeOpts_slowRequest_fromEnv(t *testing.T) {
	if err := os.Setenv("C3LSP_SLOW_REQUEST_MS", "35"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("C3LSP_SLOW_REQUEST_MS") })

	r := loadRuntimeOpts()
	if r.SlowRequestThreshold != 35*time.Millisecond {
		t.Fatalf("expected 35ms slow threshold from env, got %s", r.SlowRequestThreshold)
	}
}

func TestLoadRuntimeOpts_timeoutDump_fromFeatureFlag(t *testing.T) {
	_ = os.Unsetenv("C3LSP_REQUEST_TIMEOUT_DUMP")
	utils.SetRuntimeFeatureFlags(map[string]bool{"REQUEST_TIMEOUT_DUMP": true})
	t.Cleanup(func() { utils.SetRuntimeFeatureFlags(nil) })

	r := loadRuntimeOpts()
	if !r.RequestTimeoutDump {
		t.Fatalf("expected timeout dump enabled from feature flag")
	}
}

func TestLoadRuntimeOpts_timeoutDump_fromEnv(t *testing.T) {
	if err := os.Setenv("C3LSP_REQUEST_TIMEOUT_DUMP", "true"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("C3LSP_REQUEST_TIMEOUT_DUMP") })

	r := loadRuntimeOpts()
	if !r.RequestTimeoutDump {
		t.Fatalf("expected timeout dump enabled from env")
	}
}

func TestRunWithRequestTimeout_returnsFallbackOnTimeout(t *testing.T) {
	srv := &Server{}
	srv.options.Runtime = RuntimeOpts{
		RequestTimeoutMs: 10,
		RequestTimeout:   10 * time.Millisecond,
	}
	srv.initialized.Store(true)

	start := time.Now()
	value, err := runWithRequestTimeout[int](srv, 0, "test/method", "", 7, func(_ context.Context) (int, error) {
		time.Sleep(60 * time.Millisecond)
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 7 {
		t.Fatalf("expected fallback value 7, got %d", value)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("expected timeout fallback quickly, took %s", elapsed)
	}
}

func TestRunWithRequestTimeout_returnsFallbackWhenInflightLimitReached(t *testing.T) {
	srv := &Server{gate: requestGate{slots: make(chan struct{}, 1)}}
	srv.gate.slots <- struct{}{}

	start := time.Now()
	value, err := runWithRequestTimeout[int](srv, 0, "test/method", "", 9, func(_ context.Context) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 9 {
		t.Fatalf("expected fallback value 9, got %d", value)
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Fatalf("expected inflight backpressure wait around 250ms, got %s", elapsed)
	}
}

func TestRunGuardedRequest_rejects_when_server_not_initialized(t *testing.T) {
	srv := &Server{}

	value, err := runGuardedRequest[int](srv, nil, "textDocument/hover", "", 11, func(_ context.Context) (int, error) {
		return 42, nil
	})
	if err == nil {
		t.Fatalf("expected initialization error, got nil")
	}
	if !errors.Is(err, errServerNotReady) {
		t.Fatalf("expected errServerNotReady, got: %v", err)
	}
	if value != 11 {
		t.Fatalf("expected fallback value 11, got %d", value)
	}
}

func TestRunGuardedRequest_allows_when_initialized(t *testing.T) {
	srv := &Server{}
	srv.initialized.Store(true)

	value, err := runGuardedRequest[int](srv, nil, "textDocument/hover", "", 11, func(_ context.Context) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected handler value 42, got %d", value)
	}
}
