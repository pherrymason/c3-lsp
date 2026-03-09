package server

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
)

type DiagnosticsOpts struct {
	Enabled         bool          `json:"enabled"`
	Delay           time.Duration `json:"delay"`
	SaveFullIdle    time.Duration `json:"save-full-idle-ms"`
	FullMinInterval time.Duration `json:"full-min-interval-ms"`
}

type FormattingOpts struct {
	C3FmtPath         option.Option[string]
	Config            option.Option[string]
	WillSaveWaitUntil bool
}

// LogRotationOpts controls log file rotation to prevent unbounded log growth.
type LogRotationOpts struct {
	// MaxSizeMB is the maximum size in megabytes before the log file is rotated.
	// Defaults to 50 MB.
	MaxSizeMB int
	// MaxBackups is the maximum number of old log files to keep.
	// Defaults to 3.
	MaxBackups int
	// MaxAgeDays is the maximum number of days to retain old log files.
	// 0 means no age limit.
	MaxAgeDays int
	// Compress enables gzip compression of rotated log files.
	Compress bool
}

// DefaultLogRotationOpts returns sensible rotation defaults.
func DefaultLogRotationOpts() LogRotationOpts {
	return LogRotationOpts{
		MaxSizeMB:  50,
		MaxBackups: 3,
		MaxAgeDays: 0,
		Compress:   false,
	}
}

// loadRuntimeOpts reads all C3LSP_* environment variables once and returns a
// fully populated RuntimeOpts with derived duration fields pre-computed.
func loadRuntimeOpts() RuntimeOpts {
	r := RuntimeOpts{
		SlowRequestMs:      envInt("C3LSP_SLOW_REQUEST_MS", 0),
		RequestTimeoutMs:   envInt("C3LSP_REQUEST_TIMEOUT_MS", 0),
		RequestTimeoutDump: envBool("C3LSP_REQUEST_TIMEOUT_DUMP"),
		RequestMaxInflight: envInt("C3LSP_REQUEST_MAX_INFLIGHT", 0),
		WatchdogMs:         envInt("C3LSP_REQUEST_WATCHDOG_MS", 0),
		WatchdogDump:       envBool("C3LSP_REQUEST_WATCHDOG_DUMP"),
		IndexTimeoutMs:     envInt("C3LSP_INDEX_TIMEOUT_MS", 300000),
		PrepareRenameMs:    envInt("C3LSP_PREPARE_RENAME_TIMEOUT_MS", 2000),
	}

	// Apply feature-flag defaults when the env var was not set.
	if r.SlowRequestMs == 0 && utils.IsFeatureEnabled("REQUEST_SLOW_LOG") {
		r.SlowRequestMs = 400
	}
	if r.RequestTimeoutMs == 0 && utils.IsFeatureEnabled("REQUEST_TIMEOUT") {
		r.RequestTimeoutMs = 2000
	}
	if !r.RequestTimeoutDump && utils.IsFeatureEnabled("REQUEST_TIMEOUT_DUMP") {
		r.RequestTimeoutDump = true
	}
	if r.RequestMaxInflight == 0 && utils.IsFeatureEnabled("REQUEST_LIMIT_INFLIGHT") {
		r.RequestMaxInflight = 8
	}
	if r.WatchdogMs == 0 && utils.IsFeatureEnabled("REQUEST_WATCHDOG") {
		r.WatchdogMs = 1500
	}
	if !r.WatchdogDump && utils.IsFeatureEnabled("REQUEST_WATCHDOG_DUMP") {
		r.WatchdogDump = true
	}

	// Derive duration fields.
	r.SlowRequestThreshold = time.Duration(r.SlowRequestMs) * time.Millisecond
	r.RequestTimeout = time.Duration(r.RequestTimeoutMs) * time.Millisecond
	r.WatchdogThreshold = time.Duration(r.WatchdogMs) * time.Millisecond
	r.IndexTimeout = time.Duration(r.IndexTimeoutMs) * time.Millisecond
	r.PrepareRenameTimeout = time.Duration(r.PrepareRenameMs) * time.Millisecond

	return r
}

