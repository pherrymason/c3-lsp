package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentHover_resolves_symbol_from_recursive_root_import(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
struct BGOptions {
	int x;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_dep_test.c3")
	depDoc := document.NewDocumentFromDocURI(depURI, depSource, 1)
	state.RefreshDocumentIdentifiers(depDoc, &prs)

	appSource := `module app;
import bindgen;

fn void main() {
	BGOptions opts = {};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_app_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "BGOptions") + len("BGOpt")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for BGOptions through import bindgen")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content")
	}
	if !strings.Contains(content.Value, "BGOptions") {
		t.Fatalf("expected hover to include BGOptions, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_BGOptions_in_anonymous_module_via_bindgen_import(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
struct BGOptions {
	int x;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_dep_anon_test.c3")
	depDoc := document.NewDocumentFromDocURI(depURI, depSource, 1)
	state.RefreshDocumentIdentifiers(depDoc, &prs)

	appSource := `import bindgen;

fn void main() {
	BGOptions opts = {};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_anon_app_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "BGOptions") + len("BGOpt")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for BGOptions through import bindgen in anonymous module")
	}
}

func TestTextDocumentHover_resolves_fully_qualified_BGOptions_type(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
struct BGOptions {
	int x;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_dep_qualified_test.c3")
	depDoc := document.NewDocumentFromDocURI(depURI, depSource, 1)
	state.RefreshDocumentIdentifiers(depDoc, &prs)

	appSource := `
fn void main() {
	bindgen::bg::BGOptions opts = {};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_qualified_app_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "BGOptions") + len("BG")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for fully qualified BGOptions type")
	}
}

func TestTextDocumentHover_resolves_unqualified_BGTransCallbacks_from_bindgen_import(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
struct BGTransCallbacks {
	int x;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_dep_callbacks_test.c3")
	depDoc := document.NewDocumentFromDocURI(depURI, depSource, 1)
	state.RefreshDocumentIdentifiers(depDoc, &prs)

	appSource := `
import bindgen;

fn void main() {
	BGTransCallbacks trans_cbs = {};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_import_bindgen_callbacks_app_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	idx := strings.Index(appSource, "BGTransCallbacks") + len("BGTrans")
	pos := byteIndexToLSPPosition(appSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for BGTransCallbacks through import bindgen")
	}
}

func TestTextDocumentHover_resolves_imported_BGOptions_on_first_hover_across_cursor_positions(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	workspaceRoot := fs.GetCanonicalPath(t.TempDir())
	if err := os.WriteFile(filepath.Join(workspaceRoot, "project.json"), []byte(`{
		"dependency-search-paths": ["."],
		"dependencies": ["bindgen"],
		"sources": ["examples/**"]
	}`), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(workspaceRoot, "bindgen.c3l"), 0o755); err != nil {
		t.Fatalf("failed to create bindgen package dir: %v", err)
	}
	depSource := `<*
 Optional settings for translation.
*>
module bindgen::bg;
struct BGOptions {
	int x;
}`
	if err := os.WriteFile(filepath.Join(workspaceRoot, "bindgen.c3l", "bindgen.c3i"), []byte(depSource), 0o644); err != nil {
		t.Fatalf("failed to write dependency source: %v", err)
	}

	appSource := `import bindgen;

fn void main() {
	BGOptions opts = {};
}`
	appPath := filepath.Join(workspaceRoot, "examples", "glfw.c3")
	if err := os.MkdirAll(filepath.Dir(appPath), 0o755); err != nil {
		t.Fatalf("failed to create examples dir: %v", err)
	}
	if err := os.WriteFile(appPath, []byte(appSource), 0o644); err != nil {
		t.Fatalf("failed to write app source: %v", err)
	}

	appURI := protocol.DocumentUri(fs.ConvertPathToURI(appPath, option.None[string]()))
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)

	srv.state.SetProjectRootURI(workspaceRoot)
	srv.configureProjectForRoot(workspaceRoot)
	srv.idx.mu.Lock()
	srv.ensureIndexingStateMapsLocked()
	srv.setRootState(workspaceRoot, rootStateIndexed)
	srv.idx.mu.Unlock()

	for _, charOffset := range []int{0, 1, 2, 7, 8} {
		idx := strings.Index(appSource, "BGOptions") + charOffset
		pos := byteIndexToLSPPosition(appSource, idx)

		hover, err := srv.textDocumentHoverWithTrace(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     pos,
		}}, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected hover error for offset %d: %v", charOffset, err)
		}
		if hover == nil {
			t.Fatalf("expected hover for offset %d", charOffset)
		}

		content, ok := hover.Contents.(protocol.MarkupContent)
		if !ok {
			t.Fatalf("expected markup content for offset %d", charOffset)
		}
		if !strings.Contains(content.Value, "In module **[bindgen::bg]**") {
			t.Fatalf("expected resolved imported hover for offset %d, got: %s", charOffset, content.Value)
		}
		if strings.TrimSpace(content.Value) == "```c3\nBGOptions\n```" {
			t.Fatalf("expected non-synthetic hover for offset %d, got: %s", charOffset, content.Value)
		}
	}
}

