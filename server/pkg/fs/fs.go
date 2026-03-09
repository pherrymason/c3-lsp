package fs

import (
	"errors"
	iofs "io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/option"
)

func GetCanonicalPath(path string) string {
	path = filepath.Clean(path)

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		//fs.logger.Err(err)
	} else {
		path = resolvedPath
	}

	return path
}

func ConvertPathToURI(path string, stdlibPath option.Option[string]) string {
	path2 := strings.ReplaceAll(path, `\`, `/`)

	if stdlibPath.IsSome() {
		path2 = strings.Replace(path2, "<stdlib-path>", stdlibPath.Get()+"/", 1)
	}

	return "file:///" + strings.TrimLeft(path2, "/")
}

func ScanForC3(basePath string) ([]string, error) {
	files, _, err := ScanForC3WithOptions(basePath, ScanOptions{})
	return files, err
}

type ScanOptions struct {
	IgnoreDirs   []string
	PriorityDirs []string
}

type ScanStats struct {
	VisitedDirs int
	SkippedDirs int
	Matched     int
}

func DefaultC3ScanIgnoreDirs() []string {
	return []string{
		".git",
		".hg",
		".svn",
		"node_modules",
		"target",
		"build",
		"dist",
		"out",
		"bin",
		"obj",
		".idea",
		".vscode",
	}
}

func ScanForC3WithOptions(basePath string, options ScanOptions) ([]string, ScanStats, error) {
	var files []string
	extensions := []string{"c3", "c3i", "c3l"}
	stats := ScanStats{}

	ignore := make(map[string]struct{}, len(options.IgnoreDirs))
	for _, d := range options.IgnoreDirs {
		ignore[d] = struct{}{}
	}

	priority := make(map[string]int, len(options.PriorityDirs))
	for i, p := range options.PriorityDirs {
		priority[GetCanonicalPath(p)] = i
	}

	err := filepath.WalkDir(basePath, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			stats.VisitedDirs++
			if _, skip := ignore[d.Name()]; skip {
				stats.SkippedDirs++
				return filepath.SkipDir
			}
			return nil
		}

		for _, ext := range extensions {
			if filepath.Ext(path) == "."+ext {
				files = append(files, path)
				stats.Matched++
			}
		}

		return nil
	})

	if err != nil {
		return nil, stats, err
	}

	sort.SliceStable(files, func(i, j int) bool {
		iPri := priorityIndex(files[i], priority)
		jPri := priorityIndex(files[j], priority)
		if iPri != jPri {
			return iPri < jPri
		}
		return files[i] < files[j]
	})

	return files, stats, nil
}

func priorityIndex(path string, priority map[string]int) int {
	canonical := GetCanonicalPath(path)
	best := int(^uint(0) >> 1)
	for p, idx := range priority {
		if canonical == p || strings.HasPrefix(canonical, p+string(os.PathSeparator)) {
			if idx < best {
				best = idx
			}
		}
	}
	return best
}

func UriToPath(uri string) (string, error) {
	s := strings.ReplaceAll(uri, "%5C", "/")
	parsed, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "file" {
		return "", errors.New("URI was not a file:// URI")
	}

	if runtime.GOOS == "windows" {
		// In Windows "file:///c:/tmp/foo.md" is parsed to "/c:/tmp/foo.md".
		// Strip the first character to get a valid path.
		if strings.Contains(parsed.Path[1:], ":") {
			// url.Parse() behaves differently with "file:///c:/..." and "file://c:/..."
			return parsed.Path[1:], nil
		} else {
			// if the windows drive is not included in Path it will be in Host
			return parsed.Host + "/" + parsed.Path[1:], nil
		}
	}
	return parsed.Path, nil
}
