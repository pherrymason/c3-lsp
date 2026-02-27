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
- [x] `textDocument/completion`
- [x] `completionItem/resolve` (passthrough)
- [x] `textDocument/signatureHelp`
- [x] `textDocument/publishDiagnostics` (server notification)
- [x] `window/logMessage`
- [x] `window/showMessage`
- [x] `workspace/didChangeConfiguration`
- [x] `workspace/configuration`
- [x] `workspace/didDeleteFiles`
- [x] `workspace/didRenameFiles`

## Missing (ordered easiest -> hardest)

### Easy
- [ ] `telemetry/event`
- [ ] `workspace/workspaceFolders` request
- [ ] `workspace/willCreateFiles`
- [ ] `workspace/didCreateFiles`
- [ ] `workspace/willRenameFiles`
- [ ] `workspace/willDeleteFiles`

### Medium

- [ ] `textDocument/willSave`
- [ ] `textDocument/willSaveWaitUntil`
- [ ] `textDocument/typeDefinition`
- [ ] `textDocument/references`
- [ ] `textDocument/documentHighlight`
- [ ] `textDocument/documentLink`
- [ ] `documentLink/resolve`
- [ ] `textDocument/documentSymbol`
- [ ] `workspace/symbol`
- [ ] `workspaceSymbol/resolve`
- [ ] `textDocument/foldingRange`
- [ ] `textDocument/selectionRange`
- [ ] `textDocument/formatting`
- [ ] `textDocument/rangeFormatting`
- [ ] `textDocument/onTypeFormatting`
- [ ] `textDocument/rename`
- [ ] `textDocument/prepareRename`
- [ ] `textDocument/linkedEditingRange`
- [ ] `window/showMessageRequest`
- [ ] `window/showDocument`
- [ ] `workspace/executeCommand`
- [ ] `workspace/applyEdit`

### Hard

- [ ] Notebook document sync (`notebookDocument/*`)
- [ ] `textDocument/codeAction`
- [ ] `codeAction/resolve`
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
- [ ] `textDocument/moniker`
- [ ] Call hierarchy (`textDocument/prepareCallHierarchy`, `callHierarchy/incomingCalls`, `callHierarchy/outgoingCalls`)
- [ ] Type hierarchy (`textDocument/prepareTypeHierarchy`, `typeHierarchy/supertypes`, `typeHierarchy/subtypes`)
- [ ] `window/workDoneProgress/create`
- [ ] `window/workDoneProgress/cancel`

## Notes

- `workspace/didChangeWatchedFiles` exists but currently no-op.
- `workspace/didChangeWorkspaceFolders` exists but currently no-op.
- Initialize responses currently advertise `renameProvider`, but no `textDocument/rename` handler is wired yet.
