package symbols

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/option"
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
	isCollection      bool
	collectionSize    option.Option[int]
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

	collectionStr := ""
	if t.isCollection {
		collectionStr += "["
		if t.collectionSize.IsSome() {
			collectionStr += fmt.Sprintf("%d", t.collectionSize.Get())
		}
		collectionStr += "]"
	}

	return fmt.Sprintf("%s%s%s%s", t.name, pointerStr, collectionStr, optionalStr)
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

func NewType(baseTypeLanguage bool, baseType string, pointerCount int, isGenericArgument bool, isCollection bool, collectionSize option.Option[int], modulePath string) Type {
	return Type{
		baseTypeLanguage:  baseTypeLanguage,
		name:              baseType,
		pointer:           pointerCount,
		optional:          false,
		isGenericArgument: isGenericArgument,
		module:            modulePath,
		isCollection:      isCollection,
		collectionSize:    collectionSize,
	}
}

func NewOptionalType(baseTypeLanguage bool, baseType string, pointerCount int, isGenericArgument bool, isCollection bool, collectionSize option.Option[int], modulePath string) Type {
	return Type{
		baseTypeLanguage:  baseTypeLanguage,
		name:              baseType,
		pointer:           pointerCount,
		optional:          true,
		isGenericArgument: isGenericArgument,
		module:            modulePath,

		isCollection:   isCollection,
		collectionSize: collectionSize,
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
