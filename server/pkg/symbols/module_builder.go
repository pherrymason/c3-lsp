package symbols

type ModuleBuilder struct {
	module *Module
}

func NewModuleBuilder(moduleName string, docId string) *ModuleBuilder {
	module := NewModule(moduleName, docId, NewRange(0, 0, 0, 0), NewRange(0, 0, 0, 0))

	m := &ModuleBuilder{
		module: module,
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
