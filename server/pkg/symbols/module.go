package symbols

import (
	"slices"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Module struct {
	Variables         map[string]*Variable
	Enums             map[string]*Enum
	FaultDefs         []*FaultDef
	Structs           map[string]*Struct
	Bitstructs        map[string]*Bitstruct
	Aliases           map[string]*Alias
	TypeDefs          map[string]*TypeDef
	ChildrenFunctions []*Function
	Interfaces        map[string]*Interface
	Imports           []string // modules imported in this scope
	ImportNoRecurse   map[string]bool
	GenericParameters map[string]*GenericParameter
	GenericParamOrder []string

	BaseIndexable
}

func NewModule(name string, docId string, idRange Range, docRange Range) *Module {
	return &Module{
		Variables:         make(map[string]*Variable),
		Enums:             make(map[string]*Enum),
		FaultDefs:         []*FaultDef{},
		Structs:           make(map[string]*Struct),
		Bitstructs:        make(map[string]*Bitstruct),
		Aliases:           make(map[string]*Alias),
		TypeDefs:          make(map[string]*TypeDef),
		ChildrenFunctions: []*Function{},
		Interfaces:        make(map[string]*Interface),
		Imports:           []string{},
		ImportNoRecurse:   make(map[string]bool),

		BaseIndexable: NewBaseIndexable(
			name,
			name,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindModule,
		),
	}
}

func (m *Module) AddVariable(variable *Variable) *Module {
	m.Variables[variable.Name] = variable
	m.Insert(variable)

	return m
}

func (m *Module) AddVariables(variables []*Variable) {
	for _, variable := range variables {
		m.Variables[variable.Name] = variable
		m.Insert(variable)
	}
}

func (m *Module) AddEnum(enum *Enum) *Module {
	m.Enums[enum.Name] = enum
	m.Insert(enum)

	return m
}

func (m *Module) AddFaultDef(fault *FaultDef) *Module {
	if fault.baseType != "" {
		// Skip fault definitions that still carry transitional base-type data.
		return m
	}
	m.FaultDefs = append(m.FaultDefs, fault)
	m.Insert(fault)

	return m
}

func (m *Module) AddFunction(fun *Function) *Module {
	m.ChildrenFunctions = append(m.ChildrenFunctions, fun)
	m.InsertNestedScope(fun)

	return m
}

func (m *Module) AddInterface(_interface *Interface) *Module {
	m.Interfaces[_interface.Name] = _interface
	m.Insert(_interface)

	return m
}

func (m *Module) AddStruct(s *Struct) *Module {
	m.Structs[s.Name] = s
	m.Insert(s)

	return m
}

func (m *Module) AddBitstruct(b *Bitstruct) *Module {
	m.Bitstructs[b.Name] = b
	m.Insert(b)

	return m
}

func (m *Module) AddAlias(def *Alias) *Module {
	m.Aliases[def.GetName()] = def
	m.Insert(def)

	return m
}

func (m *Module) AddTypeDef(distinct *TypeDef) *Module {
	m.TypeDefs[distinct.GetName()] = distinct
	m.Insert(distinct)

	return m
}

func (m *Module) AddImports(imports []string) {
	m.AddImportsWithMode(imports, false)
}

func (m *Module) AddImportsWithMode(imports []string, noRecurse bool) {
	if m.ImportNoRecurse == nil {
		m.ImportNoRecurse = make(map[string]bool)
	}

	m.Imports = append(m.Imports, imports...)
	for _, imported := range imports {
		if imported == "" {
			continue
		}
		current, exists := m.ImportNoRecurse[imported]
		if !exists {
			m.ImportNoRecurse[imported] = noRecurse
			continue
		}
		m.ImportNoRecurse[imported] = current && noRecurse
	}
}

func (m *Module) IsImportNoRecurse(imported string) bool {
	if m == nil || m.ImportNoRecurse == nil {
		return false
	}

	return m.ImportNoRecurse[imported]
}

func (m *Module) ChangeModule(module string) {
	m.Name = module
	m.Module = NewModulePathFromString(module)
}

func (m *Module) SetStartPosition(position Position) {
	m.DocRange.Start = position
}

func (m *Module) SetEndPosition(position Position) {
	m.DocRange.End = position
}

func (m *Module) SetGenericParameters(generics map[string]*GenericParameter) *Module {
	m.GenericParameters = generics
	for _, gn := range generics {
		m.Insert(gn)
	}

	return m
}

func (m *Module) SetGenericParameterOrder(names []string) *Module {
	m.GenericParamOrder = names
	return m
}

func (m *Module) GetHoverInfo() string {
	if len(m.GenericParameters) == 0 {
		return m.Name
	}

	names := m.GenericParamOrder
	if len(names) == 0 {
		names = make([]string, 0, len(m.GenericParameters))
		for name := range m.GenericParameters {
			names = append(names, name)
		}
		slices.Sort(names)
	}

	return m.Name + " <" + strings.Join(names, ", ") + ">"
}

func (m *Module) GetCompletionDetail() string {
	if len(m.GenericParameters) == 0 {
		return "Module"
	}

	names := m.GenericParamOrder
	if len(names) == 0 {
		names = make([]string, 0, len(m.GenericParameters))
		for name := range m.GenericParameters {
			names = append(names, name)
		}
		slices.Sort(names)
	}

	return "Module<" + strings.Join(names, ", ") + ">"
}

func (m *Module) GetChildrenFunctionByName(name string) option.Option[*Function] {
	for _, fun := range m.ChildrenFunctions {
		if fun.GetFullName() == name {
			return option.Some(fun)
		}
	}

	return option.None[*Function]()
}
