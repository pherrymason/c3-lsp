.PHONY: *

ASSETS_DIR = assets

# NOTE: go-tree-sitter only supports 14
TREE_SITTER_GENERATE_ABI = 14
TREE_SITTER_DIR = $(ASSETS_DIR)/tree-sitter-c3
TREE_SITTER_GIT = git@github.com:c3lang/tree-sitter-c3.git
TREE_SITTER_COMMIT = 2c04e78

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
	cd $(TREE_SITTER_DIR) && tree-sitter build --wasm && tree-sitter playground

build-parser:
	cd $(TREE_SITTER_DIR) && git fetch --all && git checkout $(TREE_SITTER_COMMIT) && tree-sitter generate --abi=$(TREE_SITTER_GENERATE_ABI)
	rm -rf ./server/internal/lsp/cst/tree_sitter
	rm -f ./server/internal/lsp/cst/parser.c
	cp -r $(TREE_SITTER_DIR)/src/tree_sitter ./server/internal/lsp/cst
	cp $(TREE_SITTER_DIR)/src/parser.c ./server/internal/lsp/cst/parser.c
	cp $(TREE_SITTER_DIR)/src/scanner.c ./server/internal/lsp/cst/scanner.c

index-c3-std:
	export C3C_DIR=$(C3C_DIR) && bash ./bin/build_index.sh

build:
	bash ./bin/build.sh

build-dev:
	go build -C ./server/cmd/lsp -gcflags="all=-N -l" -o ../../bin/c3lsp

build-all: build-darwin build-linux

# Build darwin-amd64
build-darwin:
	@echo "Building darwin-amd64"
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -C ./server/cmd/lsp -o ../../bin/c3lsp
	chmod +x ./server/bin/c3lsp
	cd ./server/bin && zip ./darwin-amd64-c3lsp.zip c3lsp
	echo "darwin-amd64 built"

# Build linux
build-linux:
	bash ./bin/build_linux.sh

# Unzips github artifact + zips linux, windows and mac binaries
BIN_PATH = ./server/bin
DARWIN_PREFIX = darwin-arm64
LINUX_PREFIX = linux-amd64
WIN_PREFIX = windows-amd64
pack-release:
	unzip $(BIN_PATH)/c3-lsp.zip -d $(BIN_PATH)/release
	mv $(BIN_PATH)/release/c3-lsp-$(DARWIN_PREFIX) $(BIN_PATH)/release/c3lsp; chmod 0555 $(BIN_PATH)/release/c3lsp
	rm  -f $(BIN_PATH)/c3lsp-$(DARWIN_PREFIX).zip ; zip $(BIN_PATH)/c3lsp-$(DARWIN_PREFIX).zip $(BIN_PATH)/release/c3lsp	
	
	mv $(BIN_PATH)/release/c3-lsp-$(LINUX_PREFIX) $(BIN_PATH)/release/c3lsp; chmod 0555 $(BIN_PATH)/release/c3lsp
	rm -f $(BIN_PATH)/c3lsp-$(LINUX_PREFIX).tar.gz; tar -czvf $(BIN_PATH)/c3lsp-$(LINUX_PREFIX).tar.gz $(BIN_PATH)/release/c3lsp
	
	mv $(BIN_PATH)/release/c3-lsp-$(WIN_PREFIX).exe $(BIN_PATH)/release/c3lsp.exe
	rm  -f $(BIN_PATH)/release/c3-lsp-$(WIN_PREFIX).zip; zip $(BIN_PATH)/c3lsp-$(WIN_PREFIX).zip $(BIN_PATH)/release/c3lsp.exe
	

#attach-process:
#	dlv attach --headless --listen=:2345 $(pgrep c3lsp) ./server/c3lsp --log

test:
	cd server && go test ./...


## VS Code extension
## -----------------
build-vscode:
	cd client/vscode && npm run vscode:prepublish
	cd client/vscode && vsce package

build-vscode-dev:
	cd client/vscode && npm run vscode:prepublish-dev
