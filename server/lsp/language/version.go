package language

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/lsp/language/stdlib"
	"github.com/pherrymason/c3-lsp/lsp/unit_modules"
	"github.com/pherrymason/c3-lsp/option"
	"golang.org/x/mod/semver"
)

type stdLibFunc func() unit_modules.UnitModules

type Version struct {
	Number        string
	stdLibSymbols stdLibFunc
}

func SupportedVersions() []Version {
	return []Version{
		Version{
			Number:        "dummy",
			stdLibSymbols: stdlib.Load_vdummy_stdlib,
		},
		Version{
			Number:        "0.5.5",
			stdLibSymbols: stdlib.Load_v055_stdlib,
		},
	}
}

func GetVersion(number option.Option[string]) Version {
	versions := SupportedVersions()
	if number.IsNone() {
		return versions[len(versions)-1]
	}

	requestedVersion := number.Get()
	for _, version := range versions {
		if semver.Compare("v"+requestedVersion, "v"+version.Number) == 0 {
			return version
		}
	}

	panic(fmt.Sprintf("Requested C3 language version \"%s\" not supported", requestedVersion))
}
