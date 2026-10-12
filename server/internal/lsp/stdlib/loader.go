package stdlib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"

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

// In-memory stdlib cache to avoid re-indexing on every LSP restart.
var (
	memCacheMu sync.RWMutex
	memCache   = make(map[string][]symbols_table.UnitModules)
)

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

// LoadStdlibByFile indexes the C3 standard library from source files,
// returning one UnitModules per source file. This is the correct approach
// because multiple source files can contribute to the same module name
// (e.g. std::core::mem in mem.c3 and mem_mempool.c3), and a flat
// RegisterModule call would overwrite earlier entries.
//
// Returns an empty slice (no error) if c3cLibPath is empty.
//
// Prefer LoadOrBuildStdlibByFile which adds disk caching on top.
func LoadStdlibByFile(logger commonlog.Logger, version string, c3cLibPath string) []symbols_table.UnitModules {
	if c3cLibPath == "" {
		logger.Warningf("No stdlib path configured for version %s — stdlib completions unavailable.", version)
		logger.Warning("Set c3.path in c3lsp.json to your c3c binary, or c3.stdlib-path to the lib/c3 directory.")
		return nil
	}

	baseLibPath := fs.GetCanonicalPath(c3cLibPath)
	files, err := fs.ScanForC3(filepath.Join(baseLibPath, "std"))
	if err != nil {
		logger.Warningf("Failed to scan stdlib at %s: %v", baseLibPath, err)
		return nil
	}

	logger.Infof("Indexing %d stdlib source files from %s ...", len(files), baseLibPath)
	parser := p.NewParser(logger)

	var result []symbols_table.UnitModules
	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warningf("Could not read %s: %v", filePath, err)
			continue
		}
		doc := document.NewDocumentFromString(filePath, string(content))
		modules, _ := parser.ParseSymbols(&doc)
		result = append(result, modules)
	}

	logger.Infof("Stdlib indexed: %d files", len(result))
	return result
}

// --- Cache types and functions ---

// StdlibFileEntry is a serializable entry for one stdlib source file.
type StdlibFileEntry struct {
	DocId   string            `json:"doc_id"`
	Modules []*symbols.Module `json:"modules"`
}

// stdlibCacheFormatVersion must be bumped whenever the serialized symbol
// format changes, so caches written by older builds are discarded.
const stdlibCacheFormatVersion = 2

// StdlibCache is the disk-cache representation for a stdlib version.
type StdlibCache struct {
	Version       string            `json:"version"`
	FormatVersion int               `json:"formatVersion"`
	Fingerprint   string            `json:"fingerprint"`
	Files         []StdlibFileEntry `json:"files"`
}

// stdlibCacheFingerprint identifies what produced a cache: the LSP executable
// (whose embedded grammar/parser determine the indexed symbols) and the stdlib
// location. A cache with a different fingerprint is rebuilt from source.
func stdlibCacheFingerprint(c3cLibPath string) string {
	hash := sha256.New()
	if executable, err := os.Executable(); err == nil {
		if file, err := os.Open(executable); err == nil {
			_, _ = io.Copy(hash, file)
			file.Close()
		}
	}
	hash.Write([]byte(fs.GetCanonicalPath(c3cLibPath)))

	return hex.EncodeToString(hash.Sum(nil))
}

// GetStdlibCachePath returns the path where stdlib cache files are stored.
func GetStdlibCachePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	stdlibDir := filepath.Join(cacheDir, "c3-lsp", "stdlib")
	if err := os.MkdirAll(stdlibDir, 0755); err != nil {
		return "", err
	}
	return stdlibDir, nil
}

// GetStdlibCacheFile returns the full path to a specific version's cache file.
func GetStdlibCacheFile(version string) (string, error) {
	dir, err := GetStdlibCachePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("stdlib_%s.json", version)), nil
}

