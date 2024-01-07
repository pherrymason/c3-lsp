package lsp

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/fs"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
)

type Server struct {
	server    *glspserv.Server
	documents *documentStore
	language  Language
}

// ServerOpts holds the options to create a new Server.
type ServerOpts struct {
	Name    string
	Version string
	LogFile string
	//Logger         *util.ProxyLogger
	//Notebooks      *core.NotebookStore
	//TemplateLoader core.TemplateLoader
	FS fs.FileStorage
}

var log commonlog.Logger

func NewServer(opts ServerOpts) *Server {
	lsName := "C3-LSP"
	version := "0.0.1"

	// This increases logging verbosity (optional)
	commonlog.Configure(2, nil)

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, lsName, true)

	server := &Server{
		server:    glspServer,
		documents: newDocumentStore(opts.FS, &glspServer.Log),
		language:  NewLanguage(),
	}

	handler.Initialized = initialized
	handler.Shutdown = shutdown
	handler.SetTrace = setTrace

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		capabilities := handler.CreateServerCapabilities()

		change := protocol.TextDocumentSyncKindIncremental
		capabilities.TextDocumentSync = protocol.TextDocumentSyncOptions{
			OpenClose: boolPtr(true),
			Change:    &change,
			Save:      boolPtr(true),
		}
		capabilities.DeclarationProvider = true
		server.documents.rootURI = *params.RootURI
		server.indexWorkspace()

		return protocol.InitializeResult{
			Capabilities: capabilities,
			ServerInfo: &protocol.InitializeResultServerInfo{
				Name:    lsName,
				Version: &version,
			},
		}, nil
	}

	handler.TextDocumentDidOpen = func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		doc, err := server.documents.DidOpen(*params, context.Notify)
		if err != nil {
			glspServer.Log.Debug("COULD NOT OPEN!")
			return err
		}

		if doc != nil {
			server.language.RefreshDocumentIdentifiers(doc)
		}

		return nil
	}

	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		doc, ok := server.documents.Get(params.TextDocument.URI)
		if !ok {
			return nil
		}

		doc.ApplyChanges(params.ContentChanges)
		server.language.RefreshDocumentIdentifiers(doc)
		return nil
	}

	handler.TextDocumentDidClose = func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		server.documents.Close(params.TextDocument.URI)
		return nil
	}

	handler.TextDocumentDidSave = func(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
		return nil
	}

	handler.TextDocumentHover = func(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
		doc, ok := server.documents.Get(params.TextDocument.URI)
		if !ok {
			return nil, nil
		}

		server.server.Log.Debug(fmt.Sprint("HOVER requested on ", len(doc.Content), params.Position.IndexIn(doc.Content)))
		word, err := wordInPosition(doc.Content, params.Position.IndexIn(doc.Content))
		if word == "out" {
			err = nil
		}
		if err != nil {
			server.server.Log.Debug(fmt.Sprint("Error trying to find word: ", err))
			return nil, nil
		}

		server.server.Log.Debug(fmt.Sprint("HOVER requested: ", word))

		identifier, err := server.language.FindIdentifierDeclaration(word)
		if err != nil {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "not found",
				},
			}, nil
		}

		var hover protocol.Hover
		switch v := identifier.(type) {
		case indexables.Variable:
			hover = protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: fmt.Sprintf("%s %s", v.GetType(), v.GetName()),
				},
			}

		case indexables.FunctionIndexable:
			hover = protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: fmt.Sprintf("%s", v.GetName()),
				},
			}
		case indexables.Struct:
			fmt.Printf("Struct: %s, Members: %v\n", v.GetName(), v.Members)
		default:
			fmt.Println("Tipo desconocido")
		}

		return &hover, nil
	}

	handler.TextDocumentDeclaration = func(context *glsp.Context, params *protocol.DeclarationParams) (any, error) {
		doc, ok := server.documents.Get(params.TextDocument.URI)
		if !ok {
			return nil, nil
		}

		word, err := wordInPosition(doc.Content, params.Position.IndexIn(doc.Content))
		if err != nil {
			server.server.Log.Debug(fmt.Sprint("Error trying to find word: ", err))

			return nil, nil
		}

		identifier, err := server.language.FindIdentifierDeclaration(word)

		if err == nil {
			return protocol.Location{
				URI: identifier.GetDocumentURI(),
				Range: protocol.Range{
					protocol.Position{identifier.GetDeclarationPosition().Line, identifier.GetDeclarationPosition().Character},
					protocol.Position{identifier.GetDeclarationPosition().Line, identifier.GetDeclarationPosition().Character + 1},
				},
			}, nil
		}

		return nil, nil
	}

	handler.TextDocumentCompletion = func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		doc, ok := server.documents.Get(params.TextDocumentPositionParams.TextDocument.URI)
		if !ok {
			glspServer.Log.Debug(fmt.Sprintf("MIERDERRRR: %s", params.TextDocumentPositionParams.TextDocument.URI))
			return nil, nil
		}

		suggestions := server.language.BuildCompletionList(doc.Content, params.Position.Line+1, params.Position.Character-1)

		return suggestions, nil
	}

	handler.CompletionItemResolve = func(context *glsp.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
		return params, nil
	}

	handler.WorkspaceDidChangeWorkspaceFolders = func(context *glsp.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {

		return nil
	}

	return server
}

// Run starts the Language Server in stdio mode.
func (s *Server) Run() error {
	return errors.Wrap(s.server.RunStdio(), "lsp")
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	/*
		context.Notify(protocol.ServerWorkspaceWorkspaceFolders, protocol.PublishDiagnosticsParams{
			URI:         doc.URI,
			Diagnostics: diagnostics,
		})*/

	return nil
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func (s *Server) indexWorkspace() {
	path, _ := fs.UriToPath(s.documents.rootURI)
	files, _ := fs.ScanForC3(fs.GetCanonicalPath(path))
	s.server.Log.Debug(fmt.Sprint("Workspace FILES:", len(files), files))

	for _, filePath := range files {
		doc := NewDocumentFromFilePath(filePath)
		s.language.RefreshDocumentIdentifiers(&doc)
	}
}
