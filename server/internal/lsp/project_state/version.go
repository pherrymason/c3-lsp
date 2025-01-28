package project_state

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/internal/lsp/stdlib"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"golang.org/x/mod/semver"
)

type stdLibFunc func() symbols_table.UnitModules

type Version struct {
	Number        string
	stdLibSymbols stdLibFunc
}

func SupportedVersions() []Version {
	return []Version{
		{
			Number:        "dummy",
			stdLibSymbols: stdlib.Load_vdummy_stdlib,
		},
		{
			Number:        "0.6.4",
			stdLibSymbols: stdlib.Load_v064_stdlib,
		},
		{
			Number:        "0.6.5",
			stdLibSymbols: stdlib.Load_v065_stdlib,
		},
		{
			Number:        "0.6.6",
			stdLibSymbols: stdlib.Load_v066_stdlib,
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
