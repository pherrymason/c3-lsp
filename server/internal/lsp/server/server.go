package server

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"log"
	"time"

	"github.com/bep/debounce"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	ps "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
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
	options ServerOpts
	version string

	documents    *document.Storage
	astConverter *factory.ASTConverter
	symbolTable  *analysis.SymbolTable

	state  *ps.ProjectState // To remove on 0.4 release
	parser *p.Parser
	search search.Search // To remove on 0.4 release

	diagnosticDebounced func(func())
}

func NewServer(opts ServerOpts, appName string, version string) *Server {
	var logPath *string
	if opts.LogFilepath.IsSome() {
		v := opts.LogFilepath.Get()
		logPath = &v
	}

	commonlog.Configure(2, logPath) // This increases logging verbosity (optional)

	logger := commonlog.GetLogger(fmt.Sprintf("%s.parser", appName))

	if opts.SendCrashReports {
		logger.Debug("Sending crash reports")
	} else {
		logger.Debug("No crash reports")
	}

	if opts.C3.Version.IsSome() {
		logger.Debug(fmt.Sprintf("C3 Language version specified: %s", opts.C3.Version.Get()))
	}

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, appName, true)

	requestedLanguageVersion := checkRequestedLanguageVersion(opts.C3.Version)

	state := ps.NewProjectState(logger, option.Some(requestedLanguageVersion.Number), opts.Debug)
	parser := p.NewParser(logger)
	search := search.NewSearch(logger, opts.Debug)

	server := &Server{
		server:  glspServer,
		options: opts,
		version: version,

		documents:    document.NewStore(),
		astConverter: factory.NewASTConverter(),
		symbolTable:  analysis.NewSymbolTable(),

		state:  &state,
		parser: &parser,
		search: search,

		diagnosticDebounced: debounce.New(opts.Diagnostics.Delay * time.Millisecond),
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
			TypeDescription:    protocol.MessageTypeInfo,
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
func (srv *Server) Run() error {
	return errors.Wrap(srv.server.RunStdio(), "lsp")
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

	selectedVersion := supportedVersions[len(supportedVersions)-1]
	log.Printf("Specified c3 language version %s not supported. Default to %s", version.Get(), selectedVersion.Number)

	return selectedVersion
}
