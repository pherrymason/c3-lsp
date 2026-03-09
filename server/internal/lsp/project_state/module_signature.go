package project_state

import (
	"hash/fnv"
	"sort"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func computeModuleSignature(module *symbols.Module) uint64 {
	parts := []string{}

	imports := append([]string(nil), module.Imports...)
	sort.Strings(imports)
	parts = append(parts, "imports:"+strings.Join(imports, ","))

	funParts := []string{}
	for _, fun := range module.ChildrenFunctions {
		funParts = append(funParts, fun.DisplaySignature(true))
	}
	sort.Strings(funParts)
	parts = append(parts, "functions:"+strings.Join(funParts, ";"))

	structParts := []string{}
	for _, strukt := range module.Structs {
		memberParts := []string{}
		for _, member := range strukt.GetMembers() {
			memberParts = append(memberParts, member.GetName()+":"+member.GetType().String())
		}
		sort.Strings(memberParts)
		structParts = append(structParts, strukt.GetName()+"{"+strings.Join(memberParts, ",")+"}")
	}
	sort.Strings(structParts)
	parts = append(parts, "structs:"+strings.Join(structParts, ";"))

	enumParts := []string{}
	for _, enum := range module.Enums {
		enumeratorNames := []string{}
		for _, e := range enum.GetEnumerators() {
			enumeratorNames = append(enumeratorNames, e.GetName())
		}
		sort.Strings(enumeratorNames)
		enumParts = append(enumParts, enum.GetName()+":"+enum.GetType()+"{"+strings.Join(enumeratorNames, ",")+"}")
	}
	sort.Strings(enumParts)
	parts = append(parts, "enums:"+strings.Join(enumParts, ";"))

	defParts := make([]string, 0, len(module.Aliases))
	for _, def := range module.Aliases {
		if def.ResolvesToType() {
			defParts = append(defParts, def.GetName()+":"+def.ResolvedType().String())
		} else {
			defParts = append(defParts, def.GetName()+":"+def.GetResolvesTo())
		}
	}
	sort.Strings(defParts)
	parts = append(parts, "defs:"+strings.Join(defParts, ";"))

	distinctParts := make([]string, 0, len(module.TypeDefs))
	for _, distinct := range module.TypeDefs {
		distinctParts = append(distinctParts, distinct.GetName()+":"+distinct.GetBaseType().String())
	}
	sort.Strings(distinctParts)
	parts = append(parts, "distincts:"+strings.Join(distinctParts, ";"))

	faultParts := []string{}
	for _, fault := range module.FaultDefs {
		constants := fault.GetConstants()
		names := make([]string, 0, len(constants))
		for _, c := range constants {
			names = append(names, c.GetName())
		}
		sort.Strings(names)
		faultParts = append(faultParts, fault.GetName()+"{"+strings.Join(names, ",")+"}")
	}
	sort.Strings(faultParts)
	parts = append(parts, "faults:"+strings.Join(faultParts, ";"))

	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.Join(parts, "|")))
	return h.Sum64()
}
