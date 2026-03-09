package search

import (
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

// ResolveOneLevelSymbol resolves one indirection step for a symbol into the
// next inspectable symbol candidate (type/definition target/etc.).
//
// This helper is shared by search v1 and v2 to keep one-level resolution
// semantics aligned.
func ResolveOneLevelSymbol(symbol symbols.Indexable, projState *project_state.ProjectState, hierarchySymbols []symbols.Indexable) symbols.Indexable {
	switch s := symbol.(type) {
	case *symbols.Variable:
		return lookupTypeByFQN(projState, s.GetType().GetFullQualifiedName())

	case *symbols.StructMember:
		if s.IsStruct() {
			return s.Substruct().Get()
		}
		return lookupTypeByFQN(projState, s.GetType().GetFullQualifiedName())

	case *symbols.Function:
		resolvedType := resolveTypeForHierarchy(*s.GetReturnType(), hierarchySymbols)
		return lookupTypeByFQN(projState, resolvedType.GetFullQualifiedName())

	case *symbols.Alias:
		if s.ResolvesToType() {
			return lookupTypeByFQN(projState, s.ResolvedType().GetFullQualifiedName())
		}
		return lookupAnyByFQN(projState, s.GetModuleString()+"::"+s.GetResolvesTo())

	case *symbols.TypeDef:
		baseType := s.GetBaseType()
		if baseType == nil || baseType.GetName() == "" {
			return nil
		}
		return lookupTypeByFQN(projState, baseType.GetFullQualifiedName())

	default:
		return nil
	}
}

func lookupTypeByFQN(projState *project_state.ProjectState, fqn string) symbols.Indexable {
	if projState == nil || fqn == "" {
		return nil
	}

	module, name := splitFQN(fqn)
	if name == "" {
		return nil
	}

	if exact := findTypeSymbol(projState.SearchByFQN(fqn), name, module); exact != nil {
		return exact
	}

	snapshot := projState.Snapshot()
	if snapshot == nil {
		return nil
	}

	if module != "" && !strings.Contains(module, "::") {
		for _, full := range snapshot.ModuleNamesByShort(module) {
			if exact := findTypeSymbol(projState.SearchByFQN(full+"::"+name), name, full); exact != nil {
				return exact
			}
		}
	}

	var result symbols.Indexable
	snapshot.ForEachModuleUntil(func(mod *symbols.Module) bool {
		moduleName := mod.GetModuleString()
		if module != "" && strings.Contains(module, "::") && moduleName != module {
			return false
		}

		if exact := findTypeSymbol(projState.SearchByFQN(moduleName+"::"+name), name, moduleName); exact != nil {
			result = exact
			return true
		}
		return false
	})

	return result
}

func lookupAnyByFQN(projState *project_state.ProjectState, fqn string) symbols.Indexable {
	if projState == nil || fqn == "" {
		return nil
	}

	results := projState.SearchByFQN(fqn)
	if len(results) == 0 {
		return nil
	}

	module, name := splitFQN(fqn)

	for _, candidate := range results {
		if candidate == nil {
			continue
		}

		if candidate.GetName() == name && (module == "" || candidate.GetModuleString() == module) {
			return candidate
		}
	}

	return results[0]
}

func splitFQN(fqn string) (string, string) {
	sep := strings.LastIndex(fqn, "::")
	if sep < 0 {
		return "", fqn
	}

	module := strings.TrimSpace(fqn[:sep])
	name := strings.TrimSpace(fqn[sep+2:])
	return module, name
}

func findTypeSymbol(results []symbols.Indexable, expectedName string, expectedModule string) symbols.Indexable {
	for _, candidate := range results {
		if candidate == nil || candidate.GetName() != expectedName {
			continue
		}

		if expectedModule != "" && candidate.GetModuleString() != expectedModule {
			continue
		}

		switch typed := candidate.(type) {
		case *symbols.Struct, *symbols.Enum, *symbols.FaultDef, *symbols.TypeDef:
			return candidate
		case *symbols.Alias:
			if typed.ResolvesToType() {
				return candidate
			}
		}
	}

	return nil
}

func resolveTypeForHierarchy(_type symbols.Type, hierarchySymbols []symbols.Indexable) symbols.Type {
	if !_type.IsGenericArgument() {
		return _type
	}

	var parentType *symbols.Type
	for i := len(hierarchySymbols) - 1; i >= 0; i-- {
		switch elm := hierarchySymbols[i].(type) {
		case *symbols.StructMember:
			if elm.GetType().HasGenericArguments() {
				parentType = elm.GetType()
				i = -1
			}
		}
	}

	if parentType != nil {
		return parentType.GetGenericArgument(0)
	}

	return _type
}