// loadStdlibFromCache attempts to load stdlib UnitModules from the disk cache.
func loadStdlibFromCache(logger commonlog.Logger, version string, fingerprint string) ([]symbols_table.UnitModules, error) {
	cacheFile, err := GetStdlibCacheFile(version)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache file path: %w", err)
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache StdlibCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	if cache.Version != version {
		return nil, fmt.Errorf("cache version mismatch: expected %s, got %s", version, cache.Version)
	}

	if cache.FormatVersion != stdlibCacheFormatVersion {
		return nil, fmt.Errorf("cache format mismatch: expected %d, got %d", stdlibCacheFormatVersion, cache.FormatVersion)
	}

	if cache.Fingerprint != fingerprint {
		return nil, fmt.Errorf("cache fingerprint mismatch: built by a different c3-lsp binary or stdlib path")
	}

	var result []symbols_table.UnitModules
	for _, entry := range cache.Files {
		docId := entry.DocId
		um := symbols_table.NewParsedModules(&docId)
		for _, mod := range entry.Modules {
			mod.RebuildAfterDeserialization()
			um.RegisterModule(mod)
		}
		result = append(result, um)
	}

	logger.Infof("Loaded stdlib from cache: version %s, %d files, %d modules", version, len(result), len(cache.Files))
	return result, nil
}

// saveStdlibToCache saves stdlib UnitModules to the disk cache.
func saveStdlibToCache(logger commonlog.Logger, version string, fingerprint string, unitModules []symbols_table.UnitModules) error {
	cacheFile, err := GetStdlibCacheFile(version)
	if err != nil {
		return fmt.Errorf("failed to get cache file path: %w", err)
	}

	var entries []StdlibFileEntry
	for _, um := range unitModules {
		entries = append(entries, StdlibFileEntry{
			DocId:   um.DocId(),
			Modules: um.Modules(),
		})
	}

	cache := StdlibCache{
		Version:       version,
		FormatVersion: stdlibCacheFormatVersion,
		Fingerprint:   fingerprint,
		Files:         entries,
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	logger.Infof("Saved stdlib cache: version %s, %d files (%.2f MB)", version, len(entries), float64(len(data))/(1024*1024))
	return nil
}

// LoadOrBuildStdlibByFile loads stdlib from in-memory cache, then disk cache,
// and falls back to indexing from source. The result is cached in memory and
// persisted to disk for future sessions.
func LoadOrBuildStdlibByFile(logger commonlog.Logger, version string, c3cLibPath string) []symbols_table.UnitModules {
	// 1. In-memory cache
	memCacheMu.RLock()
	if cached, ok := memCache[version]; ok {
		memCacheMu.RUnlock()
		logger.Debugf("Stdlib %s served from in-memory cache (%d files)", version, len(cached))
		return cached
	}
	memCacheMu.RUnlock()

	// 2. Disk cache
	fingerprint := stdlibCacheFingerprint(c3cLibPath)
	if modules, err := loadStdlibFromCache(logger, version, fingerprint); err == nil {
		memCacheMu.Lock()
		memCache[version] = modules
		memCacheMu.Unlock()
		return modules
	} else {
		logger.Debugf("Stdlib disk cache miss for %s: %v", version, err)
	}

	// 3. Build from source
	modules := LoadStdlibByFile(logger, version, c3cLibPath)
	if modules == nil {
		return nil
	}

	// Store in memory
	memCacheMu.Lock()
	memCache[version] = modules
	memCacheMu.Unlock()

	// Persist to disk
	if err := saveStdlibToCache(logger, version, fingerprint, modules); err != nil {
		logger.Warningf("Failed to save stdlib cache: %v", err)
	}

	return modules
}

// InvalidateStdlibCache removes a version from the in-memory cache.
// Useful when the stdlib path changes.
func InvalidateStdlibCache(version string) {
	memCacheMu.Lock()
	delete(memCache, version)
	memCacheMu.Unlock()
}
