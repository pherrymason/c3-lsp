package server

import "testing"

func TestShouldWarnPartialWorkspaceIndexForRename_warns_once_per_unindexed_root(t *testing.T) {
	root := "/tmp/project"

	srv := &Server{
		idx:                indexingCoordinator{indexed: map[string]bool{}},
		renameWarningRoots: map[string]bool{},
	}

	if !srv.shouldWarnPartialWorkspaceIndexForRename(root) {
		t.Fatalf("expected first warning for unindexed root")
	}

	if srv.shouldWarnPartialWorkspaceIndexForRename(root) {
		t.Fatalf("expected warning to be emitted only once per root")
	}
}

func TestShouldWarnPartialWorkspaceIndexForRename_skips_when_root_indexed(t *testing.T) {
	root := "/tmp/project"

	srv := &Server{
		idx:                indexingCoordinator{indexed: map[string]bool{root: true}},
		renameWarningRoots: map[string]bool{root: true},
	}

	if srv.shouldWarnPartialWorkspaceIndexForRename(root) {
		t.Fatalf("expected no warning for indexed root")
	}

	if srv.renameWarningRoots[root] {
		t.Fatalf("expected stale warning marker to be cleared for indexed root")
	}
}
