package server

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type diagnosticsRunMode string

const (
	diagnosticsRunQuick diagnosticsRunMode = "quick"
	diagnosticsRunFull  diagnosticsRunMode = "full"
)

func (s *Server) RunDiagnosticsQuick(state *project_state.ProjectState, notify glsp.NotifyFunc, delay bool, triggerURI *protocol.DocumentUri) {
	s.runDiagnostics(state, notify, delay, triggerURI, diagnosticsRunQuick)
}

func (s *Server) RunDiagnosticsFull(state *project_state.ProjectState, notify glsp.NotifyFunc, delay bool) {
	s.runDiagnostics(state, notify, delay, nil, diagnosticsRunFull)
}

func (s *Server) scheduleDiagnosticsFullAfterSaveIdle(ctx *glsp.Context) {
	if s == nil || s.diag.saveFullDebounced == nil {
		return
	}

	notify := glsp.NotifyFunc(nil)
	if ctx != nil {
		notify = ctx.Notify
	}

	s.diag.saveFullDebounced(func() {
		now := time.Now()
		if !s.reserveSaveFullDiagnosticsSlot(now) {
			if s.server != nil {
				perfLogf(s.server.Log, "diagnostics/full", now, "trigger=save_idle skipped=min_interval")
			}
			return
		}
		s.RunDiagnosticsFull(s.state, notify, false)
	})
}

func (s *Server) reserveSaveFullDiagnosticsSlot(now time.Time) bool {
	if s == nil {
		return false
	}

	minInterval := s.options.Diagnostics.FullMinInterval
	if minInterval <= 0 {
		minInterval = 30000
	}
	interval := minInterval * time.Millisecond

	s.diag.saveMu.Lock()
	defer s.diag.saveMu.Unlock()

	if s.diag.lastSaveFullNs == 0 {
		s.diag.lastSaveFullNs = now.UnixNano()
		return true
	}

	lastRun := time.Unix(0, s.diag.lastSaveFullNs)
	if now.Sub(lastRun) < interval {
		return false
	}

	s.diag.lastSaveFullNs = now.UnixNano()
	return true
}

func (s *Server) RunDiagnostics(state *project_state.ProjectState, notify glsp.NotifyFunc, delay bool, triggerURI *protocol.DocumentUri) {
	// Backward-compatible alias; default to quick mode.
	s.RunDiagnosticsQuick(state, notify, delay, triggerURI)
}