func TestTextDocumentHover_resolves_chained_string_method_in_inline_lambda(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	asciiSource := `module std::ascii;

fn String String.strip(self, String needle) {
	return self;
}

fn String String.camel_to_snake(self, Allocator alloc) {
	return self;
}`
	asciiURI := protocol.DocumentUri("file:///tmp/hover_import_ascii_methods_test.c3")
	asciiDoc := document.NewDocumentFromDocURI(asciiURI, asciiSource, 1)
	state.RefreshDocumentIdentifiers(asciiDoc, &prs)
	asciiDocID := utils.NormalizePath(asciiURI)
	asciiUnit := state.GetUnitModulesByDoc(asciiDocID)
	if asciiUnit == nil {
		t.Fatalf("expected std::ascii unit modules")
	}
	asciiModule := asciiUnit.Get("std::ascii")
	if asciiModule == nil {
		t.Fatalf("expected std::ascii module")
	}
	hasStrip := false
	hasCamel := false
	for _, fn := range asciiModule.ChildrenFunctions {
		if fn.GetName() == "String.strip" {
			hasStrip = true
		}
		if fn.GetName() == "String.camel_to_snake" {
			hasCamel = true
		}
	}
	if !hasStrip || !hasCamel {
		t.Fatalf("expected std::ascii methods to be parsed, hasStrip=%v hasCamel=%v", hasStrip, hasCamel)
	}

	appSource := `import std::ascii;

fn void main() {
	var x = fn String(String str, Allocator alloc) =>
		str.strip("vk").camel_to_snake(alloc);
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_import_ascii_lambda_app_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)
	appDocID := utils.NormalizePath(appURI)
	appUnit := state.GetUnitModulesByDoc(appDocID)
	if appUnit == nil {
		t.Fatalf("expected app unit modules")
	}
	appModule := appUnit.Get(symbols.NormalizeModuleName(appDocID))
	if appModule == nil {
		t.Fatalf("expected anonymous app module")
	}
	foundStr := false
	strType := ""
	for _, fn := range appModule.ChildrenFunctions {
		if fn.GetName() != "main" {
			continue
		}
		if variable, ok := fn.Variables["str"]; ok {
			foundStr = true
			strType = variable.GetType().GetName()
		}
	}
	if !foundStr {
		t.Fatalf("expected lambda parameter str to be indexed in app main")
	}
	if strType != "String" {
		t.Fatalf("expected lambda parameter str type to be String, got %q", strType)
	}
	hasAsciiImport := false
	for _, imp := range appModule.Imports {
		if imp == "std::ascii" {
			hasAsciiImport = true
			break
		}
	}
	if !hasAsciiImport {
		t.Fatalf("expected std::ascii import on anonymous app module")
	}

	idx := strings.LastIndex(appSource, "camel_to_snake") + len("camel")
	pos := byteIndexToLSPPosition(appSource, idx)
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for chained string method inside inline lambda")
	}
}

func TestTextDocumentHover_resolves_qualified_enumerator_in_misc_style_code(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	clangSource := `module clang;
enum CXTypeKind {
	TYPE_POINTER,
}`
	clangURI := protocol.DocumentUri("file:///tmp/hover_misc_clang_module_test.c3")
	clangDoc := document.NewDocumentFromDocURI(clangURI, clangSource, 1)
	state.RefreshDocumentIdentifiers(clangDoc, &prs)

	miscSource := `module bgimpl::misc;
import clang;

fn bool isTypePFN(CXType type) {
	return type.kind == clang::TYPE_POINTER;
}`
	miscURI := protocol.DocumentUri("file:///tmp/hover_misc_style_app_test.c3")
	miscDoc := document.NewDocumentFromDocURI(miscURI, miscSource, 1)
	state.RefreshDocumentIdentifiers(miscDoc, &prs)

	idx := strings.Index(miscSource, "TYPE_POINTER") + len("TYPE")
	pos := byteIndexToLSPPosition(miscSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: miscURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for qualified enumerator in misc-style code")
	}
}

func TestTextDocumentHover_resolves_parent_module_symbol_in_data_methods_style_code(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	dataSource := `module bgimpl;
enum CFieldKind : char {
	NORMAL,
}`
	dataURI := protocol.DocumentUri("file:///tmp/hover_data_module_test.c3")
	dataDoc := document.NewDocumentFromDocURI(dataURI, dataSource, 1)
	state.RefreshDocumentIdentifiers(dataDoc, &prs)

	methodsSource := `module bgimpl::data_methods;

macro CFields.@foreach(
	&self;
	@body(CFieldKind* kind, usz index))
{
	usz[CFieldKind.values.len] counters;
}`
	methodsURI := protocol.DocumentUri("file:///tmp/hover_data_methods_style_app_test.c3")
	methodsDoc := document.NewDocumentFromDocURI(methodsURI, methodsSource, 1)
	state.RefreshDocumentIdentifiers(methodsDoc, &prs)

	idx := strings.LastIndex(methodsSource, "CFieldKind") + len("CField")
	pos := byteIndexToLSPPosition(methodsSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: methodsURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for parent-module symbol in data-methods-style code")
	}
}

func TestTextDocumentHover_resolves_data_methods_real_positions(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	dataSource := `module bgimpl;
enum CFieldKind : char { NORMAL }
struct CFields { CFieldKind[] kinds; }`
	dataURI := protocol.DocumentUri("file:///tmp/hover_data_methods_real_data.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(dataURI, dataSource, 1), &prs)

	methodsSource := `module bgimpl::data_methods;

macro void CFields.init(
	&self,
	Allocator alloc)
{
}

macro CFields.@foreach(
	&self;
	@body(CFieldKind* kind, usz index))
{
	usz[CFieldKind.values.len] counters;
}`
	methodsURI := protocol.DocumentUri("file:///tmp/hover_data_methods_real_methods.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(methodsURI, methodsSource, 1), &prs)

	initIdx := strings.Index(methodsSource, "CFields.init") + len("CFie")
	initHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: methodsURI},
		Position:     byteIndexToLSPPosition(methodsSource, initIdx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on CFields.init: %v", err)
	}
	if initHover == nil {
		t.Fatalf("expected hover response on CFields in CFields.init")
	}

	kindIdx := strings.LastIndex(methodsSource, "CFieldKind.values.len") + len("CField")
	kindHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: methodsURI},
		Position:     byteIndexToLSPPosition(methodsSource, kindIdx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on CFieldKind.values: %v", err)
	}
	if kindHover == nil {
		t.Fatalf("expected hover response on CFieldKind in access path")
	}
}

func TestTextDocumentHover_infers_untyped_lambda_parameter_type_from_callback_context(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
alias BGTransFn = fn String(String name, Allocator alloc);
struct BGTransCallbacks { BGTransFn func_macro; }`
	depURI := protocol.DocumentUri("file:///tmp/hover_mlir_dep_bindgen_bg.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(depURI, depSource, 1), &prs)

	stdSource := `module std::string;
fn bool String.contains(self, String value) { return true; }`
	stdURI := protocol.DocumentUri("file:///tmp/hover_mlir_dep_std_string.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(stdURI, stdSource, 1), &prs)

	appSource := `import bindgen;

fn void main() {
	BGTransCallbacks transfns = {
		.func_macro = fn (name, alloc) {
			return name.contains("DEFINE_C_API_STRUCT") ? "" : name;
		},
	};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_mlir_untyped_lambda_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "name.contains") + len("name.cont")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on untyped lambda method: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for method on untyped lambda parameter inferred from callback context")
	}

	nameIdx := strings.Index(appSource, "name.contains") + len("na")
	nameHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, nameIdx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on untyped lambda parameter: %v", err)
	}
	if nameHover == nil {
		t.Fatalf("expected hover response for untyped lambda parameter name")
	}
	content, ok := nameHover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for lambda parameter")
	}
	if !strings.Contains(content.Value, "String name") {
		t.Fatalf("expected inferred lambda parameter type in hover, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_struct_member_designator_in_initializer(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
alias BGTransFn = fn String(String name, Allocator alloc);
struct BGTransCallbacks {
	BGTransFn constant;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_mlir_dep_bindgen_designator.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(depURI, depSource, 1), &prs)

	appSource := `import bindgen;

fn void main() {
	BGTransCallbacks transfns = {
		.constant = fn (name, alloc) {
			return "";
		},
	};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_mlir_designator_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, ".constant") + len(".cons")
	pos := byteIndexToLSPPosition(appSource, idx)
	fallback := srv.resolveDesignatedStructMemberHoverFallback(utils.NormalizePath(appURI), symbols.NewPositionFromLSPPosition(pos))
	if fallback.IsNone() {
		t.Fatalf("expected designated struct member fallback to resolve .constant")
	}

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on struct member designator: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for struct member designator")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for designator member")
	}
	if !strings.Contains(content.Value, "constant") {
		t.Fatalf("expected designator member hover to include member name, got: %s", content.Value)
	}
}

