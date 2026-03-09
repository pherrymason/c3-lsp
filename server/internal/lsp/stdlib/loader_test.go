package stdlib

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
)

func TestRehydrateModule_RestoresLookupDataAfterJSONRoundTrip(t *testing.T) {
	docID := "_stdlib_0.7.10"
	original := symbols.NewModule("std::io", "/tmp/io.c3", symbols.NewRange(0, 0, 0, 0), symbols.NewRange(0, 0, 1, 0))
	docs := symbols.NewDocCommentBuilder("Print any value to stdout, appending a newline.").
		WithContract("@param", "x: The value to print").
		Build()
	fun := symbols.NewFunctionBuilder("printn", symbols.NewTypeFromString("void", "std::io"), "std::io", "/tmp/io.c3").
		WithDocs(docs).
		Build()
	original.AddFunction(fun)

	payload, err := json.Marshal(original)
	assert.NoError(t, err)

	loaded := &symbols.Module{}
	err = json.Unmarshal(payload, loaded)
	assert.NoError(t, err)

	assert.Equal(t, "", loaded.GetModule().GetName())
	assert.Len(t, loaded.NestedScopes(), 0)

	rehydrateModule(loaded)

	assert.Equal(t, "std::io", loaded.GetModule().GetName())
	assert.Len(t, loaded.NestedScopes(), 1)
	rehydratedPrintn := loaded.ChildrenFunctions[0]
	assert.NotNil(t, rehydratedPrintn.GetDocComment())
	assert.NotEmpty(t, rehydratedPrintn.GetDocComment().GetBody())
	assert.True(t, rehydratedPrintn.GetDocComment().HasContracts())

	modules := symbols_table.NewParsedModules(&docID)
	modules.RegisterModule(loaded)
	loadable := modules.GetLoadableModules(symbols.NewModulePathFromString("std::io"))
	assert.Len(t, loadable, 1)
}

func TestLoadStdlib_prefersCacheBeforeBuild(t *testing.T) {
	logger := commonlog.MockLogger{}
	docID := "_stdlib_test"
	modules := symbols_table.NewParsedModules(&docID)
	modules.RegisterModule(symbols.NewModule("std::io", "/tmp/io.c3", symbols.NewRange(0, 0, 0, 0), symbols.NewRange(0, 0, 1, 0)))

	originalLoad := loadStdlibFromCacheFn
	originalBuild := buildStdlibIndexFn
	originalSave := saveStdlibToCacheFn
	t.Cleanup(func() {
		loadStdlibFromCacheFn = originalLoad
		buildStdlibIndexFn = originalBuild
		saveStdlibToCacheFn = originalSave
	})

	buildCalls := 0
	loadStdlibFromCacheFn = func(_ commonlog.Logger, _ string) (*symbols_table.UnitModules, error) {
		return &modules, nil
	}
	buildStdlibIndexFn = func(_ string, _ string) (*symbols_table.UnitModules, error) {
		buildCalls++
		return nil, errors.New("should not build when cache exists")
	}
	saveStdlibToCacheFn = func(_ commonlog.Logger, _ string, _ *symbols_table.UnitModules) error {
		return nil
	}

	result := LoadStdlib(logger, "0.7.10", "/fake/lib")

	assert.Equal(t, docID, result.DocId())
	assert.Equal(t, 0, buildCalls)
}

func TestLoadStdlib_buildsWhenCacheMissingAndPathConfigured(t *testing.T) {
	logger := commonlog.MockLogger{}
	docID := "_stdlib_test_built"
	built := symbols_table.NewParsedModules(&docID)
	built.RegisterModule(symbols.NewModule("std::math", "/tmp/math.c3", symbols.NewRange(0, 0, 0, 0), symbols.NewRange(0, 0, 1, 0)))

	originalLoad := loadStdlibFromCacheFn
	originalBuild := buildStdlibIndexFn
	originalSave := saveStdlibToCacheFn
	t.Cleanup(func() {
		loadStdlibFromCacheFn = originalLoad
		buildStdlibIndexFn = originalBuild
		saveStdlibToCacheFn = originalSave
	})

	buildCalls := 0
	loadStdlibFromCacheFn = func(_ commonlog.Logger, _ string) (*symbols_table.UnitModules, error) {
		return nil, errors.New("cache miss")
	}
	buildStdlibIndexFn = func(_ string, _ string) (*symbols_table.UnitModules, error) {
		buildCalls++
		return &built, nil
	}
	saveStdlibToCacheFn = func(_ commonlog.Logger, _ string, _ *symbols_table.UnitModules) error {
		return nil
	}

	result := LoadStdlib(logger, "0.7.10", "/fake/lib")

	assert.Equal(t, docID, result.DocId())
	assert.Equal(t, 1, buildCalls)
}

