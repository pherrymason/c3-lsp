package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type FunctionBuilder struct {
	function Function
}

func NewFunctionBuilderARoot(module string, docId string) *FunctionBuilder {
	f := &FunctionBuilder{
		function: Function{
			fType:             Anonymous,
			Variables:         make(map[string]Variable),
			Defs:              make(map[string]Def),
			Enums:             make(map[string]Enum),
			Structs:           make(map[string]Struct),
			ChildrenFunctions: make([]Function, 0),
			BaseIndexable: BaseIndexable{
				module:      module,
				documentURI: docId,
			},
		},
	}

	f.WithDocumentRange(0, 0, 0, 1000)

	return f
}

func NewFunctionBuilder(name string, returnType string, module string, docId string) *FunctionBuilder {
	return &FunctionBuilder{
		function: Function{
			fType:             UserDefined,
			name:              name,
			returnType:        returnType,
			argumentIds:       nil,
			Variables:         make(map[string]Variable),
			Defs:              make(map[string]Def),
			Enums:             make(map[string]Enum),
			Structs:           make(map[string]Struct),
			ChildrenFunctions: make([]Function, 0),
			BaseIndexable: BaseIndexable{
				module:      module,
				documentURI: docId,
				Kind:        protocol.CompletionItemKindFunction,
			},
		},
	}
}

func (fb *FunctionBuilder) WithTypeIdentifier(typeIdentifier string) *FunctionBuilder {
	fb.function.typeIdentifier = typeIdentifier
	return fb
}

func (fb *FunctionBuilder) WithArgument(variable Variable) *FunctionBuilder {
	if fb.function.argumentIds == nil {
		fb.function.argumentIds = []string{}
	}
	fb.function.argumentIds = append(fb.function.argumentIds, variable.GetName())
	fb.function.Variables[variable.GetName()] = variable

	return fb
}

func (fb *FunctionBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FunctionBuilder {
	fb.function.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return fb
}

func (fb *FunctionBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FunctionBuilder {
	fb.function.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return fb
}

func (fb *FunctionBuilder) Build() Function {
	return fb.function
}
