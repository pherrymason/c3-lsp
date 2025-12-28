# Module Storage Refactor Design

## Current Problems

1. **Redundant storage**: Each symbol stored 3-4 times
   - Module.Structs["X"], Module.Enums["Y"], etc. (typed maps)
   - BaseIndexable.children[] (generic array)
   - BaseIndexable.childrenNames[] (string array)
   - IndexStore (global trie)

2. **Inconsistent access patterns**:
   - Structs/Enums: O(1) map lookup
   - Functions: O(n) array scan
   - Some code uses typed maps, other uses children[]

3. **Inefficient iteration**: 7+ loops to index all symbols

## New Design

### Core Structure

```go
type Module struct {
    // UNIFIED storage - single source of truth
    symbols map[string]Indexable

    // Keep only essential non-symbol data
    Imports           []string
    GenericParameters map[string]*GenericParameter

    BaseIndexable  // Will be simplified - no children storage
}
```

### Key Methods

```go
// Add any symbol type consistently
func (m *Module) AddSymbol(symbol Indexable) *Module {
    m.symbols[symbol.GetName()] = symbol
    return m
}

// Get symbol by name (O(1))
func (m *Module) GetSymbol(name string) (Indexable, bool) {
    sym, exists := m.symbols[name]
    return sym, exists
}

// Get all symbols (for iteration)
func (m *Module) AllSymbols() []Indexable {
    result := make([]Indexable, 0, len(m.symbols))
    for _, sym := range m.symbols {
        result = append(result, sym)
    }
    return result
}

// Get symbols by type (if needed)
func (m *Module) GetSymbolsByType(symbolType protocol.CompletionItemKind) []Indexable {
    var result []Indexable
    for _, sym := range m.symbols {
        if sym.GetKind() == symbolType {
            result = append(result, sym)
        }
    }
    return result
}
```

### Migration Strategy - Backward Compatibility Phase

To avoid breaking all code at once, we'll provide temporary compatibility methods:

```go
// Deprecated: Use GetSymbolsByType or type assertion
func (m *Module) GetStructs() map[string]*Struct {
    structs := make(map[string]*Struct)
    for name, sym := range m.symbols {
        if s, ok := sym.(*Struct); ok {
            structs[name] = s
        }
    }
    return structs
}

// Deprecated: Use GetSymbolsByType or type assertion
func (m *Module) GetEnums() map[string]*Enum {
    enums := make(map[string]*Enum)
    for name, sym := range m.symbols {
        if e, ok := sym.(*Enum); ok {
            enums[name] = e
        }
    }
    return enums
}

// Similar for Functions, Variables, etc.
```

### BaseIndexable Simplification

```go
type BaseIndexable struct {
    name          string
    moduleString  string
    module        ModulePath
    documentURI   string
    hasSourceCode bool
    idRange       Range
    docRange      Range
    Kind          protocol.CompletionItemKind
    docComment    *DocComment
    attributes    []string

    // REMOVED: children []Indexable
    // REMOVED: childrenNames []string
    // REMOVED: nestedScopes []Indexable
}

// Children methods removed from Indexable interface
// Each type implements its own specific methods:
// - Struct has GetMembers() []StructMember
// - Function has GetVariables() []Variable
// - Module has AllSymbols() []Indexable
```

## Migration Steps

### Phase 1: Add New Structure (Parallel)
1. Add `symbols map[string]Indexable` to Module
2. Modify Add* methods to populate BOTH old and new structures
3. Add compatibility methods (GetStructs(), GetEnums(), etc.)
4. Tests should still pass

### Phase 2: Migrate Usage
1. Update project_state.indexParsedSymbols to use AllSymbols()
2. Update search code to use GetSymbol() or AllSymbols()
3. Update tests to use new methods
4. Verify everything works

### Phase 3: Remove Old Structure
1. Remove typed maps (Structs, Enums, etc.) from Module
2. Remove children/nestedScopes from BaseIndexable
3. Remove compatibility methods
4. Clean up interface definitions

## Expected Benefits

- **Memory**: ~75% reduction (1 copy instead of 4)
- **Performance**: O(1) lookup for all symbols (not just some)
- **Code simplicity**:
  - indexParsedSymbols: 7 loops → 1 loop
  - Search code: Consistent patterns
- **Maintainability**: One pattern for all symbol types

## Risks & Mitigation

**Risk**: Type safety loss (everything is Indexable)
**Mitigation**:
- Use type assertions where specific type is needed
- Compiler will catch errors at type assertion points
- Add helper methods for common type checks

**Risk**: Breaking changes in many files
**Mitigation**:
- Phased approach with backward compatibility
- Fix one module at a time
- Keep old tests passing until Phase 3

**Risk**: Performance of type filtering
**Mitigation**:
- Benchmark before/after
- Add caching if needed (unlikely given small symbol counts)
