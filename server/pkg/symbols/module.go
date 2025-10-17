package symbols

import (
	"strings"
	"unicode"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Module struct {
	Variables         map[string]*Variable
	Enums             map[string]*Enum
	Faults            []*Fault
	Structs           map[string]*Struct
	Bitstructs        map[string]*Bitstruct
	Defs              map[string]*Def
	Distincts         map[string]*Distinct
	ChildrenFunctions []*Function
	Interfaces        map[string]*Interface
	Imports           []string // modules imported in this scope
	GenericParameters map[string]*GenericParameter

	BaseIndexable
}

func NewModule(name string, docId string, idRange Range, docRange Range) *Module {
	return &Module{
		Variables:         make(map[string]*Variable),
		Enums:             make(map[string]*Enum),
		Faults:            []*Fault{},
		Structs:           make(map[string]*Struct),
		Bitstructs:        make(map[string]*Bitstruct),
		Defs:              make(map[string]*Def),
		Distincts:         make(map[string]*Distinct),
		ChildrenFunctions: []*Function{},
		Interfaces:        make(map[string]*Interface),
		Imports:           []string{},

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
	m.Variables[variable.name] = variable
	m.Insert(variable)

	return m
}

func (m *Module) AddVariables(variables []*Variable) {
	for _, variable := range variables {
		m.Variables[variable.name] = variable
		m.Insert(variable)
	}
}

func (m *Module) AddEnum(enum *Enum) *Module {
	m.Enums[enum.name] = enum
	m.Insert(enum)

	return m
}

func (m *Module) AddFault(fault *Fault) *Module {
	//TODO: @0.7.7 Fault does not have a name anymore, but as a workaround all defined faults have name ""
	if fault.baseType != "" {
		panic("In C3 0.7.X faultdef do not have a name")
	}
	m.Faults = append(m.Faults, fault)
	m.Insert(fault)

	return m
}

func (m *Module) AddFunction(fun *Function) *Module {
	m.ChildrenFunctions = append(m.ChildrenFunctions, fun)
	m.InsertNestedScope(fun)

	return m
}

func (m *Module) AddInterface(_interface *Interface) *Module {
	m.Interfaces[_interface.name] = _interface
	m.Insert(_interface)

	return m
}

func (m *Module) AddStruct(s *Struct) *Module {
	m.Structs[s.name] = s
	m.Insert(s)

	return m
}

func (m *Module) AddBitstruct(b *Bitstruct) *Module {
	m.Bitstructs[b.name] = b
	m.Insert(b)

	return m
}

func (m *Module) AddDef(def *Def) *Module {
	m.Defs[def.GetName()] = def
	m.Insert(def)

	return m
}

func (m *Module) AddDistinct(distinct *Distinct) *Module {
	m.Distincts[distinct.GetName()] = distinct
	m.Insert(distinct)

	return m
}

func (m *Module) AddImports(imports []string) {
	m.Imports = append(m.Imports, imports...)
}

func (m *Module) ChangeModule(module string) {
	m.name = module
	m.module = NewModulePathFromString(module)
}

func (m *Module) SetStartPosition(position Position) {
	m.docRange.Start = position
}

func (m *Module) SetEndPosition(position Position) {
	m.docRange.End = position
}

func (m *Module) SetGenericParameters(generics map[string]*GenericParameter) *Module {
	m.GenericParameters = generics
	for _, gn := range generics {
		m.Insert(gn)
	}

	return m
}

func (m *Module) GetHoverInfo() string {
	return m.name
}

func (m *Module) GetCompletionDetail() string {
	return "Module"
}

func (m *Module) GetChildrenFunctionByName(name string) option.Option[*Function] {
	for _, fun := range m.ChildrenFunctions {
		if fun.GetFullName() == name {
			return option.Some(fun)
		}
	}

	return option.None[*Function]()
}

type ModulePath struct {
	tokens []string
}

func NewModulePath(path []string) ModulePath {
	return ModulePath{
		tokens: path,
	}
}

func NewModulePathFromString(module string) ModulePath {
	var modules []string
	if len(module) > 0 {
		modules = strings.Split(module, "::")
	}

	return NewModulePath(modules)
}

func (mp *ModulePath) AddPath(path string) {
	newSlice := []string{path}
	mp.tokens = append(newSlice, mp.tokens...)
}

func (mp ModulePath) GetName() string {
	return concatPaths(mp.tokens, "::")
}

func (mp ModulePath) IsEmpty() bool {
	return len(mp.tokens) == 0
}

func (mp ModulePath) IsSubModuleOf(parentModule ModulePath) bool {
	if len(mp.tokens) < len(parentModule.tokens) {
		return false
	}

	isChild := true
	for i, pm := range parentModule.tokens {
		if i > len(mp.tokens) {
			break
		}

		if mp.tokens[i] != pm {
			isChild = false
			break
		}
	}

	return isChild
}

func (mp ModulePath) IsImplicitlyImported(otherModule ModulePath) bool {
	if mp.GetName() == otherModule.GetName() {
		return true
	}

	isSubModuleOf := mp.IsSubModuleOf(otherModule)
	isParentOf := otherModule.IsSubModuleOf(mp)

	return isSubModuleOf || isParentOf
}

func concatPaths(slice []string, delimiter string) string {
	result := ""

	for i, str := range slice {
		if i > 0 {
			result += delimiter
		}
		result += str
	}
	return result
}

func NormalizeModuleName(input string) string {
	const maxLength = 31
	var modifiedName []rune

	input = strings.TrimSuffix(input, ".c3")

	// Iterar sobre cada caracter del string de entrada
	for _, char := range input {
		// Verificar si el caracter es alfanumérico y minúscula
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			if unicode.IsLower(char) {
				// Si es alfanumérico y minúscula, añadirlo al nombre modificado
				modifiedName = append(modifiedName, char)
			} else {
				// Si es alfanumérico pero no es minúscula, convertirlo a minúscula y añadirlo al nombre modificado
				modifiedName = append(modifiedName, unicode.ToLower(char))
			}
		} else {
			// Si no es alfanumérico, reemplazarlo por '_'
			modifiedName = append(modifiedName, '_')
		}

		// Verificar si la longitud del nombre modificado excede el máximo permitido
		if len(modifiedName) >= maxLength {
			break
		}
	}

	// Devolver el nombre modificado como un string
	return string(modifiedName)
}
