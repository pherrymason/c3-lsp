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
	genericArguments []Type // This type holds Generic arguments
	// TODO This should be a map to properly know which module generic parameter is refering to.
	module            string
	isGenericArgument bool // When true, this is a module generic argument.
}

func (t Type) GetName() string {
	return t.name
}

func (t Type) IsBaseTypeLanguage() bool {
	return t.baseTypeLanguage
}

func (t Type) IsGenericArgument() bool {
	return t.isGenericArgument
}

func (t Type) HasGenericArguments() bool {
	return len(t.genericArguments) > 0
}

func (t Type) GetGenericArgument(index uint) Type {
	return t.genericArguments[index]
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

func NewType(baseTypeLanguage bool, baseType string, pointerCount int, isGenericArgument bool, modulePath string) Type {
	return Type{
		baseTypeLanguage:  baseTypeLanguage,
		name:              baseType,
		pointer:           pointerCount,
		optional:          false,
		isGenericArgument: isGenericArgument,
		module:            modulePath,
	}
}

func NewOptionalType(baseTypeLanguage bool, baseType string, pointerCount int, isGenericArgument bool, modulePath string) Type {
	return Type{
		baseTypeLanguage:  baseTypeLanguage,
		name:              baseType,
		pointer:           pointerCount,
		optional:          true,
		isGenericArgument: isGenericArgument,
		module:            modulePath,
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
