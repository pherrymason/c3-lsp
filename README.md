# Language Server for the C3 language.
[![Go](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml/badge.svg)](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml)

WIP LSP for [C3 language](https://github.com/c3lang/c3c)  

## Project Goals
Writing a Language Server Protocol (LSP) can be a complex and challenging task. To manage this complexity, our initial focus is on covering the basic yet essential needs of a typical LSP implementation.

### Current target
The main current target, is to cover the most essential feature which is to scan precise information about symbols used within the source code of a project.  
This information can then be used by an IDE to enable the following features:

- **Go to Definition:** Navigate to the exact location where a symbol is defined.
- **Hover Information:** Display detailed information about symbols when hovering over them.
- **Autocomplete:** Suggest relevant symbols and code completions as you type.

These features will significantly improve the developer experience by providing accurate and efficient code navigation and assistance.

### Future plans
Once these initial objectives are completed, we will explore additional functionalities that can be added to the project, further enhancing its capabilities and usefulness.

## Server
### Supported OS
Project is written in Golang, so in theory it could be built to any OS supported by Golang.  

### Usage
Simply run `c3-lsp` to start the server.  
It supports the following options:
- **help:** Display accepted options.
- **send-reports:** If enabled (disabled by default) will send __crash__ reports to Sentry so bugs can be debugged easily.
- **lang-version:** Use it to specify a specific c3 language version. By default `c3-lsp` will select the last version supported.


### Installation
Precompiled binaries for the following operating systems are available:

- Linux x64 [download](https://github.com/pherrymason/c3-lsp/releases/download/latest/linux-amd64-c3lsp.zip)  
- MacOS x64 [download](https://github.com/pherrymason/c3-lsp/releases/download/latest/darwin-amd64-c3lsp.zip).

You can also build from source:

- Download and install golang: https://go.dev/
- Clone this repo
- Run `make build`: This will create `c3-lsp` in `server/bin` folder.

### Features
- [x] **TextDocumentCompletion.** Suggests symbols located in project's source code. See notes.
- [x] **Go to declaration.**
- [x] **Go to definition.** 
- [x] **Hover** Displays information about symbol under cursor.
- [x] **Stdlib** Offers symbol information of stdlib (0.5.5)

## IDE extensions
There's a simple vscode extension available for download here: [download vix](https://github.com/pherrymason/c3-lsp/releases/download/latest/c3-lsp-client-0.0.1.vsix)  
Be sure to configure it with the path of the lsp server binary.


**Current status**  
***Index status***
- [~] Attributes (Module privacy)
- [x] Variables & type
- [x] [Global constants]()
- [x] Functions
- [x] Function parameters
- [x] Function return type
    - [x] Enums + Enumerators
    - [x] base type 
    - [x] enum methods
- [x] [Faults](https://c3-lang.org/references/docs/types/#faults)
- [x] Structs
    - [x] Struct members
    - [x] Struct methods
    - [x] Struct implementing interface
    - [x] Anonymous bitstructs
    - [~] Struct subtyping: Only for those subtypes defined in same file.
- [x] bitstruct
- [x] Unions
    - [x] Union members
- [~] defines
- [x] Interfaces
- [~] [Macros](https://c3-lang.org/references/docs/macros/)
    - [ ] return type
- [x] imports
- [x] modules
- [x] multiple modules per file
- [x] implicit module name is assumed to be the file name, converted to lower case, with any invalid characters replaced by underscore (_).
- [x] [Generics](https://c3-lang.org/references/docs/generics/)
- [ ] [Language Builtins](https://c3-lang.org/references/docs/builtins/)

***LSP Features Status***
- [x] Index scopes and its hierarchy to improve hover and Auto Completion.
- [ ] Complete Go to declaration
  - [x] Find symbol in same scope
  - [x] Find symbol in parent scope
  - [x] Find symbol present in same module, but different file
  - [x] Find symbol in imported module
  - [x] Find symbol in implicit parent module.
  - [x] Find symbol in stdlib
- [~] TextDocumentCompletion:
    - Struct methods are not suggested until first letter is written.

## Useful links:
- LSP specification: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- treesitter
  - Queries docs: https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries
  - Queries: https://emacs-tree-sitter.github.io/syntax-highlighting/queries/
  - Tree-sitter - a new parsing system for programming tools (video) https://www.thestrangeloop.com/2018/tree-sitter---a-new-parsing-system-for-programming-tools.html
