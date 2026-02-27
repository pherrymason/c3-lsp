package server

import (
	stderrors "errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/bep/debounce"
	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_v2"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
)

type Server struct {
	server  *glspserv.Server
	options ServerOpts
	version string

	state  *l.ProjectState
	parser *p.Parser
	search search.SearchInterface

	clientCapabilities protocol.ClientCapabilities

	diagnosticDebounced func(func())
	workspaceC3Options  c3c.C3Opts
	activeConfigRoot    string
	indexedRoots        map[string]bool
}

// ServerOpts holds the options to create a new Server.
/*type ServerOpts struct {
	C3Version   option.Option[string]
	C3CPath     option.Option[string]
	LogFilepath option.Option[string]

	DiagnosticsDelay   time.Duration
	DiagnosticsEnabled bool

	SendCrashReports bool
	Debug            bool
}*/

func NewServer(opts ServerOpts, appName string, version string) *Server {
	var logpath *string
	if opts.LogFilepath.IsSome() {
		v := opts.LogFilepath.Get()
		logpath = &v
	}

	commonlog.Configure(2, logpath) // This increases logging verbosity (optional)

	logger := commonlog.GetLogger(appName)
	logger.Infof(fmt.Sprintf("%s version %s", appName, version))

	if executable, err := os.Executable(); err == nil {
		logger.Infof(fmt.Sprintf("Server executable: %s", executable))
	}

	if opts.SendCrashReports {
		logger.Debug("Crash reports enabled")
	} else {
		logger.Debug("Crash reports disabled")
	}

	if opts.C3.Version.IsSome() {
		logger.Infof(fmt.Sprintf("C3 Language version specified: %s", opts.C3.Version.Get()))
	}

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, appName, true)

	requestedLanguageVersion := checkRequestedLanguageVersion(logger, opts.C3.Version)

	state := l.NewProjectState(logger, option.Some(requestedLanguageVersion), opts.Debug)
	parser := p.NewParser(logger)

	// Instantiate search implementation based on feature flag
	var searchImpl search.SearchInterface
	if utils.IsFeatureEnabled("USE_SEARCH_V2") {
		logger.Info("Using SearchV2 implementation (new architecture)")
		searchImpl = search_v2.NewSearchV2(logger, opts.Debug)
	} else {
		logger.Info("Using original Search implementation")
		searchV1 := search.NewSearch(logger, opts.Debug)
		searchImpl = &searchV1
	}

	server := &Server{
		server:  glspServer,
		options: opts,
		version: version,

		state:  &state,
		parser: &parser,
		search: searchImpl,

		diagnosticDebounced: debounce.New(opts.Diagnostics.Delay * time.Millisecond),
		workspaceC3Options:  cloneC3Opts(opts.C3),
		indexedRoots:        make(map[string]bool),
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
		if params.Trace != nil {
			protocol.SetTraceValue(*params.Trace)
			notifyLogTrace(context, fmt.Sprintf("Initial trace set to %s", protocol.GetTraceValue()), nil)
		}

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
	handler.TextDocumentHover = func(context *glsp.Context, params *protocol.HoverParams) (result *protocol.Hover, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentHover, &err)
		return server.TextDocumentHover(context, params)
	}
	handler.TextDocumentDeclaration = func(context *glsp.Context, params *protocol.DeclarationParams) (result any, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentDeclaration, &err)
		return server.TextDocumentDeclaration(context, params)
	}
	handler.TextDocumentDefinition = func(context *glsp.Context, params *protocol.DefinitionParams) (result any, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentDefinition, &err)
		return server.TextDocumentDefinition(context, params)
	}
	handler.TextDocumentTypeDefinition = func(context *glsp.Context, params *protocol.TypeDefinitionParams) (result any, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentTypeDefinition, &err)
		return server.TextDocumentTypeDefinition(context, params)
	}
	handler.TextDocumentImplementation = func(context *glsp.Context, params *protocol.ImplementationParams) (result any, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentImplementation, &err)
		return server.TextDocumentImplementation(context, params)
	}
	handler.TextDocumentPrepareRename = func(context *glsp.Context, params *protocol.PrepareRenameParams) (result any, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentPrepareRename, &err)
		return server.TextDocumentPrepareRename(context, params)
	}
	handler.TextDocumentRename = func(context *glsp.Context, params *protocol.RenameParams) (result *protocol.WorkspaceEdit, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentRename, &err)
		return server.TextDocumentRename(context, params)
	}
	handler.TextDocumentCompletion = func(context *glsp.Context, params *protocol.CompletionParams) (result any, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentCompletion, &err)
		return server.TextDocumentCompletion(context, params)
	}
	handler.TextDocumentSignatureHelp = func(context *glsp.Context, params *protocol.SignatureHelpParams) (result *protocol.SignatureHelp, err error) {
		defer server.recoverRequestPanic(context, protocol.MethodTextDocumentSignatureHelp, &err)
		return server.TextDocumentSignatureHelp(context, params)
	}
	handler.WorkspaceDidChangeWatchedFiles = server.WorkspaceDidChangeWatchedFiles
	handler.WorkspaceDidChangeConfiguration = server.WorkspaceDidChangeConfiguration
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

func cloneC3Opts(input c3c.C3Opts) c3c.C3Opts {
	clone := input
	if input.CompileArgs != nil {
		clone.CompileArgs = append([]string(nil), input.CompileArgs...)
	}

	return clone
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
	previous := protocol.GetTraceValue()
	protocol.SetTraceValue(params.Value)

	verbose := fmt.Sprintf("Previous trace value: %s", previous)
	notifyLogTrace(context, fmt.Sprintf("Trace set to %s", protocol.GetTraceValue()), &verbose)

	return nil
}

func notifyLogTrace(context *glsp.Context, message string, verbose *string) {
	if context == nil || protocol.GetTraceValue() == protocol.TraceValueOff {
		return
	}

	params := protocol.LogTraceParams{Message: message}
	if verbose != nil && protocol.GetTraceValue() == protocol.TraceValueVerbose {
		params.Verbose = verbose
	}

	go context.Notify(protocol.MethodLogTrace, params)
}

func (s *Server) recoverRequestPanic(context *glsp.Context, method string, errp *error) {
	if recovered := recover(); recovered != nil {
		msg := fmt.Sprintf("panic recovered in %s: %v", method, recovered)
		stack := debug.Stack()
		log.Printf("%s\n%s", msg, string(stack))
		s.notifyWindowLogMessage(context, protocol.MessageTypeError, msg)
		*errp = stderrors.New(msg)
	}
}

func checkRequestedLanguageVersion(logger commonlog.Logger, version option.Option[string]) string {
	// Default to supported version if not specified
	if version.IsNone() {
		logger.Infof("Using default C3 version: %s", project_state.SupportedC3Version)
		return project_state.SupportedC3Version
	}

	requestedVersion := version.Get()

	// Warn if requested version doesn't match officially supported version
	if requestedVersion != project_state.SupportedC3Version {
		logger.Warningf("Requested C3 version %s differs from officially supported version %s",
			requestedVersion, project_state.SupportedC3Version)
		logger.Warningf("The LSP will attempt to load stdlib for version %s", requestedVersion)
		logger.Warning("Correct behavior is not guaranteed for unsupported versions.")
	}

	return requestedVersion
}
