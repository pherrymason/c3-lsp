package parser

import (
	"fmt"
	"testing"

	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
)

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func assertSameFunction(t *testing.T, expected idx.Function, actual idx.Function, functionName string) {
	assert.Equal(t, expected.FunctionType(), actual.FunctionType(), expected.GetName(), "Function type does not match")
	assert.Equal(t, expected.GetName(), actual.GetName(), "Function name does not match")
	assert.Equal(t, expected.GetReturnType(), actual.GetReturnType(), expected.GetName(), "Function return type does not match")
	assert.Equal(t, expected.ArgumentIds(), actual.ArgumentIds(), expected.GetName(), "Function arguments does not match")
	assert.Equal(t, expected.GetDocumentURI(), actual.GetDocumentURI(), expected.GetName(), "Function doc id does not match")

	assertSameRange(t, expected.GetDeclarationRange(), actual.GetDeclarationRange(), fmt.Sprint("Function declaration range:", expected.GetName()))
	assertSameRange(t, expected.GetDocumentRange(), actual.GetDocumentRange(), fmt.Sprint("Function document range:", expected.GetName()))

	assert.Equal(t, expected.GetKind(), actual.GetKind(), expected.GetName())
	//assert.Equal(t, expected.Variables, actual.Variables, expected.GetName())

	for key, value := range expected.Variables {
		assertSameVariable(t, value, actual.Variables[key], "var")
	}

	assert.Equal(t, expected.Enums, actual.Enums, expected.GetName())
	assert.Equal(t, expected.Structs, actual.Structs, expected.GetName())

	assert.Equal(t, len(expected.ChildrenFunctions), len(actual.ChildrenFunctions))
	for key, value := range expected.ChildrenFunctions {
		assertSameFunction(t, value, actual.ChildrenFunctions[key], value.GetName())
	}
}

func assertSameRange(t *testing.T, expected idx.Range, actual idx.Range, msg string) {
	assert.Equal(t, expected.Start, actual.Start, fmt.Sprint(msg, " start"))
	assert.Equal(t, expected.Start, actual.Start, fmt.Sprint(msg, " end"))
}

func assertSameVariable(t *testing.T, expected idx.Variable, actual idx.Variable, msg string) {
	assert.Equal(t, expected.GetName(), actual.GetName(), "Variable name does not match")
	assert.Equal(t, expected.GetType(), actual.GetType(), "Variable type does not match")
	//	assert.Equal(t, expected.GetDocumentRange(), actual.GetDocumentRange(), "Variable document range does not match")
	assert.Equal(t, expected.GetDeclarationRange(), actual.GetDeclarationRange(), "Variable declaration range does not match")
}
