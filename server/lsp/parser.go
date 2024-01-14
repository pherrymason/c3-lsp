package lsp

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"unsafe"
)

const VarDeclarationQuery = `(var_declaration
		name: (identifier) @variable_name
	)`
const FunctionDeclarationQuery = `(function_declaration) @function_dec`
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

	//functionsMap := make(map[string]*idx.Function)
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
					variable := p.nodeToVariable(doc, c.Node.Parent(), c.Node, sourceCode, content)
					scopeTree.AddVariables([]idx.Variable{
						variable,
					})
				}
			} else if nodeType == "function_declaration" {
				function := p.nodeToFunction(doc, c.Node, sourceCode)
				scopeTree.AddFunction(&function)
			} else if nodeType == "enum_declaration" {
				enum := p.nodeToEnum(doc, c.Node, sourceCode)
				scopeTree.AddEnum(&enum)
			} else if nodeType == "struct_declaration" {
				_struct := p.nodeToStruct(doc, c.Node, sourceCode)
				scopeTree.AddStruct(_struct)
			}
		}
	}

	return scopeTree
}

func (p *Parser) nodeToVariable(doc *Document, variableNode *sitter.Node, identifierNode *sitter.Node, sourceCode []byte, content string) idx.Variable {
	typeNode := identifierNode.PrevSibling()
	typeNodeContent := typeNode.Content(sourceCode)
	variable := idx.NewVariable(
		content,
		typeNodeContent,
		doc.URI,
		idx.NewRangeFromSitterPositions(identifierNode.StartPoint(), identifierNode.EndPoint()), idx.NewRangeFromSitterPositions(variableNode.StartPoint(), variableNode.EndPoint()))

	return variable
}

func (p *Parser) nodeToFunction(doc *Document, node *sitter.Node, sourceCode []byte) idx.Function {

	nameNode := node.ChildByFieldName("name")

	// Extract function arguments
	arguments := []idx.Variable{}
	parameters := node.ChildByFieldName("parameters")
	for i := uint32(0); i < parameters.ChildCount(); i++ {
		argNode := parameters.Child(int(i))
		switch argNode.Type() {
		case "parameter":
			arguments = append(arguments, p.nodeToArgument(doc, argNode, sourceCode))
		}
	}

	var argumentIds []string
	for _, arg := range arguments {
		argumentIds = append(argumentIds, arg.GetName())
	}

	symbol := idx.NewFunction(nameNode.Content(sourceCode), node.ChildByFieldName("return_type").Content(sourceCode), argumentIds, doc.URI, idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()), idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()), protocol.CompletionItemKindFunction)

	variables := p.FindVariableDeclarations(doc, node)
	variables = append(arguments, variables...)

	// TODO Previous node has the info about which function is belongs to.

	symbol.AddVariables(variables)

	return symbol
}

// nodeToArgument Very similar to nodeToVariable, but arguments have optional identifiers (for example when using `self` for struct methods)
func (p *Parser) nodeToArgument(doc *Document, argNode *sitter.Node, sourceCode []byte) idx.Variable {
	var identifier string = ""
	var idRange idx.Range
	var argType string = ""
	if argNode.ChildCount() == 2 {
		if argNode.Child(0).Type() == "identifier" {
			// argument without type
			idNode := argNode.Child(0)
			identifier = idNode.Content(sourceCode)
			idRange = idx.NewRangeFromSitterPositions(idNode.StartPoint(), idNode.EndPoint())
		} else {
			// first node is type
			argType = argNode.Child(0).Content(sourceCode)

			idNode := argNode.Child(1)
			identifier = idNode.Content(sourceCode)
			idRange = idx.NewRangeFromSitterPositions(idNode.StartPoint(), idNode.EndPoint())
		}
	} else if argNode.ChildCount() == 1 {
		idNode := argNode.Child(0)
		identifier = idNode.Content(sourceCode)
		idRange = idx.NewRangeFromSitterPositions(idNode.StartPoint(), idNode.EndPoint())
	}

	variable := idx.NewVariable(
		identifier,
		argType,
		doc.URI,
		idRange, idx.NewRangeFromSitterPositions(argNode.StartPoint(), argNode.EndPoint()))

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
				variable := p.nodeToVariable(doc, c.Node.Parent(), c.Node, sourceCode, content)
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
				identifier := idx.NewFunction(content, "", []string{}, doc.URI, idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), idx.NewRangeFromSitterPositions(c.Node.StartPoint(), c.Node.EndPoint()), protocol.CompletionItemKindFunction)

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}
