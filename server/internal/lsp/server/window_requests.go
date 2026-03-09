package server

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const openProjectConfigActionTitle = "Open project.json"

func (s *Server) requestWindowShowMessage(
	context *glsp.Context,
	t protocol.MessageType,
	message string,
	actions []protocol.MessageActionItem,
) (*protocol.MessageActionItem, error) {
	if context == nil || context.Call == nil {
		return nil, fmt.Errorf("window/showMessageRequest: client call channel unavailable")
	}

	params := protocol.ShowMessageRequestParams{Type: t, Message: message, Actions: actions}
	var response *protocol.MessageActionItem
	context.Call(protocol.ServerWindowShowMessageRequest, params, &response)

	return response, nil
}

func (s *Server) requestWindowShowDocument(context *glsp.Context, params protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	if context == nil || context.Call == nil {
		return nil, fmt.Errorf("window/showDocument: client call channel unavailable")
	}
	if s != nil && s.clientCapabilities.Window != nil && s.clientCapabilities.Window.ShowDocument != nil {
		if !s.clientCapabilities.Window.ShowDocument.Support {
			return nil, fmt.Errorf("window/showDocument: client does not advertise support")
		}
	}

	result := protocol.ShowDocumentResult{}
	context.Call(protocol.ServerWindowShowDocument, params, &result)

	return &result, nil
}

func (s *Server) offerProjectConfigOpen(context *glsp.Context, projectRoot string) {
	if projectRoot == "" {
		return
	}

	projectConfigPath := filepath.Join(projectRoot, "project.json")
	if _, err := os.Stat(projectConfigPath); err != nil {
		return
	}

	action, err := s.requestWindowShowMessage(
		context,
		protocol.MessageTypeError,
		"C3 project configuration error detected. Open project.json to inspect it?",
		[]protocol.MessageActionItem{{Title: openProjectConfigActionTitle}},
	)
	if err != nil || action == nil || action.Title != openProjectConfigActionTitle {
		return
	}

	_, _ = s.requestWindowShowDocument(context, protocol.ShowDocumentParams{
		URI: protocol.URI(fs.ConvertPathToURI(projectConfigPath, option.None[string]())),
	})
}
