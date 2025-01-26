package parser

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tliron/commonlog"
)

const DocCommentQuery = `(doc_comment) @doc_comment`
const VarDeclarationQuery = `(var_declaration
		name: (identifier) @variable_name
	)`
const GlobalVarDeclaration = `(global_declaration) @global_decl`
const ConstantDeclaration = `(const_declaration) @const_decl`
const LocalVarDeclaration = `(func_definition
	body: (macro_func_body (compound_stmt (declaration_stmt) @local) )
 )`
const FunctionDefinitionQuery = `(func_definition) @function_def`
const FunctionDeclarationQuery = `(func_declaration) @function_dec`
const EnumDeclaration = `(enum_declaration) @enum_dec`
const FaultDeclaration = `(fault_declaration) @fault_doc`
const StructDeclaration = `(struct_declaration) @struct_dec`
const BitstructDeclaration = `(bitstruct_declaration) @bitstruct_dec`
const DefineDeclaration = `(define_declaration) @def_dec`
const DistinctDeclaration = `(distinct_declaration) @distinct_dec`
const InterfaceDeclaration = `(interface_declaration) @interface_dec`
const MacroDeclaration = `(macro_declaration) @macro_dec`
const ModuleDeclaration = `(module) @module_dec`
const ImportDeclaration = `(import_declaration) @import_dec`

const ModuleQuery = `(source_file ` + ModuleDeclaration + `)`

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

