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

type C3Opts struct {
	Version    option.Option[string] `json:"version"`
	Path       option.Option[string] `json:"path"`
	StdlibPath option.Option[string] `json:"stdlib-path"`
}

type DiagnosticsOpts struct {
	Enabled bool          `json:"enabled"`
	Delay   time.Duration `json:"delay"`
}

// ServerOpts holds the options to create a new Server.
type ServerOpts struct {
	C3          C3Opts          `json:"C3Opts"`
	Diagnostics DiagnosticsOpts `json:"Diagnostics"`

	LogFilepath      option.Option[string]
	SendCrashReports bool
	Debug            bool
}

type ServerOptsJson struct {
	C3 struct {
		Version    *string `json:"version,omitempty"`
		Path       *string `json:"path,omitempty"`
		StdlibPath *string `json:"stdlib-path,omitempty"`
	}

	Diagnostics struct {
		Enabled bool          `json:"enabled"`
		Delay   time.Duration `json:"delay"`
	}
}

func (s *Server) loadServerConfigurationForWorkspace(path string) {
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
	log.Printf("%s", data)

	var options ServerOptsJson
	err = json.Unmarshal(data, &options)
	if err != nil {
		log.Fatalf("Error deserializing config json: %v", err)
	}

	if options.C3.StdlibPath != nil {
		s.options.C3.StdlibPath = option.Some(*options.C3.StdlibPath)
		log.Printf("Stdlib:%s", *options.C3.StdlibPath)
		log.Printf("Setted Stdlib:%s", s.options.C3.StdlibPath.Get())
	}

	if options.C3.Version != nil {
		s.options.C3.Version = option.Some(*options.C3.Version)
	}
	if options.C3.Path != nil {
		s.options.C3.Path = option.Some(*options.C3.Path)
		// Get version from binary
	}

	c3Version := c3c.GetC3Version(s.options.C3.Path)
	if c3Version.IsSome() {
		s.options.C3.Version = c3Version
	}

	requestedLanguageVersion := checkRequestedLanguageVersion(s.options.C3.Version)
	s.state.SetLanguageVersion(requestedLanguageVersion)

	// Change log filepath?
	// Should be able to do that form c3lsp.json?

	// Enable/disable sendCrashReports
	// Should be able to do that form c3lsp.json?
}
