package server

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

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

	var options ServerOpts
	err = json.Unmarshal(data, &options)
	if err != nil {
		log.Fatalf("Error deserializing config json: %v", err)
	}

	s.options = options

	// Change log filepath?
	// Should be able to do that form c3lsp.json?

	// Enable/disable sendCrashReports
	// Should be able to do that form c3lsp.json?
}
