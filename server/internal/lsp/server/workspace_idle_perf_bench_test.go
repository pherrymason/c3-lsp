package server

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const defaultPerfWorkspace = "/Users/f00lg/github/c3/c3-test"
const defaultPerfC3LibPath = "/Users/f00lg/github/c3/c3c/lib"

func BenchmarkWorkspaceIdle(b *testing.B) {
	workspaceRoot := os.Getenv("C3LSP_PERF_WORKSPACE")
	if workspaceRoot == "" {
		workspaceRoot = defaultPerfWorkspace
	}

	c3LibPath := os.Getenv("C3LSP_PERF_C3LIB")
	if c3LibPath == "" {
		c3LibPath = defaultPerfC3LibPath
	}

	workspaceRoot = fs.GetCanonicalPath(workspaceRoot)
	if workspaceRoot == "" {
		b.Skip("workspace path is empty")
	}

	if info, err := os.Stat(workspaceRoot); err != nil || !info.IsDir() {
		b.Skipf("workspace does not exist: %s", workspaceRoot)
	}

	targetPath := filepath.Join(workspaceRoot, "src", "thread.c3")
	targetSourceBytes, err := os.ReadFile(targetPath)
	if err != nil {
		b.Skipf("failed to read target file %s: %v", targetPath, err)
	}
	targetSource := string(targetSourceBytes)

	callNeedle := "io::printfn("
	callIndex := strings.Index(targetSource, callNeedle)
	if callIndex < 0 {
		b.Skipf("needle not found in %s: %q", targetPath, callNeedle)
	}

	hoverPos := byteIndexToLSPPosition(targetSource, callIndex+len("io::")+1)
	targetURI := protocol.DocumentUri(fs.ConvertPathToURI(targetPath, option.None[string]()))

	var indexTotal time.Duration
	var hoverColdTotal time.Duration
	var hoverWarmTotal time.Duration

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		srv := newWorkspacePerfServer(c3LibPath)
		srv.state.SetProjectRootURI(workspaceRoot)

		indexStart := time.Now()
		srv.indexWorkspaceAt(workspaceRoot)
		srv.markRootIndexed(workspaceRoot)
		indexTotal += time.Since(indexStart)

		doc := document.NewDocumentFromDocURI(targetURI, targetSource, 1)
		srv.state.RefreshDocumentIdentifiers(doc, srv.parser)

		hoverParams := &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: targetURI},
				Position:     hoverPos,
			},
		}

		hoverColdStart := time.Now()
		hoverCold, hoverErr := srv.TextDocumentHover(nil, hoverParams)
		hoverColdTotal += time.Since(hoverColdStart)
		if hoverErr != nil {
			b.Fatalf("cold hover failed: %v", hoverErr)
		}
		if hoverCold == nil {
			b.Fatalf("cold hover returned nil")
		}

		if i == 0 {
			content, ok := hoverCold.Contents.(protocol.MarkupContent)
			if !ok {
				b.Fatalf("cold hover did not return markdown content")
			}

			if !strings.Contains(content.Value, "printfn") {
				b.Fatalf("hover content missing function name printfn: %s", content.Value)
			}

			if !strings.Contains(content.Value, "In module **[") {
				b.Fatalf("hover content missing module section: %s", content.Value)
			}
		}

		hoverWarmStart := time.Now()
		hoverWarm, hoverWarmErr := srv.TextDocumentHover(nil, hoverParams)
		hoverWarmTotal += time.Since(hoverWarmStart)
		if hoverWarmErr != nil {
			b.Fatalf("warm hover failed: %v", hoverWarmErr)
		}
		if hoverWarm == nil {
			b.Fatalf("warm hover returned nil")
		}
	}

	ms := float64(time.Millisecond)
	b.ReportMetric(float64(indexTotal)/float64(b.N)/ms, "index_ms/op")
	b.ReportMetric(float64(hoverColdTotal)/float64(b.N)/ms, "hover_cold_ms/op")
	b.ReportMetric(float64(hoverWarmTotal)/float64(b.N)/ms, "hover_warm_ms/op")
}