func TestTextDocumentHover_prioritizes_designator_member_over_unrelated_function_name(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
alias BGTransFn = fn String(String name, Allocator alloc);
struct BGTransCallbacks {
	BGTransFn func;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_mlir_dep_bindgen_func_designator.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(depURI, depSource, 1), &prs)

	unrelatedSource := `module bgimpl::wter;
fn usz? func(OutStream out) { return out.err(); }`
	unrelatedURI := protocol.DocumentUri("file:///tmp/hover_mlir_unrelated_func_name.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(unrelatedURI, unrelatedSource, 1), &prs)

	appSource := `import bindgen;

fn void main() {
	BGTransCallbacks transfns = {
		.func = fn (name, alloc) {
			return name;
		},
	};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_mlir_designator_func_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, ".func") + len(".fu")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on .func designator: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for .func designator")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for .func designator")
	}
	if !strings.Contains(content.Value, "BGTransFn func") {
		t.Fatalf("expected .func member hover, got: %s", content.Value)
	}
	if strings.Contains(content.Value, "fn usz? func(") {
		t.Fatalf("expected to avoid unrelated global func hover, got: %s", content.Value)
	}

	dotIdx := strings.Index(appSource, ".func")
	dotHover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, dotIdx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on .func dot position: %v", err)
	}
	if dotHover == nil {
		t.Fatalf("expected hover response on .func dot position")
	}
}

func TestTextDocumentHover_on_lambda_fn_keyword_shows_callback_type_info(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bindgen::bg;
alias BGTransFn = fn String(String name, Allocator alloc);
struct BGTransCallbacks {
	BGTransFn constant;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_mlir_dep_bindgen_fn_keyword.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(depURI, depSource, 1), &prs)

	appSource := `import bindgen;

fn void main() {
	BGTransCallbacks transfns = {
		.constant = fn (name, alloc) {
			return "";
		},
	};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_mlir_fn_keyword_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "fn (name") + 1
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on fn keyword: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response on lambda fn keyword")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for fn keyword")
	}
	if !strings.Contains(content.Value, "def BGTransFn = fn String(String name, Allocator alloc)") {
		t.Fatalf("expected fn hover to show callback alias info, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_qualified_module_function_call_symbol(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	clangSource := `module clang;
fn int createIndex(int a, int b) {
	return 0;
}`
	clangURI := protocol.DocumentUri("file:///tmp/hover_qualified_clang_module_test.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(clangURI, clangSource, 1), &prs)

	bgSource := `module bgimpl;
import clang;

fn void main() {
	int index = clang::createIndex(0, 0);
}`
	bgURI := protocol.DocumentUri("file:///tmp/hover_qualified_clang_app_test.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(bgURI, bgSource, 1), &prs)

	idx := strings.Index(bgSource, "createIndex") + len("create")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: bgURI},
		Position:     byteIndexToLSPPosition(bgSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on qualified module function: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response on clang::createIndex")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content")
	}
	if !strings.Contains(content.Value, "createIndex") {
		t.Fatalf("expected qualified function hover to include createIndex, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_short_module_token_in_qualified_call(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	bgstrSource := `module bindgen::bgstr;
fn String camel_to_snake(String s, Allocator alloc) { return s; }`
	bgstrURI := protocol.DocumentUri("file:///tmp/hover_module_token_bgstr_dep.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(bgstrURI, bgstrSource, 1), &prs)

	appSource := `import bindgen;
fn void main() {
	var x = &bgstr::camel_to_snake;
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_module_token_bgstr_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "bgstr::") + len("bg")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for short module token: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for short module token bgstr")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for short module token")
	}
	if !strings.Contains(content.Value, "module ") {
		t.Fatalf("expected module-prefixed hover for module token, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_short_module_token_err(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	errSource := `module bgimpl::err;
fn void info(bool no_verbose, String fmt, String v) {}`
	errURI := protocol.DocumentUri("file:///tmp/hover_module_token_err_dep.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(errURI, errSource, 1), &prs)

	appSource := `module bgimpl;
import bgimpl::err;
fn void main() {
	err::info(false, "x", "y");
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_module_token_err_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "err::") + len("er")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for err module token: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for short module token err")
	}
}

