package stdlib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
)

// Global configuration for C3C library path
var c3cLibPath string
var detectedC3Version string

var loadStdlibFromCacheFn = LoadStdlibFromCache
var buildStdlibIndexFn = BuildStdlibIndex
var saveStdlibToCacheFn = SaveStdlibToCache

const StdlibCacheFormatVersion = 6
const StdlibParserCompatibilityKey = "symbols-v1"

const stdlibBuildLockPollInterval = 200 * time.Millisecond
const stdlibBuildLockStaleTTL = 5 * time.Minute
const stdlibBuildLockWaitTimeout = 90 * time.Second

// SetC3CLibPath sets the global C3C library path and detects the installed version
func SetC3CLibPath(logger commonlog.Logger, path string) {
	c3cLibPath = path
	// Try to detect version from version.h file in the c3c sources
	detectedC3Version = detectVersionFromPath(logger, path)
}

// GetC3CLibPath returns the configured C3C library path
func GetC3CLibPath() string {
	if c3cLibPath == "" {
		// Default fallback path
		return filepath.Join("..", "..", "..", "assets", "c3c", "lib")
	}
	return c3cLibPath
}

// GetDetectedC3Version returns the detected C3 version from the configured path
func GetDetectedC3Version() string {
	return detectedC3Version
}

// detectVersionFromPath attempts to detect C3 version from the version.h file
func detectVersionFromPath(logger commonlog.Logger, libPath string) string {
	// Try to find version.h - it's usually in ../src/version.h relative to lib/
	versionFile := filepath.Join(filepath.Dir(libPath), "src", "version.h")

	content, err := os.ReadFile(versionFile)
	if err != nil {
		logger.Debug("could not detect C3 version", "path", versionFile, "error", err)
		return ""
	}

	// Parse version from version.h
	re := regexp.MustCompile(`#define\s+COMPILER_VERSION\s+"([^"]+)"`)
	match := re.FindStringSubmatch(string(content))
	if len(match) > 1 {
		logger.Info("detected C3 version", "version", match[1])
		return match[1]
	}

	return ""
}

// StdlibCache represents the cached stdlib index
type StdlibCache struct {
	FormatVersion       int               `json:"format_version"`
	Version             string            `json:"version"`
	ParserCompatibility string            `json:"parser_compatibility"`
	DocId               string            `json:"doc_id"`
	Modules             []*symbols.Module `json:"modules"`
}

// GetStdlibCachePath returns the path where stdlib cache files are stored
func GetStdlibCachePath() (string, error) {
	// Try to get cache directory based on OS
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	stdlibDir := filepath.Join(cacheDir, "c3-lsp", "stdlib")

	// Ensure directory exists
	if err := os.MkdirAll(stdlibDir, 0755); err != nil {
		return "", err
	}

	return stdlibDir, nil
}

// GetStdlibCacheFile returns the full path to a specific version's cache file
func GetStdlibCacheFile(version string) (string, error) {
	dir, err := GetStdlibCachePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("stdlib_%s.json", version)), nil
}

func GetStdlibCacheLockFile(version string) (string, error) {
	cacheFile, err := GetStdlibCacheFile(version)
	if err != nil {
		return "", err
	}

	return cacheFile + ".lock", nil
}

// LoadStdlibFromCache attempts to load stdlib from a cache file
func LoadStdlibFromCache(logger commonlog.Logger, version string) (*symbols_table.UnitModules, error) {
	start := time.Now()
	defer logPerf(logger, "stdlib/load-cache", start, "version", version)

	cacheFile, err := GetStdlibCacheFile(version)
	if err != nil {
		logger.Debug("failed to get stdlib cache file path", "error", err)
		return nil, fmt.Errorf("failed to get cache file path: %w", err)
	}

	logger.Debug("looking for stdlib cache", "path", cacheFile)

	// Check if file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		logger.Debug("stdlib cache file does not exist", "path", cacheFile)
		return nil, fmt.Errorf("cache file does not exist: %s", cacheFile)
	}

	logger.Debug("found stdlib cache file, attempting to load")

	// Read and parse cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		logger.Warning("failed to read stdlib cache file", "error", err)
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache StdlibCache
	if err := json.Unmarshal(data, &cache); err != nil {
		logger.Warning("failed to parse stdlib cache file", "error", err)
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Verify version matches
	if cache.Version != version {
		logger.Warning("stdlib cache version mismatch", "expected", version, "got", cache.Version)
		return nil, fmt.Errorf("cache version mismatch: expected %s, got %s", version, cache.Version)
	}

	if cache.FormatVersion != StdlibCacheFormatVersion {
		logger.Warning("stdlib cache format mismatch", "expected", StdlibCacheFormatVersion, "got", cache.FormatVersion)
		return nil, fmt.Errorf("cache format mismatch: expected %d, got %d", StdlibCacheFormatVersion, cache.FormatVersion)
	}

	if cache.ParserCompatibility != StdlibParserCompatibilityKey {
		logger.Warning("stdlib parser compatibility mismatch", "expected", StdlibParserCompatibilityKey, "got", cache.ParserCompatibility)
		return nil, fmt.Errorf("cache parser compatibility mismatch: expected %s, got %s", StdlibParserCompatibilityKey, cache.ParserCompatibility)
	}

	logger.Info("successfully loaded stdlib cache", "version", version, "modules", len(cache.Modules))

	// Reconstruct UnitModules from cache
	docId := cache.DocId
	modules := symbols_table.NewParsedModules(&docId)
	for _, mod := range cache.Modules {
		rehydrateModule(mod)
		modules.RegisterModule(mod)
	}

	return &modules, nil
}