func (s *Server) runDiagnostics(state *project_state.ProjectState, notify glsp.NotifyFunc, delay bool, triggerURI *protocol.DocumentUri, mode diagnosticsRunMode) {
	if !s.options.Diagnostics.Enabled {
		return
	}
	projectRoot := s.resolveProjectRootForURI(triggerURI)
	if projectRoot == "" {
		return
	}

	runGeneration := s.nextDiagnosticsGeneration()

	runDiagnostics := func() {
		start := time.Now()
		startRootHits, startRootMisses := s.projectRootCacheCounters()
		publishedCount := 0
		clearedCount := 0
		staleCancelled := false
		defer func() {
			if s.server != nil {
				rootHits, rootMisses := s.projectRootCacheCounters()
				telemetry := computeRootCacheTelemetry(startRootHits, startRootMisses, rootHits, rootMisses)
				perfLogf(
					s.server.Log,
					"diagnostics/"+string(mode),
					start,
					"published=%d cleared=%d stale_cancelled=%t %s",
					publishedCount,
					clearedCount,
					staleCancelled,
					formatRootCacheTelemetry(telemetry),
				)
			}
		}()

		if !s.isCurrentDiagnosticsGeneration(runGeneration) {
			staleCancelled = true
			return
		}

		invalidation := state.GetLastInvalidationScope()
		relevantFiles := map[string]bool(nil)
		if mode == diagnosticsRunQuick {
			relevantFiles = diagnosticsRelevantFiles(state, triggerURI, invalidation)
		}
		hasFileFilter := len(relevantFiles) > 0

		cmdCtx, cancelCommand := context.WithCancel(context.Background())
		runID := s.swapDiagnosticsCommandCancel(cancelCommand)
		defer s.clearDiagnosticsCommandCancel(runID)

		s.configureProjectForRoot(projectRoot)

		diagnosticsCommand := s.diagnosticsCommand
		if diagnosticsCommand == nil {
			diagnosticsCommand = c3c.CheckC3ErrorsCommandContext
		}
		out, stdErr, err := diagnosticsCommand(cmdCtx, s.options.C3, projectRoot)
		_ = out
		if cmdCtx.Err() != nil {
			staleCancelled = true
			return
		}

		if !s.isCurrentDiagnosticsGeneration(runGeneration) {
			staleCancelled = true
			return
		}

		errorsInfo, diagnosticsDisabled := extractErrorDiagnostics(stdErr.String())

		if diagnosticsDisabled {
			s.options.Diagnostics.Enabled = false
			clearedCount += s.clearDiagnosticsForFiles(s.state, notify, nil)
			notify(protocol.ServerWindowShowMessage, protocol.ShowMessageParams{
				Type:    protocol.MessageTypeWarning,
				Message: "C3-LSP disabled diagnostics: compiler does not support --lsp diagnostics format",
			})
			return
		}

		// Check for fatal errors (errors not starting with > LSPERR|) e.g., project configuration issues
		if err != nil && len(errorsInfo) == 0 {
			stderrOutput := stdErr.String()
			log.Println("Diagnostics report: c3c command failed:", err)

			// Check if this is a configuration error (missing directory, invalid project.json, etc.)
			if strings.Contains(stderrOutput, "Can't open the directory") ||
				strings.Contains(stderrOutput, "No such file or directory") ||
				strings.Contains(stderrOutput, "project.json") {
				log.Println("Project configuration error detected. Please check your project.json file and ensure all referenced directories exist.")
				notify(protocol.ServerWindowShowMessage, protocol.ShowMessageParams{
					Type:    protocol.MessageTypeError,
					Message: "C3 project configuration error: please check project.json and referenced directories",
				})
			}

			// Clear old diagnostics since we can't generate new ones
			clearedCount += s.clearDiagnosticsForFiles(s.state, notify, nil)
			return
		}

		if len(errorsInfo) == 0 && err == nil {
			// No diagnostics to report, clear existing ones.
			clearedCount += s.clearDiagnosticsForFiles(s.state, notify, nil)
			return
		}

		if err != nil {
			log.Println("Diagnostics report:", err)
		}

		if !s.isCurrentDiagnosticsGeneration(runGeneration) {
			staleCancelled = true
			return
		}

		// Send empty diagnostics for those files that had previously an error, but not anymore.
		// If this is not done, the IDE will keep displaying the errors.
		for k := range s.state.GetDocumentDiagnostics() {
			if !s.isCurrentDiagnosticsGeneration(runGeneration) {
				staleCancelled = true
				return
			}

			if hasFileFilter && !relevantFiles[k] {
				continue
			}

			if !hasDiagnosticForFile(k, errorsInfo) {
				s.state.RemoveDocumentDiagnostics(k)
				clearedCount++
				notify(protocol.ServerTextDocumentPublishDiagnostics,
					protocol.PublishDiagnosticsParams{
						URI:         fs.ConvertPathToURI(k, s.options.C3.StdlibPath),
						Diagnostics: []protocol.Diagnostic{},
					})
			}
		}

		for _, errInfo := range errorsInfo {
			if !s.isCurrentDiagnosticsGeneration(runGeneration) {
				staleCancelled = true
				return
			}

			if hasFileFilter && !relevantFiles[errInfo.File] {
				continue
			}

			newDiagnostics := []protocol.Diagnostic{
				errInfo.Diagnostic,
			}
			state.SetDocumentDiagnostics(errInfo.File, newDiagnostics)
			publishedCount++
			notify(
				protocol.ServerTextDocumentPublishDiagnostics,
				protocol.PublishDiagnosticsParams{
					URI:         fs.ConvertPathToURI(errInfo.File, s.options.C3.StdlibPath),
					Diagnostics: newDiagnostics,
				})
		}
	}

	dispatch := func() {
		fingerprint := diagnosticsEnqueueFingerprint(projectRoot, mode, triggerURI, state.Revision())
		s.enqueueDiagnosticsRun(projectRoot, fingerprint, runDiagnostics)
	}

	if delay {
		if mode == diagnosticsRunFull {
			s.diag.fullDebounced(dispatch)
		} else {
			s.diag.quickDebounced(dispatch)
		}
	} else {
		dispatch()
	}
}

