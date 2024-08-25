package server

import (
	"fmt"

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
	server  *glspserv.Server
	version string

	state  *l.ProjectState
	parser *p.Parser
	search search.Search
}

// ServerOpts holds the options to create a new Server.
type ServerOpts struct {
	C3Version        option.Option[string]
	LogFilepath      option.Option[string]
	SendCrashReports bool
	Debug            bool
}

func NewServer(opts ServerOpts, appName string, version string) *Server {
	var logpath *string
	if opts.LogFilepath.IsSome() {
		v := opts.LogFilepath.Get()
		logpath = &v
	}

	commonlog.Configure(2, logpath) // This increases logging verbosity (optional)

	logger := commonlog.GetLogger(fmt.Sprintf("%s.parser", appName))

	if opts.SendCrashReports {
		logger.Debug("Sending crash reports")
	} else {
		logger.Debug("No crash reports")
	}

	if opts.C3Version.IsSome() {
		logger.Debug(fmt.Sprintf("C3 Language version specified: %s", opts.C3Version.Get()))
	}

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, appName, true)

	requestedLanguageVersion := checkRequestedLanguageVersion(opts.C3Version)

	state := l.NewProjectState(logger, option.Some(requestedLanguageVersion.Number), opts.Debug)
	parser := p.NewParser(logger)
	search := search.NewSearch(logger, opts.Debug)

	server := &Server{
		server:  glspServer,
		version: version,

		state:  &state,
		parser: &parser,
		search: search,
	}

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
		return server.Initialize(
			appName,
			server.version,
			capabilities,
			context,
			params,
		)
	}

	handler.TextDocumentDidOpen = server.TextDocumentDidOpen
	handler.TextDocumentDidChange = server.TextDocumentDidChange
	handler.TextDocumentDidClose = server.TextDocumentDidClose
	handler.TextDocumentDidSave = server.TextDocumentDidSave
	handler.TextDocumentHover = server.TextDocumentHover
	handler.TextDocumentDeclaration = server.TextDocumentDeclaration
	handler.TextDocumentDefinition = server.TextDocumentDefinition
	handler.TextDocumentCompletion = server.TextDocumentCompletion
	handler.TextDocumentSignatureHelp = server.TextDocumentSignatureHelp
	handler.WorkspaceDidChangeWatchedFiles = server.WorkspaceDidChangeWatchedFiles
	handler.WorkspaceDidDeleteFiles = server.WorkspaceDidDeleteFiles
	handler.WorkspaceDidRenameFiles = server.WorkspaceDidRenameFiles

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
		if sVersion.Number == "dummy" {
			continue
		}

		compare := semver.Compare("v"+sVersion.Number, "v"+version.Get())
		if compare == 0 {
			return sVersion
		}
	}

	panic("c3 language version not supported")
}
