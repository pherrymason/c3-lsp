package main

import (
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pherrymason/c3-lsp/internal/lsp/server"
)

const version = "0.3.0"
const prerelease = false
const appName = "C3-LSP"

func getVersion() string {
	if prerelease {
		return version + "-pre"
	}

	return version
}

func main() {
	options, showHelp := cmdLineArguments()
	commitHash := buildInfo()
	if showHelp {
		printHelp(appName, getVersion(), commitHash)

		return
	}

	if options.SendCrashReports {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              "https://76f9fe6a1d3e2be7c9083891a644b0a3@o124652.ingest.us.sentry.io/4507278372110336",
			Release:          fmt.Sprintf("c3.lsp@%s+%s", getVersion(), commitHash),
			Debug:            false,
			AttachStacktrace: true,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}

		// Flush buffered events before the program terminates.
		defer sentry.Flush(2 * time.Second)
		defer sentry.Recover()
	}

	server := server.NewServer(options, appName, version)
	server.Run()
}
