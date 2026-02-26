package server

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestInitializeWorkspaceURI_PrefersRootURI(t *testing.T) {
	rootURI := protocol.DocumentUri("file:///tmp/root")
	folderURI := protocol.DocumentUri("file:///tmp/folder")

	params := &protocol.InitializeParams{
		RootURI: &rootURI,
		WorkspaceFolders: []protocol.WorkspaceFolder{{
			URI:  folderURI,
			Name: "folder",
		}},
	}

	got := initializeWorkspaceURI(params)
	if got == nil || *got != rootURI {
		t.Fatalf("initializeWorkspaceURI should prefer RootURI, got %v", got)
	}
}

func TestInitializeWorkspaceURI_UsesFirstWorkspaceFolder(t *testing.T) {
	folderURI := protocol.DocumentUri("file:///tmp/workspace")

	params := &protocol.InitializeParams{
		WorkspaceFolders: []protocol.WorkspaceFolder{{
			URI:  folderURI,
			Name: "workspace",
		}},
	}

	got := initializeWorkspaceURI(params)
	if got == nil || *got != folderURI {
		t.Fatalf("initializeWorkspaceURI should use first workspace folder, got %v", got)
	}
}
