package server

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func (h *Server) moduleHasFunctionNamed(moduleName string, name string) bool {
	if moduleName == "" || name == "" {
		return false
	}

	return h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() != moduleName {
			return false
		}
		for _, fun := range module.ChildrenFunctions {
			if fun == nil {
				continue
			}
			if fun.GetName() == name || fun.GetMethodName() == name {
				return true
			}
		}
		return false
	})
}

func (h *Server) validateRenameNoConflict(target renameTarget, newName string) error {
	if indexableIsNil(target.declaration) {
		return nil
	}

	switch decl := target.declaration.(type) {
	case *symbols.Function:
		if decl != nil && h.functionRenameConflicts(decl, newName) {
			return fmt.Errorf("rename conflict: function '%s' already exists in module '%s'", newName, decl.GetModuleString())
		}
	case *symbols.Variable:
		if decl != nil && h.variableRenameConflicts(decl, newName) {
			return fmt.Errorf("rename conflict: variable '%s' already exists in scope", newName)
		}
	case *symbols.StructMember:
		if decl != nil && h.structMemberRenameConflicts(decl, newName) {
			return fmt.Errorf("rename conflict: member '%s' already exists in struct", newName)
		}
	case *symbols.Enumerator:
		if decl != nil && h.enumeratorRenameConflicts(decl, newName) {
			return fmt.Errorf("rename conflict: enum member '%s' already exists", newName)
		}
	case *symbols.FaultConstant:
		if decl != nil && h.faultConstantRenameConflicts(decl, newName) {
			return fmt.Errorf("rename conflict: fault constant '%s' already exists", newName)
		}
	}

	return nil
}

func (h *Server) functionRenameConflicts(target *symbols.Function, newName string) bool {
	return h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() != target.GetModuleString() {
			return false
		}
		for _, fun := range module.ChildrenFunctions {
			if fun == nil {
				continue
			}
			if fun.GetDocumentURI() == target.GetDocumentURI() && fun.GetIdRange() == target.GetIdRange() {
				continue
			}
			if fun.GetMethodName() == newName || fun.GetName() == newName {
				return true
			}
		}
		return false
	})
}

func (h *Server) variableRenameConflicts(target *symbols.Variable, newName string) bool {
	return h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		for _, fun := range module.ChildrenFunctions {
			if fun == nil {
				continue
			}

			for _, variable := range fun.Variables {
				if variable == nil {
					continue
				}
				if variable.GetDocumentURI() == target.GetDocumentURI() && variable.GetIdRange() == target.GetIdRange() {
					if existing, ok := fun.Variables[newName]; ok && existing != nil {
						if existing.GetDocumentURI() != target.GetDocumentURI() || existing.GetIdRange() != target.GetIdRange() {
							return true
						}
					}
					return false
				}
			}
		}

		if module.GetName() == target.GetModuleString() {
			if existing := module.Variables[newName]; existing != nil {
				if existing.GetDocumentURI() != target.GetDocumentURI() || existing.GetIdRange() != target.GetIdRange() {
					return true
				}
			}
		}
		return false
	})
}

func (h *Server) structMemberRenameConflicts(target *symbols.StructMember, newName string) bool {
	return h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() != target.GetModuleString() {
			return false
		}
		for _, strukt := range module.Structs {
			if strukt == nil {
				continue
			}
			containsTarget := false
			for _, member := range strukt.GetMembers() {
				if member != nil && member.GetDocumentURI() == target.GetDocumentURI() && member.GetIdRange() == target.GetIdRange() {
					containsTarget = true
					break
				}
			}
			if !containsTarget {
				continue
			}
			for _, member := range strukt.GetMembers() {
				if member == nil {
					continue
				}
				if member.GetDocumentURI() == target.GetDocumentURI() && member.GetIdRange() == target.GetIdRange() {
					continue
				}
				if member.GetName() == newName {
					return true
				}
			}
			return false
		}
		return false
	})
}

func (h *Server) enumeratorRenameConflicts(target *symbols.Enumerator, newName string) bool {
	return h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() != target.GetModuleString() {
			return false
		}
		for _, enum := range module.Enums {
			if enum == nil || enum.GetName() != target.GetEnumName() {
				continue
			}
			for _, enumerator := range enum.GetEnumerators() {
				if enumerator == nil {
					continue
				}
				if enumerator.GetDocumentURI() == target.GetDocumentURI() && enumerator.GetIdRange() == target.GetIdRange() {
					continue
				}
				if enumerator.GetName() == newName {
					return true
				}
			}
			return false
		}
		return false
	})
}

func (h *Server) faultConstantRenameConflicts(target *symbols.FaultConstant, newName string) bool {
	return h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() != target.GetModuleString() {
			return false
		}
		for _, fault := range module.FaultDefs {
			if fault == nil || fault.GetName() != target.GetFaultName() {
				continue
			}
			for _, constant := range fault.GetConstants() {
				if constant == nil {
					continue
				}
				if constant.GetDocumentURI() == target.GetDocumentURI() && constant.GetIdRange() == target.GetIdRange() {
					continue
				}
				if constant.GetName() == newName {
					return true
				}
			}
			return false
		}
		return false
	})
}
