package c3c

import (
	"bytes"
	"log"
	"os/exec"
	"regexp"

	"github.com/pherrymason/c3-lsp/pkg/option"
)

func binaryPath(c3Path option.Option[string]) string {
	binary := "c3c"
	if c3Path.IsSome() {
		binary = c3Path.Get()
	}
	return binary
}

func GetC3Version(c3Path option.Option[string]) option.Option[string] {
	binary := binaryPath(c3Path)
	command := exec.Command(binary, "--version")
	var out bytes.Buffer
	var stdErr bytes.Buffer

	// set the output to our variable
	command.Stdout = &out
	command.Stderr = &stdErr

	err := command.Run()
	if err != nil {
		// Could not get version from c3c
		log.Printf("Could not get c3c version")
	} else {
		re := regexp.MustCompile(`C3 Compiler Version:\s+(\d+\.\d+\.\d+)`)
		match := re.FindStringSubmatch(out.String())
		if len(match) > 1 {
			log.Printf("C3 Version found: %s", match[1])
			return option.Some(match[1])
		} else {
			log.Printf("C3 Version not found")
		}
	}

	return option.None[string]()
}

func CheckC3ErrorsCommand(c3Path option.Option[string], projectPath string) (bytes.Buffer, bytes.Buffer, error) {
	binary := binaryPath(c3Path)
	command := exec.Command(binary, "build", "--test")
	command.Dir = projectPath

	// set var to get the output
	var out bytes.Buffer
	var stdErr bytes.Buffer

	// set the output to our variable
	command.Stdout = &out
	command.Stderr = &stdErr

	err := command.Run()

	return out, stdErr, err
}
