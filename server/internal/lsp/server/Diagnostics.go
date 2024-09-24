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
		if err == nil {
			s.clearOldDiagnostics(s.state, notify)
			return
		}

		log.Println("An error:", err)
		errorsInfo, diagnosticsDisabled := extractErrorDiagnostics(stdErr.String())

		if diagnosticsDisabled {
			s.options.Diagnostics.Enabled = false
			s.clearOldDiagnostics(s.state, notify)
			return
		}

		// Send empty diagnostics for those files that had previously an error, but not anymore.
		// If this is not done, the IDE will keep displaying the errors.
		for k := range s.state.GetDocumentDiagnostics() {
			if !hasDiagnosticForFile(k, errorsInfo) {
				s.state.RemoveDocumentDiagnostics(k)
				notify(protocol.ServerTextDocumentPublishDiagnostics,
					protocol.PublishDiagnosticsParams{
						URI:         k,
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
		if strings.HasPrefix(line, "Error") {
			// Procesa la línea de error
			parts := strings.Split(line, "|")
			if parts[0] == "Error" {
				if len(parts) != 5 {
					// Disable future diagnostics, looks like c3c is an old version.
					diagnosticsDisabled = true
				} else {
					line, err := strconv.Atoi(parts[2])
					if err != nil {
						continue
					}
					line -= 1
					character, err := strconv.Atoi(parts[3])
					if err != nil {
						continue
					}
					character -= 1

					errorsInfo = append(errorsInfo, ErrorInfo{
						File: parts[1],
						Diagnostic: protocol.Diagnostic{
							Range: protocol.Range{
								Start: protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(character)},
								End:   protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(99)},
							},
							Severity: cast.ToPtr(protocol.DiagnosticSeverityError),
							Source:   cast.ToPtr("c3c build --test"),
							Message:  parts[4],
						},
					})
				}
			}
			break
		}
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