func (p *Parser) ParseSymbols(doc *document.Document) (symbols_table.UnitModules, symbols_table.PendingToResolve) {
	parsedModules := symbols_table.NewParsedModules(&doc.URI)
	pendingToResolve := symbols_table.NewPendingToResolve()
	//fmt.Println(doc.URI, doc.ContextSyntaxTree.RootNode())

	query := `[
(source_file ` + DocCommentQuery + `)
(source_file ` + ModuleDeclaration + `)
(source_file ` + ImportDeclaration + `)
(source_file ` + GlobalVarDeclaration + `)
(source_file ` + LocalVarDeclaration + `)
(source_file ` + ConstantDeclaration + `)
(source_file ` + FunctionDefinitionQuery + `)
(source_file ` + FunctionDeclarationQuery + `)
(source_file ` + DefineDeclaration + `)
(source_file ` + DistinctDeclaration + `)
(source_file ` + StructDeclaration + `)
(source_file ` + BitstructDeclaration + `)
(source_file ` + EnumDeclaration + `)
(source_file ` + FaultDeclaration + `)
(source_file ` + InterfaceDeclaration + `)
(source_file ` + MacroDeclaration + `)
]`

	/*
		q, err := sitter.NewQuery([]byte(query), cst.GetLanguage())
		if err != nil {
			panic(err)
		}
		qc := sitter.NewQueryCursor()
		qc.Exec(q, doc.ContextSyntaxTree.RootNode())*/
	qc := cst.RunQuery(query, doc.ContextSyntaxTree.RootNode())
	sourceCode := []byte(doc.SourceCode.Text)
	//fmt.Println(doc.URI, " ", doc.ContextSyntaxTree.RootNode())
	//fmt.Println(doc.ContextSyntaxTree.RootNode().Content(sourceCode))
	//parsed := fmt.Sprint(doc.URI, " ", doc.ContextSyntaxTree.RootNode())
	//fmt.Println(parsed)
	var moduleSymbol *idx.Module
	anonymousModuleName := true
	lastModuleName := ""
	var lastDocComment *idx.DocComment = nil
	//subtyptingToResolve := []StructWithSubtyping{}

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			nodeType := c.Node.Type()
			nodeEndPoint := idx.NewPositionFromTreeSitterPoint(c.Node.EndPoint())
			if nodeType != "module" && nodeType != "doc_comment" {
				moduleSymbol = parsedModules.GetOrInitModule(
					lastModuleName,
					&doc.URI,
					doc.ContextSyntaxTree.RootNode(),
					anonymousModuleName,
				)
			}

			switch nodeType {
			case "doc_comment":
				lastDocComment = cast.ToPtr(p.nodeToDocComment(c.Node, sourceCode))
			case "module":
				anonymousModuleName = false
				module, _, _ := p.nodeToModule(doc, c.Node, sourceCode)
				lastModuleName = module.GetName()
				moduleSymbol = parsedModules.UpdateOrInitModule(
					module,
					doc.ContextSyntaxTree.RootNode(),
				)

				start := c.Node.StartPoint()
				moduleSymbol.
					SetStartPosition(idx.NewPositionFromTreeSitterPoint(start))

				moduleSymbol.SetStartPosition(idx.NewPositionFromTreeSitterPoint(start))
				moduleSymbol.ChangeModule(lastModuleName)

				if lastDocComment != nil {
					moduleSymbol.SetDocComment(lastDocComment)
				}

			case "import_declaration":
				imports := p.nodeToImport(doc, c.Node, sourceCode)
				moduleSymbol.AddImports(imports)

			case "global_declaration":
				variables := p.globalVariableDeclarationNodeToVariable(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddVariables(variables)
				pendingToResolve.AddVariableType(variables, moduleSymbol)

				if lastDocComment != nil {
					for _, v := range variables {
						v.SetDocComment(lastDocComment)
					}
				}

			case "func_definition", "func_declaration":
				function, err := p.nodeToFunction(c.Node, moduleSymbol, &doc.URI, sourceCode)
				if err == nil {
					moduleSymbol.AddFunction(&function)
					pendingToResolve.AddFunctionTypes(&function, moduleSymbol)

					if lastDocComment != nil {
						function.SetDocComment(lastDocComment)
					}
				}

			case "enum_declaration":
				enum := p.nodeToEnum(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddEnum(&enum)

				if lastDocComment != nil {
					enum.SetDocComment(lastDocComment)
				}

			case "struct_declaration":
				strukt, membersNeedingSubtypingResolve := p.nodeToStruct(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddStruct(&strukt)
				if len(membersNeedingSubtypingResolve) > 0 {
					pendingToResolve.AddStructSubtype(&strukt, membersNeedingSubtypingResolve)
				}

				pendingToResolve.AddStructMemberTypes(&strukt, moduleSymbol)

				if lastDocComment != nil {
					strukt.SetDocComment(lastDocComment)
				}

			case "bitstruct_declaration":
				bitstruct := p.nodeToBitStruct(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddBitstruct(&bitstruct)

				if lastDocComment != nil {
					bitstruct.SetDocComment(lastDocComment)
				}

			case "define_declaration":
				def := p.nodeToDef(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddDef(&def)
				pendingToResolve.AddDefType(&def, moduleSymbol)

				if lastDocComment != nil {
					def.SetDocComment(lastDocComment)
				}

			case "distinct_declaration":
				distinct := p.nodeToDistinct(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddDistinct(&distinct)
				pendingToResolve.AddDistinctType(&distinct, moduleSymbol)

				if lastDocComment != nil {
					distinct.SetDocComment(lastDocComment)
				}

			case "const_declaration":
				_const := p.nodeToConstant(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddVariable(&_const)

				if lastDocComment != nil {
					_const.SetDocComment(lastDocComment)
				}

			case "fault_declaration":
				fault := p.nodeToFault(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddFault(&fault)

				if lastDocComment != nil {
					fault.SetDocComment(lastDocComment)
				}

			case "interface_declaration":
				interf := p.nodeToInterface(c.Node, moduleSymbol, &doc.URI, sourceCode)
				moduleSymbol.AddInterface(&interf)

				if lastDocComment != nil {
					interf.SetDocComment(lastDocComment)
				}

			case "macro_declaration":
				macro, err := p.nodeToMacro(c.Node, moduleSymbol, &doc.URI, sourceCode)
				if err == nil {
					moduleSymbol.AddFunction(&macro)

					if lastDocComment != nil {
						macro.SetDocComment(lastDocComment)
					}
				}
			default:
				// TODO test that module ends up with wrong endPosition
				// when this source code:
				// int variable = 3;
				// fn void main() {
				// int value = 4;
				// v
				// }
				lastDocComment = nil
				continue
			}

			if nodeType != "doc_comment" {
				// Ensure the next node won't receive the same doc comment
				lastDocComment = nil
				moduleSymbol.SetEndPosition(nodeEndPoint)
			}
		}
	}

	if moduleSymbol != nil {
		moduleSymbol.SetEndPosition(
			idx.NewPositionFromTreeSitterPoint(
				doc.ContextSyntaxTree.RootNode().EndPoint(),
			),
		)
	}

	// Try to resolve as many types as possible
	//p.resolveTypes(&parsedModules)

	return parsedModules, pendingToResolve
}

func (p *Parser) FindVariableDeclarations(node *sitter.Node, moduleName string, currentModule *idx.Module, docId *string, sourceCode []byte) []*idx.Variable {
	query := LocalVarDeclaration
	qc := cst.RunQuery(query, node)

	var variables []*idx.Variable
	found := make(map[string]bool)
	//sourceCode := []byte(doc.Content)
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		// Apply predicates filtering
		m = qc.FilterPredicates(m, sourceCode)
		for _, c := range m.Captures {
			content := c.Node.Content(sourceCode)

			if _, exists := found[content]; !exists {
				found[content] = true
				funcVariables := p.localVariableDeclarationNodeToVariable(c.Node, currentModule, docId, sourceCode)

				variables = append(variables, funcVariables...)
			}
		}
	}

	return variables
}
