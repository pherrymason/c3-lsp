# C3 LSP
[![Go](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml/badge.svg)](https://github.com/pherrymason/c3-lsp/actions/workflows/go.yml)

WIP LSP for [C3 language](https://github.com/c3lang/c3c)  
Using tree-sitter grammar rules from https://github.com/cbuttner/tree-sitter-c3.git

## Server
**Features**
- [x] Indexes workspace variables and function definitions
- [x] **TextDocumentCompletion.** Suggests symbols located in project's source code. See notes.
- [x] **Go to declaration.**
- [x] **Go to definition.** 
- [x] **Hover** Displays information about symbol under cursor.

**Next release**
- Index Generics
- TextDocumentCompletion: Be able to suggest `Interfaces`, `module names`

**Current status**  
***Index status***
- [ ] Attributes
- [x] Variables & type
- [x] [Global constants]()
- [x] Functions
- [x] Function parameters
- [x] Function return type
    - [x] Enums + Enumerators
    - [x] base type 
    - [ ] enum methods
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
  - [ ] Find symbol in stdlib
- [~] TextDocumentCompletion:
    - Struct methods are not suggested until first letter is written.
    - Completion of symbols defined in stdlib.

## Useful links:
- LSP specification: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- treesitter
  - Queries docs: https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries
  - Queries: https://emacs-tree-sitter.github.io/syntax-highlighting/queries/
  - Tree-sitter - a new parsing system for programming tools (video) https://www.thestrangeloop.com/2018/tree-sitter---a-new-parsing-system-for-programming-tools.html
