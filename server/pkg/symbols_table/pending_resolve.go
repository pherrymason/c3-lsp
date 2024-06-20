package symbols_table

import "github.com/pherrymason/c3-lsp/pkg/symbols"

// This structure contains stuff that could not be fully resolved and will
// need a second pass of processing, usually after more modules were being parsed.
type PendingToResolve struct {
	// types that were not found in same file/module and will be potentially defined in the wrong module.
	typesByModule map[string][]PendingTypeContext

	// inline structs found while parsing that will need to be unrolled
	subtyptingToResolve []StructWithSubtyping
}

type PendingTypeContext struct {
	vType         *symbols.Type
	contextModule *symbols.Module // Module where this Type is present
	solved        bool
}

func (pt *PendingTypeContext) Solve() {
	pt.solved = true
}
func (pt *PendingTypeContext) IsSolved() bool {
	return pt.solved
}

type StructWithSubtyping struct {
	strukt  *symbols.Struct
	members []symbols.Type
}

func NewPendingToResolve() PendingToResolve {
	return PendingToResolve{
		typesByModule:       make(map[string][]PendingTypeContext),
		subtyptingToResolve: []StructWithSubtyping{},
	}
}

// Getters ----------
func (p *PendingToResolve) GetTypesByModule(docId string) []PendingTypeContext {
	return p.typesByModule[docId]
}

// Setters ----------
func (p *PendingToResolve) AddStructSubtype(strukt *symbols.Struct, types []symbols.Type) {
	p.subtyptingToResolve = append(
		p.subtyptingToResolve,
		StructWithSubtyping{strukt: strukt, members: types},
	)
}

func (p *PendingToResolve) AddStructSubtype2(strukt *symbols.Struct) {
	inlineMembers := []symbols.Type{}
	for _, member := range strukt.GetMembers() {
		if member.IsInlinePendingToResolve() && !member.IsExpandedInline() {
			inlineMembers = append(inlineMembers, *member.GetType())
		}
	}

	if len(inlineMembers) > 0 {
		p.subtyptingToResolve = append(
			p.subtyptingToResolve,
			StructWithSubtyping{strukt: strukt, members: inlineMembers},
		)
	}
}

func (p *PendingToResolve) SolveType(moduleName string, indexSolved int) {
	p.typesByModule[moduleName] = append(p.typesByModule[moduleName][:indexSolved], p.typesByModule[moduleName][indexSolved+1:]...)

}

func (p *PendingToResolve) AddVariableType(variables []*symbols.Variable, contextModule *symbols.Module) {
	for _, variable := range variables {
		sType := variable.GetType()
		if !sType.IsBaseTypeLanguage() {
			p.typesByModule[variable.GetModuleString()] = append(
				p.typesByModule[variable.GetModuleString()],
				PendingTypeContext{
					vType:         sType,
					contextModule: contextModule,
				},
			)
		}
	}
}

func (p *PendingToResolve) AddStructMemberTypes(strukt *symbols.Struct, contextModule *symbols.Module) {
	for _, member := range strukt.GetMembers() {
		sType := member.GetType()
		if !sType.IsBaseTypeLanguage() {
			p.typesByModule[contextModule.GetName()] = append(
				p.typesByModule[contextModule.GetName()],
				PendingTypeContext{
					vType:         sType,
					contextModule: contextModule,
				},
			)
		}
	}
}

func (p *PendingToResolve) AddFunctionTypes(function *symbols.Function, contextModule *symbols.Module) {
	if !function.GetReturnType().IsBaseTypeLanguage() {
		p.typesByModule[contextModule.GetName()] = append(
			p.typesByModule[contextModule.GetName()],
			PendingTypeContext{
				vType:         function.GetReturnType(),
				contextModule: contextModule,
			},
		)
	}

	for _, arg := range function.ArgumentIds() {
		variable := function.Variables[arg]
		if !variable.GetType().IsBaseTypeLanguage() {
			p.typesByModule[contextModule.GetName()] = append(
				p.typesByModule[contextModule.GetName()],
				PendingTypeContext{
					vType:         variable.GetType(),
					contextModule: contextModule,
				},
			)
		}
	}
}

func (p *PendingToResolve) AddDefType(def *symbols.Def, contextModule *symbols.Module) {
	if !def.ResolvesToType() {
		return
	}

	if def.ResolvesToType() && def.ResolvedType().IsBaseTypeLanguage() {
		return
	}

	p.typesByModule[contextModule.GetName()] = append(
		p.typesByModule[contextModule.GetName()],
		PendingTypeContext{
			vType:         def.ResolvedType(),
			contextModule: contextModule,
		},
	)
}
