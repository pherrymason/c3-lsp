package symbols

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type FunctionType int

const (
	UserDefined = iota
	Method
	Macro
)

type Function struct {
	fType          FunctionType
	returnType     Type
	argumentIds    []string // Used to list which variables are defined in function signature. They are fully defined in Variables
	typeIdentifier string

	Variables map[string]*Variable

	BaseIndexable
}

func NewFunction(name string, returnType Type, argumentIds []string, module string, docId string, idRange Range, docRange Range) Function {
	return newFunctionType(UserDefined, "", name, returnType, argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func NewTypeFunction(typeIdentifier string, name string, returnType Type, argumentIds []string, module string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(Method, typeIdentifier, name, returnType, argumentIds, module, docId, idRange, docRange, kind)
}

func NewMacro(name string, argumentIds []string, module string, docId string, idRange Range, docRange Range) Function {
	return newFunctionType(Macro, "", name, NewTypeFromString("", module), argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func newFunctionType(fType FunctionType, typeIdentifier string, name string, returnType Type, argumentIds []string, module string, docId string, identifierRangePosition Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		fType:          fType,
		returnType:     returnType,
		argumentIds:    argumentIds,
		typeIdentifier: typeIdentifier,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			identifierRangePosition,
			docRange,
			kind,
		),
		Variables: make(map[string]*Variable),
	}
}

func (f Function) Id() string {
	return f.documentURI + f.module.GetName()
}

func (f Function) FunctionType() FunctionType {
	return f.fType
}

func (f Function) GetName() string {
	if f.typeIdentifier == "" {
		return f.name
	}

	return f.typeIdentifier + "." + f.name
}

func (f Function) GetMethodName() string {
	return f.name
}

func (f Function) GetFullName() string {
	if f.typeIdentifier == "" {
		return f.GetName()
	}

	return f.typeIdentifier + "." + f.name
}

func (f Function) GetFQN() string {
	return fmt.Sprintf("%s::%s", f.module.GetName(), f.GetName())
}

func (f Function) GetTypeIdentifier() string {
	return f.typeIdentifier
}

func (f Function) GetKind() protocol.CompletionItemKind {
	switch f.fType {
	case Method:
		return protocol.CompletionItemKindMethod

	default:
		return protocol.CompletionItemKindFunction
	}
}

func (f *Function) GetReturnType() *Type {
	return &f.returnType
}

func (f Function) ArgumentIds() []string {
	return f.argumentIds
}

func (f *Function) GetArguments() []*Variable {
	arguments := []*Variable{}
	for _, arg := range f.ArgumentIds() {
		arguments = append(arguments, f.Variables[arg])
	}

	return arguments
}

func (f *Function) AddVariables(variables []*Variable) {
	for _, variable := range variables {
		f.Variables[variable.name] = variable
		f.Insert(variable)
	}
}

func (f *Function) AddVariable(variable *Variable) {
	f.Variables[variable.name] = variable
	f.Insert(variable)
}

func (f *Function) SetDocRange(docRange Range) {
	f.docRange = docRange
}

func (f *Function) SetStartPosition(position Position) {
	f.docRange.Start = position
}

func (f *Function) SetEndPosition(position Position) {
	f.docRange.End = position
}

func (f Function) GetHoverInfo() string {

	args := []string{}
	for _, arg := range f.argumentIds {
		args = append(args, f.Variables[arg].Type.name+" "+f.Variables[arg].name)
	}

	source := fmt.Sprintf("%s %s(%s)", f.GetReturnType(), f.GetFullName(), strings.Join(args, ", "))

	return source
}