func diagnosticsEnqueueFingerprint(root string, mode diagnosticsRunMode, triggerURI *protocol.DocumentUri, revision uint64) string {
	trigger := ""
	if triggerURI != nil {
		trigger = string(*triggerURI)
	}

	return fmt.Sprintf("%s|%s|%s|%d", root, mode, trigger, revision)
}

func diagnosticsRelevantFiles(state *project_state.ProjectState, triggerURI *protocol.DocumentUri, invalidation project_state.InvalidationScope) map[string]bool {
	if triggerURI == nil {
		return nil
	}

	relevantDocs := state.GetDocumentsForModules(invalidation.ImpactedModules)
	if len(relevantDocs) == 0 {
		return nil
	}

	relevant := make(map[string]bool, len(relevantDocs))
	for _, doc := range relevantDocs {
		relevant[doc] = true
	}

	return relevant
}

func (s *Server) swapDiagnosticsCommandCancel(cancel context.CancelFunc) uint64 {
	if s == nil {
		return 0
	}

	s.diag.runMu.Lock()
	s.diag.runID++
	runID := s.diag.runID
	previous := s.diag.runCancel
	s.diag.runCancel = cancel
	s.diag.runMu.Unlock()

	if previous != nil {
		previous()
	}

	return runID
}

func (s *Server) clearDiagnosticsCommandCancel(runID uint64) {
	if s == nil {
		return
	}

	s.diag.runMu.Lock()
	if s.diag.runID == runID {
		s.diag.runCancel = nil
	}
	s.diag.runMu.Unlock()
}

type ErrorInfo struct {
	File       string
	Diagnostic protocol.Diagnostic
}

func extractErrorDiagnostics(output string) ([]ErrorInfo, bool) {
	errorsInfo := []ErrorInfo{}
	diagnosticsDisabled := false

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// > LSPERR|error|"/<path>>/test.c3"|13|47|"Expected ';'"
		if !strings.HasPrefix(line, "> LSP") {
			continue
		}

		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			// Disable future diagnostics, looks like c3c is an old version.
			diagnosticsDisabled = true
			continue
		}

		var severity protocol.DiagnosticSeverity
		switch parts[1] {
		case "error":
			severity = protocol.DiagnosticSeverityError
		case "warning":
			severity = protocol.DiagnosticSeverityWarning
		default:
			continue
		}

		errorLine, err := strconv.Atoi(parts[3])
		if err != nil || errorLine <= 0 {
			continue
		}
		errorLine--
		character, err := strconv.Atoi(parts[4])
		if err != nil || character <= 0 {
			continue
		}
		character--

		message := strings.Trim(parts[5], `"`)
		errorsInfo = append(errorsInfo, ErrorInfo{
			File: strings.Trim(parts[2], `"`),
			Diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(errorLine), Character: protocol.UInteger(character)},
					End:   protocol.Position{Line: protocol.UInteger(errorLine), Character: protocol.UInteger(character + 1)},
				},
				Severity: cast.ToPtr(severity),
				Source:   cast.ToPtr("c3c build --lsp"),
				Message:  message,
			},
		})
	}

	return errorsInfo, diagnosticsDisabled
}

func (s *Server) clearDiagnosticsForFiles(state *project_state.ProjectState, notify glsp.NotifyFunc, filter map[string]bool) int {
	cleared := 0
	for k := range state.GetDocumentDiagnostics() {
		if filter != nil && !filter[k] {
			continue
		}

		notify(protocol.ServerTextDocumentPublishDiagnostics,
			protocol.PublishDiagnosticsParams{
				URI:         fs.ConvertPathToURI(k, s.options.C3.StdlibPath),
				Diagnostics: []protocol.Diagnostic{},
			})
		state.RemoveDocumentDiagnostics(k)
		cleared++
	}

	if filter == nil {
		state.ClearDocumentDiagnostics()
	}

	return cleared
}

func hasDiagnosticForFile(file string, errorsInfo []ErrorInfo) bool {
	for _, v := range errorsInfo {
		if file == v.File {
			return true
		}
	}

	return false
}
