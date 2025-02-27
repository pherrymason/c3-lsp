package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/stretchr/testify/assert"
	"testing"
)

func mainFunctionNode() *ast.FunctionDecl {
	return &ast.FunctionDecl{
		Signature: &ast.FunctionSignature{
			Name:       &ast.Ident{Name: "main"},
			Parameters: []*ast.FunctionParameter{},
		},
	}
}

func TestContext_should_detect_word_under_cursor_correctly(t *testing.T) {
	t.Run("simple indent", func(t *testing.T) {
		source := `
	fn void main() {
		int status = 0;
	}
	`

		path := []PathStep{
			{node: &ast.Module{}, propertyName: "module"},
			{node: mainFunctionNode(), propertyName: "Declarations"},
			{node: &ast.CompoundStmt{}, propertyName: "Body"},
			{node: &ast.AssignmentExpression{}, propertyName: "Statements"},
			{node: &ast.GenDecl{}, propertyName: "Left"},
			{node: &ast.Ident{Name: "status"}, propertyName: "Names"},
		}
		pos := lsp.Position{2, 8}
		ctxt := getContextFromPosition(path, pos, source, ContextHintForGoTo)

		assert.Equal(t, "status", ctxt.identUnderCursor)
		assert.Equal(t, "status", ctxt.fullIdentUnderCursor)
		assert.False(t, ctxt.isSelExpr)
	})

	t.Run("simple selector expression", func(t *testing.T) {
		source := `
	fn void main() {
		int status = obj.property;
	}
	`

		path := []PathStep{
			{node: &ast.Module{}, propertyName: "module"},
			{node: mainFunctionNode(), propertyName: "Declarations"},
			{node: &ast.CompoundStmt{}, propertyName: "Body"},
			{node: &ast.AssignmentExpression{}, propertyName: "Statements"},
			{node: &ast.SelectorExpr{X: &ast.Ident{Name: "obj"}, Sel: &ast.Ident{Name: "property"}}, propertyName: "Right"},
			{node: &ast.Ident{Name: "property"}, propertyName: "Sel"},
		}
		pos := lsp.Position{2, 20}
		ctxt := getContextFromPosition(path, pos, source, ContextHintForGoTo)

		assert.Equal(t, "property", ctxt.identUnderCursor)
		assert.Equal(t, "property", ctxt.fullIdentUnderCursor)
		assert.True(t, ctxt.isSelExpr)
	})

	// Tests for cases with parse errors
	t.Run("parse error with not finished one level selector expression should not flag it as isSelExpr", func(t *testing.T) {
		source := `
	fn void main() {
		int status = obj.
	}
	`

		path := []PathStep{
			{node: &ast.Module{}, propertyName: "module"},
			{node: mainFunctionNode(), propertyName: "Declarations"},
			{node: &ast.CompoundStmt{}, propertyName: "Body"},
			{node: &ast.ErrorNode{
				Content: "int status = obj.",
			}, propertyName: "Statements"},
		}
		pos := lsp.NewPosition(2, 19)
		ctxt := getContextFromPosition(path, pos, source, ContextHintForGoTo)

		assert.Equal(t, "", ctxt.identUnderCursor)
		assert.Equal(t, "obj.", ctxt.fullIdentUnderCursor)
		assert.True(t, ctxt.isSelExpr)
	})

	t.Run("parse error with not finished multi level selector expression should flag it as isSelExpr", func(t *testing.T) {
		source := `
	fn void main() {
		int status = obj.prop.
	}
	`

		path := []PathStep{
			{node: &ast.Module{}, propertyName: "module"},
			{node: mainFunctionNode(), propertyName: "Declarations"},
			{node: &ast.CompoundStmt{}, propertyName: "Body"},
			{node: &ast.ErrorNode{
				Content: "int status = obj.",
			}, propertyName: "Statements"},
		}
		pos := lsp.Position{2, 24}
		ctxt := getContextFromPosition(path, pos, source, ContextHintForGoTo)

		assert.Equal(t, "", ctxt.identUnderCursor)
		assert.Equal(t, "obj.prop.", ctxt.fullIdentUnderCursor)
		assert.True(t, ctxt.isSelExpr)
	})
}
