package symbols

type ModuleBuilder struct {
	module *Module
}

func NewModuleBuilder(moduleName string, docId string) *ModuleBuilder {
	m := &ModuleBuilder{
		module: &Module{
			BaseIndexable: BaseIndexable{
				name:        moduleName,
				module:      NewModulePathFromString(moduleName),
				documentURI: docId,
			},
		},
	}

	return m
}

func (mb *ModuleBuilder) WithoutSourceCode() *ModuleBuilder {
	mb.module.BaseIndexable.hasSourceCode = false
	return mb
}

func (mb ModuleBuilder) Build() *Module {
	return mb.module
}
