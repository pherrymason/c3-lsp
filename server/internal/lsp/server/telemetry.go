package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) notifyTelemetryEvent(context *glsp.Context, event string, data any) {
	if context == nil || event == "" {
		return
	}

	payload := map[string]any{
		"event": event,
	}
	if data != nil {
		payload["data"] = data
	}

	go context.Notify(protocol.ServerTelemetryEvent, payload)
}