func BenchmarkWorkspaceCompletionWarm(b *testing.B) {
	workspaceRoot := os.Getenv("C3LSP_PERF_WORKSPACE")
	if workspaceRoot == "" {
		workspaceRoot = defaultPerfWorkspace
	}

	c3LibPath := os.Getenv("C3LSP_PERF_C3LIB")
	if c3LibPath == "" {
		c3LibPath = defaultPerfC3LibPath
	}

	workspaceRoot = fs.GetCanonicalPath(workspaceRoot)
	if workspaceRoot == "" {
		b.Skip("workspace path is empty")
	}

	if info, err := os.Stat(workspaceRoot); err != nil || !info.IsDir() {
		b.Skipf("workspace does not exist: %s", workspaceRoot)
	}

	targetPath := filepath.Join(workspaceRoot, "src", "thread.c3")
	targetSourceBytes, err := os.ReadFile(targetPath)
	if err != nil {
		b.Skipf("failed to read target file %s: %v", targetPath, err)
	}

	originalSource := string(targetSourceBytes)
	needle := "io::printfn("
	repl := "io::pr"
	callIndex := strings.Index(originalSource, needle)
	if callIndex < 0 {
		b.Skipf("needle not found in %s: %q", targetPath, needle)
	}

	completionSource := strings.Replace(originalSource, needle, repl, 1)
	completionIndex := strings.Index(completionSource, repl)
	if completionIndex < 0 {
		b.Skipf("completion needle not found after replacement in %s", targetPath)
	}

	completionPos := byteIndexToLSPPosition(completionSource, completionIndex+len(repl))
	targetURI := protocol.DocumentUri(fs.ConvertPathToURI(targetPath, option.None[string]()))

	srv := newWorkspacePerfServer(c3LibPath)
	srv.state.SetProjectRootURI(workspaceRoot)
	srv.indexWorkspaceAt(workspaceRoot)
	srv.markRootIndexed(workspaceRoot)

	doc := document.NewDocumentFromDocURI(targetURI, completionSource, 2)
	srv.state.RefreshDocumentIdentifiers(doc, srv.parser)

	params := &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: targetURI},
			Position:     completionPos,
		},
		Context: &protocol.CompletionContext{
			TriggerKind: protocol.CompletionTriggerKindInvoked,
		},
	}

	baseline, err := srv.TextDocumentCompletion(nil, params)
	if err != nil {
		b.Fatalf("warmup completion failed: %v", err)
	}
	items, ok := baseline.([]completionItemWithLabelDetails)
	if !ok || len(items) == 0 {
		b.Fatalf("unexpected completion result type or empty list: %T", baseline)
	}
	if !completionItemsContainLabel(items, "printfn") {
		b.Fatalf("completion result missing expected symbol printfn")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, completionErr := srv.TextDocumentCompletion(nil, params)
		if completionErr != nil {
			b.Fatalf("completion failed: %v", completionErr)
		}

		completionItems, castOk := result.([]completionItemWithLabelDetails)
		if !castOk || len(completionItems) == 0 {
			b.Fatalf("unexpected completion result type or empty list: %T", result)
		}
	}
}

