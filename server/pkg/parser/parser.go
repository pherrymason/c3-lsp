package parser

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/parser/queries"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/tliron/commonlog"
)

type Parser struct {
	logger commonlog.Logger
	//pendingToResolve symbols_table.PendingToResolve
}

func NewParser(logger commonlog.Logger) Parser {
	return Parser{
		logger: logger,
		//pendingToResolve: symbols_table.NewPendingToResolve(),
	}
}

func (p *Parser) ClearProject() {
	// p.pendingToResolve = symbols_table.NewPendingToResolve()
}

// declStart returns the first child position that is not a doc_comment,
// so that document ranges begin at the actual keyword rather than the doc comment.
func declStart(node *sitter.Node) sitter.Point {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() != "doc_comment" {
			return child.StartPosition()
		}
	}
	return node.StartPosition()
}

// firstNamedNonDocChild returns the index of the first named child that is not a doc_comment.
// This is needed because unnamed tokens (like the 'fn' keyword) count as children,
// and in 0.8 the optional doc_comment may appear before them.
func firstNamedNonDocChild(node *sitter.Node) uint {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil && child.IsNamed() && child.Kind() != "doc_comment" {
			return i
		}
	}
	return 0
}

// extractDocComment finds and parses a doc_comment child node, if present.
// Also checks the parent node (for cases like global_declaration wrapping declaration).
func (p *Parser) extractDocComment(node *sitter.Node, sourceCode []byte) *idx.DocComment {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "doc_comment" {
			return cast.ToPtr(p.nodeToDocComment(child, sourceCode))
		}
	}
	// Check parent: global_declaration holds doc_comment alongside declaration/const_declaration
	if parent := node.Parent(); parent != nil {
		for i := uint(0); i < parent.ChildCount(); i++ {
			child := parent.Child(i)
			if child != nil && child.Kind() == "doc_comment" {
				return cast.ToPtr(p.nodeToDocComment(child, sourceCode))
			}
		}
	}
	return nil
}