func TestTextDocumentHover_resolves_short_module_token_trans(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	transSource := `module bgimpl::trans;
fn String tokensUnderCursor(Allocator alloc, CXCursor cursor, BGTransCallbacks* fns) { return ""; }`
	transURI := protocol.DocumentUri("file:///tmp/hover_module_token_trans_dep.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(transURI, transSource, 1), &prs)

	appSource := `module bgimpl::vtor;
import bgimpl;
fn void main() {
	String v = trans::tokensUnderCursor(alloc, cursor, &g.trans_fns);
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_module_token_trans_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "trans::") + len("tr")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for trans module token: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for short module token trans")
	}
}

func TestTextDocumentHover_resolves_short_module_token_wter_from_writers_file(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	wterSource := `module bgimpl::wter;
fn void constDecl(usz out, WriteState* state, String enumName, String? transCursor, String val, WriteAttrs attrs) {}`
	wterURI := protocol.DocumentUri("file:///tmp/hover_module_token_writers_dep.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(wterURI, wterSource, 1), &prs)

	appSource := `module bgimpl::vtor;
import bgimpl;
fn void run(usz out, WriteState* ws, String enumName, String? transCursor, String val, WriteAttrs attrs) {
	wter::constDecl(out, ws, enumName, transCursor, val, attrs);
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_module_token_writers_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "wter::") + len("wt")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for wter module token: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for short module token wter")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for wter module token")
	}
	if !strings.Contains(content.Value, "module bgimpl::wter") {
		t.Fatalf("expected module-prefixed hover for wter token, got: %s", content.Value)
	}
}

