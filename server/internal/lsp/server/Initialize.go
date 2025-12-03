package server

import (
	"os"

	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (s *Server) Initialize(serverName string, serverVersion string, capabilities protocol.ServerCapabilities, context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	//capabilities := handler.CreateServerCapabilities()

	change := protocol.TextDocumentSyncKindIncremental
	capabilities.TextDocumentSync = protocol.TextDocumentSyncOptions{
		OpenClose: cast.ToPtr(true),
		Change:    &change,
		Save:      cast.ToPtr(true),
	}
	capabilities.DeclarationProvider = true
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

	if params.RootURI != nil {
		s.state.SetProjectRootURI(utils.NormalizePath(*params.RootURI))
		path, _ := fs.UriToPath(*params.RootURI)
		s.loadServerConfigurationForWorkspace(path)
		s.indexWorkspace()

		s.RunDiagnostics(s.state, context.Notify, false)
	}

	// Disable diagnostics only if the client does not support publishDiagnostics at all.
	if params.Capabilities.TextDocument == nil || params.Capabilities.TextDocument.PublishDiagnostics == nil {
		s.options.Diagnostics.Enabled = false
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
	files, _ := fs.ScanForC3(fs.GetCanonicalPath(path))

	for _, filePath := range files {
		content, _ := os.ReadFile(filePath)
		doc := document.NewDocumentFromString(filePath, string(content))
		h.state.RefreshDocumentIdentifiers(&doc, h.parser)
	}
}
