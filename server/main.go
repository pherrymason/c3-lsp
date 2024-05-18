package main

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pherrymason/c3-lsp/lsp"
	flag "github.com/spf13/pflag"
)

const version = "0.0.2"
const appName = "C3-LSP"

func main() {
	options := cmdLineArguments()
	commitHash := buildInfo()
	if options.showHelp {
		printHelp(appName, version, commitHash)

		return
	}

	if options.allowReporting {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:     "https://76f9fe6a1d3e2be7c9083891a644b0a3@o124652.ingest.us.sentry.io/4507278372110336",
			Release: fmt.Sprintf("c3.lsp@%s+%s", version, commitHash),
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}

		// Flush buffered events before the program terminates.
		defer sentry.Flush(2 * time.Second)

		sentry.CaptureMessage("It works!")
	}

	server := lsp.NewServer(lsp.ServerOpts{
		Name:        appName,
		Version:     version,
		LogFilepath: options.logFilePath,
	})
	server.Run()
}

type Options struct {
	showHelp       bool
	logFilePath    string
	allowReporting bool
}

func cmdLineArguments() Options {
	var showHelp = flag.Bool("help", false, "Shows this help")

	var allowReporting = flag.Bool("allowReporting", false, "Automatically reports crashes to server.")

	var logFilePath = flag.String("log-path", "./lsp.log", "Enables logs and sets its filepath")

	flag.Parse()

	return Options{
		showHelp:       *showHelp,
		logFilePath:    *logFilePath,
		allowReporting: *allowReporting,
	}
}

func printAppGreet(appName string, version string, commit string) {
	fmt.Printf("%s version %s (%s)\n", appName, version, commit)
}

func printHelp(appName string, version string, commit string) {
	printAppGreet(appName, version, commit)

	fmt.Println("\nOptions")
	flag.PrintDefaults()
}

func buildInfo() string {
	var Commit = func() string {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					return setting.Value
				}
			}
		}

		return ""
	}()

	return Commit
}
