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
	canonicalPath := fs.GetCanonicalPath(path)

	// Index workspace source files
	files, _ := fs.ScanForC3(canonicalPath)
	for _, filePath := range files {
		content, _ := os.ReadFile(filePath)
		doc := document.NewDocumentFromString(filePath, string(content))
		h.state.RefreshDocumentIdentifiers(&doc, h.parser)
	}

	// Index dependency libraries from project.json
	h.indexDependencies(canonicalPath)
}

// indexDependencies reads the project.json file in the workspace root,
// resolves declared dependencies by searching dependency-search-paths
// for .c3l library directories, and indexes all .c3/.c3i files found
// in those libraries.
func (h *Server) indexDependencies(projectDir string) {
	config, err := fs.ReadC3ProjectConfig(projectDir)
	if err != nil {
		h.server.Log.Warningf("Failed to read project.json: %v", err)
		return
	}

	if config == nil {
		// No project.json found — nothing to resolve
		return
	}

	if len(config.Dependencies) == 0 {
		return
	}

	h.server.Log.Infof("project.json: found %d dependencies: %v", len(config.Dependencies), config.Dependencies)
	h.server.Log.Infof("project.json: dependency search paths: %v", config.DependencySearchPaths)

	resolutions := fs.ResolveDependencies(projectDir, config)

	for _, res := range resolutions {
		if !res.Found {
			h.server.Log.Warningf("Dependency '%s' not found in any search path. Autocompletion for this library will not be available.", res.Name)
			h.server.Log.Warningf("  Searched paths: %v", config.DependencySearchPaths)
			continue
		}

		h.server.Log.Infof("Resolved dependency '%s' at: %s", res.Name, res.Path)

		depFiles, err := fs.ScanDependencyForC3(res.Path)
		if err != nil {
			h.server.Log.Warningf("Failed to scan dependency '%s' at %s: %v", res.Name, res.Path, err)
			continue
		}

		h.server.Log.Infof("  Found %d source files in dependency '%s'", len(depFiles), res.Name)

		for _, filePath := range depFiles {
			content, err := os.ReadFile(filePath)
			if err != nil {
				h.server.Log.Warningf("  Failed to read %s: %v", filePath, err)
				continue
			}
			doc := document.NewDocumentFromString(filePath, string(content))
			h.state.RefreshDocumentIdentifiers(&doc, h.parser)
		}
	}
}
