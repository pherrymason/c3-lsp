package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// C3ProjectConfig represents the relevant fields from a C3 project.json file.
type C3ProjectConfig struct {
	DependencySearchPaths []string `json:"dependency-search-paths"`
	Dependencies          []string `json:"dependencies"`
}

// DependencyResolution holds the result of resolving a dependency.
type DependencyResolution struct {
	// Name is the dependency name as declared in project.json.
	Name string
	// Path is the resolved absolute path to the .c3l directory, empty if not found.
	Path string
	// Found indicates whether the dependency was located.
	Found bool
}

// ReadC3ProjectConfig reads and parses the project.json file from the given directory.
// Returns nil (no error) if the file doesn't exist - the project may simply not have one.
func ReadC3ProjectConfig(projectDir string) (*C3ProjectConfig, error) {
	configPath := filepath.Join(projectDir, "project.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading project.json: %w", err)
	}

	var config C3ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing project.json: %w", err)
	}

	return &config, nil
}

// ResolveDependencies resolves each dependency declared in the project config by
// searching for <dep-name>.c3l directories under each dependency-search-path.
// Search paths are resolved relative to projectDir.
//
// The C3 compiler resolves dependencies by looking for a directory named
// "<dependency>.c3l" under each path listed in "dependency-search-paths".
// The search paths themselves are relative to the project.json location.
func ResolveDependencies(projectDir string, config *C3ProjectConfig) []DependencyResolution {
	if config == nil || len(config.Dependencies) == 0 {
		return nil
	}

	var results []DependencyResolution

	for _, dep := range config.Dependencies {
		resolution := DependencyResolution{Name: dep}

		for _, searchPath := range config.DependencySearchPaths {
			// Resolve search path relative to project directory
			absSearchPath := searchPath
			if !filepath.IsAbs(searchPath) {
				absSearchPath = filepath.Join(projectDir, searchPath)
			}
			absSearchPath = filepath.Clean(absSearchPath)

			// Look for <dep-name>.c3l directory
			c3lDir := filepath.Join(absSearchPath, dep+".c3l")
			info, err := os.Stat(c3lDir)
			if err == nil && info.IsDir() {
				resolution.Path = c3lDir
				resolution.Found = true
				break
			}
		}

		results = append(results, resolution)
	}

	return results
}

// ScanDependencyForC3 scans a resolved .c3l library directory for .c3 and .c3i files.
// It only scans the top-level directory (not recursively into target subdirectories)
// since .c3l libraries keep their source files at the root alongside manifest.json.
func ScanDependencyForC3(c3lPath string) ([]string, error) {
	var files []string
	extensions := map[string]bool{".c3": true, ".c3i": true}

	entries, err := os.ReadDir(c3lPath)
	if err != nil {
		return nil, fmt.Errorf("error reading .c3l directory %s: %w", c3lPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if extensions[filepath.Ext(entry.Name())] {
			files = append(files, filepath.Join(c3lPath, entry.Name()))
		}
	}

	return files, nil
}
