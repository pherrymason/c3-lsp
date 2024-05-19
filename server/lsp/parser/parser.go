package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/cst"
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

const VarDeclarationQuery = `(var_declaration
		name: (identifier) @variable_name
	)`
const GlobalVarDeclaration = `(global_declaration) @global_decl`
const ConstantDeclaration = `(const_declaration) @const_decl`
const LocalVarDeclaration = `(func_definition
	body: (macro_func_body (compound_stmt (declaration_stmt) @local) )
 )`
const FunctionDeclarationQuery = `(func_definition) @function_dec`
const EnumDeclaration = `(enum_declaration) @enum_dec`
const FaultDeclaration = `(fault_declaration) @fault_doc`
const StructDeclaration = `(struct_declaration) @struct_dec`
const BitstructDeclaration = `(bitstruct_declaration) @bitstruct_dec`
const DefineDeclaration = `(define_declaration) @def_dec`
const InterfaceDeclaration = `(interface_declaration) @interface_dec`
const MacroDeclaration = `(macro_declaration) @macro_dec`
const ModuleDeclaration = `(module) @module_dec`
const ImportDeclaration = `(import_declaration) @import_dec`

const ModuleQuery = `(source_file ` + ModuleDeclaration + `)`

type Parser struct {
	Logger interface{}
}

type StructWithSubtyping struct {
	strukt  *idx.Struct
	members []string
}

func NewParser(logger interface{}) Parser {
	return Parser{
		Logger: logger,
	}
}

func (p *Parser) ParseSymbols(doc *document.Document) ParsedModules {
	parsedModules := NewParsedModules(doc.URI)
	//fmt.Println(doc.URI, doc.ContextSyntaxTree.RootNode())

	query := `[
(source_file ` + ModuleDeclaration + `)
(source_file ` + ImportDeclaration + `)
(source_file ` + GlobalVarDeclaration + `)
(source_file ` + LocalVarDeclaration + `)
(source_file ` + ConstantDeclaration + `)
(source_file ` + FunctionDeclarationQuery + `)
(source_file ` + DefineDeclaration + `)
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
	sourceCode := []byte(doc.Content)
	//fmt.Println(doc.URI, " ", doc.ContextSyntaxTree.RootNode())
	//fmt.Println(doc.ContextSyntaxTree.RootNode().Content(sourceCode))

	var moduleSymbol *idx.Module
	anonymousModuleName := true
	lastModuleName := ""
	subtyptingToResolve := []StructWithSubtyping{}

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			nodeType := c.Node.Type()
			nodeEndPoint := idx.NewPositionFromTreeSitterPoint(c.Node.EndPoint())
			/*
				fmt.Printf("- %s\n", nodeType)
				if c.Node.HasError() {
					fmt.Printf("Node has error!\n")
					fmt.Printf(nodeType)
				}*/

			if nodeType != "module" {
				moduleSymbol = parsedModules.GetOrInitModule(lastModuleName, doc, anonymousModuleName)
			}

			switch nodeType {
			case "module":
				anonymousModuleName = false
				moduleName, generics := p.nodeToModule(doc, c.Node, sourceCode)
				lastModuleName = moduleName
				moduleSymbol = parsedModules.GetOrInitModule(lastModuleName, doc, false)
				moduleSymbol.SetGenericParameters(generics)

				start := c.Node.StartPoint()
				moduleSymbol.
					SetStartPosition(idx.NewPositionFromTreeSitterPoint(start))

				moduleSymbol.SetStartPosition(idx.NewPositionFromTreeSitterPoint(start))
				moduleSymbol.ChangeModule(lastModuleName)

			case "import_declaration":
				imports := p.nodeToImport(doc, c.Node, sourceCode)
				moduleSymbol.AddImports(imports)

			case "global_declaration":
				variables := p.globalVariableDeclarationNodeToVariable(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddVariables(variables)

			case "func_definition":
				function := p.nodeToFunction(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddFunction(&function)

			case "enum_declaration":
				enum := p.nodeToEnum(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddEnum(&enum)

			case "struct_declaration":
				_struct, membersNeedingSubtypingResolve := p.nodeToStruct(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddStruct(&_struct)
				if len(membersNeedingSubtypingResolve) > 0 {
					subtyptingToResolve = append(subtyptingToResolve,
						StructWithSubtyping{strukt: &_struct, members: membersNeedingSubtypingResolve},
					)
				}

			case "bitstruct_declaration":
				bitstruct := p.nodeToBitStruct(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddBitstruct(&bitstruct)

			case "define_declaration":
				def := p.nodeToDef(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddDef(&def)

			case "const_declaration":
				_const := p.nodeToConstant(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddVariable(&_const)

			case "fault_declaration":
				fault := p.nodeToFault(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddFault(&fault)

			case "interface_declaration":
				interf := p.nodeToInterface(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddInterface(&interf)

			case "macro_declaration":
				macro := p.nodeToMacro(c.Node, moduleSymbol.GetModuleString(), doc.URI, sourceCode)
				moduleSymbol.AddFunction(&macro)
			default:
				// TODO test that module ends up with wrong endPosition
				// when this source code:
				// int variable = 3;
				// fn void main() {
				// int value = 4;
				// v
				// }
				continue
			}

			moduleSymbol.SetEndPosition(nodeEndPoint)
		}
	}

	resolveStructSubtypes(&parsedModules, subtyptingToResolve)

	return parsedModules
}

func (p *Parser) FindVariableDeclarations(node *sitter.Node, moduleName string, docId string, sourceCode []byte) []*idx.Variable {
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
				funcVariables := p.localVariableDeclarationNodeToVariable(c.Node, moduleName, docId, sourceCode)

				for _, variable := range funcVariables {
					variables = append(variables, variable)
				}
			}
		}
	}

	return variables
}

func resolveStructSubtypes(parsedModules *ParsedModules, subtyping []StructWithSubtyping) {
	for _, struktWithSubtyping := range subtyping {
		for _, inlinedMemberName := range struktWithSubtyping.members {

			for _, module := range parsedModules.SymbolsByModule() {
				// Search
				for _, strukt := range module.Structs {
					if strukt.GetName() == inlinedMemberName {
						struktWithSubtyping.strukt.InheritMembersFrom(inlinedMemberName, strukt)
					}
				}
			}
		}
	}
}
