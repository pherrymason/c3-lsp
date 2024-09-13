package c3c

import "github.com/pherrymason/c3-lsp/pkg/option"

type C3Opts struct {
	Version     option.Option[string] `json:"version"`
	Path        option.Option[string] `json:"path"`
	StdlibPath  option.Option[string] `json:"stdlib-path"`
	CompileArgs []string              `json:"compile-args"`
}
