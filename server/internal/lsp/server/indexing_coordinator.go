package server

import (
	"context"
	"sync"
)

// indexingCoordinator groups all workspace-indexing state fields.
type indexingCoordinator struct {
	mu          sync.Mutex
	indexed     map[string]bool
	indexing    map[string]bool
	cancels     map[string]context.CancelFunc
	rootStates  map[string]workspaceIndexState
	generations map[string]uint64
}
