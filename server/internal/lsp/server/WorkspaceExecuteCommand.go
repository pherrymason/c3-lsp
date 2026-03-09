package server

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	workspaceCommandReindexWorkspace = "c3lsp.reindexWorkspace"
	workspaceCommandReloadConfig     = "c3lsp.reloadConfiguration"
	workspaceCommandClearDiagnostics = "c3lsp.clearDiagnosticsCache"
)

func workspaceExecuteCommands() []string {
	return []string{
		workspaceCommandReindexWorkspace,
		workspaceCommandReloadConfig,
		workspaceCommandClearDiagnostics,
	}
}

func (s *Server) WorkspaceExecuteCommand(context *glsp.Context, params *protocol.ExecuteCommandParams) (any, error) {
	if params == nil {
		return nil, fmt.Errorf("workspace/executeCommand: params is nil")
	}

	switch params.Command {
	case workspaceCommandReindexWorkspace:
		if err := validateWorkspaceCommandNoArgs(params.Command, params.Arguments); err != nil {
			return nil, err
		}
		return s.executeReindexWorkspace(context)
	case workspaceCommandReloadConfig:
		if err := validateWorkspaceCommandNoArgs(params.Command, params.Arguments); err != nil {
			return nil, err
		}
		return s.executeReloadConfiguration(context)
	case workspaceCommandClearDiagnostics:
		if err := validateWorkspaceCommandNoArgs(params.Command, params.Arguments); err != nil {
			return nil, err
		}
		return s.executeClearDiagnosticsCache(context)
	default:
		return nil, fmt.Errorf("workspace/executeCommand: unknown command %q", params.Command)
	}
}

func validateWorkspaceCommandNoArgs(command string, args []any) error {
	if len(args) > 0 {
		return fmt.Errorf("workspace/executeCommand: command %q does not accept arguments", command)
	}

	return nil
}

func (s *Server) executeReindexWorkspace(context *glsp.Context) (any, error) {
	root := s.currentWorkspaceCommandRoot()
	if root == "" {
		return nil, fmt.Errorf("workspace/executeCommand: %s requires an active workspace root", workspaceCommandReindexWorkspace)
	}

	s.cancelRootIndexing(root)
	s.clearRootTracking(root)
	s.indexWorkspaceAtAsync(root)

	if context != nil {
		s.RunDiagnosticsFull(s.state, context.Notify, false)
	}

	return map[string]any{
		"command": workspaceCommandReindexWorkspace,
		"root":    root,
	}, nil
}

func (s *Server) executeReloadConfiguration(context *glsp.Context) (any, error) {
	root := s.currentWorkspaceCommandRoot()
	if root == "" {
		return nil, fmt.Errorf("workspace/executeCommand: %s requires an active workspace root", workspaceCommandReloadConfig)
	}

	s.reloadWorkspaceConfiguration(context, root, "workspace/executeCommand")
	s.offerProjectConfigOpen(context, root)

	return map[string]any{
		"command": workspaceCommandReloadConfig,
		"root":    root,
	}, nil
}

func (s *Server) executeClearDiagnosticsCache(context *glsp.Context) (any, error) {
	notify := noopNotify
	if context != nil && context.Notify != nil {
		notify = context.Notify
	}

	cleared := s.clearDiagnosticsForFiles(s.state, notify, nil)
	applyEditCalled := false

	if context != nil && context.Call != nil {
		label := workspaceCommandClearDiagnostics
		_, err := s.applyWorkspaceEdit(context, protocol.ApplyWorkspaceEditParams{
			Label: &label,
			Edit:  *emptyWorkspaceEdit(),
		})
		if err != nil {
			return nil, err
		}
		applyEditCalled = true
	}

	return map[string]any{
		"command":            workspaceCommandClearDiagnostics,
		"cleared":            cleared,
		"workspaceApplyEdit": applyEditCalled,
	}, nil
}

func (s *Server) currentWorkspaceCommandRoot() string {
	root := ""
	if s != nil && s.state != nil {
		root = s.state.GetProjectRootURI()
	}
	if root == "" && s != nil {
		root = s.activeConfigRoot
	}

	return fs.GetCanonicalPath(root)
}

func (s *Server) applyWorkspaceEdit(context *glsp.Context, params protocol.ApplyWorkspaceEditParams) (protocol.ApplyWorkspaceEditResponse, error) {
	if context == nil || context.Call == nil {
		return protocol.ApplyWorkspaceEditResponse{}, fmt.Errorf("workspace/applyEdit: client call channel unavailable")
	}

	response := protocol.ApplyWorkspaceEditResponse{}
	context.Call(protocol.ServerWorkspaceApplyEdit, params, &response)
	if !response.Applied {
		if response.FailureReason != nil {
			return response, fmt.Errorf("workspace/applyEdit rejected: %s", *response.FailureReason)
		}
		return response, fmt.Errorf("workspace/applyEdit rejected")
	}

	return response, nil
}
