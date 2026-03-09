package server

import (
	"context"
	"fmt"
	"time"

	"github.com/tliron/glsp"
)

type timedResult[T any] struct {
	value T
	err   error
}

func (s *Server) acquireRequestSlot(method string, details string, wait time.Duration) (func(), bool) {
	if s == nil || s.gate.slots == nil {
		return func() {}, true
	}

	if wait <= 0 {
		wait = 250 * time.Millisecond
	}

	select {
	case s.gate.slots <- struct{}{}:
		return func() { <-s.gate.slots }, true
	case <-time.After(wait):
		if s.server != nil && s.server.Log != nil {
			s.server.Log.Warning("request dropped due to inflight limit", "method", method, "wait", wait.String(), "details", details)
		}
		return func() {}, false
	}
}

func runWithRequestTimeout[T any](s *Server, requestID uint64, method string, details string, fallback T, fn func(context.Context) (T, error)) (T, error) {
	var timeout time.Duration
	var slowThreshold time.Duration
	if s != nil {
		timeout = s.options.Runtime.RequestTimeout
		slowThreshold = s.options.Runtime.SlowRequestThreshold
	}

	startedAt := time.Now()
	logSlow := func(status string, err error) {
		if slowThreshold <= 0 {
			return
		}

		elapsed := time.Since(startedAt)
		if elapsed < slowThreshold {
			return
		}

		if s != nil && s.server != nil && s.server.Log != nil {
			if err != nil {
				s.server.Log.Warning("slow request", "method", method, "status", status, "took", elapsed.String(), "threshold", slowThreshold.String(), "err", err, "details", details)
				return
			}
			s.server.Log.Warning("slow request", "method", method, "status", status, "took", elapsed.String(), "threshold", slowThreshold.String(), "details", details)
		}
	}

	slotWait := 250 * time.Millisecond
	if timeout > 0 && timeout < slotWait {
		slotWait = timeout
	}
	releaseSlot, acquired := s.acquireRequestSlot(method, details, slotWait)
	if !acquired {
		logSlow("dropped", nil)
		return fallback, nil
	}
	defer releaseSlot()

	var baseCtx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		baseCtx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		baseCtx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// Thread request_id into the context so downstream handlers can access it.
	requestCtx := context.WithValue(baseCtx, contextKeyRequestID{}, requestID)

	if timeout <= 0 {
		value, err := fn(requestCtx)
		status := "ok"
		if err != nil {
			status = "error"
		}
		logSlow(status, err)
		return value, err
	}

	ch := make(chan timedResult[T], 1)
	go func() {
		var zero T
		defer func() {
			if recovered := recover(); recovered != nil {
				ch <- timedResult[T]{value: zero, err: fmt.Errorf("%s panic: %v", method, recovered)}
			}
		}()

		value, err := fn(requestCtx)
		ch <- timedResult[T]{value: value, err: err}
	}()

	select {
	case result := <-ch:
		status := "ok"
		if result.err != nil {
			status = "error"
		}
		logSlow(status, result.err)
		return result.value, result.err
	case <-time.After(timeout):
		cancel()
		if s != nil && s.server != nil && s.server.Log != nil {
			s.server.Log.Warning("request timeout", "method", method, "timeout", timeout.String(), "details", details)
			if s.options.Runtime.RequestTimeoutDump {
				s.server.Log.Warning("request timeout goroutine dump", "method", method, "details", details, "goroutines", dumpAllGoroutines())
			}
		}
		logSlow("timeout", nil)
		return fallback, nil
	}
}

func runGuardedRequest[T any](
	s *Server,
	glspContext *glsp.Context,
	method string,
	details string,
	fallback T,
	fn func(context.Context) (T, error),
) (result T, err error) {
	if !s.isReadyForRequests() {
		if s != nil && s.server != nil && s.server.Log != nil {
			s.server.Log.Warning("rejecting request", "method", method, "reason", errServerNotReady, "details", details)
		}
		return fallback, errServerNotReady
	}

	// Extract the request_id from the details string (it was placed there by
	// withRequestID and is always the first token: "request_id=N ...").
	var requestID uint64
	if s != nil {
		_, _ = fmt.Sscanf(details, "request_id=%d", &requestID)
	}

	finishWatchdog := s.beginRequestWatchdog(method, details)
	defer func() { finishWatchdog(err) }()
	defer s.recoverRequestPanic(glspContext, method, &err)

	return runWithRequestTimeout(s, requestID, method, details, fallback, fn)
}
