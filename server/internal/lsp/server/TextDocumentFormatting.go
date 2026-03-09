package server

import (
	"bytes"
	stdcontext "context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const formatterDefaultConfigSentinel = ":default:"
const formatterCommandTimeout = 8 * time.Second

func (s *Server) TextDocumentFormatting(_ *glsp.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	if root := s.resolveProjectRootForURI(&params.TextDocument.URI); root != "" {
		s.configureProjectForRoot(root)
	}

	docURI := utils.NormalizePath(params.TextDocument.URI)
	doc := s.state.GetDocument(docURI)
	if doc == nil {
		return []protocol.TextEdit{}, nil
	}

	formatted, err := s.formatSourceWithC3Fmt(params.TextDocument.URI, doc.SourceCode.Text)
	if err != nil {
		return nil, err
	}

	return textEditsFromFormattedDocument(doc.SourceCode.Text, formatted), nil
}

func (s *Server) TextDocumentRangeFormatting(_ *glsp.Context, params *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	if root := s.resolveProjectRootForURI(&params.TextDocument.URI); root != "" {
		s.configureProjectForRoot(root)
	}

	docURI := utils.NormalizePath(params.TextDocument.URI)
	doc := s.state.GetDocument(docURI)
	if doc == nil {
		return []protocol.TextEdit{}, nil
	}

	formatted, err := s.formatSourceWithC3Fmt(params.TextDocument.URI, doc.SourceCode.Text)
	if err != nil {
		return nil, err
	}

	edits := textEditsFromFormattedDocument(doc.SourceCode.Text, formatted)
	if len(edits) == 0 {
		return edits, nil
	}

	for _, edit := range edits {
		if rangesOverlap(edit.Range, params.Range) {
			return edits, nil
		}
	}

	return []protocol.TextEdit{}, nil
}

func (s *Server) TextDocumentOnTypeFormatting(_ *glsp.Context, params *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	if params == nil || !isSupportedOnTypeFormattingTrigger(params.Ch) {
		return []protocol.TextEdit{}, nil
	}
	if root := s.resolveProjectRootForURI(&params.TextDocument.URI); root != "" {
		s.configureProjectForRoot(root)
	}

	docURI := utils.NormalizePath(params.TextDocument.URI)
	doc := s.state.GetDocument(docURI)
	if doc == nil {
		return []protocol.TextEdit{}, nil
	}

	formatted, err := s.formatSourceWithC3Fmt(params.TextDocument.URI, doc.SourceCode.Text)
	if err != nil {
		return nil, err
	}

	return textEditsFromFormattedDocument(doc.SourceCode.Text, formatted), nil
}

func isSupportedOnTypeFormattingTrigger(ch string) bool {
	return ch == "}" || ch == ";"
}

func onTypeFormattingAdditionalTriggerCharacters() []string {
	return []string{";"}
}

func (s *Server) formatSourceWithC3Fmt(docURI protocol.DocumentUri, source string) (string, error) {

	if s.options.Formatting.C3FmtPath.IsNone() {
		return "", fmt.Errorf("c3 formatter not configured: set Formatting.c3fmt in c3lsp.json")
	}

	binaryPath, err := resolveFormatterBinary(s.options.Formatting.C3FmtPath.Get())
	if err != nil {
		return "", err
	}

	args := formatterCommandArgs(s.options.Formatting.Config)
	runCtx, cancel := stdcontext.WithTimeout(stdcontext.Background(), formatterCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, binaryPath, args...)

	if filePath, err := fs.UriToPath(string(docURI)); err == nil {
		cmd.Dir = filepath.Dir(filePath)
	}

	cmd.Stdin = strings.NewReader(source)

	var out bytes.Buffer
	var stdErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stdErr

	if err := cmd.Run(); err != nil {
		if runCtx.Err() == stdcontext.DeadlineExceeded {
			return "", fmt.Errorf("c3 formatter timed out after %s", formatterCommandTimeout)
		}

		errMsg := strings.TrimSpace(stdErr.String())
		if errMsg != "" {
			return "", fmt.Errorf("c3 formatter failed: %s", errMsg)
		}

		return "", fmt.Errorf("c3 formatter failed: %w", err)
	}

	return out.String(), nil
}

func textEditsFromFormattedDocument(original string, formatted string) []protocol.TextEdit {
	if formatted == original {
		return []protocol.TextEdit{}
	}

	if edit, ok := minimalTextEdit(original, formatted); ok {
		return []protocol.TextEdit{edit}
	}

	return []protocol.TextEdit{fullDocumentTextEdit(original, formatted)}
}

func minimalTextEdit(original string, formatted string) (protocol.TextEdit, bool) {
	if original == formatted {
		return protocol.TextEdit{}, false
	}

	prefix := 0
	maxPrefix := minInt(len(original), len(formatted))
	for prefix < maxPrefix && original[prefix] == formatted[prefix] {
		prefix++
	}

	origSuffix := len(original)
	fmtSuffix := len(formatted)
	for origSuffix > prefix && fmtSuffix > prefix && original[origSuffix-1] == formatted[fmtSuffix-1] {
		origSuffix--
		fmtSuffix--
	}

	if prefix == origSuffix && prefix == fmtSuffix {
		return protocol.TextEdit{}, false
	}

	return protocol.TextEdit{
		Range: protocol.Range{
			Start: byteIndexToLSPPosition(original, prefix),
			End:   byteIndexToLSPPosition(original, origSuffix),
		},
		NewText: formatted[prefix:fmtSuffix],
	}, true
}

func rangesOverlap(a protocol.Range, b protocol.Range) bool {
	return positionBeforeOrEqual(a.Start, b.End) && positionBeforeOrEqual(b.Start, a.End)
}

func positionBeforeOrEqual(a protocol.Position, b protocol.Position) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Character <= b.Character
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatterCommandArgs(config option.Option[string]) []string {
	args := []string{"--stdin", "--stdout"}
	if config.IsNone() {
		return args
	}

	value := strings.TrimSpace(config.Get())
	if value == "" {
		return args
	}

	if value == formatterDefaultConfigSentinel {
		return append(args, "--default")
	}

	return append(args, "--config="+value)
}

func resolveFormatterBinary(configuredPath string) (string, error) {
	trimmed := strings.TrimSpace(configuredPath)
	if trimmed == "" {
		return "", fmt.Errorf("c3 formatter path is empty")
	}

	if info, err := os.Stat(trimmed); err == nil {
		if info.IsDir() {
			return resolveFormatterFromDirectory(trimmed)
		}

		return trimmed, nil
	}

	if strings.ContainsRune(trimmed, os.PathSeparator) || filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("c3 formatter path not found: %s", trimmed)
	}

	resolved, err := exec.LookPath(trimmed)
	if err != nil {
		return "", fmt.Errorf("c3 formatter not found in PATH: %s", trimmed)
	}

	return resolved, nil
}

func resolveFormatterFromDirectory(dir string) (string, error) {
	candidates := []string{
		filepath.Join(dir, "build", "c3fmt"),
		filepath.Join(dir, "c3fmt"),
		filepath.Join(dir, "build", "c3fmt.exe"),
		filepath.Join(dir, "c3fmt.exe"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not locate c3 formatter binary under directory %s", dir)
}

func fullDocumentTextEdit(original string, formatted string) protocol.TextEdit {
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   byteIndexToLSPPosition(original, len(original)),
		},
		NewText: formatted,
	}
}
