package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) notifyWindowLogMessage(context *glsp.Context, t protocol.MessageType, message string) {
	if context == nil {
		return
	}

	go context.Notify(protocol.ServerWindowLogMessage, protocol.LogMessageParams{
		Type:    t,
		Message: message,
	})
}

func (s *Server) notifyWindowShowMessage(context *glsp.Context, t protocol.MessageType, message string) {
	if context == nil {
		return
	}

	go context.Notify(protocol.ServerWindowShowMessage, protocol.ShowMessageParams{
		Type:    t,
		Message: message,
	})
}
