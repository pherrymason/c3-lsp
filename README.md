# Language Server for the C3 language.
[![Go](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml/badge.svg)](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml)

WIP LSP for [C3 language](https://github.com/c3lang/c3c)  


## Table of Contents

- [Features](#Features)
- [Usage](#Usage)
- [Installation](#Installation)
- [Project Goals](#project-goals)

## Features
Supported Language server features:

- Completion
- Go to definition
- Go to declaration
- Hover
- Signature Help

Furthermore, the LSP is able to resolve stdlib symbols information (for supported C3c versions), allowing to use this in completion and hover functionalities.

## Usage 
### Usage with text editors / IDE's
Usually you'll use a plugin for your editor that will automatically run the server with the required arguments.  
Some plugins/extensions might already include the binaries, check their documentation.  
If they don't include it, they might allow you to specify the path to the lsp binary, so download the lsp binary and configure the plugin/extension to its location.

Read more in the [wiki](https://github.com/pherrymason/c3-lsp/wiki/Integration-with-editors).

### Standalone usage
Simply run `c3lsp` to start the server.  
It supports the following options:
- **help:** Display accepted options.
- **send-reports:** If enabled (disabled by default) will send __crash__ reports to Sentry so bugs can be debugged easily.
- **lang-version:** Use it to specify a specific c3 language version. By default `c3lsp` will select the last version supported.


## Installation
Project is written in Golang, so in theory it could be built to any OS supported by Golang.  
Precompiled binaries for the following operating systems are available: Linux, Windows 11 and MacOs.

Find latest built binaries in the [releases page](https://github.com/pherrymason/c3-lsp/releases)

You can also build from source:

- Download and install golang: https://go.dev/
- Clone this repo
- Run `make build`: This will create `c3lsp` in `server/bin` folder.


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


## Useful links:
- LSP specification: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- treesitter
  - Queries docs: https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries
  - Queries: https://emacs-tree-sitter.github.io/syntax-highlighting/queries/
  - Tree-sitter - a new parsing system for programming tools (video) https://www.thestrangeloop.com/2018/tree-sitter---a-new-parsing-system-for-programming-tools.html