func BenchmarkTextDocumentDidSaveBurst_UnchangedVersion(b *testing.B) {
	uri := protocol.DocumentUri("file:///tmp/bench_didsave_unchanged_version.c3")
	srv := newDidSavePerfServer(uri, `module app;
fn void main() {}`)
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.DidSaveTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}}

	// warm up version cache entry
	if err := srv.TextDocumentDidSave(ctx, params); err != nil {
		b.Fatalf("warmup didSave failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := srv.TextDocumentDidSave(ctx, params); err != nil {
			b.Fatalf("didSave failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentDidSaveBurst_VersionBump(b *testing.B) {
	uri := protocol.DocumentUri("file:///tmp/bench_didsave_version_bump.c3")
	source := `module app;
fn void main() {}`
	srv := newDidSavePerfServer(uri, source)
	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.DidSaveTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doc := srv.state.GetDocument(string(uri))
		if doc == nil {
			b.Fatalf("document not found: %s", uri)
		}

		nextVersion := doc.Version + 1
		srv.state.UpdateDocument(uri, nextVersion, []interface{}{}, srv.parser)
		if err := srv.TextDocumentDidSave(ctx, params); err != nil {
			b.Fatalf("didSave failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentDidSaveBurst_DiagnosticsMocked(b *testing.B) {
	root := b.TempDir()
	projectFile := filepath.Join(root, "project.json")
	if err := os.WriteFile(projectFile, []byte("{}"), 0o644); err != nil {
		b.Fatalf("failed to write project.json: %v", err)
	}

	filePath := filepath.Join(root, "bench_save.c3")
	uri := protocol.DocumentUri(fs.ConvertPathToURI(filePath, option.None[string]()))
	source := `module app;
fn void main() {}`

	srv := newDidSavePerfServer(uri, source)
	srv.options.Diagnostics.Enabled = true
	srv.options.Diagnostics.Delay = 1
	srv.options.Diagnostics.SaveFullIdle = 10
	srv.options.Diagnostics.FullMinInterval = 20
	srv.state.SetProjectRootURI(root)
	srv.resetDiagnosticsSchedulers()

	var commandRuns atomic.Int64
	srv.diagnosticsCommand = func(ctx context.Context, _ c3c.C3Opts, _ string) (bytes.Buffer, bytes.Buffer, error) {
		commandRuns.Add(1)
		select {
		case <-ctx.Done():
			return bytes.Buffer{}, bytes.Buffer{}, ctx.Err()
		case <-time.After(2 * time.Millisecond):
			return bytes.Buffer{}, bytes.Buffer{}, nil
		}
	}

	ctx := &glsp.Context{Notify: func(string, any) {}}
	params := &protocol.DidSaveTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doc := srv.state.GetDocument(string(uri))
		if doc == nil {
			b.Fatalf("document not found: %s", uri)
		}

		nextVersion := doc.Version + 1
		srv.state.UpdateDocument(uri, nextVersion, []interface{}{}, srv.parser)
		if err := srv.TextDocumentDidSave(ctx, params); err != nil {
			b.Fatalf("didSave failed: %v", err)
		}
	}

	b.StopTimer()
	time.Sleep(100 * time.Millisecond)
	commandCount := float64(commandRuns.Load())
	b.ReportMetric(commandCount, "diag_runs_total")
	if b.N > 0 {
		b.ReportMetric(commandCount/float64(b.N), "diag_runs/save")
	}
}

func BenchmarkTextDocumentDidChange_EmptyChange(b *testing.B) {
	uri := protocol.DocumentUri("file:///tmp/bench_didchange_empty.c3")
	source := `module app;
fn void main() {}`
	srv := newDidSavePerfServer(uri, source)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doc := srv.state.GetDocument(string(uri))
		if doc == nil {
			b.Fatalf("document not found: %s", uri)
		}

		nextVersion := doc.Version + 1
		err := srv.TextDocumentDidChange(nil, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
				Version:                nextVersion,
			},
			ContentChanges: []interface{}{},
		})
		if err != nil {
			b.Fatalf("didChange failed: %v", err)
		}
	}
}

