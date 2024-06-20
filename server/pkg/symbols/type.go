package symbols

import (
	"fmt"
	"strings"
)

type Type struct {
	baseTypeLanguage bool // Is a base type of the language
	name             string
	pointer          int
	optional         bool
	genericArguments []Type
	module           string
}

func (t Type) GetName() string {
	return t.name
}

func (t Type) IsBaseTypeLanguage() bool {
	return t.baseTypeLanguage
}

func (t Type) GetFullQualifiedName() string {
	if t.baseTypeLanguage {
		return t.name
	}

	return t.module + "::" + t.name
}

func (t *Type) SetModule(module string) {
	t.module = module
}

func (t Type) String() string {
	pointerStr := strings.Repeat("*", t.pointer)
	optionalStr := ""
	if t.optional {
		optionalStr = "!"
	}

	return fmt.Sprintf("%s%s%s", t.name, pointerStr, optionalStr)
}

func NewTypeFromString(_type string, modulePath string) Type {
	pointerCount := strings.Count(_type, "*")
	baseType := strings.TrimSuffix(_type, "*")

	return Type{
		name:    baseType,
		pointer: pointerCount,
		module:  modulePath,
	}
}

func NewType(baseTypeLanguage bool, baseType string, pointerCount int, modulePath string) Type {
	return Type{
		baseTypeLanguage: baseTypeLanguage,
		name:             baseType,
		pointer:          pointerCount,
		optional:         false,
		module:           modulePath,
	}
}

func NewOptionalType(baseTypeLanguage bool, baseType string, pointerCount int, modulePath string) Type {
	return Type{
		baseTypeLanguage: baseTypeLanguage,
		name:             baseType,
		pointer:          pointerCount,
		optional:         true,
		module:           modulePath,
	}
}

func NewTypeWithGeneric(baseTypeLanguage bool, isOptional bool, baseType string, pointerCount int, genericArguments []Type, modulePath string) Type {
	return Type{
		baseTypeLanguage: baseTypeLanguage,
		name:             baseType,
		pointer:          pointerCount,
		optional:         isOptional,
		genericArguments: genericArguments,
		module:           modulePath,
	}
}
