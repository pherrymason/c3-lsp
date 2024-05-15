package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Interface struct {
	methods map[string]Function
	BaseIndexable
}

func NewInterface(name string, module string, docId string, idRange Range, docRange Range) Interface {
	return Interface{
		methods: make(map[string]Function, 0),
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindInterface,
		),
	}
}

func (i Interface) GetMethod(name string) Function {
	return i.methods[name]
}

func (i *Interface) AddMethods(methods []Function) {
	for _, method := range methods {
		i.methods[method.GetName()] = method
	}
}

func (i Interface) GetHoverInfo() string {
	return fmt.Sprintf("%s", i.name)
}
