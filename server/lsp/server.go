package lsp

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/utils"
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
	FS utils.FileStorage
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
			//server.refreshDiagnosticsOfDocument(doc, context.Notify, false)
		}

		glspServer.Log.Debug("GOOD!!!!!!")

		return nil
	}

	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		doc, ok := server.documents.Get(params.TextDocument.URI)
		if !ok {
			return nil
		}

		doc.ApplyChanges(params.ContentChanges)
		//server.refreshDiagnosticsOfDocument(doc, context.Notify, true)
		return nil
	}

	handler.TextDocumentDidClose = func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		server.documents.Close(params.TextDocument.URI)
		return nil
	}

	handler.TextDocumentDidSave = func(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
		return nil
	}

	handler.TextDocumentDeclaration = func(context *glsp.Context, params *protocol.DeclarationParams) (any, error) {

		//doc, ok := server.documents.Get(params.TextDocument.URI)
		//if !ok {
		//	return nil, nil
		//}

		return nil, nil
	}

	handler.TextDocumentCompletion = func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		doc, ok := server.documents.Get(params.TextDocumentPositionParams.TextDocument.URI)
		if !ok {
			glspServer.Log.Debug(fmt.Sprintf("MIERDERRRR: %s", params.TextDocumentPositionParams.TextDocument.URI))
			return nil, nil
		}

		suggestions := server.language.BuildCompletionList(doc.Content, params.Position.Line+1, params.Position.Character-1)

		/*
			mapped := lo.Map(
				suggestions,
				func(item protocol.CompletionItem) {

				})*/

		return suggestions, nil
	}

	handler.CompletionItemResolve = func(context *glsp.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
		return params, nil
	}

	return server
}

// Run starts the Language Server in stdio mode.
func (s *Server) Run() error {
	return errors.Wrap(s.server.RunStdio(), "lsp")
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
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
