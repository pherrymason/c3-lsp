package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type FunctionType int

const (
	Anonymous = iota
	UserDefined
)

type Function struct {
	_type           FunctionType
	Name            string
	ReturnType      string
	DocumentURI     string
	identifierRange Range
	documentRange   Range
	Kind            protocol.CompletionItemKind

	Variables         map[string]Variable
	Enums             map[string]*Enum
	ChildrenFunctions map[string]*Function
}

func NewAnonymousScopeFunction(name string, docId string, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		_type:             Anonymous,
		Name:              name,
		ReturnType:        "??",
		DocumentURI:       docId,
		documentRange:     docRange,
		Kind:              kind,
		Variables:         make(map[string]Variable),
		Enums:             make(map[string]*Enum),
		ChildrenFunctions: make(map[string]*Function),
	}
}

func NewFunction(name string, docId string, identifierRangePosition Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		_type:             UserDefined,
		Name:              name,
		ReturnType:        "??",
		DocumentURI:       docId,
		identifierRange:   identifierRangePosition,
		documentRange:     docRange,
		Kind:              kind,
		Variables:         make(map[string]Variable),
		ChildrenFunctions: make(map[string]*Function),
	}
}

func (f Function) GetName() string {
	return f.Name
}

func (f Function) GetKind() protocol.CompletionItemKind {
	return f.Kind
}

func (f Function) GetDocumentURI() string {
	return f.DocumentURI
}

func (f Function) GetDeclarationRange() Range {
	return f.identifierRange
}

func (f Function) GetDocumentRange() Range {
	return f.documentRange
}

func (f *Function) SetEndRange(endPosition Position) {
	f.documentRange.End = endPosition
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
	f.ChildrenFunctions[f2.Name] = f2
}
