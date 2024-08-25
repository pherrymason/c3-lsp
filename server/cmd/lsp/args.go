package main

import (
	"flag"
	"fmt"
	"runtime/debug"

	"github.com/pherrymason/c3-lsp/internal/lsp/server"
	"github.com/pherrymason/c3-lsp/pkg/option"
)

func cmdLineArguments() (server.ServerOpts, bool) {
	var showHelp = flag.Bool("help", false, "Shows this help")
	var sendCrashReports = flag.Bool("send-crash-reports", false, "Automatically reports crashes to server.")

	var logFilePath = flag.String("log-path", "", "Enables logs and sets its filepath")
	var debug = flag.Bool("debug", false, "Enables debug mode")

	var c3Version = flag.String("lang-version", "", "Specify C3 language version.")
	var c3cPath = flag.String("c3c-path", "", "Path where c3c is located.")
	var diagnosticsDelay = flag.Int("diagnostics-delay", 2000, "Delay calculation of code diagnostics after modifications in source. In milliseconds, default 2000 ms.")

	flag.Parse()

	c3VersionOpt := option.None[string]()
	if *c3Version != "" {
		c3VersionOpt = option.Some(*c3Version)
	}
	c3cPathOpt := option.None[string]()
	if *c3cPath != "" {
		c3cPathOpt = option.Some(*c3cPath)
	}
	logFilePathOpt := option.None[string]()
	if *logFilePath != "" {
		logFilePathOpt = option.Some(*logFilePath)
	}

	return server.ServerOpts{
		C3Version:        c3VersionOpt,
		C3CPath:          c3cPathOpt,
		DiagnosticsDelay: uint(*diagnosticsDelay),
		LogFilepath:      logFilePathOpt,
		Debug:            *debug,
		SendCrashReports: *sendCrashReports,
	}, *showHelp
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