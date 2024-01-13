package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type FunctionType int

const (
	Anonymous = iota
	UserDefined
)

type Function struct {
	fType      FunctionType
	name       string
	returnType string
	arguments  []string // Used to list which variables are defined in function signature. They are fully defined in Variables

	Variables         map[string]Variable
	Enums             map[string]*Enum
	Structs           map[string]Struct
	ChildrenFunctions map[string]*Function

	BaseIndexable
}

func NewAnonymousScopeFunction(name string, docId string, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(Anonymous, name, "", docId, Range{}, docRange, kind)
}

func NewFunction(name string, returnType string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(UserDefined, name, returnType, docId, idRange, docRange, kind)
}

func newFunctionType(fType FunctionType, name string, returnType string, docId string, identifierRangePosition Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		fType:      fType,
		name:       name,
		returnType: returnType,
		BaseIndexable: BaseIndexable{
			documentURI:     docId,
			identifierRange: identifierRangePosition,
			documentRange:   docRange,
			Kind:            kind,
		},
		Variables:         make(map[string]Variable),
		Enums:             make(map[string]*Enum),
		Structs:           make(map[string]Struct),
		ChildrenFunctions: make(map[string]*Function),
	}
}

func (f Function) GetName() string {
	return f.name
}

func (f Function) GetReturnType() string {
	return f.returnType
}

func (f Function) GetKind() protocol.CompletionItemKind {
	return f.Kind
}

func (f Function) GetDocumentURI() string {
	return f.documentURI
}

func (f Function) GetDeclarationRange() Range {
	return f.identifierRange
}

func (f Function) GetDocumentRange() Range {
	return f.documentRange
}

func (f *Function) AddVariables(variables []Variable) {
	for _, variable := range variables {
		f.Variables[variable.Name] = variable
	}
}

func (f *Function) AddEnum(enum *Enum) {
	f.Enums[enum.name] = enum
}

func (f Function) AddFunction(f2 *Function) {
	f.ChildrenFunctions[f2.name] = f2
}

func (f Function) AddStruct(s Struct) {
	f.Structs[s.name] = s
}
