package search_v2

import "github.com/pherrymason/c3-lsp/pkg/symbols"

// AccessContext tracks what is accessible at each step of resolution
// Immutable
type AccessContext struct {
	FromDistinct    int  // One of NotFromDistinct, InlineDistinct, NonInlineDistinct
	MembersReadable bool // Can access enum variants, fault constants, struct members
	MethodsReadable bool // Can access methods
}

// Constants for FromDistinct field (defined in search package)
const (
	NotFromDistinct   = 0
	InlineDistinct    = 1
	NonInlineDistinct = 2
)

func NewAccessContext() AccessContext {
	return AccessContext{
		FromDistinct:    NotFromDistinct,
		MembersReadable: true,
		MethodsReadable: true,
	}
}

// AfterResolving returns a new context after resolving a symbol to its type
func (ctx AccessContext) AfterResolving(from, to symbols.Indexable) AccessContext {
	newCtx := ctx

	// When resolving to a type, members become readable
	// When resolving to a non-type (shouldn't happen, but be safe), members are not readable
	newCtx.MembersReadable = isTypeSymbol(to)

	// Handle distinct type resolution
	if distinct, ok := from.(*symbols.Distinct); ok {
		if distinct.IsInline() {
			newCtx.FromDistinct = InlineDistinct
			// Methods on inline distincts only accessible on instances, not on the type itself
			wasInstance := !isTypeSymbol(from)
			newCtx.MethodsReadable = ctx.MethodsReadable && wasInstance
		} else {
			newCtx.FromDistinct = NonInlineDistinct
			newCtx.MethodsReadable = false
		}
	}

	return newCtx
}

// AfterFindingMember returns a new context after successfully finding a member
func (ctx AccessContext) AfterFindingMember(member symbols.Indexable) AccessContext {
	newCtx := ctx
	// Members become unreadable only for non-type members (like enum variants)
	// StructMembers, Variables, etc. need to resolve to their types first
	newCtx.MembersReadable = !isTypeSymbol(member)
	newCtx.FromDistinct = NotFromDistinct
	return newCtx
}

func isTypeSymbol(s symbols.Indexable) bool {
	switch s.(type) {
	case *symbols.Struct, *symbols.Enum, *symbols.Fault, *symbols.Distinct:
		return true
	case *symbols.Def:
		return s.(*symbols.Def).ResolvesToType()
	default:
		return false
	}
}
