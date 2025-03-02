package analysis

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis/symbol"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
)

func GetHoverInfo(document *document.Document, pos lsp.Position, storage *document.Storage, symbolTable *symbols.SymbolTable) *protocol.Hover {
	symbolResult := FindSymbolAtPosition(pos, document.Uri, symbolTable, document.Ast, document.Text)
	if symbolResult.IsNone() {
		return nil
	}

	symbol := symbolResult.Get()
	hover := buildHover(symbol)

	return &hover
}

func buildHover(symbol *symbols.Symbol) protocol.Hover {
	var description string
	var sizeInfo string // TODO reimplement this
	var extraLine string
	switch symbol.Kind {
	case ast.VAR, ast.CONST:
		description = fmt.Sprintf("%s %s", symbol.TypeDef.Name, symbol.Identifier)

	case ast.STRUCT, ast.AnonymousStructField:
		description = fmt.Sprintf("%s", symbol.Identifier)

	case ast.FIELD:
		switch symbol.TypeDef.NodeDecl.(type) {
		case *ast.TypeInfo:
			description = fmt.Sprintf("%s %s", symbol.TypeDef.Name, symbol.Identifier)
		}

	case ast.FUNCTION:
		description = functionDescriptionString(symbol)

	case ast.MACRO:
		description = macroDescriptionString(symbol, true)
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

func macroDescriptionString(symbol *symbols.Symbol, includeMacroName bool) string {
	macro := symbol.NodeDecl.(*ast.MacroDecl)
	typeMethod := ""
	if macro.Signature.ParentTypeId.IsSome() {
		typeMethod = macro.Signature.ParentTypeId.Get().Name + "."
	}
	args := []string{}
	for _, arg := range macro.Signature.Parameters {
		typeName := ""
		if arg.Type != nil {
			typeName = arg.Type.String() + " "
		}
		args = append(args, typeName+arg.Name.Name)
	}
	trailing := ""
	if macro.Signature.TrailingBlockParam != nil {
		trailing = "; " + macro.Signature.TrailingBlockParam.Name.Name
		paramCount := len(macro.Signature.TrailingBlockParam.Parameters)
		if paramCount > 0 {
			params := []string{}
			for i := 0; i < paramCount; i++ {
				parameter := macro.Signature.TrailingBlockParam.Parameters[i]
				typeName := ""
				if parameter.Type != nil {
					typeName = parameter.Type.String() + " "
				}
				paramString := fmt.Sprintf("%s%s", typeName, parameter.Name.Name)
				params = append(params, paramString)
			}

			trailing += "(" + strings.Join(params, ", ") + ")"
		}
	}

	macroName := ""
	if !includeMacroName {
		macroName = ""
	} else {
		if typeMethod != "" {
			macroName = " " + typeMethod + macro.Signature.Name.Name
		} else {
			macroName = " " + macro.Signature.Name.Name
		}
	}

	description := fmt.Sprintf(
		"macro%s(%s)",
		macroName,
		strings.Join(args, ", ")+trailing,
	)
	return description
}

func functionDescriptionString(symbol *symbols.Symbol) string {
	f := symbol.NodeDecl.(*ast.FunctionDecl)
	args := []string{}
	for _, arg := range f.Signature.Parameters {
		args = append(args, arg.Type.Identifier.String()+" "+arg.Name.Name)
	}

	description := fmt.Sprintf(
		"fn %s %s(%s)",
		f.Signature.ReturnType.Identifier.Name,
		f.Signature.Name.Name,
		strings.Join(args, ", "),
	)
	return description
}