func TestLoadStdlibWithBackgroundRefresh_rebuildsAsyncOnCacheHit(t *testing.T) {
	logger := commonlog.MockLogger{}
	cachedDocID := "_stdlib_cached"
	builtDocID := "_stdlib_rebuilt"
	cached := symbols_table.NewParsedModules(&cachedDocID)
	built := symbols_table.NewParsedModules(&builtDocID)

	originalLoad := loadStdlibFromCacheFn
	originalBuild := buildStdlibIndexFn
	originalSave := saveStdlibToCacheFn
	t.Cleanup(func() {
		loadStdlibFromCacheFn = originalLoad
		buildStdlibIndexFn = originalBuild
		saveStdlibToCacheFn = originalSave
	})

	rebuildStarted := make(chan struct{}, 1)
	rebuildRelease := make(chan struct{})
	rebuildDone := make(chan symbols_table.UnitModules, 1)

	loadStdlibFromCacheFn = func(_ commonlog.Logger, _ string) (*symbols_table.UnitModules, error) {
		return &cached, nil
	}
	buildStdlibIndexFn = func(_ string, _ string) (*symbols_table.UnitModules, error) {
		rebuildStarted <- struct{}{}
		<-rebuildRelease
		return &built, nil
	}
	saveStdlibToCacheFn = func(_ commonlog.Logger, _ string, _ *symbols_table.UnitModules) error {
		return nil
	}

	result := LoadStdlibWithBackgroundRefresh(logger, "0.7.10", "/fake/lib", func(mod symbols_table.UnitModules) {
		rebuildDone <- mod
	})

	assert.Equal(t, cachedDocID, result.DocId())

	select {
	case <-rebuildStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected background rebuild to start")
	}

	close(rebuildRelease)

	select {
	case mod := <-rebuildDone:
		assert.Equal(t, builtDocID, mod.DocId())
	case <-time.After(1 * time.Second):
		t.Fatalf("expected rebuilt stdlib callback to be called")
	}
}

func TestTryAcquireFileLock_enforcesExclusivity(t *testing.T) {
	lockFile := filepath.Join(t.TempDir(), "stdlib_0.7.11.lock")

	unlock1, acquired1, err := tryAcquireFileLock(lockFile)
	if err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}
	if !acquired1 {
		t.Fatalf("expected first lock acquisition to succeed")
	}

	_, acquired2, err := tryAcquireFileLock(lockFile)
	if err != nil {
		t.Fatalf("unexpected second lock error: %v", err)
	}
	if acquired2 {
		t.Fatalf("expected second lock acquisition to fail while first lock is held")
	}

	unlock1()

	unlock3, acquired3, err := tryAcquireFileLock(lockFile)
	if err != nil {
		t.Fatalf("unexpected third lock error: %v", err)
	}
	if !acquired3 {
		t.Fatalf("expected lock acquisition to succeed after release")
	}
	unlock3()
}

func TestIsStaleLockFile_detectsOldMetadataTimestamp(t *testing.T) {
	lockFile := filepath.Join(t.TempDir(), "stdlib_0.7.11.lock")
	content := []byte("pid=123\nstarted_unix=1\n")
	if err := os.WriteFile(lockFile, content, 0644); err != nil {
		t.Fatalf("failed to seed lock file: %v", err)
	}

	stale, err := isStaleLockFile(lockFile, 1*time.Second)
	if err != nil {
		t.Fatalf("unexpected stale check error: %v", err)
	}
	if !stale {
		t.Fatalf("expected lock file to be stale")
	}
}
