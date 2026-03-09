package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestStability_ColdStartBurst_NoHang(t *testing.T) {
	workspaceRoot, c3LibPath := stabilityWorkspaceConfig(t)
	srv := newWorkspacePerfServer(c3LibPath)
	srv.state.SetProjectRootURI(workspaceRoot)

	uri, content, hoverPos, renamePos := stabilityRequestTargets(t, workspaceRoot)
	doc := document.NewDocumentFromDocURI(uri, content, 1)
	srv.state.RefreshDocumentIdentifiers(doc, srv.parser)

	durations := make([]time.Duration, 0, 30)
	for i := 0; i < 10; i++ {
		durations = append(durations, mustCompleteWithin(t, 2*time.Second, "hover", func() error {
			_, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     hoverPos,
			}})
			return err
		}))

		durations = append(durations, mustCompleteWithin(t, 2*time.Second, "definition", func() error {
			_, err := srv.TextDocumentDefinition(nil, &protocol.DefinitionParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     hoverPos,
			}})
			return err
		}))

		durations = append(durations, mustCompleteWithin(t, 2*time.Second, "prepareRename", func() error {
			_, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     renamePos,
			}})
			return err
		}))
	}

	p95 := percentileDuration(durations, 95)
	if p95 > 1500*time.Millisecond {
		t.Fatalf("cold burst p95 too high: %s", p95)
	}
}

func TestStability_ConcurrentMixedRequests_DuringBackgroundIndexing(t *testing.T) {
	workspaceRoot, c3LibPath := stabilityWorkspaceConfig(t)
	srv := newWorkspacePerfServer(c3LibPath)
	srv.state.SetProjectRootURI(workspaceRoot)

	uri, content, hoverPos, renamePos := stabilityRequestTargets(t, workspaceRoot)
	doc := document.NewDocumentFromDocURI(uri, content, 1)
	srv.state.RefreshDocumentIdentifiers(doc, srv.parser)

	srv.indexWorkspaceAtAsync(workspaceRoot)

	var wg sync.WaitGroup
	errCh := make(chan error, 60)

	for i := 0; i < 20; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			mustCompleteWithinChan(2*time.Second, errCh, "hover", func() error {
				_, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     hoverPos,
				}})
				return err
			})
		}()

		go func() {
			defer wg.Done()
			mustCompleteWithinChan(2*time.Second, errCh, "definition", func() error {
				_, err := srv.TextDocumentDefinition(nil, &protocol.DefinitionParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     hoverPos,
				}})
				return err
			})
		}()

		go func() {
			defer wg.Done()
			mustCompleteWithinChan(2*time.Second, errCh, "prepareRename", func() error {
				_, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     renamePos,
				}})
				return err
			})
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestStability_WarmRequestSLOs_AndVariance(t *testing.T) {
	workspaceRoot, c3LibPath := stabilityWorkspaceConfig(t)
	srv := newWorkspacePerfServer(c3LibPath)
	srv.state.SetProjectRootURI(workspaceRoot)

	uri, content, hoverPos, renamePos := stabilityRequestTargets(t, workspaceRoot)
	srv.indexWorkspaceAt(workspaceRoot)
	srv.markRootIndexed(workspaceRoot)
	doc := document.NewDocumentFromDocURI(uri, content, 1)
	srv.state.RefreshDocumentIdentifiers(doc, srv.parser)

	hoverRounds := make([]time.Duration, 0, 5)
	definitionRounds := make([]time.Duration, 0, 5)
	renameRounds := make([]time.Duration, 0, 5)

	for round := 0; round < 5; round++ {
		hoverDurations := make([]time.Duration, 0, 20)
		definitionDurations := make([]time.Duration, 0, 20)
		renameDurations := make([]time.Duration, 0, 20)

		for i := 0; i < 20; i++ {
			hoverDurations = append(hoverDurations, mustCompleteWithin(t, 1200*time.Millisecond, "hover", func() error {
				_, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     hoverPos,
				}})
				return err
			}))

			definitionDurations = append(definitionDurations, mustCompleteWithin(t, 1200*time.Millisecond, "definition", func() error {
				_, err := srv.TextDocumentDefinition(nil, &protocol.DefinitionParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     hoverPos,
				}})
				return err
			}))

			renameDurations = append(renameDurations, mustCompleteWithin(t, 1200*time.Millisecond, "prepareRename", func() error {
				_, err := srv.TextDocumentPrepareRename(nil, &protocol.PrepareRenameParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     renamePos,
				}})
				return err
			}))
		}

		hoverRounds = append(hoverRounds, percentileDuration(hoverDurations, 95))
		definitionRounds = append(definitionRounds, percentileDuration(definitionDurations, 95))
		renameRounds = append(renameRounds, percentileDuration(renameDurations, 95))
	}

	hoverP95 := percentileDuration(hoverRounds, 95)
	definitionP95 := percentileDuration(definitionRounds, 95)
	renameP95 := percentileDuration(renameRounds, 95)

	if hoverP95 > 120*time.Millisecond {
		t.Fatalf("warm hover p95 too high: %s", hoverP95)
	}
	if definitionP95 > 150*time.Millisecond {
		t.Fatalf("warm definition p95 too high: %s", definitionP95)
	}
	if renameP95 > 200*time.Millisecond {
		t.Fatalf("warm prepareRename p95 too high: %s", renameP95)
	}

	if varianceRatio(hoverRounds) > 2.0 && percentileDuration(hoverRounds, 100) > 5*time.Millisecond {
		t.Fatalf("hover p95 variance too high across rounds: %v", hoverRounds)
	}
	if varianceRatio(definitionRounds) > 2.0 && percentileDuration(definitionRounds, 100) > 5*time.Millisecond {
		t.Fatalf("definition p95 variance too high across rounds: %v", definitionRounds)
	}
	if varianceRatio(renameRounds) > 2.0 && percentileDuration(renameRounds, 100) > 5*time.Millisecond {
		t.Fatalf("prepareRename p95 variance too high across rounds: %v", renameRounds)
	}
}

