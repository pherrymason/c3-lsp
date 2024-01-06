# C3 LSP
![build result](https://github.com/pherrymason/c3-lsp/actions/workflows/main.yml/badge.svg)  

WIP LSP for [C3 language](https://github.com/c3lang/c3c)  
Using tree-sitter grammar rules from https://github.com/zweimach/tree-sitter-c3

## Server
Features:
- Naïve auto completion items from same document.
- Go to declaration (variables and functions, only on same file).



## YACC to TreeSitter 
Experiment based on https://github.com/miks1965/yacc-to-tree-sitter to convert C3 yacc grammar file to treesitter grammar.js

It is not complete, as it ignores lexical rules.










## Useful links:
- LSP specification: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- treesitter
  - Queries docs: https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries
  - Queries: https://emacs-tree-sitter.github.io/syntax-highlighting/queries/
  - Tree-sitter - a new parsing system for programming tools (video) https://www.thestrangeloop.com/2018/tree-sitter---a-new-parsing-system-for-programming-tools.html
