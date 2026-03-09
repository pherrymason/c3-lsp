package server

import "time"

type diagnosticsWorkerState struct {
	pending             diagnosticsQueuedRun
	inFlightFingerprint string
	wakeCh              chan struct{}
}

type diagnosticsQueuedRun struct {
	fingerprint string
	run         func()
}

func (s *Server) enqueueDiagnosticsRun(root string, fingerprint string, run func()) bool {
	if s == nil || run == nil {
		return false
	}

	if root == "" {
		run()
		return true
	}

	s.diag.queueMu.Lock()
	if s.diag.workers == nil {
		s.diag.workers = make(map[string]*diagnosticsWorkerState)
	}
	worker, ok := s.diag.workers[root]
	if !ok {
		worker = &diagnosticsWorkerState{wakeCh: make(chan struct{}, 1)}
		s.diag.workers[root] = worker
		go s.runDiagnosticsWorker(root, worker)
	}
	if worker.pending.run != nil && worker.pending.fingerprint == fingerprint {
		s.diag.queueMu.Unlock()
		return false
	}
	if worker.inFlightFingerprint != "" && worker.inFlightFingerprint == fingerprint {
		s.diag.queueMu.Unlock()
		return false
	}
	worker.pending = diagnosticsQueuedRun{fingerprint: fingerprint, run: run}
	s.diag.queueMu.Unlock()

	select {
	case worker.wakeCh <- struct{}{}:
	default:
	}

	return true
}

func (s *Server) runDiagnosticsWorker(root string, worker *diagnosticsWorkerState) {
	idleTTL := s.diag.workerIdleTTL
	if idleTTL <= 0 {
		idleTTL = 2 * time.Minute
	}

	idleTimer := time.NewTimer(idleTTL)
	defer idleTimer.Stop()

	for {
		select {
		case <-worker.wakeCh:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			for {
				queued := s.dequeueDiagnosticsRun(root, worker)
				if queued.run == nil {
					break
				}
				queued.run()
				s.finishDiagnosticsRun(root, worker, queued.fingerprint)
			}
			idleTimer.Reset(idleTTL)
		case <-idleTimer.C:
			if s.tryRetireDiagnosticsWorker(root, worker) {
				return
			}
			idleTimer.Reset(idleTTL)
		}
	}
}

func (s *Server) dequeueDiagnosticsRun(root string, worker *diagnosticsWorkerState) diagnosticsQueuedRun {
	s.diag.queueMu.Lock()
	defer s.diag.queueMu.Unlock()

	current, ok := s.diag.workers[root]
	if !ok || current != worker {
		return diagnosticsQueuedRun{}
	}

	queued := worker.pending
	worker.pending = diagnosticsQueuedRun{}
	if queued.run != nil {
		worker.inFlightFingerprint = queued.fingerprint
	}
	return queued
}

func (s *Server) finishDiagnosticsRun(root string, worker *diagnosticsWorkerState, fingerprint string) {
	if fingerprint == "" {
		return
	}

	s.diag.queueMu.Lock()
	defer s.diag.queueMu.Unlock()

	current, ok := s.diag.workers[root]
	if !ok || current != worker {
		return
	}
	if worker.inFlightFingerprint == fingerprint {
		worker.inFlightFingerprint = ""
	}
}

func (s *Server) tryRetireDiagnosticsWorker(root string, worker *diagnosticsWorkerState) bool {
	s.diag.queueMu.Lock()
	defer s.diag.queueMu.Unlock()

	current, ok := s.diag.workers[root]
	if !ok || current != worker {
		return true
	}

	if worker.pending.run != nil || worker.inFlightFingerprint != "" {
		return false
	}

	delete(s.diag.workers, root)
	return true
}

func (s *Server) diagnosticsQueueWorkerCount() int {
	if s == nil {
		return 0
	}

	s.diag.queueMu.Lock()
	defer s.diag.queueMu.Unlock()
	return len(s.diag.workers)
}
