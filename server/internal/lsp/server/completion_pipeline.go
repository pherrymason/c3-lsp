package server

import (
	stdctx "context"
	"strings"
	"time"

	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type completionContext struct {
	cursor             ctx.CursorContext
	suggestions        []protocol.CompletionItem
	searchDuration     time.Duration
	buildContextTime   time.Duration
	snippetSupport     bool
	docText            string
	docURI             protocol.DocumentUri
	docVersion         int32
	position           symbols.Position
	symbolInPosition   string
	symbolStartIndex   int
	replaceRange       protocol.Range
	cursorIndex        int
	explicitModuleName string
}

type completionRenderStats struct {
	structFieldLookups int
}

func (h *Server) buildCompletionContextWithCancel(params *protocol.CompletionParams, requestCtx stdctx.Context) completionContext {
	if requestCtx == nil {
		requestCtx = stdctx.Background()
	}

	buildStart := time.Now()
	select {
	case <-requestCtx.Done():
		return completionContext{}
	default:
	}

	cursorContext := ctx.BuildFromDocumentPosition(
		params.Position,
		utils.NormalizePath(params.TextDocument.URI),
		h.state,
	)

	snippetSupport := clientSupportsCompletionSnippets(h.clientCapabilities)

	select {
	case <-requestCtx.Done():
		return completionContext{}
	default:
	}

	doc := h.state.GetDocument(cursorContext.DocURI)
	if doc == nil {
		return completionContext{
			buildContextTime: time.Since(buildStart),
			snippetSupport:   snippetSupport,
		}
	}

	unitModules := h.state.GetUnitModulesByDoc(doc.URI)
	if unitModules == nil {
		return completionContext{
			cursor:           cursorContext,
			buildContextTime: time.Since(buildStart),
			snippetSupport:   snippetSupport,
			docText:          doc.SourceCode.Text,
			docURI:           doc.URI,
			docVersion:       doc.Version,
			position:         cursorContext.Position,
		}
	}
	rewoundPos := doc.RewindPosition(cursorContext.Position)
	symbol := doc.SourceCode.SymbolInPosition(rewoundPos, unitModules)

	return completionContext{
		cursor:             cursorContext,
		buildContextTime:   time.Since(buildStart),
		snippetSupport:     snippetSupport,
		docText:            doc.SourceCode.Text,
		docURI:             doc.URI,
		docVersion:         doc.Version,
		position:           cursorContext.Position,
		symbolInPosition:   symbol.GetFullQualifiedName(),
		symbolStartIndex:   symbol.FullTextRange().Start.IndexIn(doc.SourceCode.Text),
		replaceRange:       symbol.FullTextRange().ToLSP(),
		cursorIndex:        cursorContext.Position.IndexIn(doc.SourceCode.Text),
		explicitModuleName: modulePathNameFromWord(symbol),
	}
}

func (h *Server) renderCompletionItemsWithStats(c completionContext, requestCtx stdctx.Context) ([]completionItemWithLabelDetails, completionRenderStats, bool) {
	if requestCtx == nil {
		requestCtx = stdctx.Background()
	}

	items := make([]completionItemWithLabelDetails, 0, len(c.suggestions))
	stats := completionRenderStats{}
	structFieldMemo := map[string][]string{}
	structMode := structCompletionContext(c.docText, c.symbolStartIndex, c.cursorIndex)
	trailing := chooseTrailingToken(c.docText, c.cursorIndex)

	for _, suggestion := range c.suggestions {
		select {
		case <-requestCtx.Done():
			return nil, completionRenderStats{}, true
		default:
		}

		item := completionItemWithLabelDetails{CompletionItem: suggestion}

		if suggestion.Detail != nil {
			item.LabelDetails = &completionItemLabelDetails{
				Description: cast.ToPtr(" " + *suggestion.Detail),
				Detail:      completionKindDescription(suggestion.Kind),
			}
		}

		if suggestion.Detail != nil && item.Documentation == nil {
			if suggestion.Kind != nil && (*suggestion.Kind == protocol.CompletionItemKindFunction || *suggestion.Kind == protocol.CompletionItemKindMethod || *suggestion.Kind == protocol.CompletionItemKindStruct) {
				doc := signatureDocumentation(*suggestion.Detail)
				item.Documentation = doc
			}
		}

		isCallable := false
		isStructSnippet := false
		if suggestion.Kind != nil && suggestion.Detail != nil {
			if *suggestion.Kind == protocol.CompletionItemKindFunction || *suggestion.Kind == protocol.CompletionItemKindMethod {
				if snippet, ok := buildCallableSnippet(suggestion.Label, *suggestion.Detail); ok {
					isCallable = true
					insertText := snippet
					if !c.snippetSupport {
						insertText = snippetToPlainInsertText(snippet)
					}

					if textEdit, ok := item.TextEdit.(protocol.TextEdit); ok {
						textEdit.NewText = insertText
						item.TextEdit = textEdit
					} else {
						item.InsertText = cast.ToPtr(insertText)
					}

					if c.snippetSupport {
						snippetFormat := protocol.InsertTextFormatSnippet
						item.InsertTextFormat = &snippetFormat
					}
				}
			}
		}

		if suggestion.Kind != nil && *suggestion.Kind == protocol.CompletionItemKindStruct {
			if structMode != structCompletionNone {
				fields := extractStructFieldsFromData(suggestion.Data)
				if len(fields) == 0 {
					memoKey := c.explicitModuleName + "|" + suggestion.Label
					if cached, ok := structFieldMemo[memoKey]; ok {
						fields = cached
					} else {
						stats.structFieldLookups++
						fields = structFieldsInScope(h.state, c.docURI, c.position, suggestion.Label, c.explicitModuleName)
						structFieldMemo[memoKey] = fields
					}
				}

				structTypeName := completedStructTypeName(c.symbolInPosition, suggestion.Label)

				if snippet, ok := buildStructSnippet(structMode, structTypeName, fields); ok {
					isStructSnippet = true
					insertText := snippet
					if !c.snippetSupport {
						insertText = snippetToPlainInsertText(snippet)
					}

					item.TextEdit = protocol.TextEdit{
						NewText: insertText,
						Range:   c.replaceRange,
					}
					item.InsertText = nil

					if c.snippetSupport {
						snippetFormat := protocol.InsertTextFormatSnippet
						item.InsertTextFormat = &snippetFormat
					}
				}
			}
		}

		shouldAddTrailing := isCallable || (isStructSnippet && structMode == structCompletionValue)
		if shouldAddTrailing && trailing != "" {
			if textEdit, ok := item.TextEdit.(protocol.TextEdit); ok {
				if !strings.HasSuffix(textEdit.NewText, trailing) {
					textEdit.NewText += trailing
					item.TextEdit = textEdit
				}
			} else if item.InsertText != nil {
				if !strings.HasSuffix(*item.InsertText, trailing) {
					item.InsertText = cast.ToPtr(*item.InsertText + trailing)
				}
			} else {
				item.InsertText = cast.ToPtr(item.Label + trailing)
			}
		}

		items = append(items, item)
	}

	return items, stats, false
}
