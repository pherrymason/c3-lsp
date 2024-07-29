package lsp

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/internal/lsp/handlers"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
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
	LogFilepath      option.Option[string]
	SendCrashReports bool
	Debug            bool
}

func NewServer(opts ServerOpts) *Server {
	var logpath *string
	if opts.LogFilepath.IsSome() {
		v := opts.LogFilepath.Get()
		logpath = &v
	}

	commonlog.Configure(2, logpath) // This increases logging verbosity (optional)

	logger := commonlog.GetLogger(fmt.Sprintf("%s.parser", opts.Name))

	if opts.SendCrashReports {
		logger.Debug("Sending crash reports")
	} else {
		logger.Debug("No crash reports")
	}

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, opts.Name, true)

	//documents := document.NewDocumentStore(fs.FileStorage{})

	requestedLanguageVersion := checkRequestedLanguageVersion(opts.C3Version)

	state := l.NewProjectState(logger, option.Some(requestedLanguageVersion.Number), opts.Debug)
	parser := p.NewParser(logger)
	search := search.NewSearch(logger, opts.Debug)
	handlers := handlers.NewHandlers(&state, &parser, search)

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
	handler.TextDocumentSignatureHelp = handlers.TextDocumentSignatureHelp
	handler.WorkspaceDidChangeWatchedFiles = handlers.WorkspaceDidChangeWatchedFiles
	handler.WorkspaceDidDeleteFiles = handlers.WorkspaceDidDeleteFiles
	handler.WorkspaceDidRenameFiles = handlers.WorkspaceDidRenameFiles

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

func checkRequestedLanguageVersion(version option.Option[string]) project_state.Version {
	supportedVersions := project_state.SupportedVersions()

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
