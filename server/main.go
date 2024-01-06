package main

import (
	"github.com/pherrymason/c3-lsp/lsp"
	_ "github.com/tliron/commonlog/simple"
)

func main() {
	server := lsp.NewServer(lsp.ServerOpts{
		Name:    "C3 LSP",
		Version: "0.0.1",
		LogFile: "/Volumes/Development/raul/c3/go-lsp/lsp.log",
	})
	server.Run()
}
