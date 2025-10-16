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

func (t *Type) IsOptional() bool {
	return t.optional
}

func (t *Type) IsCollection() bool {
	return t.isCollection
}

func (t *Type) GetCollectionSize() option.Option[int] {
	return t.collectionSize
}

func (t *Type) GetPointerCount() int {
	return t.pointer
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

func (t Type) GetGenericArguments() []Type {
	return t.genericArguments
}

func (t Type) GetFullQualifiedName() string {
	if t.baseTypeLanguage {
		return t.name
	}

	return t.module + "::" + t.name
}

func (t *Type) GetModule() string {
	return t.module
}

func (t *Type) SetModule(module string) {
	t.module = module
}

func (t Type) UnsizedCollectionOf() Type {
	t.isCollection = true
	t.collectionSize = option.None[int]()
	return t
}

func (t *Type) IsPointer() bool {
	return t.pointer > 0
}

func (t Type) String() string {
	pointerStr := strings.Repeat("*", t.pointer)
	optionalStr := ""
	if t.optional {
		optionalStr = "?"
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
	baseType := strings.TrimSuffix(_type, "*")

	// Only consider '*'s at the end
	pointerCount := strings.Count(strings.TrimPrefix(_type, baseType), "*")

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

type TypeBuilder struct {
	type_ Type
}

func NewTypeBuilder(repr string, module string) *TypeBuilder {
	return &TypeBuilder{
		type_: NewTypeFromString(repr, module),
	}
}

func NewBaseTypeBuilder(repr string, module string) *TypeBuilder {
	return NewTypeBuilder(repr, module).IsBaseTypeLanguage()
}

func NewGenericTypeBuilder(repr string, module string) *TypeBuilder {
	return NewTypeBuilder(repr, module).IsGenericArgument()
}

func (b *TypeBuilder) IsBaseTypeLanguage() *TypeBuilder {
	b.type_.baseTypeLanguage = true
	return b
}

func (tb *TypeBuilder) IsOptional() *TypeBuilder {
	tb.type_.optional = true
	return tb
}

func (b *TypeBuilder) IsUnsizedCollection() *TypeBuilder {
	b.type_.isCollection = true
	b.type_.collectionSize = option.None[int]()
	return b
}

func (b *TypeBuilder) IsCollectionWithSize(size int) *TypeBuilder {
	b.type_.isCollection = true
	b.type_.collectionSize = option.Some(size)
	return b
}

func (b *TypeBuilder) IsGenericArgument() *TypeBuilder {
	b.type_.isGenericArgument = true
	return b
}

func (b *TypeBuilder) WithGenericArguments(types ...Type) *TypeBuilder {
	b.type_.genericArguments = append(b.type_.genericArguments, types...)
	return b
}

func (b *TypeBuilder) Build() Type {
	return b.type_
}
