package search_v2

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

type MemberFinder struct {
	projState *project_state.ProjectState
	search    *SearchV2
}

func NewMemberFinder(projState *project_state.ProjectState, search *SearchV2) *MemberFinder {
	return &MemberFinder{
		projState: projState,
		search:    search,
	}
}

// FindMemberOrMethod searches for a member or method in the given parent symbol
func (f *MemberFinder) FindMemberOrMethod(
	parent symbols.Indexable,
	name string,
	ctx AccessContext,
	docId option.Option[string],
	contextModule string,
) (symbols.Indexable, bool) {

	// Try to find direct member first (if readable)
	// Inline distinct types allow member access (that's the point of inline)
	if ctx.MembersReadable && (ctx.FromDistinct == NotFromDistinct || ctx.FromDistinct == InlineDistinct) {
		if member := f.findDirectMember(parent, name); member != nil {
			return member, true
		}
	}

	// Try associated values for instances (enums)
	if !ctx.MembersReadable {
		if member := f.findAssociatedValue(parent, name); member != nil {
			return member, true
		}
	}

	// Try methods (if readable)
	if ctx.MethodsReadable {
		if method := f.findMethod(parent, name, docId, contextModule); method != nil {
			return method, true
		}
	}

	return nil, false
}

func (f *MemberFinder) findDirectMember(parent symbols.Indexable, name string) symbols.Indexable {
	switch p := parent.(type) {
	case *symbols.Enum:
		for _, enumerator := range p.GetEnumerators() {
			if enumerator.GetName() == name {
				return enumerator
			}
		}
		// Also check associated values (accessible on enum instances)
		for i := range p.GetAssociatedValues() {
			if p.GetAssociatedValues()[i].GetName() == name {
				return &p.GetAssociatedValues()[i]
			}
		}

	case *symbols.Fault:
		for _, constant := range p.GetConstants() {
			if constant.GetName() == name {
				return constant
			}
		}

	case *symbols.Struct:
		for _, member := range p.GetMembers() {
			if member.GetName() == name {
				return member
			}
		}

	case *symbols.Enumerator:
		for i := range p.AssociatedValues {
			if p.AssociatedValues[i].GetName() == name {
				return &p.AssociatedValues[i]
			}
		}
	}

	return nil
}

func (f *MemberFinder) findAssociatedValue(parent symbols.Indexable, name string) symbols.Indexable {
	if enum, ok := parent.(*symbols.Enum); ok {
		for i := range enum.GetAssociatedValues() {
			if enum.GetAssociatedValues()[i].GetName() == name {
				return &enum.GetAssociatedValues()[i]
			}
		}
	}
	return nil
}

func (f *MemberFinder) findMethod(
	parent symbols.Indexable,
	name string,
	docId option.Option[string],
	contextModule string,
) symbols.Indexable {

	parentName := ""
	parentFQN := ""

	switch p := parent.(type) {
	case *symbols.Struct:
		parentName = p.GetName()
	case *symbols.Enum:
		parentName = p.GetName()
	case *symbols.Fault:
		parentName = p.GetName()
	case *symbols.Enumerator:
		if p.GetModuleString() == "" || p.GetEnumName() == "" {
			return nil
		}
		parentFQN = p.GetEnumFQN()
	case *symbols.FaultConstant:
		if p.GetModuleString() == "" || p.GetFaultName() == "" {
			return nil
		}
		parentFQN = p.GetFaultFQN()
	default:
		return nil
	}

	// For enumerator/fault constants, we need to get the parent type first
	if parentFQN != "" {
		parentSymbols := f.projState.SearchByFQN(parentFQN)
		if len(parentSymbols) == 0 {
			return nil
		}
		parentName = parentSymbols[0].GetName()
	}

	// Search for the method
	methodFQN := parentName + "." + name
	docIdStr := ""
	if docId.IsSome() {
		docIdStr = docId.Get()
	}
	searchParams := search_params.NewSearchParamsBuilder().
		WithText(methodFQN, symbols.NewRange(0, 0, 0, 0)).
		WithDocId(docIdStr).
		WithContextModuleName(contextModule).
		WithScopeMode(search_params.InModuleRoot).
		Build()

	result := f.search.FindSimpleSymbol(searchParams, f.projState)
	if result.IsSome() {
		return result.Get()
	}
	return nil
}
