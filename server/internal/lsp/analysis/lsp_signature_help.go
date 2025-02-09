package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
)

func BuildSignatureHelp(document *document.Document, pos lsp.Position, storage *document.Storage, symbolTable *SymbolTable) *protocol.SignatureHelp {
	// Search callExpr under cursor
	visitor := &SignatureHelpVisitor{pos: pos}
	walk.Walk(visitor, document.Ast, "")

	if visitor.callExpr == nil {
		return nil
	}

	ident := visitor.callExpr.Identifier.(*ast.Ident)
	explicitIdentModule := option.None[string]()
	if ident.ModulePath != nil {
		explicitIdentModule = option.Some(ident.ModulePath.Name)
	}
	symbolResult := symbolTable.FindSymbolByPosition(
		ident.Name,
		explicitIdentModule,
		NewLocation(
			document.Uri,
			visitor.callExpr.StartPosition(),
			NewModuleName(visitor.module),
		),
	)

	if symbolResult.IsNone() {
		return nil
	}

	symbol := symbolResult.Get()
	if symbol.Kind != ast.FUNCTION {
		return nil
	}

	parameters := []protocol.ParameterInformation{}
	argsToStringify := []string{}
	activeParameterIndex := option.None[int]()
	numWrittenArguments := len(visitor.callExpr.Arguments)

	switch fnc := symbol.NodeDecl.(type) {
	case *ast.FunctionDecl:
		for idx, param := range fnc.Signature.Parameters {
			if param.Range.HasPosition(pos) {
				activeParameterIndex = option.Some(idx)
			}
			if idx == numWrittenArguments {
				activeParameterIndex = option.Some(idx)
			}

			argsToStringify = append(
				argsToStringify,
				param.Type.Identifier.String()+" "+param.Name.Name,
			)

			parameters = append(
				parameters,
				protocol.ParameterInformation{
					Label: param.Type.Identifier.String() + " " + param.Name.Name,
					// TODO: Parse '@param' contract text to get param docs
					Documentation: nil,
				},
			)
		}
	}

	var docs any = nil
	docComment := symbol.NodeDecl.GetDocComment()
	if docComment.IsSome() {
		docs = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: docComment.Get().DisplayBodyWithContracts(),
		}
	}

	signature := protocol.SignatureInformation{
		Label:         symbol.Identifier + "(" + strings.Join(argsToStringify, ", ") + ")",
		Parameters:    parameters,
		Documentation: docs,
	}

	if activeParameterIndex.IsSome() {
		index := protocol.UInteger(activeParameterIndex.Get())
		signature.ActiveParameter = cast.ToPtr(index)
	}

	signatureHelp := protocol.SignatureHelp{
		Signatures: []protocol.SignatureInformation{signature},
	}

	return &signatureHelp
}

type SignatureHelpVisitor struct {
	pos        lsp.Position
	callExpr   *ast.CallExpr
	module     string
	stopSearch bool
}

func (v *SignatureHelpVisitor) Enter(node ast.Node, propertyName string) walk.Visitor {
	if node == nil {
		return nil
	}

	module, ok := node.(*ast.Module)
	if !v.stopSearch && ok {
		v.module = module.Name
	}

	if node.GetRange().HasPosition(v.pos) {
		if call, ok := node.(*ast.CallExpr); ok {
			// Verify the cursor is just after "("
			if call.Lparen <= v.pos.Column && call.Rparen <= v.pos.Column {
				// Store function identifier
				v.callExpr = call
				v.stopSearch = true
			}
		}
	}

	return v
}

func (v *SignatureHelpVisitor) Exit(n ast.Node, propertyName string) {
	if v.stopSearch {
		return
	}
}
