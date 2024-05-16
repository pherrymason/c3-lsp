package symbols

import (
	"strings"
	"unicode"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Module struct {
	Variables         map[string]Variable
	Enums             map[string]Enum
	Faults            map[string]Fault
	Structs           map[string]Struct
	Bitstructs        map[string]Bitstruct
	Defs              map[string]Def
	ChildrenFunctions []Function
	Interfaces        map[string]Interface
	Imports           []string // modules imported in this scope

	BaseIndexable
	ChildrenIndexable
}

func NewModule(name string, docId string, idRange Range, docRange Range) *Module {
	return &Module{
		Variables:         make(map[string]Variable),
		Enums:             make(map[string]Enum),
		Faults:            make(map[string]Fault),
		Structs:           make(map[string]Struct),
		Bitstructs:        make(map[string]Bitstruct),
		Defs:              make(map[string]Def),
		ChildrenFunctions: []Function{},
		Interfaces:        make(map[string]Interface),
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

func (m *Module) AddVariable(variable Variable) {
	m.Variables[variable.name] = variable
}

func (m *Module) AddVariables(variables []Variable) {
	for _, variable := range variables {
		m.Variables[variable.name] = variable
	}
}

func (m *Module) AddEnum(enum Enum) {
	m.Enums[enum.name] = enum
}

func (m *Module) AddFault(fault Fault) {
	m.Faults[fault.name] = fault
}

func (m *Module) AddFunction(f2 Function) {
	m.ChildrenFunctions = append(m.ChildrenFunctions, f2)
}

func (m *Module) AddInterface(_interface Interface) {
	m.Interfaces[_interface.name] = _interface
}

func (m Module) AddStruct(s Struct) {
	m.Structs[s.name] = s
}

func (m Module) AddBitstruct(b Bitstruct) {
	m.Bitstructs[b.name] = b
}

func (m Module) AddDef(def Def) {
	m.Defs[def.GetName()] = def
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

func (m Module) GetHoverInfo() string {
	return m.name
}

func (m Module) GetChildrenFunctionByName(name string) (fn Function, found bool) {
	for _, fun := range m.ChildrenFunctions {
		if fun.GetFullName() == name {
			return fun, true
		}
	}

	return Function{}, false
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
