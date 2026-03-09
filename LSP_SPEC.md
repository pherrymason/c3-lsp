# LSP 3.17 Support Checklist

Scope: `server/internal/lsp/server` in this repository, mapped against the LSP 3.17 spec.

## Implemented

- [x] `initialize`
- [x] `initialized`
- [x] `shutdown`
- [x] `$/setTrace`
- [x] `$/logTrace`
- [x] `textDocument/didOpen`
- [x] `textDocument/didChange`
- [x] `textDocument/didClose`
- [x] `textDocument/didSave`
- [x] `textDocument/hover`
- [x] `textDocument/declaration`
- [x] `textDocument/definition`
- [x] `textDocument/implementation`
- [x] `textDocument/typeDefinition`
- [x] `textDocument/completion`
- [x] `completionItem/resolve` (passthrough)
- [x] `textDocument/signatureHelp`
- [x] `textDocument/formatting`
- [x] `textDocument/rename`
- [x] `textDocument/prepareRename`
- [x] `textDocument/references`
- [x] `textDocument/documentHighlight`
- [x] `textDocument/documentSymbol`
- [x] `textDocument/foldingRange`
- [x] `textDocument/selectionRange`
- [x] `textDocument/rangeFormatting`
- [x] `textDocument/onTypeFormatting`
- [x] `textDocument/documentLink`
- [x] `documentLink/resolve`
- [x] `textDocument/publishDiagnostics` (server notification)
- [x] `window/logMessage`
- [x] `window/showMessage`
- [x] `workspace/didChangeConfiguration`
- [x] `workspace/configuration`
- [x] `workspace/symbol`
- [x] `workspace/didDeleteFiles`
- [x] `workspace/didRenameFiles`

## Missing (ordered easiest -> hardest)

### Easy
- [x] `telemetry/event`
- [x] `workspace/workspaceFolders` request
- [x] `workspace/willCreateFiles`
- [x] `workspace/didCreateFiles`
- [x] `workspace/willRenameFiles`
- [x] `workspace/willDeleteFiles`

### Medium

- [x] `textDocument/willSave`
- [x] `textDocument/willSaveWaitUntil`
- [x] `workspaceSymbol/resolve`
- [x] `textDocument/linkedEditingRange`
- [x] `window/showMessageRequest`
- [x] `window/showDocument`
- [x] `workspace/executeCommand`
- [x] `workspace/applyEdit`

### Hard

- [ ] Notebook document sync (`notebookDocument/*`)
- [x] `textDocument/codeAction`
- [x] `codeAction/resolve`
- [ ] `textDocument/codeLens`
- [ ] `codeLens/resolve`
- [ ] `workspace/codeLens/refresh`
- [ ] `textDocument/semanticTokens/*`
- [ ] `textDocument/inlayHint`
- [ ] `inlayHint/resolve`
- [ ] `workspace/inlayHint/refresh`
- [ ] `textDocument/inlineValue`
- [ ] `workspace/inlineValue/refresh`
- [ ] `textDocument/documentColor`
- [ ] `textDocument/colorPresentation`
- [x] `textDocument/moniker`
- [x] `textDocument/prepareCallHierarchy`
- [x] `callHierarchy/incomingCalls`
- [x] `callHierarchy/outgoingCalls`
- [ ] Type hierarchy (`textDocument/prepareTypeHierarchy`, `typeHierarchy/supertypes`, `typeHierarchy/subtypes`)
- [x] `window/workDoneProgress/create`
- [x] `window/workDoneProgress/cancel`

## Notes

- `workspace/didChangeWatchedFiles` updates deleted/changed documents, reloads project config markers, and triggers scoped reindexing for external source changes.
- `workspace/didChangeWorkspaceFolders` now updates root tracking, cancels removed-root indexing, and schedules indexing for added buildable roots.
- `window/showMessageRequest` and `window/showDocument` are wrapped via request helpers; reload-configuration command uses them in an actionable `project.json` open prompt flow.
- `workspace/executeCommand` supports `c3lsp.reindexWorkspace`, `c3lsp.reloadConfiguration`, and `c3lsp.clearDiagnosticsCache`; diagnostics-cache command uses internal `workspace/applyEdit` helper when client request channel is available.
- `textDocument/linkedEditingRange` returns deterministic identifier-linked ranges for module/symbol targets in the active document.
- `workspaceSymbol/resolve` is handled through a protocol-extension dispatch path (for clients that send it despite 3.16 surface); current behavior is deterministic pass-through of symbol payload.
- `window/workDoneProgress/create` and `window/workDoneProgress/cancel` are handled via request/cancel helpers and deterministic token tracking.
- `textDocument/moniker` returns deterministic C3 moniker identifiers for module and symbol targets with stable kind/uniqueness mapping.
- Call hierarchy is implemented for function symbols with deterministic prepare/incoming/outgoing behavior based on indexed references.
- `textDocument/codeAction` currently returns a deterministic empty list and `codeAction/resolve` is a deterministic pass-through for forward-compatible client flows.
- `renameProvider` is wired to `textDocument/prepareRename` and `textDocument/rename`.