func TestTextDocumentHover_module_token_force_preload_ignores_stale_preload_done_cache(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	visitorsPath := filepath.Join(srcDir, "visitors.c3")
	visitorsSource := `module bgimpl::vtor;
import bgimpl;
fn void run(usz out, WriteState* ws, String enumName, String? transCursor, String val, WriteAttrs attrs) {
	wter::constDecl(out, ws, enumName, transCursor, val, attrs);
}`
	if err := os.WriteFile(visitorsPath, []byte(visitorsSource), 0o644); err != nil {
		t.Fatalf("failed to write visitors file: %v", err)
	}

	writersPath := filepath.Join(srcDir, "writers.c3")
	writersSource := `module bgimpl::wter;
fn void constDecl(usz out, WriteState* state, String enumName, String? transCursor, String val, WriteAttrs attrs) {}`
	if err := os.WriteFile(writersPath, []byte(writersSource), 0o644); err != nil {
		t.Fatalf("failed to write writers file: %v", err)
	}

	visitorsURI := protocol.DocumentUri(fs.ConvertPathToURI(visitorsPath, option.None[string]()))
	visitorsDoc := document.NewDocumentFromDocURI(visitorsURI, visitorsSource, 1)
	state.RefreshDocumentIdentifiers(visitorsDoc, &prs)

	root := fs.GetCanonicalPath(tmpDir)
	staleKey := root + "|bgimpl|recurse"
	srv := &Server{
		state:             &state,
		parser:            &prs,
		search:            &searchImpl,
		rootCache:         projectRootCacheState{cache: map[string]string{}},
		importPreloadDone: map[string]struct{}{staleKey: {}},
	}

	idx := strings.Index(visitorsSource, "wter::") + len("wt")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: visitorsURI},
		Position:     byteIndexToLSPPosition(visitorsSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for wter module token with stale preload cache: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for wter module token after forced preload")
	}
}

