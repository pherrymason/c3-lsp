package search

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

type builtinSymbolCase struct {
	Module string
	Name   string
	Kind   string
}

func TestBuiltinSymbols_accessible_with_import_or_global(t *testing.T) {
	stdRoot := resolveStdRootForBuiltinMatrix(t)
	state := NewTestState()

	loadStdDocsIntoState(t, &state, stdRoot)
	cases := collectBuiltinSymbolCases(t, &state)
	if len(cases) == 0 {
		t.Fatalf("no @builtin symbols discovered under std root: %s", stdRoot)
	}

	search := NewSearchWithoutLog()
	failures := []string{}

	for i, tc := range cases {
		imported := resolveBuiltinCase(t, &state, search, tc, true, i)
		global := resolveBuiltinCase(t, &state, search, tc, false, i)
		if !imported && !global {
			failures = append(failures, fmt.Sprintf("%s [%s] (module=%s)", tc.Name, tc.Kind, tc.Module))
		}
	}

	if len(failures) > 0 {
		t.Fatalf("unresolved @builtin symbols (%d):\n%s", len(failures), strings.Join(failures, "\n"))
	}
}

func resolveStdRootForBuiltinMatrix(t *testing.T) string {
	t.Helper()

	candidates := []string{}
	if env := strings.TrimSpace(os.Getenv("C3LSP_BUILTIN_STD_ROOT")); env != "" {
		candidates = append(candidates, env)
	}

	if _, thisFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../../../.."))
		candidates = append(candidates, filepath.Join(repoRoot, "../c3c/lib/std"))
	}

	candidates = append(candidates, "/Users/f00lg/github/c3/c3c/lib/std")

	for _, candidate := range candidates {
		canonical := fs.GetCanonicalPath(candidate)
		if canonical == "" {
			continue
		}
		info, err := os.Stat(canonical)
		if err == nil && info.IsDir() {
			return canonical
		}
	}

	t.Fatalf("could not resolve std root for builtin matrix; checked: %v", candidates)
	return ""
}

func loadStdDocsIntoState(t *testing.T, state *TestState, stdRoot string) {
	t.Helper()

	files, _, err := fs.ScanForC3WithOptions(stdRoot, fs.ScanOptions{IgnoreDirs: fs.DefaultC3ScanIgnoreDirs()})
	if err != nil {
		t.Fatalf("failed to scan std root: %v", err)
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read std file %s: %v", file, err)
		}
		state.registerDoc(file, string(content))
	}
}

func collectBuiltinSymbolCases(t *testing.T, state *TestState) []builtinSymbolCase {
	t.Helper()

	caseMap := map[string]builtinSymbolCase{}
	snapshot := state.state.Snapshot()
	if snapshot == nil {
		t.Fatalf("project snapshot is nil")
	}

	for _, modulesByDoc := range snapshot.AllUnitModulesView() {
		for _, module := range modulesByDoc.Modules() {
			for _, child := range module.Children() {
				addBuiltinCaseFromSymbol(caseMap, module.GetName(), child)
			}
			for _, nested := range module.NestedScopes() {
				addBuiltinCaseFromSymbol(caseMap, module.GetName(), nested)
			}
			for _, fault := range module.FaultDefs {
				if !hasBuiltinAttribute(fault) {
					continue
				}
				for _, constant := range fault.GetConstants() {
					if constant == nil {
						continue
					}
					key := fmt.Sprintf("%s|%s|fault_constant", constant.GetModuleString(), constant.GetName())
					caseMap[key] = builtinSymbolCase{Module: constant.GetModuleString(), Name: constant.GetName(), Kind: "fault_constant"}
				}
			}
		}
	}

	cases := make([]builtinSymbolCase, 0, len(caseMap))
	for _, tc := range caseMap {
		cases = append(cases, tc)
	}

	sort.Slice(cases, func(i, j int) bool {
		if cases[i].Module != cases[j].Module {
			return cases[i].Module < cases[j].Module
		}
		if cases[i].Kind != cases[j].Kind {
			return cases[i].Kind < cases[j].Kind
		}
		return cases[i].Name < cases[j].Name
	})

	return cases
}

func addBuiltinCaseFromSymbol(caseMap map[string]builtinSymbolCase, moduleName string, symbol symbols.Indexable) {
	if symbol == nil || !hasBuiltinAttribute(symbol) {
		return
	}
	if symbol.GetName() == "" {
		return
	}

	kind := ""
	switch s := symbol.(type) {
	case *symbols.Function:
		kind = "function"
		_ = s
	case *symbols.Variable:
		if s.IsConstant() {
			kind = "const"
		}
	case *symbols.Alias:
		kind = "alias"
	case *symbols.FaultDef:
		kind = "fault"
	default:
		kind = "other"
	}

	if kind == "" {
		return
	}

	key := fmt.Sprintf("%s|%s|%s", moduleName, symbol.GetName(), kind)
	caseMap[key] = builtinSymbolCase{Module: moduleName, Name: symbol.GetName(), Kind: kind}
}

func hasBuiltinAttribute(symbol symbols.Indexable) bool {
	type builtinAttr interface {
		GetAttributes() []string
	}

	attrSource, ok := symbol.(builtinAttr)
	if !ok {
		return false
	}

	for _, attr := range attrSource.GetAttributes() {
		if strings.TrimSpace(attr) == "@builtin" {
			return true
		}
	}

	return false
}

func resolveBuiltinCase(t *testing.T, state *TestState, search Search, tc builtinSymbolCase, imported bool, index int) bool {
	t.Helper()

	usage, cursorInUsage := builtinUsage(tc)
	if usage == "" || cursorInUsage < 0 {
		t.Fatalf("unsupported @builtin usage for case: %+v", tc)
	}

	importLine := ""
	if imported {
		importLine = "import " + tc.Module + ";\n"
	}

	body := "module app;\n" + importLine + "fn void main() {\n\t" + usage + "\n}"
	docID := fmt.Sprintf("builtin_access_%d_%t.c3", index, imported)
	state.registerDoc(docID, body)

	line := uint(2)
	if imported {
		line = 3
	}
	position := symbols.NewPosition(line, uint(1+cursorInUsage))
	result := search.FindSymbolDeclarationInWorkspace(docID, position, state.state)
	return result.IsSome()
}

func builtinUsage(tc builtinSymbolCase) (string, int) {
	name := tc.Name
	if name == "" {
		return "", -1
	}

	cursorOffset := strings.LastIndex(name, ".") + 1
	if cursorOffset < 0 {
		cursorOffset = 0
	}

	switch tc.Kind {
	case "fault_constant":
		usage := "void? e = " + name + "?;"
		return usage, strings.Index(usage, name) + cursorOffset
	case "function":
		if strings.HasPrefix(name, "@") {
			usage := name + "();"
			return usage, strings.Index(usage, name) + 1
		}
		usage := "var probe = " + name + ";"
		return usage, strings.Index(usage, name) + cursorOffset
	case "const", "alias", "fault", "other":
		usage := "var probe = " + name + ";"
		return usage, strings.Index(usage, name) + cursorOffset
	default:
		return "", -1
	}
}
