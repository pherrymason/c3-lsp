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

func (s *Server) RefreshDiagnostics(state *project_state.ProjectState, notify glsp.NotifyFunc, delay bool) {
	if state.IsCalculatingDiagnostics() {
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
	err := command.Run()
	log.Println("output:", out.String())
	log.Println("output:", stdErr.String())
	if err != nil {
		log.Println("An error:", err)
		errorInfo := extractErrors(stdErr.String())

		diagnostics := []protocol.Diagnostic{
			errorInfo.Diagnostic,
		}

		go notify(
			protocol.ServerTextDocumentPublishDiagnostics,
			protocol.PublishDiagnosticsParams{
				URI:         state.GetProjectRootURI() + "/src/" + errorInfo.File,
				Diagnostics: diagnostics,
			})
	}

	state.SetCalculateDiagnostics(false)
}

type ErrorInfo struct {
	File       string
	Diagnostic protocol.Diagnostic
}

func extractErrors(output string) ErrorInfo {
	var errorInfo ErrorInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Error") {
			// Procesa la línea de error
			parts := strings.Split(line, "|")
			if len(parts) == 4 {
				line, err := strconv.Atoi(parts[2])
				if err != nil {
					continue
				}
				line -= 1

				errorInfo = ErrorInfo{
					File: parts[1],
					Diagnostic: protocol.Diagnostic{
						Range: protocol.Range{
							Start: protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(0)},
							End:   protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(99)},
						},
						Severity: cast.ToPtr(protocol.DiagnosticSeverityError),
						Source:   cast.ToPtr("c3c build --test"),
						Message:  parts[3],
					},
				}
			}
			break // Asumimos que solo te interesa el primer error
		}
	}

	return errorInfo
}