func envInt(key string, defaultVal int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}

	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return defaultVal
	}

	return v
}

func envBool(key string) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return raw == "1" || raw == "true" || raw == "yes"
}

// RuntimeOpts holds tuning parameters read once from environment variables at
// startup. They are intentionally not exposed in c3lsp.json because they are
// meant for advanced debugging and profiling, not end-user configuration.
type RuntimeOpts struct {
	// Request lifecycle
	SlowRequestMs      int  // C3LSP_SLOW_REQUEST_MS      — log requests slower than this (0 = off)
	RequestTimeoutMs   int  // C3LSP_REQUEST_TIMEOUT_MS    — hard kill per-request (0 = off)
	RequestTimeoutDump bool // C3LSP_REQUEST_TIMEOUT_DUMP  — goroutine dump on timeout
	RequestMaxInflight int  // C3LSP_REQUEST_MAX_INFLIGHT  — max concurrent requests (0 = off)
	WatchdogMs         int  // C3LSP_REQUEST_WATCHDOG_MS   — watchdog alarm threshold (0 = off)
	WatchdogDump       bool // C3LSP_REQUEST_WATCHDOG_DUMP — goroutine dump on watchdog
	IndexTimeoutMs     int  // C3LSP_INDEX_TIMEOUT_MS      — per-root indexing timeout (0 = off, default 300000 = 5 min)
	PrepareRenameMs    int  // C3LSP_PREPARE_RENAME_TIMEOUT_MS — prepare-rename timeout (default 2000)

	// Derived durations (computed once from the ms fields above)
	SlowRequestThreshold time.Duration
	RequestTimeout       time.Duration
	WatchdogThreshold    time.Duration
	IndexTimeout         time.Duration
	PrepareRenameTimeout time.Duration
}

// ServerOpts holds the options to create a new Server.
type ServerOpts struct {
	C3          c3c.C3Opts      `json:"C3Opts"`
	Diagnostics DiagnosticsOpts `json:"Diagnostics"`
	Formatting  FormattingOpts
	Runtime     RuntimeOpts

	LogFilepath      option.Option[string]
	LogRotation      LogRotationOpts
	SendCrashReports bool
	Debug            bool
}

type ServerOptsJson struct {
	C3 struct {
		Version     *string  `json:"version,omitempty"`
		Path        *string  `json:"path,omitempty"`
		StdlibPath  *string  `json:"stdlib-path,omitempty"`
		CompileArgs []string `json:"compile-args"`
	}

	Diagnostics struct {
		Enabled         bool          `json:"enabled"`
		Delay           time.Duration `json:"delay"`
		SaveFullIdle    time.Duration `json:"save-full-idle-ms"`
		FullMinInterval time.Duration `json:"full-min-interval-ms"`
	}

	Formatting struct {
		C3Fmt             *string `json:"c3fmt,omitempty"`
		Config            *string `json:"config,omitempty"`
		WillSaveWaitUntil *bool   `json:"will-save-wait-until,omitempty"`
	}

	Env map[string]bool

	LogFilepath *string `json:"log-filepath,omitempty"`
	Debug       *bool   `json:"debug,omitempty"`

	// Log rotation settings. Omitted fields use the defaults from DefaultLogRotationOpts().
	LogRotation struct {
		MaxSizeMB  *int  `json:"max-size-mb,omitempty"`
		MaxBackups *int  `json:"max-backups,omitempty"`
		MaxAgeDays *int  `json:"max-age-days,omitempty"`
		Compress   *bool `json:"compress,omitempty"`
	} `json:"log-rotation,omitempty"`
}

