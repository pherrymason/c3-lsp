package lsp

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	"fmt"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"unsafe"
)

const VarDeclarationQuery = `(var_declaration
		name: (identifier) @variable_name
	)`
const FunctionDeclarationQuery = `(function_declaration
        name: (identifier) @function_name
        body: (_) @body
    )`
const EnumDeclaration = `(enum_declaration) @enum_dec`
const StructDeclaration = `(struct_declaration) @struct_dec`

type Parser struct {
	logger commonlog.Logger
}

func getParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(getLanguage())

	return parser
}

func getLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_c3())
	return sitter.NewLanguage(ptr)
}

func GetParsedTree(source []byte) *sitter.Tree {
	parser := getParser()
	n := parser.Parse(nil, source)

	return n
}

func GetParsedTreeFromString(source string) *sitter.Tree {
	sourceCode := []byte(source)
	parser := getParser()
	n := parser.Parse(nil, sourceCode)

	return n
}

func (p *Parser) ExtractSymbols(doc *Document) idx.Function {
	query := `[
	(source_file ` + VarDeclarationQuery + `)
	(source_file ` + EnumDeclaration + `)	
	(source_file ` + StructDeclaration + `)
	` + FunctionDeclarationQuery + `]`

	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())
	sourceCode := []byte(doc.Content)

	functionsMap := make(map[string]*idx.Function)
	scopeTree := idx.NewAnonymousScopeFunction(
		"main",
		doc.URI,
		idx.NewRangeFromSitterPositions(doc.parsedTree.RootNode().StartPoint(), doc.parsedTree.RootNode().EndPoint()),
		protocol.CompletionItemKindModule, // Best value found
	)

	//var tempEnum *idx.Enum

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
					variable := p.nodeToVariable(doc, c.Node, sourceCode, content)
					scopeTree.AddVariables([]idx.Variable{
						variable,
					})
				case "function_declaration":
					identifier := idx.NewFunction(
						content,
						c.Node.Parent().ChildByFieldName("return_type").Content(sourceCode),
						doc.URI, idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), protocol.CompletionItemKindFunction)
					functionsMap[content] = &identifier
					scopeTree.AddFunction(&identifier)
				}
			} else if nodeType == "enum_declaration" {
				enum := p.nodeToEnum(doc, c.Node, sourceCode)
				scopeTree.AddEnum(&enum)
			} else if nodeType == "struct_declaration" {
				_struct := p.nodeToStruct(doc, c.Node, sourceCode)
				scopeTree.AddStruct(_struct)
			} else if nodeType == "compound_statement" {
				variables := p.FindVariableDeclarations(doc, c.Node)

				// TODO Previous node has the info about which function is belongs to.
				idNode := c.Node.Parent().ChildByFieldName("name")
				functionName := idNode.Content(sourceCode)

				function, ok := functionsMap[functionName]
				if !ok {
					panic(fmt.Sprint("Could not find definition for ", functionName))
				}
				function.SetEndRange(idx.NewPositionFromSitterPoint(c.Node.EndPoint()))
				function.AddVariables(variables)
			}
			//fmt.Println(c.Node.String(), content)
		}
	}

	return scopeTree
}

func (p *Parser) nodeToVariable(doc *Document, node *sitter.Node, sourceCode []byte, content string) idx.Variable {
	typeNode := node.PrevSibling()
	typeNodeContent := typeNode.Content(sourceCode)
	variable := idx.NewVariable(
		content,
		typeNodeContent,
		doc.URI,
		idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()),
		idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()), // TODO Should this include the var type range?
		protocol.CompletionItemKindVariable,
	)

	return variable
}

func (p *Parser) nodeToStruct(doc *Document, node *sitter.Node, sourceCode []byte) idx.Struct {
	nameNode := node.Child(1)
	name := nameNode.Content(sourceCode)
	// TODO parse attributes
	bodyNode := node.Child(2)

	fields := make([]idx.StructMember, 0)

	for i := uint32(0); i < bodyNode.ChildCount(); i++ {
		child := bodyNode.Child(int(i))
		switch child.Type() {
		case "field_declaration":
			fieldName := child.ChildByFieldName("name").Content(sourceCode)
			fieldType := child.ChildByFieldName("type").Content(sourceCode)
			fields = append(fields, idx.NewStructMember(fieldName, fieldType, idx.NewRangeFromSitterPositions(child.StartPoint(), child.EndPoint())))

		case "field_struct_declaration":
		case "field_union_declaration":
		}
	}

	_struct := idx.NewStruct(name, fields, doc.URI, idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()))

	return _struct
}

func (p *Parser) nodeToEnum(doc *Document, node *sitter.Node, sourceCode []byte) idx.Enum {
	// TODO parse attributes
	nodesCount := node.ChildCount()
	nameNode := node.Child(1)

	baseType := ""
	bodyIndex := int(nodesCount - 1)

	//	p.logger.Debug(fmt.Sprint(node.Content(sourceCode), ": Child count:", nodesCount))
	enum := idx.NewEnum(
		nameNode.Content(sourceCode),
		baseType,
		[]idx.Enumerator{},
		idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()),
		doc.URI,
	)

	enumeratorsNode := node.Child(bodyIndex)

	for i := uint32(0); i < enumeratorsNode.ChildCount(); i++ {
		enumeratorNode := enumeratorsNode.Child(int(i))
		if enumeratorNode.Type() == "enumerator" {
			enum.RegisterEnumerator(
				enumeratorNode.Child(0).Content(sourceCode),
				"",
				idx.NewRangeFromSitterPositions(enumeratorNode.StartPoint(), enumeratorNode.EndPoint()),
			)
		}
	}

	return enum
}

func (p *Parser) FindVariableDeclarations(doc *Document, node *sitter.Node) []idx.Variable {
	query := VarDeclarationQuery
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, node)

	var identifiers []idx.Variable
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
				variable := p.nodeToVariable(doc, c.Node, sourceCode, content)
				identifiers = append(identifiers, variable)
			}
		}
	}

	return identifiers
}

func (p *Parser) FindFunctionDeclarations(doc *Document) []idx.Indexable {
	query := FunctionDeclarationQuery //`(function_declaration name: (identifier) @function_name)`
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())

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
				identifier := idx.NewFunction(content, "", doc.URI, idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), protocol.CompletionItemKindFunction)

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}
