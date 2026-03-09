package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// diagnosticsCoordinator groups all diagnostics-scheduling fields.
type diagnosticsCoordinator struct {
	quickDebounced    func(func())
	fullDebounced     func(func())
	saveFullDebounced func(func())
	generation        atomic.Uint64

	saveMu          sync.Mutex
	saveDocVersions map[string]int32
	lastSaveFullNs  int64

	queueMu       sync.Mutex
	workers       map[string]*diagnosticsWorkerState
	workerIdleTTL time.Duration

	runMu     sync.Mutex
	runCancel context.CancelFunc
	runID     uint64
}
