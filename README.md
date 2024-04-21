# C3 LSP
[![Go](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml/badge.svg)](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml)

WIP LSP for [C3 language](https://github.com/c3lang/c3c)  
Using tree-sitter grammar rules from https://github.com/zweimach/tree-sitter-c3

## Server
**Features**
- [x] Indexes workspace variables and function definitions
- [x] Naïve auto completion items of variables and function names defined in the workspace.
- [x] Go to declaration.
- [x] Hover

**TODO list**
- Parser:
  - [x] Variables & type
  - [ ] [Global constants]()
  - [x] Functions
    - [x] Function parameters
    - [x] Function return type
  - [x] Enums + Enumerators
    - [x] base type 
  - [ ] [Faults](https://c3-lang.org/references/docs/types/#faults)
  - [x] Structs
    - [x] Struct members
    - [x] Struct methods
  - [ ] Unions
    - [ ] Union members
  - [~] defines
  - [ ] Interfaces
  - [ ] macros
  - [ ] imports
  - [ ] [Generics](https://c3-lang.org/references/docs/generics/)
  - [ ] [Language Builtins](https://c3-lang.org/references/docs/builtins/)

- [x] Index scopes and its hierarchy to improve hover and Auto Completion.
- [ ] Complete Hover
- [ ] Complete Go to declaration
- [ ] Offer information about stdlib?
- [ ] Go to definition
- [ ] Go to type definition
- [ ] Go to implementation
- [ ] Find references
- [ ] Improve Completion feature by having context into account
- [ ] Rename


## Useful links:
- LSP specification: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- treesitter
  - Queries docs: https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries
  - Queries: https://emacs-tree-sitter.github.io/syntax-highlighting/queries/
  - Tree-sitter - a new parsing system for programming tools (video) https://www.thestrangeloop.com/2018/tree-sitter---a-new-parsing-system-for-programming-tools.html
