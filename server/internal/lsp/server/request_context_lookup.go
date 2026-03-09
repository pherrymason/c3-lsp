package server

import (
	"context"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func (s *Server) findSymbolDeclarationWithContext(ctx context.Context, docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	if ctx == nil {
		return s.search.FindSymbolDeclarationInWorkspace(docID, pos, s.state)
	}

	resultCh := make(chan option.Option[symbols.Indexable], 1)
	go func() {
		resultCh <- s.search.FindSymbolDeclarationInWorkspace(docID, pos, s.state)
	}()

	select {
	case <-ctx.Done():
		return option.None[symbols.Indexable]()
	case result := <-resultCh:
		return result
	}
}
