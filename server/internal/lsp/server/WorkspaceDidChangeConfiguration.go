package server

import (
	"encoding/json"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type runtimeSettings struct {
	C3 struct {
		Version     *string  `json:"version,omitempty"`
		Path        *string  `json:"path,omitempty"`
		StdlibPath  *string  `json:"stdlib-path,omitempty"`
		CompileArgs []string `json:"compile-args,omitempty"`
	} `json:"C3"`

	Diagnostics struct {
		Enabled         *bool          `json:"enabled,omitempty"`
		Delay           *time.Duration `json:"delay,omitempty"`
		SaveFullIdle    *time.Duration `json:"save-full-idle-ms,omitempty"`
		FullMinInterval *time.Duration `json:"full-min-interval-ms,omitempty"`
	} `json:"Diagnostics"`

	Formatting struct {
		C3Fmt             *string `json:"c3fmt,omitempty"`
		Config            *string `json:"config,omitempty"`
		WillSaveWaitUntil *bool   `json:"will-save-wait-until,omitempty"`
	} `json:"Formatting"`
}

type runtimeSettingsLower struct {
	C3 struct {
		Version     *string  `json:"version,omitempty"`
		Path        *string  `json:"path,omitempty"`
		StdlibPath  *string  `json:"stdlib-path,omitempty"`
		CompileArgs []string `json:"compile-args,omitempty"`
	} `json:"c3"`

	Diagnostics struct {
		Enabled         *bool          `json:"enabled,omitempty"`
		Delay           *time.Duration `json:"delay,omitempty"`
		SaveFullIdle    *time.Duration `json:"save-full-idle-ms,omitempty"`
		FullMinInterval *time.Duration `json:"full-min-interval-ms,omitempty"`
	} `json:"diagnostics"`

	Formatting struct {
		C3Fmt             *string `json:"c3fmt,omitempty"`
		Config            *string `json:"config,omitempty"`
		WillSaveWaitUntil *bool   `json:"will-save-wait-until,omitempty"`
	} `json:"formatting"`
}

func (s *Server) WorkspaceDidChangeConfiguration(context *glsp.Context, params *protocol.DidChangeConfigurationParams) error {
	if !s.shouldProcessNotification(protocol.MethodWorkspaceDidChangeConfiguration) {
		return nil
	}
	if params == nil {
		return nil
	}

	if !s.applyRuntimeSettings(params.Settings, context) {
		s.server.Log.Info("workspace/didChangeConfiguration received, reloading c3lsp.json")
	} else {
		s.server.Log.Info("workspace/didChangeConfiguration applied runtime settings")
	}

	if root := s.state.GetProjectRootURI(); root != "" {
		s.invalidateProjectRootCache()
		s.activeConfigRoot = ""
		s.configureProjectForRootWithContext(root, context)
	}

	if isBuildableProjectRoot(s.state.GetProjectRootURI()) {
		s.cancelRootIndexing(fs.GetCanonicalPath(s.state.GetProjectRootURI()))
		s.indexWorkspaceWithLSPContext(context)
		s.RunDiagnosticsFull(s.state, context.Notify, false)
	} else {
		s.server.Log.Info("workspace/didChangeConfiguration skipped full workspace indexing: aggregate root")
	}

	return nil
}

func (s *Server) loadClientRuntimeConfiguration(context *glsp.Context, rootURI *protocol.DocumentUri) {
	if context == nil || !supportsWorkspaceConfiguration(s.clientCapabilities) {
		return
	}

	items := []protocol.ConfigurationItem{}
	sections := []string{"C3", "Diagnostics", "Formatting", "c3", "diagnostics", "formatting", "c3lsp"}
	for _, section := range sections {
		sec := section
		items = append(items, protocol.ConfigurationItem{
			ScopeURI: rootURI,
			Section:  &sec,
		})
	}

	result := []any{}
	context.Call(protocol.ServerWorkspaceConfiguration, protocol.ConfigurationParams{Items: items}, &result)

	for _, entry := range result {
		_ = s.applyRuntimeSettings(entry, context)
	}
}

func supportsWorkspaceConfiguration(capabilities protocol.ClientCapabilities) bool {
	if capabilities.Workspace == nil || capabilities.Workspace.Configuration == nil {
		return false
	}

	return *capabilities.Workspace.Configuration
}

func (s *Server) applyRuntimeSettings(settings any, context *glsp.Context) bool {
	runtime, ok := parseRuntimeSettings(settings)
	if !ok {
		return false
	}

	applied := false

	if runtime.C3.Path != nil {
		s.options.C3.Path = option.Some(*runtime.C3.Path)
		s.workspaceC3Options.Path = option.Some(*runtime.C3.Path)
		applied = true
	}
	if runtime.C3.StdlibPath != nil {
		s.options.C3.StdlibPath = option.Some(*runtime.C3.StdlibPath)
		s.workspaceC3Options.StdlibPath = option.Some(*runtime.C3.StdlibPath)
		applied = true
	}
	if len(runtime.C3.CompileArgs) > 0 {
		s.options.C3.CompileArgs = append([]string(nil), runtime.C3.CompileArgs...)
		s.workspaceC3Options.CompileArgs = append([]string(nil), runtime.C3.CompileArgs...)
		applied = true
	}

	version := s.options.C3.Version
	if runtime.C3.Version != nil {
		version = option.Some(*runtime.C3.Version)
		s.options.C3.Version = version
		s.workspaceC3Options.Version = version
		applied = true
	}

	if runtime.Diagnostics.Enabled != nil {
		s.options.Diagnostics.Enabled = *runtime.Diagnostics.Enabled
		applied = true
	}
	if runtime.Diagnostics.Delay != nil {
		s.options.Diagnostics.Delay = *runtime.Diagnostics.Delay
		s.resetDiagnosticsSchedulers()
		applied = true
	}
	if runtime.Diagnostics.SaveFullIdle != nil {
		s.options.Diagnostics.SaveFullIdle = *runtime.Diagnostics.SaveFullIdle
		s.resetDiagnosticsSchedulers()
		applied = true
	}
	if runtime.Diagnostics.FullMinInterval != nil {
		s.options.Diagnostics.FullMinInterval = *runtime.Diagnostics.FullMinInterval
		s.resetDiagnosticsSchedulers()
		applied = true
	}

	if runtime.Formatting.C3Fmt != nil {
		s.options.Formatting.C3FmtPath = optionFromPtr(runtime.Formatting.C3Fmt)
		applied = true
	}

	if runtime.Formatting.Config != nil {
		s.options.Formatting.Config = optionFromPtr(runtime.Formatting.Config)
		applied = true
	}

	if runtime.Formatting.WillSaveWaitUntil != nil {
		s.options.Formatting.WillSaveWaitUntil = *runtime.Formatting.WillSaveWaitUntil
		applied = true
	}

	if applied {
		s.applyVersionAndLoadStdlibWithProgress(context, version)
	}

	return applied
}

func parseRuntimeSettings(settings any) (runtimeSettings, bool) {
	var runtime runtimeSettings
	if !decodeRuntimeSettings(settings, &runtime) {
		return runtimeSettings{}, false
	}

	if hasRuntimeSettings(runtime) {
		return runtime, true
	}

	var lower runtimeSettingsLower
	if !decodeRuntimeSettings(settings, &lower) {
		return runtimeSettings{}, false
	}

	runtime.C3.Version = lower.C3.Version
	runtime.C3.Path = lower.C3.Path
	runtime.C3.StdlibPath = lower.C3.StdlibPath
	runtime.C3.CompileArgs = lower.C3.CompileArgs
	runtime.Diagnostics.Enabled = lower.Diagnostics.Enabled
	runtime.Diagnostics.Delay = lower.Diagnostics.Delay
	runtime.Formatting.C3Fmt = lower.Formatting.C3Fmt
	runtime.Formatting.Config = lower.Formatting.Config
	runtime.Formatting.WillSaveWaitUntil = lower.Formatting.WillSaveWaitUntil

	return runtime, hasRuntimeSettings(runtime)
}

func decodeRuntimeSettings(settings any, out any) bool {
	bytes, err := json.Marshal(settings)
	if err != nil {
		return false
	}

	if err := json.Unmarshal(bytes, out); err != nil {
		return false
	}

	return true
}

func hasRuntimeSettings(runtime runtimeSettings) bool {
	return runtime.C3.Version != nil ||
		runtime.C3.Path != nil ||
		runtime.C3.StdlibPath != nil ||
		len(runtime.C3.CompileArgs) > 0 ||
		runtime.Diagnostics.Enabled != nil ||
		runtime.Diagnostics.Delay != nil ||
		runtime.Formatting.C3Fmt != nil ||
		runtime.Formatting.Config != nil ||
		runtime.Formatting.WillSaveWaitUntil != nil
}
