package parser

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/parser/queries"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tliron/commonlog"
)

type Parser struct {
	logger commonlog.Logger
}

func NewParser(logger commonlog.Logger) Parser {
	return Parser{
		logger: logger,
	}
}

func (p *Parser) ClearProject() {}

// parseState holds the mutable context shared across all captures in a single
// ParseSymbols call.  Grouping it here makes the signature of processCapture
// clean and explicit — there is no implicit/closure state.
type parseState struct {
	doc              *document.Document
	sourceCode       []byte
	parsedModules    *symbols_table.UnitModules
	pendingToResolve *symbols_table.PendingToResolve
	moduleSymbol     *idx.Module
	lastModuleName   string
	anonymousModule  bool
	lastDocComment   *idx.DocComment
}

// ParseSymbols parses all top-level symbols from doc and returns indexed
// modules together with unresolved type references.
func (p *Parser) ParseSymbols(doc *document.Document) (symbols_table.UnitModules, symbols_table.PendingToResolve) {
	parsedModules := symbols_table.NewParsedModules(&doc.URI)
	pendingToResolve := symbols_table.NewPendingToResolve()

	qc := cst.RunQuery(queries.SymbolsQuery, doc.ContextSyntaxTree.RootNode())

	st := &parseState{
		doc:              doc,
		sourceCode:       []byte(doc.SourceCode.Text),
		parsedModules:    &parsedModules,
		pendingToResolve: &pendingToResolve,
		anonymousModule:  true,
	}

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			p.processCapture(st, c.Node)
		}
	}

	// Extend the last module to cover the entire document.
	if st.moduleSymbol != nil {
		st.moduleSymbol.SetEndPosition(
			idx.NewPositionFromTreeSitterPoint(
				doc.ContextSyntaxTree.RootNode().EndPoint(),
			),
		)
	}

	return parsedModules, pendingToResolve
}

