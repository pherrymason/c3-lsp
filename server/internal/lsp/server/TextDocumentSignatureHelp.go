package server

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis"
	"github.com/pherrymason/c3-lsp/pkg/featureflags"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// textDocument/signatureHelp: {"context":{"isRetrigger":false,"triggerCharacter":"(","triggerKind":2},"position":{"character":20,"line":8},"textDocument":{"uri":"file:///Volumes/Development/raul/projects/game-dev/raul-game-documents/murder-c3/src/main.c3"}}
func (srv *Server) TextDocumentSignatureHelp(context *glsp.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	if featureflags.IsActive(featureflags.UseGeneratedAST) {
		doc, _ := srv.documents.GetDocument(params.TextDocument.URI)
		signatureHelp := analysis.BuildSignatureHelp(
			doc,
			lsp.NewPositionFromProtocol(params.Position),
			srv.documents,
			srv.symbolTable,
		)

		return signatureHelp, nil
	}

	// -----------------------
	// Old implementation
	// -----------------------

	// Rewind position after previous "("
	docId := utils.NormalizePath(params.TextDocument.URI)
	doc := srv.state.GetDocument(docId)
	posOption := doc.SourceCode.RewindBeforePreviousParenthesis(symbols.NewPositionFromLSPPosition(params.Position))

	if posOption.IsNone() {
		return nil, nil
	}

	foundSymbolOption := srv.search.FindSymbolDeclarationInWorkspace(
		docId,
		posOption.Get(),
		srv.state,
	)
	if foundSymbolOption.IsNone() {
		return nil, nil
	}

	foundSymbol := foundSymbolOption.Get()
	function, ok := foundSymbol.(*symbols.Function)
	if !ok {
		return nil, nil
	}

	parameters := []protocol.ParameterInformation{}
	argsToStringify := []string{}
	for _, arg := range function.GetArguments() {
		argsToStringify = append(
			argsToStringify,
			arg.GetType().String()+" "+arg.GetName(),
		)
		parameters = append(
			parameters,
			protocol.ParameterInformation{
				Label: arg.GetType().String() + " " + arg.GetName(),

				// TODO: Parse '@param' contract text to get param docs
				Documentation: nil,
			},
		)
	}

	// Count number of commas (,) written from previous `(`
	activeParameter := countWrittenArguments(posOption.Get(), doc.SourceCode)

	var docs any = nil
	docComment := function.GetDocComment()
	if docComment != nil {
		docs = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: docComment.DisplayBodyWithContracts(),
		}
	}

	signature := protocol.SignatureInformation{
		Label:         function.GetFQN() + "(" + strings.Join(argsToStringify, ", ") + ")",
		Parameters:    parameters,
		Documentation: docs,
	}
	if activeParameter.IsSome() {
		arg := activeParameter.Get()
		signature.ActiveParameter = &arg
	}

	signatureHelp := protocol.SignatureHelp{
		Signatures: []protocol.SignatureInformation{signature},
	}

	return &signatureHelp, nil
}

func countWrittenArguments(startArgumentsPosition symbols.Position, s sourcecode.SourceCode) option.Option[uint32] {
	index := startArgumentsPosition.IndexIn(s.Text)
	commas := uint32(0)
	length := len(s.Text)
	for {
		if index >= length {
			break
		}

		if rune(s.Text[index]) == ')' {
			return option.None[uint32]()
		}

		if rune(s.Text[index]) == ',' {
			commas++
		}

		index++
	}

	return option.Some(commas)
}
