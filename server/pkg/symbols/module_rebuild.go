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
		rebuildSymbolAfterDeserialization(child)
		m.Insert(child)
	}

	for _, fn := range m.ChildrenFunctions {
		fn.rebuildAfterDeserialization()
		m.InsertNestedScope(fn)
	}
}

// rebuildSymbolAfterDeserialization mirrors the Insert/InsertNestedScope calls
// done by each symbol constructor, so the rebuilt tree matches a fresh parse.
func rebuildSymbolAfterDeserialization(symbol Indexable) {
	if r, ok := symbol.(modulePathRebuilder); ok {
		r.ReconstructModulePath()
	}

	switch s := symbol.(type) {
	case *Enum:
		s.resetDerivedIndexes()
		for _, enumerator := range s.enumerators {
			rebuildSymbolAfterDeserialization(enumerator)
			s.Insert(enumerator)
		}
	case *Enumerator:
		s.resetDerivedIndexes()
		for i := range s.AssociatedValues {
			s.AssociatedValues[i].ReconstructModulePath()
			s.InsertNestedScope(&s.AssociatedValues[i])
		}
	case *Struct:
		s.resetDerivedIndexes()
		for _, member := range s.members {
			rebuildSymbolAfterDeserialization(member)
			s.Insert(member)
		}
	case *StructMember:
		s.resetDerivedIndexes()
		if s.subStruct.IsSome() {
			subStruct := s.subStruct.Get()
			rebuildSymbolAfterDeserialization(subStruct)
			s.Insert(subStruct)
		}
	case *Bitstruct:
		s.resetDerivedIndexes()
		for _, member := range s.members {
			rebuildSymbolAfterDeserialization(member)
			s.Insert(member)
		}
	case *Fault:
		s.resetDerivedIndexes()
		for _, constant := range s.constants {
			rebuildSymbolAfterDeserialization(constant)
			s.Insert(constant)
		}
	case *Interface:
		s.resetDerivedIndexes()
		methods := make([]Indexable, 0, len(s.methods))
		for _, method := range s.methods {
			methods = append(methods, method)
		}
		sortBySourcePosition(methods)
		for _, method := range methods {
			method.(*Function).rebuildAfterDeserialization()
			s.Insert(method)
		}
	case *Function:
		s.rebuildAfterDeserialization()
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
