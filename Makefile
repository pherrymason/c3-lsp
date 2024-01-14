init:
	$(MAKE) clone-tree-sitter
	cp tree-sitter-c3/src/parser.c server/c3/parser.c
	$(MAKE) copy-parser

clone-tree-sitter:
	[ ! -d "tree-sitter-c3" ] && git clone git@github.com:zweimach/tree-sitter-c3.git tree-sitter-c3  || true


build-parser:
	cd tree-sitter-c3 && tree-sitter generate
	$(MAKE) copy-parser
	#cp yacc-2-treesitter/grammar.js ./grammar.js
	#tree-sitter generate

copy-parser:
	cp -r tree-sitter-c3/src/tree_sitter server/lsp/tree_sitter
	cp tree-sitter-c3/src/parser.c server/lsp/parser.c

build-dev:
	cd server && go build -gcflags="all=-N -l" -o c3-lsp

test:
	cd server && go test ./...