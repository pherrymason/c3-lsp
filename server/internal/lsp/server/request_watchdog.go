package server

import (
	"fmt"
	"runtime"
	"time"
)

func (s *Server) beginRequestWatchdog(method string, details string) func(err error) {
	threshold := s.options.Runtime.WatchdogThreshold
	dumpGoroutines := s.options.Runtime.WatchdogDump

	if threshold <= 0 {
		return func(err error) {}
	}

	startedAt := time.Now()
	done := make(chan struct{})

	go func() {
		timer := time.NewTimer(threshold)
		defer timer.Stop()

		select {
		case <-done:
			return
		case <-timer.C:
			s.logWatchdog("request exceeded threshold", "threshold", threshold.String(), "method", method, "details", details)
			if dumpGoroutines {
				s.logWatchdog("goroutine dump", "method", method, "details", details, "goroutines", dumpAllGoroutines())
			}
		}
	}()

	return func(err error) {
		close(done)
		elapsed := time.Since(startedAt)
		if elapsed >= threshold {
			status := "ok"
			if err != nil {
				status = "error"
			}
			s.logWatchdog("request completed", "method", method, "status", status, "took", elapsed.String(), "details", details)
		}
	}
}

func (s *Server) logWatchdog(message string, keysAndValues ...any) {
	if s != nil && s.server != nil && s.server.Log != nil {
		s.server.Log.Warning(fmt.Sprintf("[watchdog] %s", message), keysAndValues...)
		return
	}

	// Fallback to stdlib for situations where commonlog is unavailable.
	fmt.Printf("[watchdog] %s %v\n", message, keysAndValues)
}

func dumpAllGoroutines() string {
	buf := make([]byte, 2*1024*1024)
	n := runtime.Stack(buf, true)
	if n <= 0 {
		return ""
	}

	return string(buf[:n])
}
