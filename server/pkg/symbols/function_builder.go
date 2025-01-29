package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type FunctionBuilder struct {
	function Function
}

func NewFunctionBuilder(name string, returnType Type, module string, docId string) *FunctionBuilder {
	return &FunctionBuilder{
		function: Function{
			fType:       UserDefined,
			returnType:  returnType,
			argumentIds: nil,
			Variables:   make(map[string]*Variable),
			BaseIndexable: NewBaseIndexable(
				name,
				module,
				docId,
				NewRange(0, 0, 0, 0),
				NewRange(0, 0, 0, 0),
				protocol.CompletionItemKindFunction,
			),
		},
	}
}

func (fb *FunctionBuilder) IsMacro() *FunctionBuilder {
	fb.function.fType = Macro
	return fb
}

func (fb *FunctionBuilder) WithTypeIdentifier(typeIdentifier string) *FunctionBuilder {
	fb.function.typeIdentifier = typeIdentifier
	return fb
}

func (fb *FunctionBuilder) WithArgument(variable *Variable) *FunctionBuilder {
	if fb.function.argumentIds == nil {
		fb.function.argumentIds = []string{}
	}
	fb.function.argumentIds = append(fb.function.argumentIds, variable.GetName())
	fb.function.AddVariable(variable)

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

func (fb *FunctionBuilder) WithDocs(docComment DocComment) *FunctionBuilder {
	fb.function.BaseIndexable.docComment = &docComment
	return fb
}

func (fb *FunctionBuilder) WithoutSourceCode() *FunctionBuilder {
	fb.function.BaseIndexable.hasSourceCode = false
	return fb
}
func (fb *FunctionBuilder) Build() *Function {
	return &fb.function
}
