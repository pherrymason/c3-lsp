package fs

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// FileStorage implements the port core.FileStorage.
type FileStorage struct {
	// Current working directory.
	workingDir string
	//logger     util.Logger
}

// NewFileStorage creates a new instance of FileStorage using the given working
// directory as reference point for relative paths.
func NewFileStorage(workingDir string /*, logger util.Logger*/) (*FileStorage, error) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	return &FileStorage{workingDir /*, logger*/}, nil
}

func (fs *FileStorage) WorkingDir() string {
	return fs.workingDir
}

func (fs *FileStorage) SetWorkingDir(path string) {
	fs.workingDir = path
}

func (fs *FileStorage) Abs(path string) (string, error) {
	var err error
	if !filepath.IsAbs(path) {
		path = filepath.Join(fs.workingDir, path)
		path, err = filepath.Abs(path)
		if err != nil {
			return path, err
		}
	}

	return path, nil
}

func (fs *FileStorage) Rel(path string) (string, error) {
	return filepath.Rel(fs.workingDir, path)
}

func (fs *FileStorage) FileExists(path string) (bool, error) {
	fi, err := fs.fileInfo(path)
	if err != nil {
		return false, err
	} else {
		return fi != nil && (*fi).Mode().IsRegular(), nil
	}
}

func (fs *FileStorage) DirExists(path string) (bool, error) {
	fi, err := fs.fileInfo(path)
	return !os.IsNotExist(err) && fi != nil && (*fi).Mode().IsDir(), nil
}

func (fs *FileStorage) fileInfo(path string) (*os.FileInfo, error) {
	if fi, err := os.Stat(path); err == nil {
		return &fi, nil
	} else if os.IsNotExist(err) {
		return nil, nil
	} else {
		return nil, err
	}
}

func (fs *FileStorage) IsDescendantOf(dir string, path string) (bool, error) {
	dir, err := fs.Abs(dir)
	if err != nil {
		return false, err
	}
	dir = GetCanonicalPath(dir)

	path, err = fs.Abs(path)
	if err != nil {
		return false, err
	}
	path = GetCanonicalPath(path)

	path, err = filepath.Rel(dir, path)
	if err != nil {
		return false, err
	}

	return !strings.HasPrefix(path, ".."), nil
}

func (fs *FileStorage) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *FileStorage) Write(path string, content []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != ".." {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()
	_, err = f.Write(content)
	return err
}

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

func ScanForC3(basePath string) ([]string, error) {
	var files []string
	extension := "c3"

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == "."+extension {
			files = append(files, path)
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
