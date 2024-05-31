package lsp

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/fs"
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/handlers"
	"github.com/pherrymason/c3-lsp/lsp/language"
	l "github.com/pherrymason/c3-lsp/lsp/language"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/pherrymason/c3-lsp/option"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
	"golang.org/x/mod/semver"
)

type Server struct {
	server *glspserv.Server
}

// ServerOpts holds the options to create a new Server.
type ServerOpts struct {
	Name             string
	Version          string
	C3Version        option.Option[string]
	LogFilepath      string
	FS               fs.FileStorage
	logger           commonlog.Logger
	SendCrashReports bool
}

func NewServer(opts ServerOpts) *Server {
	// This increases logging verbosity (optional)
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger(fmt.Sprintf("%s.parser", opts.Name))

	if opts.SendCrashReports {
		logger.Debug("Sending crash reports")
	} else {
		logger.Debug("No crash reports")
	}

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, opts.Name, true)

	documents := document.NewDocumentStore(opts.FS, &glspServer.Log)

	requestedLanguageVersion := checkRequestedLanguageVersion(opts.C3Version)

	language := l.NewLanguage(logger, option.Some(requestedLanguageVersion.Number))
	parser := p.NewParser(logger)
	handlers := handlers.NewHandlers(documents, &language, &parser)

	handler.Initialized = func(context *glsp.Context, params *protocol.InitializedParams) error {
		/*
			context.Notify(protocol.ServerWorkspaceWorkspaceFolders, protocol.PublishDiagnosticsParams{
				URI:         doc.URI,
				Diagnostics: diagnostics,
			})*/
		/*sendCrashStatus := "disabled"
		if opts.SendCrashReports {
			sendCrashStatus = "enabled"
		}

		context.Notify(protocol.ServerWindowShowMessage, protocol.ShowMessageParams{
			Type:    protocol.MessageTypeInfo,
			Message: fmt.Sprintf("SendCrash: %s", sendCrashStatus),
		})
		*/
		return nil
	}
	handler.Shutdown = shutdown
	handler.SetTrace = setTrace

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		capabilities := handler.CreateServerCapabilities()
		return handlers.Initialize(
			opts.Name,
			opts.Version,
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

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func checkRequestedLanguageVersion(version option.Option[string]) language.Version {
	supportedVersions := language.SupportedVersions()

	if version.IsNone() {
		return supportedVersions[len(supportedVersions)-1]
	}

	for _, sVersion := range supportedVersions {
		compare := semver.Compare(sVersion.Number, version.Get())
		if compare == 0 {
			return sVersion
		}
	}

	panic("c3 language not supported")
}
