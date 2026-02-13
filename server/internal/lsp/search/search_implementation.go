package search

import (
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func (s Search) FindImplementationsInWorkspace(
	docId string,
	position symbols.Position,
	state *project_state.ProjectState,
) []symbols.Indexable {
	target := s.FindSymbolDeclarationInWorkspace(docId, position, state)
	if target.IsNone() {
		return []symbols.Indexable{}
	}

	decl := target.Get()

	switch symbol := decl.(type) {
	case *symbols.Interface:
		return findInterfaceImplementations(symbol, state)
	case *symbols.Function:
		return findMethodImplementations(symbol, state)
	default:
		return []symbols.Indexable{}
	}
}

func findInterfaceImplementations(symbol *symbols.Interface, state *project_state.ProjectState) []symbols.Indexable {
	results := []symbols.Indexable{}

	for _, modules := range state.GetAllUnitModules() {
		for _, module := range modules.Modules() {
			for _, strukt := range module.Structs {
				if implementsInterface(strukt.GetInterfaces(), symbol) {
					results = append(results, strukt)
				}
			}

			for _, bitstruct := range module.Bitstructs {
				if implementsInterface(bitstruct.GetInterfaces(), symbol) {
					results = append(results, bitstruct)
				}
			}
		}
	}

	return results
}

func findMethodImplementations(symbol *symbols.Function, state *project_state.ProjectState) []symbols.Indexable {
	interfaceSymbol, ok := findOwningInterface(symbol, state)
	if !ok {
		return []symbols.Indexable{}
	}

	results := []symbols.Indexable{}
	methodName := symbol.GetMethodName()

	for _, modules := range state.GetAllUnitModules() {
		for _, module := range modules.Modules() {
			for _, strukt := range module.Structs {
				if !implementsInterface(strukt.GetInterfaces(), interfaceSymbol) {
					continue
				}
				if method := findMethodInModule(module, strukt.GetName(), methodName); method != nil {
					results = append(results, method)
				}
			}

			for _, bitstruct := range module.Bitstructs {
				if !implementsInterface(bitstruct.GetInterfaces(), interfaceSymbol) {
					continue
				}
				if method := findMethodInModule(module, bitstruct.GetName(), methodName); method != nil {
					results = append(results, method)
				}
			}
		}
	}

	return results
}

func findOwningInterface(method *symbols.Function, state *project_state.ProjectState) (*symbols.Interface, bool) {
	for _, modules := range state.GetAllUnitModules() {
		for _, module := range modules.Modules() {
			for _, iface := range module.Interfaces {
				candidate := iface.GetMethod(method.GetMethodName())
				if candidate == nil {
					continue
				}
				if candidate.GetDocumentURI() != method.GetDocumentURI() {
					continue
				}
				if candidate.GetIdRange() != method.GetIdRange() {
					continue
				}
				return iface, true
			}
		}
	}

	return nil, false
}

func findMethodInModule(module *symbols.Module, typeName string, methodName string) *symbols.Function {
	for _, fun := range module.ChildrenFunctions {
		if fun.FunctionType() != symbols.Method {
			continue
		}
		if fun.GetTypeIdentifier() != typeName {
			continue
		}
		if fun.GetMethodName() != methodName {
			continue
		}
		return fun
	}

	return nil
}

func implementsInterface(implemented []string, iface *symbols.Interface) bool {
	for _, impl := range implemented {
		if impl == iface.GetName() || impl == iface.GetFQN() {
			return true
		}
		if strings.HasSuffix(impl, "::"+iface.GetName()) {
			return true
		}
	}

	return false
}
