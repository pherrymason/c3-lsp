package symbols

import "sort"

// modulePathRebuilder is implemented by every symbol embedding BaseIndexable.
type modulePathRebuilder interface {
	ReconstructModulePath()
}

// RebuildAfterDeserialization restores the derived state that is not persisted
// in the stdlib disk cache (module paths, children and nested scopes), using the
// typed collections that are serialized. Without it, searches relying on
// Children()/NestedScopes() find nothing in modules loaded from the cache.
func (m *Module) RebuildAfterDeserialization() {
	m.ReconstructModulePath()
	m.resetDerivedIndexes()

	var children []Indexable
	for _, v := range m.Variables {
		children = append(children, v)
	}
	for _, e := range m.Enums {
		children = append(children, e)
	}
	for _, f := range m.Faults {
		children = append(children, f)
	}
	for _, s := range m.Structs {
		children = append(children, s)
	}
	for _, b := range m.Bitstructs {
		children = append(children, b)
	}
	for _, d := range m.Defs {
		children = append(children, d)
	}
	for _, d := range m.Distincts {
		children = append(children, d)
	}
	for _, i := range m.Interfaces {
		children = append(children, i)
	}
	for _, g := range m.GenericParameters {
		children = append(children, g)
	}

	// The parser inserts children in source order; maps lose it.
	sortBySourcePosition(children)
	for _, child := range children {
		if r, ok := child.(modulePathRebuilder); ok {
			r.ReconstructModulePath()
		}
		m.Insert(child)
	}

	for _, fn := range m.ChildrenFunctions {
		fn.rebuildAfterDeserialization()
		m.InsertNestedScope(fn)
	}
}

func (f *Function) rebuildAfterDeserialization() {
	f.ReconstructModulePath()
	f.resetDerivedIndexes()

	variables := make([]Indexable, 0, len(f.Variables))
	for _, v := range f.Variables {
		v.ReconstructModulePath()
		variables = append(variables, v)
	}
	sortBySourcePosition(variables)
	for _, v := range variables {
		f.Insert(v)
	}
}

func (b *BaseIndexable) resetDerivedIndexes() {
	b.Children_ = nil
	b.ChildrenNames_ = nil
	b.NestedScopes_ = nil
}

func sortBySourcePosition(symbols []Indexable) {
	sort.SliceStable(symbols, func(i, j int) bool {
		a, b := symbols[i].GetDocumentRange().Start, symbols[j].GetDocumentRange().Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Character < b.Character
	})
}
