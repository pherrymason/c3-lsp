# C3 LSP
[![Go](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml/badge.svg)](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml)

WIP LSP for [C3 language](https://github.com/c3lang/c3c)  
Using tree-sitter grammar rules from https://github.com/zweimach/tree-sitter-c3

## Server
**Features**
- [x] Indexes workspace variables and function definitions
- [x] Naïve auto completion items of variables and function names defined in the workspace.
- [x] Go to declaration.
  - Variable
  - Enum
  - Struct
  - Function
- [x] Hover:
  - Variable usages.
  - Function calls.

**TODO list**
- [x] Index scopes and its hierarchy to improve hover and Auto Completion.
- [ ] Index Symbols
  - [x] Variables & type
  - [x] Functions
    - [x] Function arguments
    - [x] Function return type
  - [x] Enums + Enumerators
    - [ ] base type 
  - [x] Structs
    - [x] Struct members
    - [ ] Struct methods
  - [ ] imports
  - [~] defines
  - [ ] macros: **Needs to update grammar.js**
- [ ] Hover information
  - [x] Variable declarations
  - [x] Function calls
  - [ ] enumerators
  - [ ] struct properties
  - [ ] struct methods
- [ ] Offer information about stdlib?
- [ ] Go to definition
- [ ] Go to type definition
- [ ] Go to implementation
- [ ] Find references
- [ ] Improve Completion feature by having context into account
- [ ] Rename


## YACC to TreeSitter 
**Experiment** based on https://github.com/miks1965/yacc-to-tree-sitter to convert C3 yacc grammar file to treesitter grammar.js

It is not complete, as it ignores lexical rules.




## Useful links:
- LSP specification: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- treesitter
  - Queries docs: https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries
  - Queries: https://emacs-tree-sitter.github.io/syntax-highlighting/queries/
  - Tree-sitter - a new parsing system for programming tools (video) https://www.thestrangeloop.com/2018/tree-sitter---a-new-parsing-system-for-programming-tools.html
