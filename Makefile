init:
	$(MAKE) clone-tree-sitter
	cp tree-sitter-c3/src/parser.c server/c3/parser.c
	cp -r tree-sitter-c3/src/tree_sitter server/c3/tree_sitter
	#cd yacc-2-treesitter && go build

	#git clone https://github.com/zweimach/tree-sitter-c3
	#cargo install tree-sitter-cli

clone-tree-sitter:
	[ ! -d "tree-sitter-c3" ] && git clone git@github.com:zweimach/tree-sitter-c3.git tree-sitter-c3  || true


#build-parser:
#	cp yacc-2-treesitter/grammar.js ./grammar.js
#	tree-sitter generate

build-dev:
	cd server && go build -gcflags="all=-N -l" -o c3-lsp
