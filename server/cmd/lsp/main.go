package main

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/pkg/option"
	flag "github.com/spf13/pflag"
)

const version = "0.0.7"
const prerelease = true
const appName = "C3-LSP"

func getVersion() string {
	if prerelease {
		return version + "-pre"
	}

	return version
}

func main() {
	options := cmdLineArguments()
	commitHash := buildInfo()
	if options.showHelp {
		printHelp(appName, getVersion(), commitHash)

		return
	}

	if options.sendCrashReports {
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

	c3Version := option.None[string]()
	if options.c3Version != "" {
		c3Version = option.Some(options.c3Version)
	}

	logFilePath := option.None[string]()
	if options.logFilePath != "" {
		logFilePath = option.Some(options.logFilePath)
	}

	server := lsp.NewServer(lsp.ServerOpts{
		Name:             appName,
		Version:          version,
		C3Version:        c3Version,
		LogFilepath:      logFilePath,
		SendCrashReports: options.sendCrashReports,
		Debug:            options.debug,
	})
	server.Run()
}

type Options struct {
	showHelp         bool
	c3Version        string
	logFilePath      string
	debug            bool
	sendCrashReports bool
}

func cmdLineArguments() Options {
	var showHelp = flag.Bool("help", false, "Shows this help")

	var sendCrashReports = flag.Bool("send-crash-reports", false, "Automatically reports crashes to server.")

	var logFilePath = flag.String("log-path", "", "Enables logs and sets its filepath")
	var debug = flag.Bool("debug", false, "Enables debug mode")

	var c3Version = flag.String("lang-version", "", "Specify C3 language version.")

	flag.Parse()

	return Options{
		showHelp:         *showHelp,
		c3Version:        *c3Version,
		logFilePath:      *logFilePath,
		debug:            *debug,
		sendCrashReports: *sendCrashReports,
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
