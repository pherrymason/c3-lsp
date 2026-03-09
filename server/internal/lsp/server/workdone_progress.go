package server

import (
	"fmt"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) requestWindowWorkDoneProgressCreate(context *glsp.Context, token protocol.ProgressToken) error {
	if context == nil || context.Call == nil {
		return fmt.Errorf("window/workDoneProgress/create: client call channel unavailable")
	}

	context.Call(protocol.ServerWindowWorkDoneProgressCreate, map[string]any{"token": token.Value}, nil)
	s.markWorkDoneProgressActive(token)
	return nil
}

func (s *Server) supportsWorkDoneProgress() bool {
	if s == nil || s.clientCapabilities.Window == nil || s.clientCapabilities.Window.WorkDoneProgress == nil {
		return false
	}

	return *s.clientCapabilities.Window.WorkDoneProgress
}

func (s *Server) beginWorkDoneProgress(context *glsp.Context, title string, message string, cancellable bool) (protocol.ProgressToken, bool) {
	if context == nil || context.Notify == nil || !s.supportsWorkDoneProgress() {
		return protocol.ProgressToken{}, false
	}

	tokenValue := fmt.Sprintf("c3lsp-progress-%d", s.gate.sequence.Add(1))
	token := protocol.ProgressToken{Value: tokenValue}

	if err := s.requestWindowWorkDoneProgressCreate(context, token); err != nil {
		return protocol.ProgressToken{}, false
	}

	var messagePtr *string
	if message != "" {
		messagePtr = &message
	}

	var cancellablePtr *bool
	if cancellable {
		c := true
		cancellablePtr = &c
	}

	context.Notify(protocol.MethodProgress, map[string]any{
		"token": token.Value,
		"value": map[string]any{
			"kind":        "begin",
			"title":       title,
			"message":     messagePtr,
			"cancellable": cancellablePtr,
		},
	})

	return token, true
}

func (s *Server) reportWorkDoneProgress(context *glsp.Context, token protocol.ProgressToken, message string, percentage *protocol.UInteger) {
	if context == nil || context.Notify == nil || !s.supportsWorkDoneProgress() || s.workDoneProgressWasCanceled(token) {
		return
	}

	var messagePtr *string
	if message != "" {
		messagePtr = &message
	}

	context.Notify(protocol.MethodProgress, map[string]any{
		"token": token.Value,
		"value": map[string]any{
			"kind":       "report",
			"message":    messagePtr,
			"percentage": percentage,
		},
	})
}

func (s *Server) endWorkDoneProgress(context *glsp.Context, token protocol.ProgressToken, message string) {
	if context == nil || context.Notify == nil || !s.supportsWorkDoneProgress() {
		return
	}

	var messagePtr *string
	if message != "" {
		messagePtr = &message
	}

	context.Notify(protocol.MethodProgress, map[string]any{
		"token": token.Value,
		"value": map[string]any{
			"kind":    "end",
			"message": messagePtr,
		},
	})

	s.workDoneProgressMu.Lock()
	delete(s.workDoneProgressActive, progressTokenKey(token))
	s.workDoneProgressMu.Unlock()
}

func (s *Server) WindowWorkDoneProgressCancel(_ *glsp.Context, params *protocol.WorkDoneProgressCancelParams) error {
	if params == nil {
		return nil
	}

	s.workDoneProgressMu.Lock()
	if s.workDoneProgressCanceled == nil {
		s.workDoneProgressCanceled = make(map[string]struct{})
	}
	if s.workDoneProgressActive == nil {
		s.workDoneProgressActive = make(map[string]struct{})
	}
	token := progressTokenKey(params.Token)
	delete(s.workDoneProgressActive, token)
	s.workDoneProgressCanceled[token] = struct{}{}
	s.workDoneProgressMu.Unlock()

	return nil
}

func (s *Server) markWorkDoneProgressActive(token protocol.ProgressToken) {
	if s == nil {
		return
	}

	s.workDoneProgressMu.Lock()
	if s.workDoneProgressActive == nil {
		s.workDoneProgressActive = make(map[string]struct{})
	}
	key := progressTokenKey(token)
	delete(s.workDoneProgressCanceled, key)
	s.workDoneProgressActive[key] = struct{}{}
	s.workDoneProgressMu.Unlock()
}

func (s *Server) workDoneProgressWasCanceled(token protocol.ProgressToken) bool {
	if s == nil {
		return false
	}

	s.workDoneProgressMu.Lock()
	defer s.workDoneProgressMu.Unlock()
	_, ok := s.workDoneProgressCanceled[progressTokenKey(token)]
	return ok
}

func progressTokenKey(token protocol.ProgressToken) string {
	return fmt.Sprintf("%v", token.Value)
}
