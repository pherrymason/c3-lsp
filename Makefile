ASSETS_DIR = assets
TREE_SITTER_DIR = $(ASSETS_DIR)/tree-sitter-c3
TREE_SITTER_GIT = git@github.com:c3lang/tree-sitter-c3.git
TREE_SITTER_COMMIT = ef09c89e498b70e4dfbf81d00e8f4086fa8d1c0a
C3C_DIR = $(ASSETS_DIR)/c3c
C3C_GIT = git@github.com:c3lang/c3c.git


init:
	$(MAKE) clone-tree-sitter
	$(MAKE) clone-c3c
	$(MAKE) build-parser

clone-tree-sitter:
	[ ! -d $(TREE_SITTER_DIR) ] && git clone $(TREE_SITTER_GIT) $(TREE_SITTER_DIR) || true

clone-c3c:
	[ ! -d $(C3C_DIR) ] && git clone $(C3C_GIT) $(C3C_DIR) || true

treesitter-playground:
	cd $(TREE_SITTER_DIR) && tree-sitter build-wasm && tree-sitter playground

build-parser:
	cd $(TREE_SITTER_DIR) && git fetch --all && git checkout $(TREE_SITTER_COMMIT) && tree-sitter generate
	rm -rf server/internal/lsp/cst/tree_sitter
	rm -f server/internal/lsp/cst/parser.c
	cp -r $(TREE_SITTER_DIR)/src/tree_sitter server/internal/lsp/cst
	cp $(TREE_SITTER_DIR)/src/parser.c server/internal/lsp/cst/parser.c
	cp $(TREE_SITTER_DIR)/src/scanner.c server/internal/lsp/cst/scanner.c

index-c3-std:
ifndef VERSION
	$(error VERSION is not set. Usage: make index-c3-std VERSION=x.y.z)
endif
	cd $(C3C_DIR) && git fetch --all && git reset --hard origin/master && git checkout tags/v$(VERSION)
	cd server/cmd/stdlib_indexer && go run main.go blurp.go --$(VERSION)

# cp server/stdlib_indexer/stdlib/*.go server/lsp/language/stdlib

build:
	go build -C server/cmd/lsp -o ../../bin/c3lsp

build-dev:
	go build -C server/cmd/lsp -gcflags="all=-N -l" -o ../../bin/c3lsp

build-all: build-darwin build-linux

# Build darwin-amd64
build-darwin:
	@echo "Building darwin-amd64"
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -C server/cmd/lsp -o ../../bin/c3lsp
	cd server/bin && zip ./darwin-amd64-c3lsp.zip c3lsp
	echo "darwin-amd64 built"

# Build linux
build-linux:
	@echo "Building linux-amd64"
ifeq ($(shell uname -s), Darwin)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-linux-musl-gcc" go build -C server/cmd/lsp -o ../../bin/c3lsp
else 
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -C server/cmd/lsp -o ../../bin/c3lsp
endif
	cd server/bin && zip ./linux-amd64-c3lsp.zip c3lsp
	@echo "linux-amd64 built"


#attach-process:
#	dlv attach --headless --listen=:2345 $(pgrep c3lsp) ./server/c3lsp --log

test:
	cd server && go test ./...


## VS Code extension
build-vscode:
	cd client/vscode && npm run vscode:prepublish
	cd client/vscode && vsce package

build-vscode-dev:
	cd client/vscode && npm run vscode:prepublish-dev
