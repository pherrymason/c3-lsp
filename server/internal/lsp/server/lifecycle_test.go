package server

import (
	"errors"
	"testing"
)

func TestShouldProcessNotification_requires_initialized_state(t *testing.T) {
	srv := &Server{}

	if srv.shouldProcessNotification("textDocument/didOpen") {
		t.Fatalf("expected notifications to be ignored before initialize")
	}

	srv.initialized.Store(true)
	if !srv.shouldProcessNotification("textDocument/didOpen") {
		t.Fatalf("expected notifications to be processed after initialize")
	}

	srv.shutdownRequested.Store(true)
	if srv.shouldProcessNotification("textDocument/didOpen") {
		t.Fatalf("expected notifications to be ignored after shutdown")
	}
}

func TestShutdown_marks_server_not_ready_for_requests(t *testing.T) {
	srv := &Server{}
	srv.initialized.Store(true)

	if err := srv.shutdown(nil); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}

	if srv.isReadyForRequests() {
		t.Fatalf("expected server to be not ready after shutdown")
	}
}

func TestExit_requires_prior_shutdown(t *testing.T) {
	srv := &Server{}

	if err := srv.exit(nil); err == nil {
		t.Fatalf("expected exit-before-shutdown to return error")
	} else if !errors.Is(err, errExitBeforeShutdown) {
		t.Fatalf("expected errExitBeforeShutdown, got: %v", err)
	}

	if err := srv.shutdown(nil); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
	if err := srv.exit(nil); err != nil {
		t.Fatalf("expected clean exit after shutdown, got: %v", err)
	}
}
