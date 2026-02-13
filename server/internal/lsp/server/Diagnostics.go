package server

import (
	"log"
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) RunDiagnostics(state *project_state.ProjectState, notify glsp.NotifyFunc, delay bool) {
	if !s.options.Diagnostics.Enabled {
		return
	}

	runDiagnostics := func() {
		out, stdErr, err := c3c.CheckC3ErrorsCommand(s.options.C3, state.GetProjectRootURI())
		log.Println("output:", out.String())
		log.Println("output:", stdErr.String())

		errorsInfo, diagnosticsDisabled := extractErrorDiagnostics(stdErr.String())

		if diagnosticsDisabled {
			s.options.Diagnostics.Enabled = false
			s.clearOldDiagnostics(s.state, notify)
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
			s.clearOldDiagnostics(s.state, notify)
			return
		}

		if len(errorsInfo) == 0 && err == nil {
			// No diagnostics to report, clear existing ones.
			s.clearOldDiagnostics(s.state, notify)
			return
		}

		if err != nil {
			log.Println("Diagnostics report:", err)
		}
		// Send empty diagnostics for those files that had previously an error, but not anymore.
		// If this is not done, the IDE will keep displaying the errors.
		for k := range s.state.GetDocumentDiagnostics() {
			if !hasDiagnosticForFile(k, errorsInfo) {
				s.state.RemoveDocumentDiagnostics(k)
				notify(protocol.ServerTextDocumentPublishDiagnostics,
					protocol.PublishDiagnosticsParams{
						URI:         fs.ConvertPathToURI(k, s.options.C3.StdlibPath),
						Diagnostics: []protocol.Diagnostic{},
					})
			}
		}

		for _, errInfo := range errorsInfo {
			newDiagnostics := []protocol.Diagnostic{
				errInfo.Diagnostic,
			}
			state.SetDocumentDiagnostics(errInfo.File, newDiagnostics)
			notify(
				protocol.ServerTextDocumentPublishDiagnostics,
				protocol.PublishDiagnosticsParams{
					URI:         fs.ConvertPathToURI(errInfo.File, s.options.C3.StdlibPath),
					Diagnostics: newDiagnostics,
				})
		}
	}

	if delay {
		s.diagnosticDebounced(runDiagnostics)
	} else {
		runDiagnostics()
	}
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

func (s *Server) clearOldDiagnostics(state *project_state.ProjectState, notify glsp.NotifyFunc) {
	for k := range state.GetDocumentDiagnostics() {
		go notify(protocol.ServerTextDocumentPublishDiagnostics,
			protocol.PublishDiagnosticsParams{
				URI:         fs.ConvertPathToURI(k, s.options.C3.StdlibPath),
				Diagnostics: []protocol.Diagnostic{},
			})
	}
	state.ClearDocumentDiagnostics()
}

func hasDiagnosticForFile(file string, errorsInfo []ErrorInfo) bool {
	for _, v := range errorsInfo {
		if file == v.File {
			return true
		}
	}

	return false
}
