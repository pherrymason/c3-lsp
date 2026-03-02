package stdlib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/tliron/commonlog"
)

// Global configuration for C3C library path
var c3cLibPath string
var detectedC3Version string

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
		logger.Debugf("Could not detect C3 version from %s: %v", versionFile, err)
		return ""
	}

	// Parse version from version.h
	re := regexp.MustCompile(`#define\s+COMPILER_VERSION\s+"([^"]+)"`)
	match := re.FindStringSubmatch(string(content))
	if len(match) > 1 {
		logger.Infof("Detected C3 version: %s", match[1])
		return match[1]
	}

	return ""
}

// StdlibCache represents the cached stdlib index
type StdlibCache struct {
	Version string            `json:"version"`
	DocId   string            `json:"doc_id"`
	Modules []*symbols.Module `json:"modules"`
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

// LoadStdlibFromCache attempts to load stdlib from a cache file
func LoadStdlibFromCache(logger commonlog.Logger, version string) (*symbols_table.UnitModules, error) {
	cacheFile, err := GetStdlibCacheFile(version)
	if err != nil {
		logger.Debugf("Failed to get stdlib cache file path: %v", err)
		return nil, fmt.Errorf("failed to get cache file path: %w", err)
	}

	logger.Debugf("Looking for stdlib cache at: %s", cacheFile)

	// Check if file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		logger.Debugf("Stdlib cache file does not exist: %s", cacheFile)
		return nil, fmt.Errorf("cache file does not exist: %s", cacheFile)
	}

	logger.Debugf("Found stdlib cache file, attempting to load...")

	// Read and parse cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		logger.Warningf("Failed to read stdlib cache file: %v", err)
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache StdlibCache
	if err := json.Unmarshal(data, &cache); err != nil {
		logger.Warningf("Failed to parse stdlib cache file: %v", err)
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Verify version matches
	if cache.Version != version {
		logger.Warningf("Stdlib cache version mismatch: expected %s, got %s", version, cache.Version)
		return nil, fmt.Errorf("cache version mismatch: expected %s, got %s", version, cache.Version)
	}

	logger.Infof("Successfully loaded stdlib cache for version %s (%d modules)", version, len(cache.Modules))

	// Reconstruct UnitModules from cache
	docId := cache.DocId
	modules := symbols_table.NewParsedModules(&docId)
	for _, mod := range cache.Modules {
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

	logger.Debugf("Saving stdlib cache to: %s", cacheFile)

	cache := StdlibCache{
		Version: version,
		DocId:   modules.DocId(),
		Modules: modules.Modules(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	logger.Infof("Successfully saved stdlib cache for version %s (%d modules, %.2f MB)",
		version, len(cache.Modules), float64(len(data))/(1024*1024))

	return nil
}

// BuildStdlibIndex builds the stdlib index from C3 source files
func BuildStdlibIndex(c3cLibPath string, version string) (*symbols_table.UnitModules, error) {
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
			parsedModules.RegisterModule(mod)
		}

		// Note: pendingTypes handling might need adjustment based on your needs
		_ = pendingTypes
	}

	return &parsedModules, nil
}

// LoadOrBuildStdlib attempts to load stdlib from cache, or builds it if not found
func LoadOrBuildStdlib(logger commonlog.Logger, c3cLibPath string, version string) (symbols_table.UnitModules, error) {
	// Try to load from cache first
	modules, err := LoadStdlibFromCache(logger, version)
	if err == nil {
		return *modules, nil
	}

	logger.Infof("Cache not found or invalid for stdlib %s, building index...", version)

	// Build index from source using provided path
	modules, err = BuildStdlibIndex(c3cLibPath, version)
	if err != nil {
		return symbols_table.UnitModules{}, fmt.Errorf("failed to build stdlib index: %w", err)
	}

	// Save to cache for future use
	if err := SaveStdlibToCache(logger, version, modules); err != nil {
		logger.Warningf("Failed to save stdlib cache: %v", err)
		// Continue anyway - we have the index in memory
	}

	return *modules, nil
}

// LoadStdlib tries to load stdlib from cache first, then builds from sources if needed
func LoadStdlib(logger commonlog.Logger, version string, c3cLibPath string) symbols_table.UnitModules {
	// Check if we have a detected C3 version
	detectedVersion := GetDetectedC3Version()

	// Try to build from sources if we have c3c path configured
	if c3cLibPath != "" {
		// Try to load from cache or build from sources
		logger.Infof("Attempting to load or build stdlib index for version %s...", version)
		modules, err := LoadOrBuildStdlib(logger, c3cLibPath, version)
		if err == nil {
			return modules
		}
		logger.Warningf("Failed to load/build stdlib for version %s: %v", version, err)
	}

	// Try to load from cache only (user may have manually created it)
	modules, err := LoadStdlibFromCache(logger, version)
	if err == nil {
		return *modules
	}

	// No stdlib available - return empty
	logger.Warningf("No stdlib available for version %s.", version)
	if detectedVersion != "" {
		logger.Warningf("C3 binary version %s detected but stdlib could not be indexed.", detectedVersion)
		logger.Warning("Please ensure c3.path in c3lsp.json points to a valid c3c installation.")
	} else {
		logger.Warning("To enable stdlib support, configure c3.path in c3lsp.json.")
	}
	docId := "_stdlib_" + version
	return symbols_table.NewParsedModules(&docId)
}
