package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadC3ProjectConfig_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := ReadC3ProjectConfig(tmpDir)
	assert.NoError(t, err)
	assert.Nil(t, config)
}

func TestReadC3ProjectConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectJSON := `{
		"langrev": "1",
		"dependency-search-paths": ["../lib", "../.."],
		"dependencies": ["mylib", "otherlib"],
		"sources": ["src/**"]
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte(projectJSON), 0644)
	require.NoError(t, err)

	config, err := ReadC3ProjectConfig(tmpDir)
	assert.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, []string{"../lib", "../.."}, config.DependencySearchPaths)
	assert.Equal(t, []string{"mylib", "otherlib"}, config.Dependencies)
}

func TestReadC3ProjectConfig_NoDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	projectJSON := `{
		"langrev": "1",
		"sources": ["src/**"]
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte(projectJSON), 0644)
	require.NoError(t, err)

	config, err := ReadC3ProjectConfig(tmpDir)
	assert.NoError(t, err)
	require.NotNil(t, config)
	assert.Empty(t, config.DependencySearchPaths)
	assert.Empty(t, config.Dependencies)
}

func TestReadC3ProjectConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("not json"), 0644)
	require.NoError(t, err)

	config, err := ReadC3ProjectConfig(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestResolveDependencies_NilConfig(t *testing.T) {
	results := ResolveDependencies("/some/path", nil)
	assert.Nil(t, results)
}

func TestResolveDependencies_NoDependencies(t *testing.T) {
	config := &C3ProjectConfig{
		DependencySearchPaths: []string{"lib"},
	}
	results := ResolveDependencies("/some/path", config)
	assert.Nil(t, results)
}

func TestResolveDependencies_Found(t *testing.T) {
	// Create temp structure:
	//   projectDir/
	//   lib/
	//     mylib.c3l/
	//       types.c3
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	libDir := filepath.Join(tmpDir, "lib", "mylib.c3l")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	require.NoError(t, os.MkdirAll(libDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "types.c3"), []byte("module mylib;"), 0644))

	config := &C3ProjectConfig{
		DependencySearchPaths: []string{"../lib"},
		Dependencies:          []string{"mylib"},
	}

	results := ResolveDependencies(projectDir, config)
	require.Len(t, results, 1)
	assert.True(t, results[0].Found)
	assert.Equal(t, "mylib", results[0].Name)
	assert.Equal(t, libDir, results[0].Path)
}

func TestResolveDependencies_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	config := &C3ProjectConfig{
		DependencySearchPaths: []string{"../lib"},
		Dependencies:          []string{"nonexistent"},
	}

	results := ResolveDependencies(projectDir, config)
	require.Len(t, results, 1)
	assert.False(t, results[0].Found)
	assert.Equal(t, "nonexistent", results[0].Name)
	assert.Empty(t, results[0].Path)
}

func TestResolveDependencies_MultipleSearchPaths(t *testing.T) {
	// Dependency is in the second search path
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "examples", "myproject")
	// First search path has nothing
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "lib1"), 0755))
	// Second search path has the lib
	lib2Dir := filepath.Join(tmpDir, "lib2", "mylib.c3l")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	require.NoError(t, os.MkdirAll(lib2Dir, 0755))

	config := &C3ProjectConfig{
		DependencySearchPaths: []string{"../../lib1", "../../lib2"},
		Dependencies:          []string{"mylib"},
	}

	results := ResolveDependencies(projectDir, config)
	require.Len(t, results, 1)
	assert.True(t, results[0].Found)
	assert.Equal(t, lib2Dir, results[0].Path)
}

func TestResolveDependencies_AbsoluteSearchPath(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	libDir := filepath.Join(tmpDir, "abs_lib", "mylib.c3l")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	require.NoError(t, os.MkdirAll(libDir, 0755))

	config := &C3ProjectConfig{
		DependencySearchPaths: []string{filepath.Join(tmpDir, "abs_lib")},
		Dependencies:          []string{"mylib"},
	}

	results := ResolveDependencies(projectDir, config)
	require.Len(t, results, 1)
	assert.True(t, results[0].Found)
	assert.Equal(t, libDir, results[0].Path)
}

func TestScanDependencyForC3(t *testing.T) {
	tmpDir := t.TempDir()
	c3lDir := filepath.Join(tmpDir, "mylib.c3l")
	require.NoError(t, os.MkdirAll(c3lDir, 0755))

	// Create various files
	require.NoError(t, os.WriteFile(filepath.Join(c3lDir, "manifest.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(c3lDir, "types.c3"), []byte("module mylib;"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(c3lDir, "api.c3i"), []byte("module mylib;"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(c3lDir, "stubs.c"), []byte("// c code"), 0644))

	// Create a subdirectory (should not be scanned)
	subDir := filepath.Join(c3lDir, "linux-x64")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "platform.c3"), []byte("module mylib;"), 0644))

	files, err := ScanDependencyForC3(c3lDir)
	assert.NoError(t, err)
	assert.Len(t, files, 2)

	// Check we got the right files (order may vary)
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	assert.Contains(t, names, "types.c3")
	assert.Contains(t, names, "api.c3i")
}

func TestScanDependencyForC3_NonexistentDir(t *testing.T) {
	files, err := ScanDependencyForC3("/nonexistent/path")
	assert.Error(t, err)
	assert.Nil(t, files)
}
