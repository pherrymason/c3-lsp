package server

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestEnqueueDiagnosticsRun_coalescesLatestPendingPerRoot(t *testing.T) {
	srv := &Server{diag: diagnosticsCoordinator{workers: make(map[string]*diagnosticsWorkerState)}}

	firstStarted := make(chan struct{}, 1)
	releaseFirst := make(chan struct{})
	thirdDone := make(chan struct{}, 1)

	var firstCount atomic.Int32
	var secondCount atomic.Int32
	var thirdCount atomic.Int32

	srv.enqueueDiagnosticsRun("/tmp/root", "f1", func() {
		firstCount.Add(1)
		select {
		case firstStarted <- struct{}{}:
		default:
		}
		<-releaseFirst
	})

	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatalf("first diagnostics run did not start")
	}

	srv.enqueueDiagnosticsRun("/tmp/root", "f2", func() {
		secondCount.Add(1)
	})
	srv.enqueueDiagnosticsRun("/tmp/root", "f3", func() {
		thirdCount.Add(1)
		select {
		case thirdDone <- struct{}{}:
		default:
		}
	})

	close(releaseFirst)

	select {
	case <-thirdDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("latest pending diagnostics run did not execute")
	}

	if firstCount.Load() != 1 {
		t.Fatalf("expected first diagnostics run once, got %d", firstCount.Load())
	}
	if secondCount.Load() != 0 {
		t.Fatalf("expected intermediate pending run to be coalesced, got %d", secondCount.Load())
	}
	if thirdCount.Load() != 1 {
		t.Fatalf("expected latest pending run to execute once, got %d", thirdCount.Load())
	}
}

func TestEnqueueDiagnosticsRun_dedupesEquivalentPendingAndInFlightFingerprint(t *testing.T) {
	srv := &Server{diag: diagnosticsCoordinator{workers: make(map[string]*diagnosticsWorkerState)}}

	started := make(chan struct{}, 1)
	releaseRun := make(chan struct{})
	finished := make(chan struct{}, 2)

	var runCount atomic.Int32

	firstAccepted := srv.enqueueDiagnosticsRun("/tmp/root", "same", func() {
		runCount.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-releaseRun
		select {
		case finished <- struct{}{}:
		default:
		}
	})
	if !firstAccepted {
		t.Fatalf("expected first enqueue to be accepted")
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected diagnostics run to start")
	}

	if accepted := srv.enqueueDiagnosticsRun("/tmp/root", "same", func() { runCount.Add(1) }); accepted {
		t.Fatalf("expected in-flight duplicate fingerprint to be dropped")
	}

	if accepted := srv.enqueueDiagnosticsRun("/tmp/root", "other", func() {
		runCount.Add(1)
		select {
		case finished <- struct{}{}:
		default:
		}
	}); !accepted {
		t.Fatalf("expected different fingerprint to be accepted")
	}

	if accepted := srv.enqueueDiagnosticsRun("/tmp/root", "other", func() { runCount.Add(1) }); accepted {
		t.Fatalf("expected pending duplicate fingerprint to be dropped")
	}

	close(releaseRun)

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected first diagnostics run to finish")
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected second diagnostics run to finish")
	}

	if runCount.Load() != 2 {
		t.Fatalf("expected only first and distinct pending runs to execute, got %d", runCount.Load())
	}
}

func TestDiagnosticsWorker_retiresAfterIdleTimeout(t *testing.T) {
	srv := &Server{
		diag: diagnosticsCoordinator{
			workers:       make(map[string]*diagnosticsWorkerState),
			workerIdleTTL: 20 * time.Millisecond,
		},
	}

	done := make(chan struct{}, 1)
	accepted := srv.enqueueDiagnosticsRun("/tmp/root-idle", "one", func() {
		select {
		case done <- struct{}{}:
		default:
		}
	})
	if !accepted {
		t.Fatalf("expected diagnostics run to be accepted")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected diagnostics run to execute")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if srv.diagnosticsQueueWorkerCount() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected idle diagnostics worker to retire")
}
