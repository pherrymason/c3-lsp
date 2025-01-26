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
	var description string
	var sizeInfo string
	var extraLine string
	switch symbol.Kind {
	case ast.VAR, ast.CONST:
		description = fmt.Sprintf("%s %s", symbol.Type.Name, symbol.Name)

	case ast.STRUCT, ast.AnonymousStructField:
		description = fmt.Sprintf("%s", symbol.Name)

	case ast.FIELD:
		switch symbol.Type.NodeDecl.(type) {
		case ast.TypeInfo:
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
	}

	isModule := false
	if !isModule {
		extraLine += "\n\nIn module **[" + symbol.Module.String() + "]**"
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: "```c3" + "\n" +
				sizeInfo +
				description + "\n```" +
				extraLine,
		},
	}

	return &hover
}
