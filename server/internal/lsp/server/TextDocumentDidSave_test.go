package server

import (
	"bytes"
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
)

func TestShouldScheduleDiagnosticsForSave_DeduplicatesSameVersion(t *testing.T) {
	srv := &Server{diag: diagnosticsCoordinator{saveDocVersions: map[string]int32{}}}

	if !srv.shouldScheduleDiagnosticsForSave("/tmp/app.c3", 5) {
		t.Fatalf("expected first save version to schedule diagnostics")
	}
	if srv.shouldScheduleDiagnosticsForSave("/tmp/app.c3", 5) {
		t.Fatalf("expected repeated save with same version to be skipped")
	}
	if !srv.shouldScheduleDiagnosticsForSave("/tmp/app.c3", 6) {
		t.Fatalf("expected newer save version to schedule diagnostics")
	}
}

func TestReserveSaveFullDiagnosticsSlot_RespectsMinInterval(t *testing.T) {
	srv := &Server{}
	srv.options.Diagnostics.FullMinInterval = 1000

	base := time.Unix(100, 0)
	if !srv.reserveSaveFullDiagnosticsSlot(base) {
		t.Fatalf("expected first save-full slot reservation to pass")
	}
	if srv.reserveSaveFullDiagnosticsSlot(base.Add(200 * time.Millisecond)) {
		t.Fatalf("expected reservation inside min interval to be rejected")
	}
	if !srv.reserveSaveFullDiagnosticsSlot(base.Add(1200 * time.Millisecond)) {
		t.Fatalf("expected reservation after min interval to pass")
	}
}

func TestScheduleDiagnosticsFullAfterSaveIdle_RespectsMinIntervalWindow(t *testing.T) {
	root := filepath.Clean(t.TempDir())

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	state.SetProjectRootURI(root)

	srv := &Server{
		state:            &state,
		activeConfigRoot: root,
		options:          ServerOpts{Diagnostics: DiagnosticsOpts{Enabled: true, Delay: 1, SaveFullIdle: 500, FullMinInterval: 1000}},
		diag: diagnosticsCoordinator{
			workers:         make(map[string]*diagnosticsWorkerState),
			saveDocVersions: make(map[string]int32),
		},
		rootCache: projectRootCacheState{cache: make(map[string]string)},
	}
	srv.resetDiagnosticsSchedulers()

	var commandRuns atomic.Int32
	srv.diagnosticsCommand = func(ctx context.Context, _ c3c.C3Opts, _ string) (bytes.Buffer, bytes.Buffer, error) {
		commandRuns.Add(1)
		select {
		case <-ctx.Done():
			return bytes.Buffer{}, bytes.Buffer{}, ctx.Err()
		case <-time.After(1 * time.Millisecond):
			return bytes.Buffer{}, bytes.Buffer{}, nil
		}
	}

	glspCtx := &glsp.Context{Notify: func(string, any) {}}

	srv.scheduleDiagnosticsFullAfterSaveIdle(glspCtx)
	if !waitForAtLeastRuns(&commandRuns, 1, 3*time.Second) {
		t.Fatalf("expected first deferred full diagnostics run")
	}

	srv.scheduleDiagnosticsFullAfterSaveIdle(glspCtx)
	time.Sleep(650 * time.Millisecond)
	if got := commandRuns.Load(); got != 1 {
		t.Fatalf("expected second run to be blocked by full-min-interval, got runs=%d", got)
	}

	time.Sleep(550 * time.Millisecond)
	srv.scheduleDiagnosticsFullAfterSaveIdle(glspCtx)
	if !waitForAtLeastRuns(&commandRuns, 2, 3*time.Second) {
		t.Fatalf("expected third run after min interval elapsed")
	}
}

func waitForAtLeastRuns(counter *atomic.Int32, target int32, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if counter.Load() >= target {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}

	return counter.Load() >= target
}
