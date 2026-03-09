package server

import "sync/atomic"

// requestGate controls request concurrency and assigns monotonic request IDs.
type requestGate struct {
	slots    chan struct{}
	sequence atomic.Uint64
}
