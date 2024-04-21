package indexables

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type FunctionType int

const (
	Anonymous = iota
	UserDefined
)

type Function struct {
	fType          FunctionType
	name           string
	returnType     string
	argumentIds    []string // Used to list which variables are defined in function signature. They are fully defined in Variables
	typeIdentifier string

	Variables         map[string]Variable
	Enums             map[string]Enum
	Structs           map[string]Struct
	Defs              map[string]Def
	ChildrenFunctions []Function

	BaseIndexable
}

func NewAnonymousScopeFunction(name string, module string, docId string, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(Anonymous, "", name, module, nil, module, docId, Range{}, docRange, kind)
}

func NewFunction(name string, returnType string, argumentIds []string, module string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(UserDefined, "", name, returnType, argumentIds, module, docId, idRange, docRange, kind)
}

func NewTypeFunction(typeIdentifier string, name string, returnType string, argumentIds []string, module string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(UserDefined, typeIdentifier, name, returnType, argumentIds, module, docId, idRange, docRange, kind)
}

func newFunctionType(fType FunctionType, typeIdentifier string, name string, returnType string, argumentIds []string, module string, docId string, identifierRangePosition Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		fType:          fType,
		name:           name,
		returnType:     returnType,
		argumentIds:    argumentIds,
		typeIdentifier: typeIdentifier,
		BaseIndexable: BaseIndexable{
			module:      module,
			documentURI: docId,
			idRange:     identifierRangePosition,
			docRange:    docRange,
			Kind:        kind,
		},
		Variables:         make(map[string]Variable),
		Enums:             make(map[string]Enum),
		Structs:           make(map[string]Struct),
		Defs:              make(map[string]Def),
		ChildrenFunctions: []Function{},
	}
}

func (f Function) FunctionType() FunctionType {
	return f.fType
}

func (f Function) GetName() string {
	return f.name
}

func (f Function) GetFullName() string {
	if f.typeIdentifier == "" {
		return f.GetName()
	}

	return f.typeIdentifier + "." + f.name
}

func (f Function) GetReturnType() string {
	return f.returnType
}

func (f Function) ArgumentIds() []string {
	return f.argumentIds
}

func (f Function) GetKind() protocol.CompletionItemKind {
	return f.Kind
}

func (f Function) GetModule() string {
	return f.module
}

func (f Function) GetDocumentURI() string {
	return f.documentURI
}

func (f Function) GetIdRange() Range {
	return f.idRange
}

func (f Function) GetDocumentRange() Range {
	return f.docRange
}

func (f *Function) AddVariables(variables []Variable) {
	for _, variable := range variables {
		f.Variables[variable.name] = variable
	}
}

func (f *Function) AddVariable(variable Variable) {
	f.Variables[variable.name] = variable
}

func (f *Function) AddEnum(enum Enum) {
	f.Enums[enum.name] = enum
}

func (f *Function) AddFunction(f2 Function) {
	/*id := f2.name
	if f2.typeIdentifier != "" {
		id = f2.typeIdentifier + "." + f2.name
	}*/

	f.ChildrenFunctions = append(f.ChildrenFunctions, f2)
}

func (f Function) GetChildrenFunctionByName(name string) (fn Function, found bool) {
	for _, fun := range f.ChildrenFunctions {
		if fun.GetFullName() == name {
			return fun, true
		}
	}

	//panic("Function not found")
	return Function{}, false
}

func (f Function) AddStruct(s Struct) {
	f.Structs[s.name] = s
}

func (f Function) GetHoverInfo() string {
	if f.typeIdentifier == "" {
		return fmt.Sprintf("%s %s()", f.GetReturnType(), f.GetName())
	}

	return fmt.Sprintf("%s %s.%s()", f.GetReturnType(), f.typeIdentifier, f.GetName())
}

func (f Function) AddDef(def Def) {
	f.Defs[def.GetName()] = def
}
