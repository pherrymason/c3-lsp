package indexables

import (
	"strings"
	"unicode"
)

type Module struct {
	Name  string
	Alias string
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
