package symbols

import (
	"fmt"

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
	returnType     string
	argumentIds    []string // Used to list which variables are defined in function signature. They are fully defined in Variables
	typeIdentifier string

	Variables map[string]*Variable

	BaseIndexable
}

func NewFunction(name string, returnType string, argumentIds []string, module string, docId string, idRange Range, docRange Range) Function {
	return newFunctionType(UserDefined, "", name, returnType, argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func NewTypeFunction(typeIdentifier string, name string, returnType string, argumentIds []string, module string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(Method, typeIdentifier, name, returnType, argumentIds, module, docId, idRange, docRange, kind)
}

func NewMacro(name string, argumentIds []string, module string, docId string, idRange Range, docRange Range) Function {
	return newFunctionType(Macro, "", name, "", argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func newFunctionType(fType FunctionType, typeIdentifier string, name string, returnType string, argumentIds []string, module string, docId string, identifierRangePosition Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		fType:          fType,
		returnType:     returnType,
		argumentIds:    argumentIds,
		typeIdentifier: typeIdentifier,
		BaseIndexable: BaseIndexable{
			name:         name,
			moduleString: module,
			module:       NewModulePathFromString(module),
			documentURI:  docId,
			idRange:      identifierRangePosition,
			docRange:     docRange,
			Kind:         kind,
		},
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

func (f Function) GetFullName() string {
	if f.typeIdentifier == "" {
		return f.GetName()
	}

	return f.typeIdentifier + "." + f.name
}

func (f Function) GetFQN() string {
	return fmt.Sprintf("%s::%s", f.module.GetName(), f.GetName())
}

func (f Function) GetKind() protocol.CompletionItemKind {
	switch f.fType {
	case Method:
		return protocol.CompletionItemKindMethod

	default:
		return protocol.CompletionItemKindFunction
	}
}

func (f Function) GetReturnType() string {
	return f.returnType
}

func (f Function) ArgumentIds() []string {
	return f.argumentIds
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
	if f.typeIdentifier == "" {
		return fmt.Sprintf("%s %s()", f.GetReturnType(), f.GetName())
	}

	return fmt.Sprintf("%s %s.%s()", f.GetReturnType(), f.typeIdentifier, f.GetName())
}
