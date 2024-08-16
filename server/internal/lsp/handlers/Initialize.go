package handlers

import (
	"os"

	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (h *Handlers) Initialize(serverName string, serverVersion string, capabilities protocol.ServerCapabilities, context *glsp.Context, params *protocol.InitializeParams) (any, error) {
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
		h.state.SetProjectRootURI(*params.RootURI)
		h.indexWorkspace()
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &serverVersion,
		},
	}, nil
}

func (h *Handlers) indexWorkspace() {
	path, _ := fs.UriToPath(h.state.GetProjectRootURI())
	files, _ := fs.ScanForC3(fs.GetCanonicalPath(path))
	//s.server.Log.Debug(fmt.Sprint("Workspace FILES:", len(files), files))

	for _, filePath := range files {
		//h.language.Debug(fmt.Sprint("Parsing ", filePath))

		content, _ := os.ReadFile(filePath)
		doc := document.NewDocumentFromString(filePath, string(content))
		h.state.RefreshDocumentIdentifiers(&doc, h.parser)
	}
}
