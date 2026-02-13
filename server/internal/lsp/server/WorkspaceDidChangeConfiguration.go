package server

import (
	"encoding/json"
	"time"

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
		Enabled *bool          `json:"enabled,omitempty"`
		Delay   *time.Duration `json:"delay,omitempty"`
	} `json:"Diagnostics"`
}

type runtimeSettingsLower struct {
	C3 struct {
		Version     *string  `json:"version,omitempty"`
		Path        *string  `json:"path,omitempty"`
		StdlibPath  *string  `json:"stdlib-path,omitempty"`
		CompileArgs []string `json:"compile-args,omitempty"`
	} `json:"c3"`

	Diagnostics struct {
		Enabled *bool          `json:"enabled,omitempty"`
		Delay   *time.Duration `json:"delay,omitempty"`
	} `json:"diagnostics"`
}

func (s *Server) WorkspaceDidChangeConfiguration(context *glsp.Context, params *protocol.DidChangeConfigurationParams) error {
	if !s.applyRuntimeSettings(params.Settings) {
		s.server.Log.Infof("workspace/didChangeConfiguration received, reloading c3lsp.json")
		if root := s.state.GetProjectRootURI(); root != "" {
			s.loadServerConfigurationForWorkspace(root)
		}
	} else {
		s.server.Log.Infof("workspace/didChangeConfiguration applied runtime settings")
	}

	s.indexWorkspace()
	s.RunDiagnostics(s.state, context.Notify, false)

	return nil
}

func (s *Server) loadClientRuntimeConfiguration(context *glsp.Context, rootURI *protocol.DocumentUri) {
	if context == nil || !supportsWorkspaceConfiguration(s.clientCapabilities) {
		return
	}

	items := []protocol.ConfigurationItem{}
	sections := []string{"C3", "Diagnostics", "c3", "diagnostics", "c3lsp"}
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
		_ = s.applyRuntimeSettings(entry)
	}
}

func supportsWorkspaceConfiguration(capabilities protocol.ClientCapabilities) bool {
	if capabilities.Workspace == nil || capabilities.Workspace.Configuration == nil {
		return false
	}

	return *capabilities.Workspace.Configuration
}

func (s *Server) applyRuntimeSettings(settings any) bool {
	runtime, ok := parseRuntimeSettings(settings)
	if !ok {
		return false
	}

	applied := false

	if runtime.C3.Path != nil {
		s.options.C3.Path = option.Some(*runtime.C3.Path)
		applied = true
	}
	if runtime.C3.StdlibPath != nil {
		s.options.C3.StdlibPath = option.Some(*runtime.C3.StdlibPath)
		applied = true
	}
	if len(runtime.C3.CompileArgs) > 0 {
		s.options.C3.CompileArgs = runtime.C3.CompileArgs
		applied = true
	}

	version := s.options.C3.Version
	if runtime.C3.Version != nil {
		version = option.Some(*runtime.C3.Version)
		s.options.C3.Version = version
		applied = true
	}

	if runtime.Diagnostics.Enabled != nil {
		s.options.Diagnostics.Enabled = *runtime.Diagnostics.Enabled
		applied = true
	}
	if runtime.Diagnostics.Delay != nil {
		s.options.Diagnostics.Delay = *runtime.Diagnostics.Delay
		applied = true
	}

	if applied {
		s.applyVersionAndLoadStdlib(version)
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
		runtime.Diagnostics.Delay != nil
}