func (s *Server) loadServerConfigurationForWorkspace(path string, context *glsp.Context) {
	file, err := os.Open(path + "/c3lsp.json")
	if err != nil {
		// No configuration project file found - use defaults
		s.server.Log.Info("no project configuration found", "path", path+"/c3lsp.json")
		utils.SetRuntimeFeatureFlags(nil)
		if s.server != nil && s.server.Log != nil {
			s.server.Log.Info("feature flags enabled", "flags", "(none)")
		}
		// Still need to initialize stdlib with default version
		s.applyVersionAndLoadStdlibWithProgress(context, option.None[string]())
		return
	}
	defer func() {
		_ = file.Close()
	}()

	s.server.Log.Info("reading project configuration", "path", path+"/c3lsp.json")

	// Lee el archivo
	data, err := io.ReadAll(file)
	if err != nil {
		s.server.Log.Error("error reading config json", "err", err)
	}
	s.server.Log.Debug("config file contents", "data", string(data))

	var options ServerOptsJson
	err = json.Unmarshal(data, &options)
	if err != nil {
		s.server.Log.Error("error deserializing config json", "err", err)
	}

	utils.SetRuntimeFeatureFlags(options.Env)
	if s.server != nil && s.server.Log != nil {
		enabledFlags := utils.EnabledFeatureFlags()
		if len(enabledFlags) == 0 {
			s.server.Log.Info("feature flags enabled", "flags", "(none)")
		} else {
			s.server.Log.Info("feature flags enabled", "flags", strings.Join(enabledFlags, ", "))
		}
	}

	nextLogPath := s.options.LogFilepath
	if options.LogFilepath != nil {
		nextLogPath = optionFromPtr(options.LogFilepath)
	}
	// Apply log rotation overrides from JSON before reconfiguring the logger.
	if options.LogRotation.MaxSizeMB != nil {
		s.options.LogRotation.MaxSizeMB = *options.LogRotation.MaxSizeMB
	}
	if options.LogRotation.MaxBackups != nil {
		s.options.LogRotation.MaxBackups = *options.LogRotation.MaxBackups
	}
	if options.LogRotation.MaxAgeDays != nil {
		s.options.LogRotation.MaxAgeDays = *options.LogRotation.MaxAgeDays
	}
	if options.LogRotation.Compress != nil {
		s.options.LogRotation.Compress = *options.LogRotation.Compress
	}

	// Apply Debug from JSON only when the CLI did not set --debug.
	if options.Debug != nil && !s.cliDebug {
		s.options.Debug = *options.Debug
	}

	if logPathChanged(s.options.LogFilepath, nextLogPath) {
		s.reconfigureLoggerOutput(nextLogPath)
	}
	s.options.LogFilepath = nextLogPath

	// Apply c3c settings from JSON only when the CLI did not provide them.
	// Precedence: CLI > c3lsp.json.
	if options.C3.StdlibPath != nil && !s.workspaceC3Options.StdlibPath.IsSome() {
		s.options.C3.StdlibPath = option.Some(*options.C3.StdlibPath)
		s.server.Log.Info("stdlib path set from config", "path", *options.C3.StdlibPath)
	}

	// Store user-configured version (from c3lsp.json)
	var userConfiguredVersion option.Option[string]
	if options.C3.Version != nil {
		userConfiguredVersion = option.Some(*options.C3.Version)
	}

	if options.C3.Path != nil && !s.workspaceC3Options.Path.IsSome() {
		s.options.C3.Path = option.Some(*options.C3.Path)
	}

	if len(options.C3.CompileArgs) > 0 {
		s.options.C3.CompileArgs = options.C3.CompileArgs
	}

	if options.Formatting.C3Fmt != nil {
		s.options.Formatting.C3FmtPath = optionFromPtr(options.Formatting.C3Fmt)
	}

	if options.Formatting.Config != nil {
		s.options.Formatting.Config = optionFromPtr(options.Formatting.Config)
	}

	if options.Formatting.WillSaveWaitUntil != nil {
		s.options.Formatting.WillSaveWaitUntil = *options.Formatting.WillSaveWaitUntil
	}

	// Apply version and load stdlib
	s.applyVersionAndLoadStdlibWithProgress(context, userConfiguredVersion)

	if options.Diagnostics.SaveFullIdle > 0 {
		s.options.Diagnostics.SaveFullIdle = options.Diagnostics.SaveFullIdle
	}
	if options.Diagnostics.FullMinInterval > 0 {
		s.options.Diagnostics.FullMinInterval = options.Diagnostics.FullMinInterval
	}
	s.resetDiagnosticsSchedulers()

	// Change log filepath?
	// Should be able to do that from c3lsp.json?

	// Enable/disable sendCrashReports
	// Should be able to do that from c3lsp.json?
}

