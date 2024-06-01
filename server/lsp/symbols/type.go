package symbols

import (
	"fmt"
	"strings"
)

type Type struct {
	baseTypeLanguage bool // Is a base type of the language
	name             string
	pointer          int
	module           string
}

func (t Type) GetName() string {
	return t.name
}

func (t Type) String() string {
	pointerStr := strings.Repeat("*", t.pointer)

	return fmt.Sprintf("%s%s", t.name, pointerStr)
}

func NewTypeFromString(_type string) Type {
	pointerCount := strings.Count(_type, "*")
	baseType := strings.TrimSuffix(_type, "*")

	return Type{
		name:    baseType,
		pointer: pointerCount,
	}
}

func NewType(baseTypeLanguage bool, baseType string, pointerCount int, modulePath string) Type {
	return Type{
		baseTypeLanguage: baseTypeLanguage,
		name:             baseType,
		pointer:          pointerCount,
		module:           modulePath,
	}
}
