init:
	$(MAKE) clone-tree-sitter
	cp tree-sitter-c3/src/parser.c server/c3/parser.c
	$(MAKE) copy-parser

clone-tree-sitter:
	[ ! -d "tree-sitter-c3" ] && git clone git@github.com:zweimach/tree-sitter-c3.git tree-sitter-c3 || true

build-parser:
	cd tree-sitter-c3 && tree-sitter generate
	$(MAKE) copy-parser

clone-tree-sitter-alt:
	[ ! -d "tree-sitter-c3" ] && git clone git@github.com:cbuttner/tree-sitter-c3.git tree-sitter-c3-alt || true

build-parser-alt:
	cd tree-sitter-c3-alt && tree-sitter generate
	rm -rf server/lsp/cst/tree_sitter
	rm -f server/lsp/cst/parser.c
	cp -r tree-sitter-c3-alt/src/tree_sitter server/lsp/cst
	cp tree-sitter-c3-alt/src/parser.c server/lsp/cst/parser.c
	cp tree-sitter-c3-alt/src/scanner.c server/lsp/cst/scanner.c


copy-parser:
	cp -r tree-sitter-c3/src/tree_sitter server/lsp/cst
	cp tree-sitter-c3/src/parser.c server/lsp/cst/parser.c

build:
	cd server && go build

build-dev:
	cd server && go build -gcflags="all=-N -l" -o c3-lsp

test:
	cd server && go test ./...