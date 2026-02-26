package server

import (
	"fmt"
	"os"

	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func initializeWorkspaceURI(params *protocol.InitializeParams) *protocol.DocumentUri {
	if params.RootURI != nil {
		return params.RootURI
	}

	if len(params.WorkspaceFolders) > 0 {
		uri := params.WorkspaceFolders[0].URI
		return &uri
	}

	if params.RootPath != nil {
		uri := protocol.DocumentUri(fs.ConvertPathToURI(*params.RootPath, option.None[string]()))
		return &uri
	}

	return nil
}

// Support "Hover"
func (s *Server) Initialize(serverName string, serverVersion string, capabilities protocol.ServerCapabilities, context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	s.clientCapabilities = params.Capabilities
	//capabilities := handler.CreateServerCapabilities()

	change := protocol.TextDocumentSyncKindIncremental
	capabilities.TextDocumentSync = protocol.TextDocumentSyncOptions{
		OpenClose: cast.ToPtr(true),
		Change:    &change,
		Save:      cast.ToPtr(true),
	}
	capabilities.DeclarationProvider = true
	capabilities.DefinitionProvider = true
	capabilities.TypeDefinitionProvider = true
	capabilities.ImplementationProvider = true
	capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: cast.ToPtr(true)}
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{".", ":"},
	}
	capabilities.SignatureHelpProvider = &protocol.SignatureHelpOptions{
		TriggerCharacters:   []string{"(", ","},
		RetriggerCharacters: []string{")"},
	}
	capabilities.Workspace = &protocol.ServerCapabilitiesWorkspace{
		FileOperations: &protocol.ServerCapabilitiesWorkspaceFileOperations{
			DidDelete: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i}",
					},
				}},
			},
			DidRename: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i}",
					},
				}},
			},
		},
	}

	workspaceURI := initializeWorkspaceURI(params)
	if workspaceURI != nil {
		s.state.SetProjectRootURI(utils.NormalizePath(*workspaceURI))
		path, _ := fs.UriToPath(*workspaceURI)
		s.configureProjectForRoot(path)
		s.loadClientRuntimeConfiguration(context, workspaceURI)
		s.notifyWindowLogMessage(context, protocol.MessageTypeInfo, fmt.Sprintf("C3-LSP loaded workspace: %s", path))
		if isBuildableProjectRoot(path) {
			s.indexWorkspaceAt(path)
			s.RunDiagnostics(s.state, context.Notify, false, nil)
		} else {
			s.notifyWindowLogMessage(context, protocol.MessageTypeInfo, "C3-LSP detected aggregate workspace root; deferring indexing to opened C3 project files")
		}

		if !isBuildableProjectRoot(path) {
			s.notifyWindowLogMessage(context, protocol.MessageTypeInfo, "C3-LSP skipped initial diagnostics: workspace root is not a C3 project root")
		}

		s.indexedRoots[fs.GetCanonicalPath(path)] = true
	}

	// Disable diagnostics only if the client does not support publishDiagnostics at all.
	if params.Capabilities.TextDocument == nil || params.Capabilities.TextDocument.PublishDiagnostics == nil {
		s.options.Diagnostics.Enabled = false
		s.notifyWindowShowMessage(context, protocol.MessageTypeWarning, "C3-LSP diagnostics disabled: client does not support publishDiagnostics")
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &serverVersion,
		},
	}, nil
}

func (h *Server) indexWorkspace() {
	path := h.state.GetProjectRootURI()
	h.indexWorkspaceAt(path)
}

func (h *Server) indexWorkspaceAt(path string) {
	if path == "" {
		return
	}

	files, _ := fs.ScanForC3(fs.GetCanonicalPath(path))

	for _, filePath := range files {
		content, _ := os.ReadFile(filePath)
		doc := document.NewDocumentFromString(filePath, string(content))
		h.state.RefreshDocumentIdentifiers(&doc, h.parser)
	}
}
