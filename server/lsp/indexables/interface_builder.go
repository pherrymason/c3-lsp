package indexables

type InterfaceBuilder struct {
	_interface Interface
}

func NewInterfaceBuilder(name string, module string, docId string) *InterfaceBuilder {
	f := &InterfaceBuilder{
		_interface: Interface{
			name:    name,
			methods: make(map[string]Function, 0),
			BaseIndexable: BaseIndexable{
				module:      module,
				documentURI: docId,
			},
		},
	}

	f.WithDocumentRange(0, 0, 0, 1000)

	return f
}

func (ib *InterfaceBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *InterfaceBuilder {
	ib._interface.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return ib
}

func (ib *InterfaceBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *InterfaceBuilder {
	ib._interface.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return ib
}

func (ib *InterfaceBuilder) Build() Interface {
	return ib._interface
}
