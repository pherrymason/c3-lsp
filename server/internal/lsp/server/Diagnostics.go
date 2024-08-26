package server

import (
	"bytes"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) RunDiagnostics(state *project_state.ProjectState, notify glsp.NotifyFunc, delay bool) {
	if state.IsCalculatingDiagnostics() {
		return
	}
	if !s.options.DiagnosticsEnabled {
		return
	}

	state.SetCalculateDiagnostics(true)

	binary := "c3c"
	if s.options.C3CPath.IsSome() {
		binary = s.options.C3CPath.Get()
	}
	command := exec.Command(binary, "build", "--test")
	command.Dir = state.GetProjectRootURI()

	// set var to get the output
	var out bytes.Buffer
	var stdErr bytes.Buffer

	// set the output to our variable
	command.Stdout = &out
	command.Stderr = &stdErr

	runDiagnostics := func() {
		err := command.Run()
		log.Println("output:", out.String())
		log.Println("output:", stdErr.String())
		if err != nil {
			log.Println("An error:", err)
			errorInfo, diagnosticsDisabled := extractErrors(stdErr.String())

			if diagnosticsDisabled {
				s.options.DiagnosticsEnabled = false
			} else {
				diagnostics := []protocol.Diagnostic{
					errorInfo.Diagnostic,
				}

				go notify(
					protocol.ServerTextDocumentPublishDiagnostics,
					protocol.PublishDiagnosticsParams{
						URI:         errorInfo.File,
						Diagnostics: diagnostics,
					})
			}
		}

		state.SetCalculateDiagnostics(false)
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

func extractErrors(output string) (ErrorInfo, bool) {
	var errorInfo ErrorInfo
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

					errorInfo = ErrorInfo{
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
					}
				}
			}
			break
		}
	}

	return errorInfo, diagnosticsDisabled
}
