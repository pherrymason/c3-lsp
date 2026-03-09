package project_state

import (
	"fmt"
	"sync"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRefreshDocumentIdentifiers_should_clear_cached_stuff_test(t *testing.T) {

	var logger commonlog.Logger
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)
	doc := document.NewDocumentFromString(
		"doc-id",
		`module app::something_app;
		fn void main() {}
		`)
	s.RefreshDocumentIdentifiers(&doc, &p)
	result := s.fqnIndex.Search("app::something_app.main")
	assert.Equal(t, 1, len(result))

	// Force a modification
	doc = document.NewDocumentFromString(
		"doc-id",
		`module app::something_new;
		fn void main() {}
		`)
	s.RefreshDocumentIdentifiers(&doc, &p)

	result = s.fqnIndex.Search("app::something_app.main")
	assert.Equal(t, 0, len(result))

	result = s.fqnIndex.Search("app::something_new.main")
	assert.Equal(t, 1, len(result))
}

func TestProjectState_tracks_module_dependencies_and_impacted_modules(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	docA := document.NewDocumentFromString(
		"a.c3",
		`module app::a;
		 import app::b;
		 fn void main() {}`,
	)
	docB := document.NewDocumentFromString(
		"b.c3",
		`module app::b;
		 import app::c;
		 fn void helper() {}`,
	)
	docC := document.NewDocumentFromString(
		"c.c3",
		`module app::c;
		 fn void leaf() {}`,
	)

	s.RefreshDocumentIdentifiers(&docA, &p)
	s.RefreshDocumentIdentifiers(&docB, &p)
	s.RefreshDocumentIdentifiers(&docC, &p)

	assert.Equal(t, []string{"app::b"}, s.GetModuleImports("app::a"))
	assert.Equal(t, []string{"app::c"}, s.GetModuleImports("app::b"))
	assert.Equal(t, []string{"app::a"}, s.GetModuleDependents("app::b"))
	assert.Equal(t, []string{"app::b"}, s.GetModuleDependents("app::c"))

	assert.Equal(t, []string{"app::a", "app::b", "app::c"}, s.GetImpactedModules([]string{"app::c"}))
}

func TestProjectState_dependency_graph_updates_when_document_changes(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	docA := document.NewDocumentFromString(
		"a.c3",
		`module app::a;
		 import app::b;
		 fn void main() {}`,
	)
	s.RefreshDocumentIdentifiers(&docA, &p)

	assert.Equal(t, []string{"app::b"}, s.GetModuleImports("app::a"))

	docAUpdated := document.NewDocumentFromString(
		"a.c3",
		`module app::a;
		 import app::c;
		 fn void main() {}`,
	)
	s.RefreshDocumentIdentifiers(&docAUpdated, &p)

	assert.Equal(t, []string{"app::c"}, s.GetModuleImports("app::a"))
	assert.Equal(t, []string(nil), s.GetModuleDependents("app::b"))
	assert.Equal(t, []string{"app::a"}, s.GetModuleDependents("app::c"))
}

func TestProjectState_invalidation_scope_distinguishes_signature_and_local_changes(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	docA := document.NewDocumentFromString(
		"a.c3",
		`module app::a;
		 import app::b;
		 fn void use_b() {}`,
	)
	docB := document.NewDocumentFromString(
		"b.c3",
		`module app::b;
		 fn void helper() {
			int x = 1;
		 }`,
	)

	s.RefreshDocumentIdentifiers(&docA, &p)
	s.RefreshDocumentIdentifiers(&docB, &p)

	localChange := document.NewDocumentFromString(
		"b.c3",
		`module app::b;
		 fn void helper() {
			int x = 2;
		 }`,
	)
	s.RefreshDocumentIdentifiers(&localChange, &p)

	scope := s.GetLastInvalidationScope()
	assert.Equal(t, []string{"app::b"}, scope.ChangedModules)
	assert.Equal(t, []string(nil), scope.SignatureChangedModules)
	assert.Equal(t, []string{"app::b"}, scope.ImpactedModules)

	revAfterLocal := s.Revision()

	signatureChange := document.NewDocumentFromString(
		"b.c3",
		`module app::b;
		 fn int helper(int value) {
			return value;
		 }`,
	)
	s.RefreshDocumentIdentifiers(&signatureChange, &p)

	scope = s.GetLastInvalidationScope()
	assert.Equal(t, []string{"app::b"}, scope.ChangedModules)
	assert.Equal(t, []string{"app::b"}, scope.SignatureChangedModules)
	assert.Equal(t, []string{"app::a", "app::b"}, scope.ImpactedModules)
	assert.Greater(t, s.Revision(), revAfterLocal)
}