// SaveStdlibToCache saves stdlib index to a cache file
func SaveStdlibToCache(logger commonlog.Logger, version string, modules *symbols_table.UnitModules) error {
	cacheFile, err := GetStdlibCacheFile(version)
	if err != nil {
		return fmt.Errorf("failed to get cache file path: %w", err)
	}

	logger.Debug("saving stdlib cache", "path", cacheFile)

	cache := StdlibCache{
		FormatVersion:       StdlibCacheFormatVersion,
		Version:             version,
		ParserCompatibility: StdlibParserCompatibilityKey,
		DocId:               modules.DocId(),
		Modules:             modules.Modules(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	tempFile := fmt.Sprintf("%s.tmp.%d.%d", cacheFile, os.Getpid(), time.Now().UnixNano())
	defer func() {
		_ = os.Remove(tempFile)
	}()

	f, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary cache file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write temporary cache file: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to sync temporary cache file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary cache file: %w", err)
	}

	if err := os.Rename(tempFile, cacheFile); err != nil {
		return fmt.Errorf("failed to atomically replace cache file: %w", err)
	}

	logger.Info("successfully saved stdlib cache", "version", version, "modules", len(cache.Modules), "size_mb", fmt.Sprintf("%.2f", float64(len(data))/(1024*1024)))

	return nil
}

// BuildStdlibIndex builds the stdlib index from C3 source files
func BuildStdlibIndex(c3cLibPath string, version string) (*symbols_table.UnitModules, error) {
	start := time.Now()
	defer func() {
		logger := commonlog.GetLogger("")
		logPerf(logger, "stdlib/build-index", start, "version", version, "path", c3cLibPath)
	}()

	baseLibPath := fs.GetCanonicalPath(c3cLibPath)
	files, err := fs.ScanForC3(filepath.Join(baseLibPath, "std"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan for C3 files: %w", err)
	}

	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("")
	parser := p.NewParser(logger)

	docId := "_stdlib_" + version
	parsedModules := symbols_table.NewParsedModules(&docId)

	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
		}

		doc := document.NewDocumentFromString(filePath, string(content))
		modules, pendingTypes := parser.ParseSymbols(&doc)

		// Merge modules into parsedModules
		for _, mod := range modules.Modules() {
			mergeOrRegisterModule(&parsedModules, mod)
		}

		// Note: pendingTypes handling might need adjustment based on your needs
		_ = pendingTypes
	}

	return &parsedModules, nil
}

func mergeOrRegisterModule(parsedModules *symbols_table.UnitModules, mod *symbols.Module) {
	existing := parsedModules.Get(mod.GetName())
	if existing == nil {
		parsedModules.RegisterModule(mod)
		return
	}

	for _, variable := range mod.Variables {
		existing.AddVariable(variable)
	}
	for _, enum := range mod.Enums {
		existing.AddEnum(enum)
	}
	for _, fault := range mod.FaultDefs {
		existing.AddFaultDef(fault)
	}
	for _, strukt := range mod.Structs {
		existing.AddStruct(strukt)
	}
	for _, bitstruct := range mod.Bitstructs {
		existing.AddBitstruct(bitstruct)
	}
	for _, def := range mod.Aliases {
		existing.AddAlias(def)
	}
	for _, distinct := range mod.TypeDefs {
		existing.AddTypeDef(distinct)
	}
	for _, fun := range mod.ChildrenFunctions {
		existing.AddFunction(fun)
	}
	for _, iface := range mod.Interfaces {
		existing.AddInterface(iface)
	}

	for _, imported := range mod.Imports {
		existing.AddImportsWithMode([]string{imported}, mod.IsImportNoRecurse(imported))
	}
}

