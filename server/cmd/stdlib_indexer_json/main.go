package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/stdlib"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/tliron/commonlog"
)

func main() {
	// Allow custom path via argument
	var c3cPath string
	if len(os.Args) > 1 {
		c3cPath = os.Args[1]
	} else {
		c3cPath = filepath.Join("..", "..", "..", "assets", "c3c")
	}

	c3cVersion := getC3Version(c3cPath)
	fmt.Printf("C3 Version detected: %s\n", c3cVersion)

	baseLibPath := fs.GetCanonicalPath(filepath.Join(c3cPath, "lib"))
	files, err := fs.ScanForC3(filepath.Join(baseLibPath, "std"))
	if err != nil {
		panic(fmt.Errorf("failed to scan for C3 files: %v", err))
	}
	fmt.Printf("Found %d C3 files to parse\n", len(files))

	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("")
	parser := p.NewParser(logger)

	docId := "_stdlib_" + c3cVersion
	parsedModules := symbols_table.NewParsedModules(&docId)

	for i, filePath := range files {
		relPath, _ := filepath.Rel(baseLibPath, filePath)
		fmt.Printf("Parsing (%03d / %03d): %s\n", i+1, len(files), relPath)

		content, err := os.ReadFile(filePath)
		if err != nil {
			panic(fmt.Errorf("could not read file %s: %v", filePath, err))
		}

		// Replace absolute paths with placeholder
		normalizedPath := strings.ReplaceAll(filePath, baseLibPath, "<stdlib-path>")
		doc := document.NewDocumentFromString(normalizedPath, string(content))

		modules, pendingTypes := parser.ParseSymbols(&doc)
		// Merge modules into parsedModules
		for _, mod := range modules.Modules() {
			if !mod.IsPrivate() {
				parsedModules.RegisterModule(mod)
			}
		}
		// Note: pendingTypes might need special handling
		_ = pendingTypes
	}

	fmt.Printf("\nParsed %d modules\n", len(parsedModules.Modules()))

	// Save to cache
	if err := stdlib.SaveStdlibToCache(logger, c3cVersion, &parsedModules); err != nil {
		panic(fmt.Errorf("failed to save stdlib cache: %v", err))
	}

	cacheFile, _ := stdlib.GetStdlibCacheFile(c3cVersion)
	fmt.Printf("\n✓ Successfully generated stdlib cache at:\n  %s\n", cacheFile)

	// Print cache file size
	if info, err := os.Stat(cacheFile); err == nil {
		fmt.Printf("  Size: %.2f MB\n", float64(info.Size())/(1024*1024))
	}

	// Verify the cache can be loaded
	fmt.Println("\nVerifying cache...")
	loaded, err := stdlib.LoadStdlibFromCache(logger, c3cVersion)
	if err != nil {
		panic(fmt.Errorf("failed to verify cache: %v", err))
	}

	fmt.Printf("✓ Cache verified - loaded %d modules\n", len(loaded.Modules()))
}

func getC3Version(path string) string {
	versionFile := filepath.Join(path, "src", "version.h")
	content, err := os.ReadFile(versionFile)
	if err != nil {
		panic(fmt.Sprintf("Could not find c3c version: Could not open %s file: %s", versionFile, err))
	}

	text := string(content)
	versionRegex := regexp.MustCompile(`#define\s+COMPILER_VERSION\s+"([^"]+)"`)
	versionMatch := versionRegex.FindStringSubmatch(text)
	if len(versionMatch) > 1 {
		return versionMatch[1]
	}

	panic("Could not find c3c version: Did not find COMPILER_VERSION in version.h")
}
