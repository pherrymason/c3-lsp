package indexables

import (
	"fmt"
	"strings"
)

type Type struct {
	name    string
	pointer int
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
