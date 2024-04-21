package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type Interface struct {
	name    string
	methods map[string]Function
	BaseIndexable
}

func NewInterface(name string, module string, docId string, idRange Range, docRange Range) Interface {
	return Interface{
		name:    name,
		methods: make(map[string]Function, 0),
		BaseIndexable: BaseIndexable{
			module:      module,
			documentURI: docId,
			idRange:     idRange,
			docRange:    docRange,
			Kind:        protocol.CompletionItemKindInterface,
		},
	}
}

func (i Interface) GetName() string {
	return i.name
}

func (i Interface) GetMethod(name string) Function {
	return i.methods[name]
}

func (i *Interface) AddMethods(methods []Function) {
	for _, method := range methods {
		i.methods[method.GetName()] = method
	}
}

func (i Interface) GetModule() string {
	return i.module
}

func (i Interface) GetKind() protocol.CompletionItemKind {
	return i.Kind
}

func (i Interface) GetDocumentURI() string {
	return i.documentURI
}

func (i Interface) GetIdRange() Range {
	return i.idRange
}

func (i Interface) GetDocumentRange() Range {
	return i.docRange
}
