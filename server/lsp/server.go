package lsp

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/fs"
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/handlers"
	l "github.com/pherrymason/c3-lsp/lsp/language"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
)

type Server struct {
	server *glspserv.Server
}

// ServerOpts holds the options to create a new Server.
type ServerOpts struct {
	Name        string
	Version     string
	LogFilepath string
	FS          fs.FileStorage
}

func NewServer(opts ServerOpts) *Server {
	serverName := "C3-LSP"
	serverVersion := "0.0.1"

	// This increases logging verbosity (optional)
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger(fmt.Sprintf("%s.parser", serverName))

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, serverName, true)

	documents := document.NewDocumentStore(opts.FS, &glspServer.Log)
	language := l.NewLanguage(logger)
	parser := p.NewParser(&logger)
	handlers := handlers.NewHandlers(documents, &language, &parser)

	handler.Initialized = initialized
	handler.Shutdown = shutdown
	handler.SetTrace = setTrace

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		capabilities := handler.CreateServerCapabilities()
		return handlers.Initialize(
			serverName,
			serverVersion,
			capabilities,
			context,
			params,
		)
	}

	handler.TextDocumentDidOpen = handlers.TextDocumentDidOpen
	handler.TextDocumentDidChange = handlers.TextDocumentDidChange
	handler.TextDocumentDidClose = handlers.TextDocumentDidClose
	handler.TextDocumentDidSave = handlers.TextDocumentDidSave
	handler.TextDocumentHover = handlers.TextDocumentHover
	handler.TextDocumentDeclaration = handlers.TextDocumentDeclaration
	handler.TextDocumentDefinition = handlers.TextDocumentDefinition
	handler.TextDocumentCompletion = handlers.TextDocumentCompletion

	handler.CompletionItemResolve = func(context *glsp.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
		return params, nil
	}

	handler.WorkspaceDidChangeWorkspaceFolders = func(context *glsp.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {

		return nil
	}

	server := &Server{
		server: glspServer,
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