func rehydrateModule(mod *symbols.Module) {
	mod.ChangeModule(mod.GetName())
	mod.SetGenericParameters(mod.GenericParameters)

	for _, variable := range mod.Variables {
		variable.Module = symbols.NewModulePathFromString(variable.GetModuleString())
		mod.AddVariable(variable)
	}
	for _, enum := range mod.Enums {
		enum.Module = symbols.NewModulePathFromString(enum.GetModuleString())
		mod.AddEnum(enum)
	}
	for _, fault := range mod.FaultDefs {
		fault.Module = symbols.NewModulePathFromString(fault.GetModuleString())
		mod.AddFaultDef(fault)
	}
	for _, strukt := range mod.Structs {
		strukt.Module = symbols.NewModulePathFromString(strukt.GetModuleString())
		mod.AddStruct(strukt)
	}
	for _, bitstruct := range mod.Bitstructs {
		bitstruct.Module = symbols.NewModulePathFromString(bitstruct.GetModuleString())
		mod.AddBitstruct(bitstruct)
	}
	for _, def := range mod.Aliases {
		def.Module = symbols.NewModulePathFromString(def.GetModuleString())
		mod.AddAlias(def)
	}
	for _, distinct := range mod.TypeDefs {
		distinct.Module = symbols.NewModulePathFromString(distinct.GetModuleString())
		mod.AddTypeDef(distinct)
	}
	for _, fun := range mod.ChildrenFunctions {
		fun.Module = symbols.NewModulePathFromString(fun.GetModuleString())
		mod.AddFunction(fun)
	}
	for _, iface := range mod.Interfaces {
		iface.Module = symbols.NewModulePathFromString(iface.GetModuleString())
		mod.AddInterface(iface)
	}
}

// LoadOrBuildStdlib attempts to load stdlib from cache, or builds it if not found
func LoadOrBuildStdlib(logger commonlog.Logger, c3cLibPath string, version string) (symbols_table.UnitModules, error) {
	// Try to load from cache first
	modules, err := loadStdlibFromCacheFn(logger, version)
	if err == nil {
		return *modules, nil
	}

	logger.Info("cache not found or invalid for stdlib, building index", "version", version)

	// Build index from source using provided path
	modules, err = buildStdlibIndexFn(c3cLibPath, version)
	if err != nil {
		return symbols_table.UnitModules{}, fmt.Errorf("failed to build stdlib index: %w", err)
	}

	// Save to cache for future use
	if err := saveStdlibToCacheFn(logger, version, modules); err != nil {
		logger.Warning("failed to save stdlib cache", "error", err)
		// Continue anyway - we have the index in memory
	}

	return *modules, nil
}

// LoadStdlib tries to load stdlib from cache first, then builds from sources if needed
func LoadStdlib(logger commonlog.Logger, version string, c3cLibPath string) symbols_table.UnitModules {
	return LoadStdlibWithBackgroundRefresh(logger, version, c3cLibPath, nil)
}