func stabilityWorkspaceConfig(t *testing.T) (string, string) {
	t.Helper()

	workspaceRoot := os.Getenv("C3LSP_PERF_WORKSPACE")
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = defaultPerfWorkspace
	}
	workspaceRoot = fs.GetCanonicalPath(workspaceRoot)
	if info, err := os.Stat(workspaceRoot); err != nil || !info.IsDir() {
		t.Skipf("workspace not available: %s", workspaceRoot)
	}

	c3LibPath := os.Getenv("C3LSP_PERF_C3LIB")
	if strings.TrimSpace(c3LibPath) == "" {
		c3LibPath = defaultPerfC3LibPath
	}

	return workspaceRoot, c3LibPath
}

func stabilityRequestTargets(t *testing.T, workspaceRoot string) (protocol.DocumentUri, string, protocol.Position, protocol.Position) {
	t.Helper()

	targetPath := filepath.Join(workspaceRoot, "src", "thread.c3")
	contentBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Skipf("target file not available: %s", targetPath)
	}
	content := string(contentBytes)
	uri := protocol.DocumentUri(fs.ConvertPathToURI(targetPath, option.None[string]()))

	hoverNeedle := "io::printfn("
	hoverIndex := strings.Index(content, hoverNeedle)
	if hoverIndex < 0 {
		t.Skipf("hover needle not found in %s", targetPath)
	}
	hoverPos := byteIndexToLSPPosition(content, hoverIndex+len("io::")+1)

	renameNeedle := "main(String[] args)"
	renameIndex := strings.Index(content, renameNeedle)
	if renameIndex < 0 {
		t.Skipf("rename needle not found in %s", targetPath)
	}
	renamePos := byteIndexToLSPPosition(content, renameIndex+len("main(String[] args"))

	return uri, content, hoverPos, renamePos
}

func mustCompleteWithin(t *testing.T, timeout time.Duration, op string, fn func() error) time.Duration {
	t.Helper()
	start := time.Now()
	done := make(chan error, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- fmt.Errorf("%s panic: %v", op, recovered)
			}
		}()
		done <- fn()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s failed: %v", op, err)
		}
		return time.Since(start)
	case <-time.After(timeout):
		t.Fatalf("%s timed out after %s", op, timeout)
		return 0
	}
}

func mustCompleteWithinChan(timeout time.Duration, errCh chan<- error, op string, fn func() error) {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- fmt.Errorf("%s panic: %v", op, recovered)
			}
		}()
		done <- fn()
	}()

	select {
	case err := <-done:
		if err != nil {
			errCh <- fmt.Errorf("%s failed: %w", op, err)
		}
	case <-time.After(timeout):
		errCh <- fmt.Errorf("%s timed out after %s", op, timeout)
	}
}

func percentileDuration(values []time.Duration, p int) time.Duration {
	if len(values) == 0 {
		return 0
	}

	copyValues := append([]time.Duration(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	if p <= 0 {
		return copyValues[0]
	}
	if p >= 100 {
		return copyValues[len(copyValues)-1]
	}

	idx := int(float64(p)/100.0*float64(len(copyValues)-1) + 0.5)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(copyValues) {
		idx = len(copyValues) - 1
	}

	return copyValues[idx]
}

func varianceRatio(values []time.Duration) float64 {
	if len(values) == 0 {
		return 0
	}

	minValue := values[0]
	maxValue := values[0]
	for _, v := range values[1:] {
		if v < minValue {
			minValue = v
		}
		if v > maxValue {
			maxValue = v
		}
	}

	if minValue <= 0 {
		return 0
	}

	return float64(maxValue) / float64(minValue)
}
