package server

import "errors"

var (
	errServerNotReady     = errors.New("server not initialized or already shutting down")
	errExitBeforeShutdown = errors.New("received exit notification before shutdown")
)
