package indexables

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type FunctionType int

const (
	ModuleScope = iota
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
	Faults            map[string]Fault
	Structs           map[string]Struct
	Defs              map[string]Def
	ChildrenFunctions []Function
	Interfaces        map[string]Interface
	Imports           []string // modules imported in this scope

	BaseIndexable
}

func NewModuleScopeFunction(module string, docId string, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(ModuleScope, "", "main", module, nil, module, docId, Range{}, docRange, kind)
}

func NewFunction(name string, returnType string, argumentIds []string, module string, docId string, idRange Range, docRange Range) Function {
	return newFunctionType(UserDefined, "", name, returnType, argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func NewTypeFunction(typeIdentifier string, name string, returnType string, argumentIds []string, module string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(UserDefined, typeIdentifier, name, returnType, argumentIds, module, docId, idRange, docRange, kind)
}

func NewMacro(name string, argumentIds []string, module string, docId string, idRange Range, docRange Range) Function {
	return newFunctionType(UserDefined, "", name, "", argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func newFunctionType(fType FunctionType, typeIdentifier string, name string, returnType string, argumentIds []string, module string, docId string, identifierRangePosition Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		fType:          fType,
		name:           name,
		returnType:     returnType,
		argumentIds:    argumentIds,
		typeIdentifier: typeIdentifier,
		BaseIndexable: BaseIndexable{
			moduleString: module,
			module:       NewModulePathFromString(module),
			documentURI:  docId,
			idRange:      identifierRangePosition,
			docRange:     docRange,
			Kind:         kind,
		},
		Faults:            make(map[string]Fault),
		Variables:         make(map[string]Variable),
		Enums:             make(map[string]Enum),
		Structs:           make(map[string]Struct),
		Defs:              make(map[string]Def),
		ChildrenFunctions: []Function{},
		Interfaces:        make(map[string]Interface),
		Imports:           []string{},
	}
}

func (f Function) Id() string {
	return f.documentURI + f.module.GetName()
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

func (f Function) GetModuleString() string {
	return f.moduleString
}

func (f Function) GetModule() ModulePath {
	return f.module
}

func (f Function) IsSubModuleOf(module ModulePath) bool {
	return f.module.IsSubModuleOf(module)
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

func (f *Function) AddFault(fault Fault) {
	f.Faults[fault.name] = fault
}

func (f *Function) AddFunction(f2 Function) {
	f.ChildrenFunctions = append(f.ChildrenFunctions, f2)
}

func (f *Function) AddInterface(_interface Interface) {
	f.Interfaces[_interface.name] = _interface
}

func (f *Function) ChangeModule(module string) {
	f.moduleString = module
	f.module = NewModulePathFromString(module)
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

func (f *Function) AddImports(imports []string) {
	f.Imports = append(f.Imports, imports...)
}
