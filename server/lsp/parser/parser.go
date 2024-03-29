package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/cst"
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"path/filepath"
	"regexp"
	"strings"
)

const ModuleQuery = `(source_file (module_declaration) @module)`
const VarDeclarationQuery = `(var_declaration
		name: (identifier) @variable_name
	)`
const FunctionDeclarationQuery = `(function_declaration) @function_dec`
const EnumDeclaration = `(enum_declaration) @enum_dec`
const StructDeclaration = `(struct_declaration) @struct_dec`
const DefineDeclaration = `(define_declaration) @def_dec`

type Parser struct {
	Logger interface{}
}

func NewParser(logger interface{}) Parser {
	return Parser{
		Logger: logger,
	}
}

func (p *Parser) ExtractSymbols(doc *document.Document) idx.Function {
	parsedSymbols := NewParsedSymbols()

	query := `[
	(source_file ` + VarDeclarationQuery + `)
	(source_file ` + EnumDeclaration + `)	
	(source_file ` + StructDeclaration + `)
	(source_file ` + DefineDeclaration + `)
	` + FunctionDeclarationQuery + `]`
	/*
		q, err := sitter.NewQuery([]byte(query), cst.GetLanguage())
		if err != nil {
			panic(err)
		}
		qc := sitter.NewQueryCursor()
		qc.Exec(q, doc.ContextSyntaxTree.RootNode())*/
	qc := cst.RunQuery(query, doc.ContextSyntaxTree.RootNode())
	sourceCode := []byte(doc.Content)

	scopeTree := idx.NewAnonymousScopeFunction("main", doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(doc.ContextSyntaxTree.RootNode().StartPoint(), doc.ContextSyntaxTree.RootNode().EndPoint()), protocol.CompletionItemKindModule)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			content := c.Node.Content(sourceCode)
			nodeType := c.Node.Type()
			if nodeType == "identifier" {
				switch c.Node.Parent().Type() {
				case "var_declaration":
					variable := p.nodeToVariable(doc, c.Node.Parent(), c.Node, sourceCode, content)
					scopeTree.AddVariable(variable)
				}
			} else if nodeType == "function_declaration" {
				function := p.nodeToFunction(doc, c.Node, sourceCode)
				scopeTree.AddFunction(function)
			} else if nodeType == "enum_declaration" {
				enum := p.nodeToEnum(doc, c.Node, sourceCode)
				scopeTree.AddEnum(enum)
			} else if nodeType == "struct_declaration" {
				_struct := p.nodeToStruct(doc, c.Node, sourceCode)
				scopeTree.AddStruct(_struct)
			} else if nodeType == "define_declaration" {
				def := p.nodeToDef(doc, c.Node, sourceCode)
				scopeTree.AddDef(def)
			}
		}
	}

	parsedSymbols.scopedFunction = scopeTree

	return scopeTree
}

func (p *Parser) FindVariableDeclarations(doc *document.Document, node *sitter.Node) []idx.Variable {
	query := VarDeclarationQuery
	/*
		q, err := sitter.NewQuery([]byte(query), cst.GetLanguage())
		if err != nil {
			panic(err)
		}

		qc := sitter.NewQueryCursor()
		qc.Exec(q, node)
	*/
	qc := cst.RunQuery(query, node)

	var variables []idx.Variable
	found := make(map[string]bool)
	sourceCode := []byte(doc.Content)
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
				variable := p.nodeToVariable(doc, c.Node.Parent(), c.Node, sourceCode, content)
				variables = append(variables, variable)
			}
		}
	}

	return variables
}

func (p *Parser) FindFunctionDeclarations(doc *document.Document) []idx.Indexable {
	query := FunctionDeclarationQuery //`(function_declaration name: (identifier) @function_name)`
	/*q, err := sitter.NewQuery([]byte(query), cst.GetLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.ContextSyntaxTree.RootNode())*/
	qc := cst.RunQuery(query, doc.ContextSyntaxTree.RootNode())

	var identifiers []idx.Indexable
	found := make(map[string]bool)
	sourceCode := []byte(doc.Content)
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		// Apply predicates filtering
		m = qc.FilterPredicates(m, sourceCode)
		for _, c := range m.Captures {
			content := c.Node.Content(sourceCode)
			c.Node.Parent().Type()
			if _, exists := found[content]; !exists {
				found[content] = true
				identifier := idx.NewFunction(content, "", []string{}, doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), protocol.CompletionItemKindFunction)

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}

func (p *Parser) ExtractModuleName(doc *document.Document) string {
	/*
		q, err := sitter.NewQuery([]byte(ModuleQuery), cst.GetLanguage())
		if err != nil {
			panic(err)
		}
		qc := sitter.NewQueryCursor()
		qc.Exec(q, doc.ContextSyntaxTree.RootNode())
	*/
	qc := cst.RunQuery(ModuleQuery, doc.ContextSyntaxTree.RootNode())

	sourceCode := []byte(doc.Content)

	var moduleName string
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			moduleName = c.Node.Content(sourceCode)
			moduleName = moduleName
		}
	}

	if moduleName == "" {
		moduleName = filepath.Base(doc.URI)
		moduleName = strings.TrimSuffix(moduleName, filepath.Ext(moduleName))
		regexpPattern := regexp.MustCompile(`[^_0-9a-z]`)
		moduleName = regexpPattern.ReplaceAllString(moduleName, "_")
	}

	return moduleName
}
