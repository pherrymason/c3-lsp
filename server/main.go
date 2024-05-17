package main

import (
	"github.com/pherrymason/c3-lsp/lsp"
)

const version = "0.0.1"

func main() {
	server := lsp.NewServer(lsp.ServerOpts{
		Name:        "C3 LSP",
		Version:     version,
		LogFilepath: "./lsp.log",
	})
	server.Run()
}