func (p *Parser) ParseSymbols(doc *document.Document) (symbols_table.UnitModules, symbols_table.PendingToResolve) {
	parsedModules := symbols_table.NewParsedModules(&doc.URI)
	pendingToResolve := symbols_table.NewPendingToResolve()

	qm := cst.RunQuery(queries.SymbolsQuery, doc.ContextSyntaxTree.RootNode())
	sourceCode := []byte(doc.SourceCode.Text)
	var moduleSymbol *idx.Module
	anonymousModuleName := true
	lastModuleName := ""

	for {
		m := qm.Next()
		if m == nil {
			break
		}

		for _, c := range m.Captures {
			nodeType := c.Node.Kind()
			nodeEndPoint := idx.NewPositionFromTreeSitterPoint(c.Node.EndPosition())
			if nodeType != "module_declaration" {
				moduleSymbol = parsedModules.GetOrInitModule(
					lastModuleName,
					&doc.URI,
					doc.ContextSyntaxTree.RootNode(),
					anonymousModuleName,
				)
			}

			switch nodeType {
			case "module_declaration":
				anonymousModuleName = false
				module, _, _ := p.nodeToModule(doc, &c.Node, sourceCode)
				lastModuleName = module.GetName()
				moduleSymbol = parsedModules.UpdateOrInitModule(
					module,
					doc.ContextSyntaxTree.RootNode(),
				)

				moduleSymbol.SetStartPosition(idx.NewPositionFromTreeSitterPoint(declStart(&c.Node)))
				moduleSymbol.ChangeModule(lastModuleName)
				moduleSymbol.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			case "import_declaration":
				imports := p.nodeToImport(doc, &c.Node, sourceCode)
				moduleSymbol.AddImports(imports)

			case "declaration":
				variables := p.variableDeclarationNodeToVariable(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddVariables(variables)
				pendingToResolve.AddVariableType(variables, moduleSymbol)
				docComment := p.extractDocComment(&c.Node, sourceCode)
				if docComment != nil {
					for _, v := range variables {
						v.SetDocComment(docComment)
					}
				}

			case "func_definition", "func_declaration":
				function, err := p.nodeToFunction(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				if err == nil {
					moduleSymbol.AddFunction(&function)
					pendingToResolve.AddFunctionTypes(&function, moduleSymbol)
					function.SetDocComment(p.extractDocComment(&c.Node, sourceCode))
				}

			// constdef (C3 0.8) shares enum_spec/enum_body with enums and its
			// constants are accessed the same way (Type.NAME), so index it as an enum.
			case "enum_declaration", "constdef_declaration":
				enum := p.nodeToEnum(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddEnum(&enum)
				enum.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			case "struct_declaration":
				strukt, membersNeedingSubtypingResolve := p.nodeToStruct(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddStruct(&strukt)
				if len(membersNeedingSubtypingResolve) > 0 {
					pendingToResolve.AddStructSubtype(&strukt, membersNeedingSubtypingResolve)
				}
				pendingToResolve.AddStructMemberTypes(&strukt, moduleSymbol)
				strukt.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			case "bitstruct_declaration":
				bitstruct := p.nodeToBitStruct(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddBitstruct(&bitstruct)
				bitstruct.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			// TODO: @0.7.7 rename internal methods/structs from Def -> Alias
			case "alias_declaration":
				def := p.nodeToDef(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddDef(&def)
				pendingToResolve.AddDefType(&def, moduleSymbol)
				def.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			// TODO: @0.7.7 rename internal methods/structs from Distinct -> TypeDef
			case "typedef_declaration":
				distinct := p.nodeToDistinct(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddDistinct(&distinct)
				pendingToResolve.AddDistinctType(&distinct, moduleSymbol)
				distinct.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			case "const_declaration":
				_const := p.nodeToConstant(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddVariable(&_const)
				_const.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			// TODO: @0.7.7 rename internal methods/structs from Fault -> FaultDef
			case "faultdef_declaration":
				fault := p.nodeToFault(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddFault(&fault)
				fault.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			case "interface_declaration":
				interf := p.nodeToInterface(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddInterface(&interf)
				interf.SetDocComment(p.extractDocComment(&c.Node, sourceCode))

			case "macro_declaration":
				macro, err := p.nodeToMacro(&c.Node, moduleSymbol, &doc.URI, sourceCode)
				if err == nil {
					moduleSymbol.AddFunction(&macro)
					macro.SetDocComment(p.extractDocComment(&c.Node, sourceCode))
				}

			case "attrdef_declaration":
				// new 0.8 declaration — not yet handled, skip

			default:
				continue
			}

			moduleSymbol.SetEndPosition(nodeEndPoint)
		}
	}

	if moduleSymbol != nil {
		moduleSymbol.SetEndPosition(
			idx.NewPositionFromTreeSitterPoint(
				doc.ContextSyntaxTree.RootNode().EndPosition(),
			),
		)
	}

	// Try to resolve as many types as possible
	//p.resolveTypes(&parsedModules)

	return parsedModules, pendingToResolve
}

func (p *Parser) FindVariableDeclarations(node *sitter.Node, moduleName string, currentModule *idx.Module, docId *string, sourceCode []byte) []*idx.Variable {
	qm := cst.RunQuery(queries.LocalVarDeclQuery, node)

	var variables []*idx.Variable
	found := make(map[string]bool)
	//sourceCode := []byte(doc.Content)
	for {
		m := qm.Next()
		if m == nil {
			break
		}
		for _, c := range m.Captures {
			if c.Node.Kind() != "declaration" {
				continue
			}
			content := c.Node.Utf8Text(sourceCode)

			if _, exists := found[content]; !exists {
				found[content] = true
				funcVariables := p.variableDeclarationNodeToVariable(&c.Node, currentModule, docId, sourceCode)

				variables = append(variables, funcVariables...)
			}
		}
	}

	return variables
}
