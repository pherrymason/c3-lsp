# C3LSP Release Notes

## 0.3.4
- Fix crash while writing an inline struct member. (#97)
- Support `<*` and `*>` comments. (#91)
- Show documentation on hover and completion. Credit to @PgBiel
- Optimizations to reduce CPU usage by 6-7x. Credit to @PgBiel [More info](https://github.com/pherrymason/c3-lsp/pull/99)
- Improve syntax highlighting in function information on hovering. Credit to @PgBiel
- Adds type information as well as other information to completions. Credit to @PgBiel
- Fix parsing of non-type alias def (Credit to @PgBiel)
- Improve macro handling. [More info](https://github.com/pherrymason/c3-lsp/pull/103). Credit to @PgBiel

## 0.3.3
- Support named and anonymous sub structs.
- Fix clearing old diagnostics. (#89, #83, #71, #62)
- Fixed crash in some scenarios where no results were found.
- Fix crash when using with Helix editor (#87)

## 0.3.2

- Fix function unnamed argument types not being resolved correctly. Thanks to @insertt
- Fix not displaying correctly collection types on hover.

## 0.3.1

- Added compatibility with stdlib symbols for C3 0.6.2.
- Fixes diagnostic errors persisting after they were fixed (#62).
- C3 Version argument is deprecated. It will try to load last C3 supported version.
- Fallback to last C3 supported version if provided version is not supported.
- Binary name changed to `c3lsp` (before it was `c3-lsp`).

## 0.3

- LSP configuration per project: Place a c3lsp.json in your c3 project to customize LSP configuration. This will also allow to use new configuration settings without needing to wait for IDE extensions to be updated. See Configuration for details.
- Inspect stdlib. Use go to declaration/definition on stdlib symbols by configuring its path in c3lsp.json.
- Fixes diagnostics in Windows platform.

## 0.2.1

- Fixes once an error is detected, it does not disappear once the error is resolved (https://github.com/pherrymason/c3-lsp/issues/59)
- Fixes Go to definition / Declaration.

## 0.2

- New feature: error diagnostics. 
  In order to work, there are some requirements:  
  You will need last version of c3c (>=0.6.2) as some fixes had to be done there (Thanks @clerno !)  
  Either you have c3c in your PATH, or you use set its path with the new argument c3c-path.
- A new argument has been added to control the delay of diagnostics being run: diagnostics-delay. By default is 2 seconds.

## 0.1.0

- C3 language keywords are now suggested in TextDocumentCompletion operation. Thanks @nikpivkin for suggestion.
- Fix server crashing when writing a dot inside a string literal. #44 Thanks @nikpivkin for reporting.
- Fix server crashing when creating a new new untitled c3 file from VS Code without workspaces. Thanks @nikpivkin for fixing.
- Fix server crashing when not finding symbol. Thanks @tclesius for reporting.

## 0.0.8

- Fix Go to definition / Go to declaration on Windows.

## 0.0.7

- Include symbols for stdlib 0.6.1
- New argument --lang-version to specify which c3 version to use
- Improvements indexing stdlib symbols: Generic parameters were ignored.
- Improvements in resolving types referencing generic parameters.
- Go to declaration/definintion failed in Windows.

## 0.0.6
- New LSP feature: SignatureHelper
- Improved resolution of references to types inside indexed symbols (Examples: variable types, function argument and return types, definitions...)
- Support optional types.
- Include Stdlib v0.6 symbols.
- c3i files were ignored.
- Difficulties parsing symbols on files that contained bitstructs with defined ranges.
- Stdlib was not properly indexed, which resulted in stdlib symbols not resolved in some situations.
- Fix completion on stdlib struct methods.
- Clear obsolete indexed symbols as documents are modified.
- Remove indexed elements when a file is removed.
– Symbols were holding old filenames after files being renamed.

## 0.0.5
- Added supports for Enum methods and associative values.
- Parse type information of defs and struct members.
- Properly resolve access path when a def is traversed.
- PENDING: Properly resolve struct members when they include implicit module path
- Resolve partial module paths properly.
- Resolve stdlib symbols: This version includes stdlib symbols from c3 0.5.5.
- Fix passing arguments to server.

## 0.0.4
- Index function declarations. Useful for bindings for example.
- Improve Hover content by enabling syntax highlighting and including function arguments.
- Fix wrong module being resolved.
- Fix finding symbols on imported modules.

## 0.0.3
- TextDocumentCompletion:
  - Added support for fault and module names.
  - Improve completion of struct methods.
- Fix resolving wrong symbol when trying to find module name.
- Fix resolving wrong symbol when there are two or more with the same name in different scopes.
- Fix processing files without symbols.
- Added cli arguments to display configure server behavior.
- Send (optionally) crash reports to Sentry. Only stack traces will sent.

## 0.0.2
- Fix crash finding symbols.
- Generics.
- Added partial support for struct subtyping.

## 0.0.1
- Implements 
  - TextDocumentDeclaration
  - TextDocumentDefinition
  - TextDocumentHover
  - TextDocumentCompletion

