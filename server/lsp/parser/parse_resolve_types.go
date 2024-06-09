package parser

// parsedModules is justParsedModules
/*
func (p *Parser) resolveTypes(parsedModules *symbols_table.UnitModules) {
	for moduleName, typesContext := range p.pendingToResolve.typesByModule {
		for x := len(typesContext) - 1; x >= 0; x-- {
			typeContext := typesContext[x]

			if len(typeContext.contextModule.Imports) > 0 {
				// Check if any of the imported modules in the Type context is present in parsedModules
				for _, imported := range typeContext.contextModule.Imports {
					mpath := symbols.NewModulePathFromString(imported)
					if !parsedModules.HasExplicitlyImportedModules(mpath) {
						continue
					}

					moduleOption := findTypeInModule(typeContext.vType, parsedModules)
					if moduleOption.IsSome() {
						// Found in same file! Fix it
						typeContext.vType.SetModule(moduleOption.Get())
						p.pendingToResolve.SolveType(moduleName, x)
					}
				}
				// Not found! Keep it registered as pending
			} else {
				moduleOption := findTypeInModule(typeContext.vType, parsedModules)
				if moduleOption.IsSome() {
					// Found in same file! Fix it
					typeContext.vType.SetModule(moduleOption.Get())
					p.pendingToResolve.SolveType(moduleName, x)
				}
			}
		}
	}

	resolveStructSubtypes(parsedModules, p.pendingToResolve.subtyptingToResolve)
}

func findTypeInModule(vType *symbols.Type, parsedModules *symbols_table.UnitModules) option.Option[string] {
	for _, module := range parsedModules.Modules() {
		for _, x := range module.Children() {
			if x.GetName() == vType.GetName() {
				// Found!
				return option.Some(module.GetName())
			}
		}
	}

	return option.None[string]()
}

// Resolves inline sub structs
func resolveStructSubtypes(parsedModules *symbols_table.UnitModules, subtyping []StructWithSubtyping) {
	for _, struktWithSubtyping := range subtyping {
		for _, inlinedMemberName := range struktWithSubtyping.members {

			for _, module := range parsedModules.Modules() {
				// Search
				for _, strukt := range module.Structs {
					if strukt.GetName() == inlinedMemberName.GetName() {
						struktWithSubtyping.strukt.InheritMembersFrom(inlinedMemberName.GetName(), strukt)
					}
				}
			}
		}
	}
}
*/
