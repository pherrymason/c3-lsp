package fs

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
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
	var files []string
	extensions := []string{"c3", "c3i"}

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, ext := range extensions {
			if !info.IsDir() && filepath.Ext(path) == "."+ext {
				files = append(files, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
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
