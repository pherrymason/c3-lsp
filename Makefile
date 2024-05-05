init:
	$(MAKE) clone-tree-sitter
	$(MAKE) build-parser

clone-tree-sitter:
	[ ! -d "tree-sitter-c3" ] && git clone git@github.com:cbuttner/tree-sitter-c3.git tree-sitter-c3 || true

build-parser:
	cd tree-sitter-c3 && tree-sitter generate
	rm -rf server/lsp/cst/tree_sitter
	rm -f server/lsp/cst/parser.c
	cp -r tree-sitter-c3/src/tree_sitter server/lsp/cst
	cp tree-sitter-c3/src/parser.c server/lsp/cst/parser.c
	cp tree-sitter-c3/src/scanner.c server/lsp/cst/scanner.c

build:
	cd server && go build

build-dev:
	cd server && go build -gcflags="all=-N -l" -o c3-lsp

#attach-process:
#	dlv attach --headless --listen=:2345 $(pgrep c3-lsp) ./server/c3-lsp --log

test:
	cd server && go test ./...