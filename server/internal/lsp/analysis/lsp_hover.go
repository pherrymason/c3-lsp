package analysis

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
)

func GetHoverInfo(document *document.Document, pos lsp.Position, storage *document.Storage, symbolTable *SymbolTable) *protocol.Hover {
	symbolResult := FindSymbolAtPosition(pos, document.Uri, symbolTable, document.Ast)
	if symbolResult.IsNone() {
		return nil
	}

	symbol := symbolResult.Get()
	hover := buildHover(symbol)

	return &hover
}

func buildHover(symbol *Symbol) protocol.Hover {
	var description string
	var sizeInfo string // TODO reimplement this
	var extraLine string
	switch symbol.Kind {
	case ast.VAR, ast.CONST:
		description = fmt.Sprintf("%s %s", symbol.Type.Name, symbol.Name)

	case ast.STRUCT, ast.AnonymousStructField:
		description = fmt.Sprintf("%s", symbol.Name)

	case ast.FIELD:
		switch symbol.Type.NodeDecl.(type) {
		case *ast.TypeInfo:
			description = fmt.Sprintf("%s %s", symbol.Type.Name, symbol.Name)
		}

	case ast.FUNCTION:
		f := symbol.NodeDecl.(*ast.FunctionDecl)
		args := []string{}
		for _, arg := range f.Signature.Parameters {
			args = append(args, arg.Type.Identifier.String()+" "+arg.Name.Name)
		}

		description = fmt.Sprintf(
			"fn %s %s(%s)",
			f.Signature.ReturnType.Identifier.Name,
			f.Signature.Name.Name,
			strings.Join(args, ", "),
		)

	case ast.MACRO:
		macro := symbol.NodeDecl.(*ast.MacroDecl)
		typeMethod := ""
		if macro.Signature.ParentTypeId.IsSome() {
			typeMethod = macro.Signature.ParentTypeId.Get().Name + "."
		}
		args := []string{}
		for _, arg := range macro.Signature.Parameters {
			args = append(args, arg.Type.Identifier.String()+" "+arg.Name.Name)
		}
		trailing := ""
		if macro.Signature.TrailingBlockParam != nil {
			trailing = "; " + macro.Signature.TrailingBlockParam.Name.Name
			paramCount := len(macro.Signature.TrailingBlockParam.Parameters)
			if paramCount > 0 {
				params := []string{}
				for i := 0; i < paramCount; i++ {
					params = append(params, macro.Signature.TrailingBlockParam.Parameters[0].Name.Name)
				}

				trailing += "(" + strings.Join(params, ", ") + ")"
			}
		}

		description = fmt.Sprintf(
			"macro %s%s(%s)",
			typeMethod,
			macro.Signature.Name.Name,
			strings.Join(args, ", ")+trailing,
		)
	}

	isModule := false
	if !isModule {
		extraLine += "\n\nIn module **[" + symbol.Module.String() + "]**"
	}

	docComment := symbol.NodeDecl.GetDocComment()
	if docComment.IsSome() {
		extraLine += "\n\n" + docComment.Get().DisplayBodyWithContracts()
	}

	return protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: "```c3" + "\n" +
				sizeInfo +
				description + "\n```" +
				extraLine,
		},
	}
}
