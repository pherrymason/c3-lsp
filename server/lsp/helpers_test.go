package lsp

import (
	"fmt"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	"testing"
)

func assertSameRange(t *testing.T, expected idx.Range, actual idx.Range, msg string) {
	assert.Equal(t, expected.Start, actual.Start, fmt.Sprint(msg, " start"))
	assert.Equal(t, expected.Start, actual.Start, fmt.Sprint(msg, " end"))
}

func assertSameFunction(t *testing.T, expected *idx.Function, actual *idx.Function) {
	assert.Equal(t, expected.FunctionType(), actual.FunctionType(), expected.GetName())
	assert.Equal(t, expected.GetName(), actual.GetName())
	assert.Equal(t, expected.GetReturnType(), actual.GetReturnType(), expected.GetName())
	assert.Equal(t, expected.ArgumentIds(), actual.ArgumentIds(), expected.GetName())
	assert.Equal(t, expected.GetDocumentURI(), actual.GetDocumentURI(), expected.GetName())

	assertSameRange(t, expected.GetDeclarationRange(), actual.GetDeclarationRange(), fmt.Sprint("Function declaration range:", expected.GetName()))
	assertSameRange(t, expected.GetDocumentRange(), actual.GetDocumentRange(), fmt.Sprint("Function document range:", expected.GetName()))

	assert.Equal(t, expected.GetKind(), actual.GetKind(), expected.GetName())
	assert.Equal(t, expected.Variables, actual.Variables, expected.GetName())
	assert.Equal(t, expected.Enums, actual.Enums, expected.GetName())
	assert.Equal(t, expected.Structs, actual.Structs, expected.GetName())

	assert.Equal(t, Keys(expected.ChildrenFunctions), Keys(actual.ChildrenFunctions))
	for key, value := range expected.ChildrenFunctions {
		assertSameFunction(t, value, actual.ChildrenFunctions[key])
	}
}

func createParser() Parser {
	return Parser{
		logger: commonlog.MockLogger{},
	}
}

func createStruct(docId string, module string, name string, members []idx.StructMember, idRange idx.Range) idx.Indexable {
	return idx.NewStruct(name, members, module, docId, idRange)
}

func assertSameVariable(t *testing.T, expected idx.Variable, actual idx.Variable) {
	assert.Equal(t, expected.GetName(), actual.GetName())
	assert.Equal(t, expected.GetType(), actual.GetType(), expected.GetName())
	assert.Equal(t, expected.GetDocumentURI(), actual.GetDocumentURI(), expected.GetName())
	assertSameRange(t, expected.GetDeclarationRange(), actual.GetDeclarationRange(), fmt.Sprint("Variable  declaration range:", expected.GetName()))
	assertSameRange(t, expected.GetDocumentRange(), actual.GetDocumentRange(), fmt.Sprint("Variable document range:", expected.GetName()))
	assert.Equal(t, expected.GetKind(), actual.GetKind(), expected.GetName())
}
