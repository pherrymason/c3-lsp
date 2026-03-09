package server

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/fs"
)

var dependencySearchPathsPattern = regexp.MustCompile(`(?s)"dependency-search-paths"\s*:\s*\[(.*?)\]`)
var jsonStringPattern = regexp.MustCompile(`"((?:\\.|[^"\\])*)"`)

func loadDependencySearchPathsForRoot(root string) []string {
	projectPath := filepath.Join(root, "project.json")
	content, err := os.ReadFile(projectPath)
	if err != nil {
		return nil
	}

	return parseDependencySearchPaths(string(content), root)
}

func parseDependencySearchPaths(projectJSON string, root string) []string {
	match := dependencySearchPathsPattern.FindStringSubmatch(projectJSON)
	if len(match) < 2 {
		return nil
	}

	entries := jsonStringPattern.FindAllStringSubmatch(match[1], -1)
	if len(entries) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	paths := make([]string, 0, len(entries))

	for _, entry := range entries {
		if len(entry) < 2 {
			continue
		}

		value, err := strconv.Unquote("\"" + entry[1] + "\"")
		if err != nil {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		resolved := value
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(root, resolved)
		}
		resolved = fs.GetCanonicalPath(resolved)
		if resolved == "" {
			continue
		}

		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			continue
		}

		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		paths = append(paths, resolved)
	}

	sort.Strings(paths)
	return paths
}
