package symbols

import "strings"

// ModulePath represents the hierarchical path of a C3 module (e.g., "foo::bar::baz").
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
