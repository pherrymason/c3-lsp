package server

import (
	"path/filepath"
	"time"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var noopNotify glsp.NotifyFunc = func(string, any) {}

// Support "Hover"
func (s *Server) TextDocumentDidSave(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	traceEnabled := s.server != nil && perfEnabled()
	start := time.Time{}
	startRootHits := uint64(0)
	startRootMisses := uint64(0)
	if traceEnabled {
		start = time.Now()
		startRootHits, startRootMisses = s.projectRootCacheCounters()
	}
	quickScheduled := false
	fullScheduled := false
	skipReason := ""
	docVersion := int32(0)
	if traceEnabled {
		defer func() {
			if params == nil {
				return
			}
			rootHits, rootMisses := s.projectRootCacheCounters()
			telemetry := computeRootCacheTelemetry(startRootHits, startRootMisses, rootHits, rootMisses)
			perfLogf(
				s.server.Log,
				"textDocument/didSave",
				start,
				"uri=%s doc_version=%d quick_scheduled=%t full_scheduled=%t skip_reason=%s %s",
				params.TextDocument.URI,
				docVersion,
				quickScheduled,
				fullScheduled,
				skipReason,
				formatRootCacheTelemetry(telemetry),
			)
		}()
	}

	if !s.shouldProcessNotification(protocol.MethodTextDocumentDidSave) {
		return nil
	}
	if params == nil {
		return nil
	}

	docID := s.normalizedDocIDFromURI(params.TextDocument.URI)
	if isWorkspaceConfigFile(docID) {
		skipReason = "workspace_config"
		s.reloadWorkspaceConfiguration(ctx, filepath.Dir(docID), "textDocument/didSave")
		return nil
	}
	if doc := s.state.GetDocumentByNormalizedID(docID); doc != nil {
		docVersion = doc.Version
		if !s.shouldScheduleDiagnosticsForSave(docID, docVersion) {
			skipReason = "unchanged_version"
			return nil
		}
	}

	notify := noopNotify
	if ctx != nil {
		notify = ctx.Notify
	}

	s.RunDiagnosticsQuick(s.state, notify, true, &params.TextDocument.URI)
	quickScheduled = true
	s.scheduleDiagnosticsFullAfterSaveIdle(ctx)
	fullScheduled = true
	return nil
}

func (s *Server) shouldScheduleDiagnosticsForSave(docID string, version int32) bool {
	if s == nil || docID == "" || version <= 0 {
		return true
	}

	s.diag.saveMu.Lock()
	defer s.diag.saveMu.Unlock()

	if previousVersion, ok := s.diag.saveDocVersions[docID]; ok && previousVersion == version {
		return false
	}

	s.diag.saveDocVersions[docID] = version
	return true
}
