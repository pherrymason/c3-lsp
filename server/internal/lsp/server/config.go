package server

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/pkg/option"
)

type DiagnosticsOpts struct {
	Enabled bool          `json:"enabled"`
	Delay   time.Duration `json:"delay"`
}

// ServerOpts holds the options to create a new Server.
type ServerOpts struct {
	C3          c3c.C3Opts      `json:"C3Opts"`
	Diagnostics DiagnosticsOpts `json:"Diagnostics"`

	LogFilepath      option.Option[string]
	SendCrashReports bool
	Debug            bool
}

type ServerOptsJson struct {
	C3 struct {
		Version     *string  `json:"version,omitempty"`
		Path        *string  `json:"path,omitempty"`
		StdlibPath  *string  `json:"stdlib-path,omitempty"`
		CompileArgs []string `json:"compile-args"`
	}

	Diagnostics struct {
		Enabled bool          `json:"enabled"`
		Delay   time.Duration `json:"delay"`
	}
}

func (srv *Server) loadServerConfigurationForWorkspace(path string) {
	file, err := os.Open(path + "/c3lsp.json")
	if err != nil {
		// No configuration project file found.
		log.Print("No configuration " + path + "/c3lsp.json found")
		return
	}
	defer file.Close()

	log.Print("Reading configuration " + path + "/c3lsp.json")

	// Lee el archivo
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Error reading config json: %v", err)
	}
	log.Printf("%srv", data)

	var options ServerOptsJson
	err = json.Unmarshal(data, &options)
	if err != nil {
		log.Fatalf("Error deserializing config json: %v", err)
	}

	if options.C3.StdlibPath != nil {
		srv.options.C3.StdlibPath = option.Some(*options.C3.StdlibPath)
		log.Printf("Stdlib:%srv", *options.C3.StdlibPath)
		log.Printf("Setted Stdlib:%srv", srv.options.C3.StdlibPath.Get())
	}

	if options.C3.Version != nil {
		srv.options.C3.Version = option.Some(*options.C3.Version)
	}

	if options.C3.Path != nil {
		srv.options.C3.Path = option.Some(*options.C3.Path)
		// Get version from binary
	}

	if len(options.C3.CompileArgs) > 0 {
		srv.options.C3.CompileArgs = options.C3.CompileArgs
	}

	c3Version := c3c.GetC3Version(srv.options.C3.Path)
	if c3Version.IsSome() {
		srv.options.C3.Version = c3Version
	}

	requestedLanguageVersion := checkRequestedLanguageVersion(srv.options.C3.Version)
	srv.state.SetLanguageVersion(requestedLanguageVersion)

	// Change log filepath?
	// Should be able to do that form c3lsp.json?

	// Enable/disable sendCrashReports
	// Should be able to do that form c3lsp.json?
}