func TestTextDocumentHover_qualified_symbol_token_loads_module_by_declaration_scan(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	visitorsPath := filepath.Join(srcDir, "visitors.c3")
	visitorsSource := `module bgimpl::vtor;
import bgimpl;
fn void run(GlobalVisitData* g, CXCursor cursor) {
	ttor::unionDecl(g, cursor);
}`
	if err := os.WriteFile(visitorsPath, []byte(visitorsSource), 0o644); err != nil {
		t.Fatalf("failed to write visitors file: %v", err)
	}

	translatorsPath := filepath.Join(srcDir, "translators.c3")
	translatorsSource := `module bgimpl::ttor;
fn CXChildVisitResult unionDecl(GlobalVisitData* vd, CXCursor cursor) { return 0; }`
	if err := os.WriteFile(translatorsPath, []byte(translatorsSource), 0o644); err != nil {
		t.Fatalf("failed to write translators file: %v", err)
	}

	visitorsURI := protocol.DocumentUri(fs.ConvertPathToURI(visitorsPath, option.None[string]()))
	visitorsDoc := document.NewDocumentFromDocURI(visitorsURI, visitorsSource, 1)
	state.RefreshDocumentIdentifiers(visitorsDoc, &prs)

	srv := &Server{
		state:             &state,
		parser:            &prs,
		search:            &searchImpl,
		rootCache:         projectRootCacheState{cache: map[string]string{}},
		importPreloadDone: map[string]struct{}{},
	}

	idx := strings.Index(visitorsSource, "unionDecl") + len("union")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: visitorsURI},
		Position:     byteIndexToLSPPosition(visitorsSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for ttor::unionDecl symbol token: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for ttor::unionDecl symbol token")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for ttor::unionDecl symbol token")
	}
	if !strings.Contains(content.Value, "unionDecl") {
		t.Fatalf("expected unionDecl hover content, got: %s", content.Value)
	}
}

func TestTextDocumentHover_resolves_designator_member_in_typed_variable_assignment_initializer(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bgimpl;
struct WriteAttrs {
	String if_condition;
	bool private;
}`
	depURI := protocol.DocumentUri("file:///tmp/hover_write_attrs_dep.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(depURI, depSource, 1), &prs)

	appSource := `module bgimpl::data_methods @test;

fn void test_WriteAttrs_to_format() {
	WriteAttrs attrs;
	attrs = {
		.if_condition = "true",
	};
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_write_attrs_test_initializer.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, ".if_condition") + len(".if_con")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error on designator member in assignment initializer: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for .if_condition in assignment initializer")
	}
}

func TestTextDocumentHover_resolves_module_tokens_in_visitors_style_calls(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	clangSource := `module clang;
fn void disposeString(CXString s) {}
const int CURSOR_FUNCTION_DECL = 1;`
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(protocol.DocumentUri("file:///tmp/hover_visitors_clang_dep.c3"), clangSource, 1), &prs)

	miscSource := `module bgimpl::misc;
fn String convStr(CXString s) { return ""; }`
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(protocol.DocumentUri("file:///tmp/hover_visitors_misc_dep.c3"), miscSource, 1), &prs)

	ttorSource := `module bgimpl::ttor;
fn void func(void* vd, CXCursor cursor) {}`
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(protocol.DocumentUri("file:///tmp/hover_visitors_ttor_dep.c3"), ttorSource, 1), &prs)

	visitorsSource := `module bgimpl::vtor;
import bgimpl, std::io, clang;

fn void run(CXString file_spell, void* vd, CXCursor cursor) {
	defer clang::disposeString(file_spell);
	String file_str = misc::convStr(file_spell);
	case clang::CURSOR_FUNCTION_DECL: ttor::func(vd, cursor);
}`
	visitorsURI := protocol.DocumentUri("file:///tmp/hover_visitors_style_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(visitorsURI, visitorsSource, 1), &prs)

	for _, tc := range []struct {
		needle string
		offset int
		label  string
	}{
		{needle: "clang::disposeString", offset: len("cl"), label: "clang module token"},
		{needle: "misc::convStr", offset: len("mi"), label: "misc module token"},
		{needle: "ttor::func", offset: len("tt"), label: "ttor module token"},
	} {
		idx := strings.Index(visitorsSource, tc.needle) + tc.offset
		hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: visitorsURI},
			Position:     byteIndexToLSPPosition(visitorsSource, idx),
		}})
		if err != nil {
			t.Fatalf("unexpected hover error for %s: %v", tc.label, err)
		}
		if hover == nil {
			t.Fatalf("expected hover response for %s", tc.label)
		}
	}
}