func TestProjectState_revision_stays_for_local_only_change(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	doc := document.NewDocumentFromString(
		"x.c3",
		`module app::x;
		 fn void helper() {
			int a = 1;
		 }`,
	)
	s.RefreshDocumentIdentifiers(&doc, &p)
	baseline := s.Revision()

	docLocalEdit := document.NewDocumentFromString(
		"x.c3",
		`module app::x;
		 fn void helper() {
			int a = 2;
		 }`,
	)
	s.RefreshDocumentIdentifiers(&docLocalEdit, &p)

	assert.Equal(t, baseline, s.Revision())
}

func TestProjectState_snapshot_is_rebuilt_after_refresh(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	doc := document.NewDocumentFromString(
		"snap.c3",
		`module app::snap;
		 fn void hello() {}`,
	)
	s.RefreshDocumentIdentifiers(&doc, &p)

	snapshot := s.Snapshot()
	if assert.NotNil(t, snapshot) {
		modules := snapshot.GetUnitModulesByDoc("snap.c3")
		if assert.NotNil(t, modules) {
			assert.NotNil(t, modules.Get("app::snap"))
		}
	}
}

func TestProjectState_snapshot_concurrent_read_write(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	seed := document.NewDocumentFromString(
		"concurrent.c3",
		`module app::concurrent;
		 fn int value() { return 0; }`,
	)
	s.RefreshDocumentIdentifiers(&seed, &p)

	var wg sync.WaitGroup

	reader := func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = s.SearchByFQN("app::concurrent.value")
			_ = s.GetAllUnitModules()
			_ = s.GetUnitModulesByDoc("concurrent.c3")
			_ = s.GetDocument("concurrent.c3")
		}
	}

	writer := func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			doc := document.NewDocumentFromString(
				"concurrent.c3",
				fmt.Sprintf(`module app::concurrent;
				 fn int value() { return %d; }`, i),
			)
			s.RefreshDocumentIdentifiers(&doc, &p)
		}
	}

	wg.Add(5)
	go reader()
	go reader()
	go reader()
	go reader()
	go writer()
	wg.Wait()

	result := s.SearchByFQN("app::concurrent.value")
	assert.NotNil(t, result)
}

func TestProjectState_snapshot_indexes_modules_and_docs(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	docA := document.NewDocumentFromString(
		"a.c3",
		`module app::net;
		 fn void a() {}`,
	)
	docB := document.NewDocumentFromString(
		"b.c3",
		`module core::net;
		 fn void b() {}`,
	)

	s.RefreshDocumentIdentifiers(&docA, &p)
	s.RefreshDocumentIdentifiers(&docB, &p)

	snapshot := s.Snapshot()
	if assert.NotNil(t, snapshot) {
		modules := snapshot.ModulesByName("app::net")
		if assert.Len(t, modules, 1) {
			assert.Equal(t, "app::net", modules[0].GetName())
		}

		docs := snapshot.DocsByModule("core::net")
		if assert.Len(t, docs, 1) {
			assert.Equal(t, "b.c3", string(docs[0]))
		}

		names := snapshot.ModuleNamesByShort("net")
		assert.Equal(t, []string{"app::net", "core::net"}, names)
	}
}