func (s *Server) applyVersionAndLoadStdlibWithProgress(context *glsp.Context, userConfiguredVersion option.Option[string]) {
	// Detect version from binary (either from configured path or system PATH)
	detectedBinaryVersion := c3c.GetC3Version(s.options.C3.Path)

	// Validate and set the final version to use
	var finalVersion option.Option[string]
	if userConfiguredVersion.IsSome() && detectedBinaryVersion.IsSome() {
		// Both configured and detected - check for mismatch
		if userConfiguredVersion.Get() != detectedBinaryVersion.Get() {
			s.server.Log.Warning("version mismatch detected",
				"configured", userConfiguredVersion.Get(),
				"detected", detectedBinaryVersion.Get(),
				"using", detectedBinaryVersion.Get())
		}
		// Use detected version (the actual binary version)
		finalVersion = detectedBinaryVersion
	} else if detectedBinaryVersion.IsSome() {
		// Only detected version available
		finalVersion = detectedBinaryVersion
	} else if userConfiguredVersion.IsSome() {
		// Only user configured version available (binary not found or version not detected)
		s.server.Log.Warning("could not detect c3c binary version, using configured version", "version", userConfiguredVersion.Get())
		finalVersion = userConfiguredVersion
	}
	// else: no version at all, will default to latest supported

	s.options.C3.Version = finalVersion

	requestedLanguageVersion := checkRequestedLanguageVersion(s.server.Log, s.options.C3.Version)

	// Determine c3cLibPath to pass to SetLanguageVersion
	// Priority: explicit stdlib-path > c3c-path/lib > empty string
	c3cLibPath := ""
	if s.options.C3.StdlibPath.IsSome() {
		normalizedStdlibPath := normalizeStdlibRootPath(s.options.C3.StdlibPath.Get())
		s.options.C3.StdlibPath = option.Some(normalizedStdlibPath)
		c3cLibPath = normalizedStdlibPath
	} else if s.options.C3.Path.IsSome() {
		c3cLibPath = s.options.C3.Path.Get() + "/lib"
	}

	token, hasProgress := s.beginWorkDoneProgress(context, "C3 stdlib", "Loading standard library symbols", false)
	if hasProgress {
		s.reportWorkDoneProgress(context, token, "Resolving cache/build state", nil)
		defer s.endWorkDoneProgress(context, token, "Stdlib ready")
	}

	s.state.SetLanguageVersion(requestedLanguageVersion, c3cLibPath)
}

func normalizeStdlibRootPath(path string) string {
	clean := filepath.Clean(path)
	if strings.EqualFold(filepath.Base(clean), "std") {
		return filepath.Dir(clean)
	}

	return clean
}

func optionFromPtr(input *string) option.Option[string] {
	if input == nil {
		return option.None[string]()
	}

	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return option.None[string]()
	}

	return option.Some(trimmed)
}

func logPathChanged(current option.Option[string], next option.Option[string]) bool {
	if current.IsSome() != next.IsSome() {
		return true
	}

	if !current.IsSome() {
		return false
	}

	return strings.TrimSpace(current.Get()) != strings.TrimSpace(next.Get())
}