func BenchmarkTextDocumentDidChangeSaveBurst_DiagnosticsMocked(b *testing.B) {
	root := b.TempDir()
	projectFile := filepath.Join(root, "project.json")
	if err := os.WriteFile(projectFile, []byte("{}"), 0o644); err != nil {
		b.Fatalf("failed to write project.json: %v", err)
	}

	filePath := filepath.Join(root, "bench_change_save.c3")
	uri := protocol.DocumentUri(fs.ConvertPathToURI(filePath, option.None[string]()))
	baseSource := `module app;
fn void main() {}`

	srv := newDidSavePerfServer(uri, baseSource)
	srv.options.Diagnostics.Enabled = true
	srv.options.Diagnostics.Delay = 1
	srv.options.Diagnostics.SaveFullIdle = 500
	srv.options.Diagnostics.FullMinInterval = 1000
	srv.state.SetProjectRootURI(root)
	srv.resetDiagnosticsSchedulers()

	var commandRuns atomic.Int64
	srv.diagnosticsCommand = func(ctx context.Context, _ c3c.C3Opts, _ string) (bytes.Buffer, bytes.Buffer, error) {
		commandRuns.Add(1)
		select {
		case <-ctx.Done():
			return bytes.Buffer{}, bytes.Buffer{}, ctx.Err()
		case <-time.After(2 * time.Millisecond):
			return bytes.Buffer{}, bytes.Buffer{}, nil
		}
	}

	ctx := &glsp.Context{Notify: func(string, any) {}}
	samples := make([]time.Duration, 0, 2048)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doc := srv.state.GetDocument(string(uri))
		if doc == nil {
			b.Fatalf("document not found: %s", uri)
		}

		nextVersion := doc.Version + 1
		updatedSource := "module app;\nfn void main() {\n\tint v = " + strconv.Itoa(i) + ";\n}"

		iterStart := time.Now()
		changeErr := srv.TextDocumentDidChange(ctx, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
				Version:                nextVersion,
			},
			ContentChanges: []interface{}{protocol.TextDocumentContentChangeEventWhole{Text: updatedSource}},
		})
		if changeErr != nil {
			b.Fatalf("didChange failed: %v", changeErr)
		}

		saveErr := srv.TextDocumentDidSave(ctx, &protocol.DidSaveTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}})
		if saveErr != nil {
			b.Fatalf("didSave failed: %v", saveErr)
		}

		if len(samples) < cap(samples) {
			samples = append(samples, time.Since(iterStart))
		}
	}

	b.StopTimer()
	time.Sleep(1200 * time.Millisecond)

	commandCount := float64(commandRuns.Load())
	if b.N > 0 {
		b.ReportMetric(commandCount/float64(b.N), "diag_runs/save")
	}
	b.ReportMetric(commandCount, "diag_runs_total")

	hits, misses := srv.projectRootCacheCounters()
	hitRate := float64(0)
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	b.ReportMetric(hitRate, "root_cache_hit_rate")

	p50 := durationPercentile(samples, 0.50)
	p95 := durationPercentile(samples, 0.95)
	b.ReportMetric(float64(p50.Microseconds()), "burst_p50_us")
	b.ReportMetric(float64(p95.Microseconds()), "burst_p95_us")
}

func durationPercentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(values))
	copy(sorted, values)
	slices.Sort(sorted)

	idx := int(float64(len(sorted)-1) * percentile)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

func newWorkspacePerfServer(c3LibPath string) *Server {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	state.SetLanguageVersion(project_state.SupportedC3Version, c3LibPath)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	return &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
		options: ServerOpts{
			C3: c3c.C3Opts{
				Version:     option.None[string](),
				Path:        option.None[string](),
				StdlibPath:  option.None[string](),
				CompileArgs: []string{},
			},
			Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 2000},
			Formatting: FormattingOpts{
				C3FmtPath: option.None[string](),
				Config:    option.None[string](),
			},
			LogFilepath:      option.None[string](),
			SendCrashReports: false,
			Debug:            false,
		},
		workspaceC3Options: c3c.C3Opts{
			Version:     option.None[string](),
			Path:        option.None[string](),
			StdlibPath:  option.None[string](),
			CompileArgs: []string{},
		},
		idx: indexingCoordinator{
			indexed:  make(map[string]bool),
			indexing: make(map[string]bool),
			cancels:  make(map[string]context.CancelFunc),
		},
		diag: diagnosticsCoordinator{
			saveDocVersions: make(map[string]int32),
		},
	}
}

func newDidSavePerfServer(uri protocol.DocumentUri, source string) *Server {
	srv := newWorkspacePerfServer(defaultPerfC3LibPath)
	srv.initialized.Store(true)
	srv.options.Diagnostics.Enabled = false
	srv.options.Diagnostics.Delay = 5
	srv.options.Diagnostics.SaveFullIdle = 10
	srv.options.Diagnostics.FullMinInterval = 100
	srv.resetDiagnosticsSchedulers()

	doc := document.NewDocumentFromDocURI(uri, source, 1)
	srv.state.RefreshDocumentIdentifiers(doc, srv.parser)

	return srv
}