func TestProjectState_GetDocumentsForModules_uses_snapshot_index(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	docA := document.NewDocumentFromString(
		"a.c3",
		`module app::one;
		 fn void a() {}`,
	)
	docB := document.NewDocumentFromString(
		"b.c3",
		`module app::two;
		 fn void b() {}`,
	)
	s.RefreshDocumentIdentifiers(&docA, &p)
	s.RefreshDocumentIdentifiers(&docB, &p)

	docs := s.GetDocumentsForModules([]string{"app::two", "app::one"})
	assert.Equal(t, []string{"a.c3", "b.c3"}, docs)
}

func TestProjectState_UpdateDocument_ignores_stale_versions(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	doc := document.NewDocumentFromDocURI(
		"versioned.c3",
		`module app::v;
		 fn void main() {}`,
		3,
	)
	s.RefreshDocumentIdentifiers(doc, &p)

	changes := []interface{}{protocol.TextDocumentContentChangeEventWhole{Text: `module app::v;
		 fn void changed() {}`}}
	s.UpdateDocument(protocol.DocumentUri("versioned.c3"), 2, changes, &p)

	current := s.GetDocument("versioned.c3")
	if assert.NotNil(t, current) {
		assert.Contains(t, current.SourceCode.Text, "main")
		assert.NotContains(t, current.SourceCode.Text, "changed")
		assert.Equal(t, int32(3), current.Version)
	}
}

func TestProjectState_UpdateDocument_empty_changes_updates_version_only(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	doc := document.NewDocumentFromDocURI(
		"empty-change.c3",
		`module app::v;
		 fn void main() {}`,
		1,
	)
	s.RefreshDocumentIdentifiers(doc, &p)

	s.UpdateDocument(protocol.DocumentUri("empty-change.c3"), 2, []interface{}{}, &p)

	current := s.GetDocument("empty-change.c3")
	if assert.NotNil(t, current) {
		assert.Contains(t, current.SourceCode.Text, "main")
		assert.Equal(t, int32(2), current.Version)
	}
}

func TestProjectState_DeleteDocument_removes_document_from_store(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	doc := document.NewDocumentFromString(
		"delete-me.c3",
		`module app::delete_me;
		 fn void main() {}`,
	)
	s.RefreshDocumentIdentifiers(&doc, &p)
	assert.NotNil(t, s.GetDocument("delete-me.c3"))

	s.DeleteDocument("delete-me.c3")

	assert.Nil(t, s.GetDocument("delete-me.c3"))
}

func TestProjectState_RenameDocument_renames_document_store_entry(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	doc := document.NewDocumentFromString(
		"old-name.c3",
		`module app::renamed;
		 fn void main() {}`,
	)
	s.RefreshDocumentIdentifiers(&doc, &p)

	s.RenameDocument("old-name.c3", "new-name.c3")

	assert.Nil(t, s.GetDocument("old-name.c3"))
	if renamed := s.GetDocument("new-name.c3"); assert.NotNil(t, renamed) {
		assert.Equal(t, "new-name.c3", renamed.URI)
	}
}

func TestProjectState_RenameDocument_ignores_missing_destination_modules(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)

	assert.NotPanics(t, func() {
		s.RenameDocument("missing-old.c3", "missing-new.c3")
	})
}

func TestProjectState_CloseDocument_removes_document_from_store(t *testing.T) {
	logger := commonlog.MockLogger{}
	s := NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	doc := document.NewDocumentFromString(
		"close-doc.c3",
		`module app::close_doc;
		 fn void hello() {}`,
	)
	s.RefreshDocumentIdentifiers(&doc, &p)
	assert.NotNil(t, s.GetDocument("close-doc.c3"))

	s.CloseDocument(protocol.DocumentUri("close-doc.c3"))

	assert.Nil(t, s.GetDocument("close-doc.c3"))
}