func TestTextDocumentHover_resolves_type_token_in_visitors_style_cast(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	depSource := `module bgimpl;
struct ConstVisitData { int x; }`
	depURI := protocol.DocumentUri("file:///tmp/hover_visitors_constvisitdata_dep.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(depURI, depSource, 1), &prs)

	appSource := `module bgimpl::vtor;
import bgimpl;
fn void run(CXClientData client_data) {
	ConstVisitData* vd = (ConstVisitData*) client_data;
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_visitors_constvisitdata_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "(ConstVisitData*)") + len("(ConstV")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for ConstVisitData cast token: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for ConstVisitData cast token")
	}
}

func TestTextDocumentHover_module_token_check_does_not_resolve_to_unrelated_macro(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	checkSource := `module bgimpl::check;
fn bool apply(String s, String include) { return true; }`
	checkURI := protocol.DocumentUri("file:///tmp/hover_visitors_check_dep.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(checkURI, checkSource, 1), &prs)

	unrelatedSource := `module std::core::test;
macro @check(#condition, String format = "", any*... args) {}`
	unrelatedURI := protocol.DocumentUri("file:///tmp/hover_visitors_unrelated_check_macro.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(unrelatedURI, unrelatedSource, 1), &prs)

	appSource := `module bgimpl::vtor;
import bgimpl;
fn void run(String file_str, String include_file) {
	if (!check::apply(file_str, include_file)) return;
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_visitors_check_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "check::apply") + len("chec")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for check module token: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for check module token")
	}
	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover content for check module token")
	}
	if strings.Contains(content.Value, "macro @check") {
		t.Fatalf("expected module token hover not to resolve unrelated @check macro, got: %s", content.Value)
	}
	if !strings.Contains(content.Value, "module bgimpl::check") {
		t.Fatalf("expected resolved full module hover for check token, got: %s", content.Value)
	}
}

func TestTextDocumentHover_unresolved_module_token_does_not_fallback_to_macro_symbol(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	unrelatedSource := `module std::core::test;
macro @check(#condition, String format = "", any*... args) {}`
	unrelatedURI := protocol.DocumentUri("file:///tmp/hover_visitors_unresolved_check_macro_only.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(unrelatedURI, unrelatedSource, 1), &prs)

	appSource := `module bgimpl::vtor;
import bgimpl;
fn void run(String file_str, String include_file) {
	if (!check::apply(file_str, include_file)) return;
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_visitors_unresolved_check_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "check::apply") + len("chec")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for unresolved check module token: %v", err)
	}
	if hover != nil {
		content, _ := hover.Contents.(protocol.MarkupContent)
		t.Fatalf("expected nil hover for unresolved module token, got: %v", content.Value)
	}
}

func TestTextDocumentHover_resolves_right_symbol_for_short_module_call(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)
	srv := &Server{state: &state, parser: &prs, search: &searchImpl}

	transSource := `module bgimpl::trans;
fn String tokensUnderCursor(Allocator alloc, CXCursor cursor, BGTransCallbacks* tfns) { return ""; }`
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(protocol.DocumentUri("file:///tmp/hover_right_symbol_trans_dep.c3"), transSource, 1), &prs)

	appSource := `module bgimpl::vtor;
import bgimpl;
fn void run(Allocator alloc, CXCursor cursor, GlobalVisitData* vd) {
	String v = trans::tokensUnderCursor(alloc, cursor, &vd.trans_fns);
}`
	appURI := protocol.DocumentUri("file:///tmp/hover_right_symbol_trans_app.c3")
	state.RefreshDocumentIdentifiers(document.NewDocumentFromDocURI(appURI, appSource, 1), &prs)

	idx := strings.Index(appSource, "tokensUnderCursor") + len("tokens")
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
		Position:     byteIndexToLSPPosition(appSource, idx),
	}})
	if err != nil {
		t.Fatalf("unexpected hover error for right symbol in short module call: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response for trans::tokensUnderCursor symbol token")
	}
}