// processCapture dispatches a single query capture to the appropriate handler.
// It updates st in-place: moduleSymbol, lastDocComment, lastModuleName, etc.
func (p *Parser) processCapture(st *parseState, node *sitter.Node) {
	nodeType := node.Type()
	nodeEndPoint := idx.NewPositionFromTreeSitterPoint(node.EndPoint())

	// Peek at a preceding doc_comment child before the module is ensured,
	// so that module_declaration can receive it too.
	if nodeType != "doc_comment" {
		if dc := p.docCommentFromNode(node, st.sourceCode); dc != nil {
			st.lastDocComment = dc
		}
	}

	// Ensure a module context exists for every non-module, non-comment capture.
	if nodeType != "module_declaration" && nodeType != "doc_comment" {
		st.moduleSymbol = st.parsedModules.GetOrInitModule(
			st.lastModuleName,
			&st.doc.URI,
			st.doc.ContextSyntaxTree.RootNode(),
			st.anonymousModule,
		)
	}

	switch nodeType {
	case "doc_comment":
		dc := cast.ToPtr(p.nodeToDocComment(node, st.sourceCode))
		st.lastDocComment = dc
		return // doc comments don't advance the module end-position

	case "module_declaration":
		p.handleModuleDeclaration(st, node)

	case "import_declaration":
		imports, noRecurse := p.nodeToImport(st.doc, node, st.sourceCode)
		st.moduleSymbol.AddImportsWithMode(imports, noRecurse)

	case "declaration":
		variables := p.variableDeclarationNodeToVariable(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddVariables(variables)
		st.pendingToResolve.AddVariableType(variables, st.moduleSymbol)
		applyDocComment(st.lastDocComment, func(dc *idx.DocComment) {
			for _, v := range variables {
				v.SetDocComment(dc)
			}
		})

	case "func_definition", "func_declaration":
		function, err := p.nodeToFunction(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		if err == nil {
			st.moduleSymbol.AddFunction(&function)
			st.pendingToResolve.AddFunctionTypes(&function, st.moduleSymbol)
			applyDocComment(st.lastDocComment, function.SetDocComment)
		}

	case "enum_declaration":
		enum := p.nodeToEnum(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddEnum(&enum)
		applyDocComment(st.lastDocComment, enum.SetDocComment)

	case "struct_declaration":
		strukt, membersNeedingSubtypingResolve := p.nodeToStruct(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddStruct(&strukt)
		if len(membersNeedingSubtypingResolve) > 0 {
			st.pendingToResolve.AddStructSubtype(&strukt, membersNeedingSubtypingResolve)
		}
		st.pendingToResolve.AddStructMemberTypes(&strukt, st.moduleSymbol)
		applyDocComment(st.lastDocComment, strukt.SetDocComment)

	case "bitstruct_declaration":
		bitstruct := p.nodeToBitStruct(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddBitstruct(&bitstruct)
		applyDocComment(st.lastDocComment, bitstruct.SetDocComment)

	case "alias_declaration":
		alias := p.nodeToAlias(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddAlias(&alias)
		st.pendingToResolve.AddAliasType(&alias, st.moduleSymbol)
		applyDocComment(st.lastDocComment, alias.SetDocComment)

	case "typedef_declaration":
		typeDef := p.nodeToTypeDef(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddTypeDef(&typeDef)
		st.pendingToResolve.AddTypeDefType(&typeDef, st.moduleSymbol)
		applyDocComment(st.lastDocComment, typeDef.SetDocComment)

	case "const_declaration":
		_const := p.nodeToConstant(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddVariable(&_const)
		applyDocComment(st.lastDocComment, _const.SetDocComment)

	case "faultdef_declaration":
		faultDef := p.nodeToFaultDef(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddFaultDef(&faultDef)
		applyDocComment(st.lastDocComment, faultDef.SetDocComment)

	case "interface_declaration":
		interf := p.nodeToInterface(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		st.moduleSymbol.AddInterface(&interf)
		applyDocComment(st.lastDocComment, interf.SetDocComment)

	case "macro_declaration":
		macro, err := p.nodeToMacro(node, st.moduleSymbol, &st.doc.URI, st.sourceCode)
		if err == nil {
			st.moduleSymbol.AddFunction(&macro)
			applyDocComment(st.lastDocComment, macro.SetDocComment)
		}

	default:
		st.lastDocComment = nil
		return
	}

	// Every successfully handled node consumes the pending doc comment and
	// advances the module's end position.
	st.lastDocComment = nil
	st.moduleSymbol.SetEndPosition(nodeEndPoint)
}

// handleModuleDeclaration processes a module_declaration node, updating the
// module context in st.
func (p *Parser) handleModuleDeclaration(st *parseState, node *sitter.Node) {
	st.anonymousModule = false
	module, _, _ := p.nodeToModule(st.doc, node, st.sourceCode)
	st.lastModuleName = module.GetName()
	st.moduleSymbol = st.parsedModules.UpdateOrInitModule(
		module,
		st.doc.ContextSyntaxTree.RootNode(),
	)

	start := startPointSkippingDocComment(node)
	startPosition := idx.NewPositionFromTreeSitterPoint(start)
	currentStart := st.moduleSymbol.GetDocumentRange().Start
	if startPosition.Line < currentStart.Line ||
		(startPosition.Line == currentStart.Line && startPosition.Character < currentStart.Character) {
		st.moduleSymbol.SetStartPosition(startPosition)
	}
	st.moduleSymbol.ChangeModule(st.lastModuleName)
	applyDocComment(st.lastDocComment, st.moduleSymbol.SetDocComment)
}

// applyDocComment calls set(dc) when dc is non-nil.  This eliminates the
// repeated `if lastDocComment != nil { ... }` pattern throughout the switch.
func applyDocComment(dc *idx.DocComment, set func(*idx.DocComment)) {
	if dc != nil {
		set(dc)
	}
}

func (p *Parser) FindVariableDeclarations(node *sitter.Node, moduleName string, currentModule *idx.Module, docId *string, sourceCode []byte) []*idx.Variable {
	var variables []*idx.Variable

	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		if n.Type() == "declaration" {
			funcVariables := p.variableDeclarationNodeToVariable(n, currentModule, docId, sourceCode)
			variables = append(variables, funcVariables...)
		}

		if n.Type() == "param" && !isTopLevelDeclarationParam(n) {
			lambdaParam := p.nodeToArgument(n, "", currentModule, docId, sourceCode, 0)
			if lambdaParam != nil {
				variables = append(variables, lambdaParam)
			}
		}

		if n.Type() == "foreach_var" {
			foreachVariables := p.foreachVarNodeToVariables(n, moduleName, docId, sourceCode)
			variables = append(variables, foreachVariables...)
		}

		if n.Type() == "try_unwrap" || n.Type() == "catch_unwrap" {
			if unwrapVariable := p.unwrapBindingNodeToVariable(n, currentModule, docId, sourceCode); unwrapVariable != nil {
				variables = append(variables, unwrapVariable)
			}
		}

		for i := uint32(0); i < n.ChildCount(); i++ {
			walk(n.Child(int(i)))
		}
	}

	walk(node)

	return variables
}

func isTopLevelDeclarationParam(paramNode *sitter.Node) bool {
	if paramNode == nil {
		return false
	}

	for parent := paramNode.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Type() != "func_param_list" && parent.Type() != "macro_param_list" {
			continue
		}

		signatureOwner := parent.Parent()
		if signatureOwner == nil {
			return false
		}

		switch signatureOwner.Type() {
		case "func_definition", "func_declaration", "macro_declaration":
			return true
		default:
			return false
		}
	}

	return false
}

func (p *Parser) foreachVarNodeToVariables(node *sitter.Node, moduleName string, docId *string, sourceCode []byte) []*idx.Variable {
	variables := []*idx.Variable{}

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil || child.Type() != "ident" {
			continue
		}

		name := child.Content(sourceCode)
		if name == "" {
			continue
		}

		idRange := idx.NewRangeFromTreeSitterPositions(child.StartPoint(), child.EndPoint())
		variable := idx.NewVariable(name, idx.Type{}, moduleName, *docId, idRange, idRange)
		variables = append(variables, &variable)
	}

	return variables
}

func (p *Parser) docCommentFromNode(node *sitter.Node, sourceCode []byte) *idx.DocComment {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "doc_comment" {
			return cast.ToPtr(p.nodeToDocComment(child, sourceCode))
		}
	}

	return nil
}