func LoadStdlibWithBackgroundRefresh(logger commonlog.Logger, version string, c3cLibPath string, onRebuilt func(symbols_table.UnitModules)) symbols_table.UnitModules {
	start := time.Now()
	defer logPerf(logger, "stdlib/load", start, "version", version, "path", c3cLibPath)

	// Check if we have a detected C3 version
	detectedVersion := GetDetectedC3Version()

	// Cache-first strategy for fast startup.
	modules, err := loadStdlibFromCacheFn(logger, version)
	if err == nil {
		logger.Info("loaded stdlib from cache", "version", version)

		if c3cLibPath != "" {
			go func() {
				refreshStart := time.Now()
				unlock, acquired, lockErr := acquireStdlibBuildLock(logger, version, stdlibBuildLockWaitTimeout)
				if lockErr != nil {
					logger.Warning("background stdlib rebuild lock failed", "version", version, "error", lockErr)
					return
				}
				if !acquired {
					logger.Info("skipping background stdlib rebuild, lock busy", "version", version)
					return
				}
				defer unlock()

				built, buildErr := buildStdlibIndexFn(c3cLibPath, version)
				if buildErr != nil {
					logger.Warning("background stdlib rebuild failed", "version", version, "error", buildErr)
					return
				}

				if saveErr := saveStdlibToCacheFn(logger, version, built); saveErr != nil {
					logger.Warning("failed to save rebuilt stdlib cache", "error", saveErr)
				}

				logPerf(logger, "stdlib/background-rebuild", refreshStart, "version", version, "path", c3cLibPath)

				if onRebuilt != nil {
					onRebuilt(*built)
				}
			}()
		}

		return *modules
	}

	if c3cLibPath != "" {
		logger.Info("stdlib cache unavailable, building from sources", "version", version)

		unlock, acquired, lockErr := acquireStdlibBuildLock(logger, version, stdlibBuildLockWaitTimeout)
		if lockErr != nil {
			logger.Warning("failed to acquire stdlib build lock", "version", version, "error", lockErr)
		} else if acquired {
			defer unlock()

			if cachedAfterLock, cacheErr := loadStdlibFromCacheFn(logger, version); cacheErr == nil {
				logger.Info("stdlib cache became available while waiting for lock", "version", version)
				return *cachedAfterLock
			}
		} else {
			if cachedWhileWaiting, waitErr := waitForStdlibCacheAvailability(logger, version, stdlibBuildLockWaitTimeout); waitErr == nil {
				logger.Info("loaded stdlib cache after waiting for lock holder", "version", version)
				return *cachedWhileWaiting
			}
		}

		modules, err := buildStdlibIndexFn(c3cLibPath, version)
		if err == nil {
			if lockErr == nil && acquired {
				if saveErr := saveStdlibToCacheFn(logger, version, modules); saveErr != nil {
					logger.Warning("failed to save stdlib cache", "error", saveErr)
				}
			} else {
				logger.Warning("skipping stdlib cache write, lock unavailable", "version", version)
			}
			return *modules
		}
		logger.Warning("failed to build stdlib from sources", "version", version, "error", err)
	}

	// No stdlib available - return empty
	logger.Warning("no stdlib available", "version", version)
	if detectedVersion != "" {
		logger.Warning("C3 binary version detected but stdlib could not be indexed", "version", detectedVersion)
		logger.Warning("Please ensure c3.path in c3lsp.json points to a valid c3c installation.")
	} else {
		logger.Warning("To enable stdlib support, configure c3.path in c3lsp.json.")
	}
	docId := "_stdlib_" + version
	return symbols_table.NewParsedModules(&docId)
}

func acquireStdlibBuildLock(logger commonlog.Logger, version string, waitTimeout time.Duration) (func(), bool, error) {
	lockFile, err := GetStdlibCacheLockFile(version)
	if err != nil {
		return nil, false, err
	}

	deadline := time.Now().Add(waitTimeout)
	for {
		unlock, acquired, err := tryAcquireFileLock(lockFile)
		if err != nil {
			return nil, false, err
		}
		if acquired {
			return unlock, true, nil
		}

		stale, staleErr := isStaleLockFile(lockFile, stdlibBuildLockStaleTTL)
		if staleErr != nil {
			logger.Warning("could not inspect stdlib lock file", "path", lockFile, "error", staleErr)
		} else if stale {
			logger.Warning("removing stale stdlib lock file", "path", lockFile)
			_ = os.Remove(lockFile)
			continue
		}

		if time.Now().After(deadline) {
			return nil, false, nil
		}

		time.Sleep(stdlibBuildLockPollInterval)
	}
}

func waitForStdlibCacheAvailability(logger commonlog.Logger, version string, waitTimeout time.Duration) (*symbols_table.UnitModules, error) {
	deadline := time.Now().Add(waitTimeout)
	for {
		modules, err := loadStdlibFromCacheFn(logger, version)
		if err == nil {
			return modules, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for stdlib cache")
		}

		time.Sleep(stdlibBuildLockPollInterval)
	}
}

func tryAcquireFileLock(lockFile string) (func(), bool, error) {
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	lockMeta := fmt.Sprintf("pid=%d\nstarted_unix=%d\n", os.Getpid(), time.Now().Unix())
	if _, err := io.WriteString(f, lockMeta); err != nil {
		_ = f.Close()
		_ = os.Remove(lockFile)
		return nil, false, err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(lockFile)
		return nil, false, err
	}

	return func() {
		_ = f.Close()
		_ = os.Remove(lockFile)
	}, true, nil
}

func isStaleLockFile(lockFile string, staleTTL time.Duration) (bool, error) {
	info, err := os.Stat(lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if time.Since(info.ModTime()) > staleTTL {
		return true, nil
	}

	content, err := os.ReadFile(lockFile)
	if err != nil {
		return false, nil
	}

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "started_unix=") {
			continue
		}

		raw := strings.TrimPrefix(line, "started_unix=")
		startedUnix, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return false, nil
		}

		startedAt := time.Unix(startedUnix, 0)
		if time.Since(startedAt) > staleTTL {
			return true, nil
		}
	}

	return false, nil
}

func logPerf(logger commonlog.Logger, operation string, startedAt time.Time, keysAndValues ...any) {
	if !utils.IsFeatureEnabled("PERF_TRACE") {
		return
	}

	logger.Info("[perf] operation completed", append([]any{"operation", operation, "duration", time.Since(startedAt).String()}, keysAndValues...)...)
}
