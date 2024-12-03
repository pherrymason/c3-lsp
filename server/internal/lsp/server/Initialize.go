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
func (srv *Server) Initialize(serverName string, serverVersion string, capabilities protocol.ServerCapabilities, context *glsp.Context, params *protocol.InitializeParams) (any, error) {
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
		srv.state.SetProjectRootURI(utils.NormalizePath(*params.RootURI))
		path, _ := fs.UriToPath(*params.RootURI)
		srv.loadServerConfigurationForWorkspace(path)
		srv.indexWorkspace()
		srv.RunDiagnostics(srv.state, context.Notify, false)
	}

	if params.Capabilities.TextDocument.PublishDiagnostics.RelatedInformation == nil || *params.Capabilities.TextDocument.PublishDiagnostics.RelatedInformation == false {
		srv.options.Diagnostics.Enabled = false
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &serverVersion,
		},
	}, nil
}

func (srv *Server) indexWorkspace() {
	path := srv.state.GetProjectRootURI()
	files, _ := fs.ScanForC3(fs.GetCanonicalPath(path))

	for _, filePath := range files {
		content, _ := os.ReadFile(filePath)
		doc := document.NewDocumentFromString(filePath, string(content))
		srv.state.RefreshDocumentIdentifiers(&doc, srv.parser)
	}
}
