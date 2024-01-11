package lsp

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_c3();
import "C"
import (
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
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

//	name: (type_identifier) @enum_name
//	body: (_) @body
//)`

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

func FindSymbols(doc *Document) indexables.Function {
	query := `[
	(source_file ` + VarDeclarationQuery + `)
	(source_file ` + EnumDeclaration + `)		
	` + FunctionDeclarationQuery + `]`

	//fmt.Println(doc.parsedTree.RootNode())

	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())
	sourceCode := []byte(doc.Content)

	functionsMap := make(map[string]*indexables.Function)
	scopeTree := indexables.NewAnonymousScopeFunction(
		"main",
		doc.URI,
		treeSitterPoints2Range(doc.parsedTree.RootNode().StartPoint(), doc.parsedTree.RootNode().EndPoint()),
		protocol.CompletionItemKindModule, // Best value found
	)

	//var tempEnum *indexables.Enum

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
					variable := nodeToVariable(doc, c.Node, sourceCode, content)
					scopeTree.AddVariables([]indexables.Variable{
						variable,
					})
				case "function_declaration":
					identifier := indexables.NewFunction(
						content,
						doc.URI,
						treeSitterPoints2Range(c.Node.StartPoint(), c.Node.EndPoint()),
						treeSitterPoints2Range(c.Node.StartPoint(), c.Node.EndPoint()),
						protocol.CompletionItemKindFunction)
					functionsMap[content] = &identifier
					scopeTree.AddFunction(&identifier)
				}
			} else if c.Node.Type() == "enum_declaration" {
				enum := nodeToEnum(doc, c.Node, sourceCode)
				scopeTree.AddEnum(&enum)
			} else if nodeType == "compound_statement" {
				variables := FindVariableDeclarations(doc, c.Node)

				// TODO Previous node has the info about which function is belongs to.
				idNode := c.Node.Parent().ChildByFieldName("name")
				functionName := idNode.Content(sourceCode)

				function, ok := functionsMap[functionName]
				if !ok {
					panic(fmt.Sprint("Could not find definition for ", functionName))
				}
				function.SetEndRange(treeSitterPoint2Position(c.Node.EndPoint()))
				function.AddVariables(variables)
			}
			//fmt.Println(c.Node.String(), content)
		}
	}

	return scopeTree
}

func nodeToVariable(doc *Document, node *sitter.Node, sourceCode []byte, content string) indexables.Variable {
	typeNode := node.PrevSibling()
	typeNodeContent := typeNode.Content(sourceCode)
	variable := indexables.NewVariable(
		content,
		typeNodeContent,
		doc.URI,
		treeSitterPoints2Range(node.StartPoint(), node.EndPoint()),
		treeSitterPoints2Range(node.StartPoint(), node.EndPoint()), // TODO Should this include the var type range?
		protocol.CompletionItemKindVariable,
	)

	return variable
}

func nodeToEnum(doc *Document, node *sitter.Node, sourceCode []byte) indexables.Enum {
	nodesCount := node.ChildCount()
	nameNode := node.Child(1)

	baseType := ""
	bodyIndex := 3
	if nodesCount == 4 {
		// Enum without base_type
	} else {
		// Enum with base_type
		baseType = "?"
		bodyIndex = 4
	}

	enumeratorsNode := node.Child(bodyIndex)
	enumerators := []indexables.Enumerator{}
	for i := uint32(0); i < enumeratorsNode.ChildCount(); i++ {
		enumeratorNode := enumeratorsNode.Child(int(i))
		if enumeratorNode.Type() == "enumerator" {
			enumerators = append(
				enumerators,
				indexables.NewEnumerator(
					enumeratorNode.Child(0).Content(sourceCode),
					"",
					treeSitterPoints2Range(enumeratorNode.StartPoint(), enumeratorNode.EndPoint()),
				),
			)
		}
	}

	enum := indexables.NewEnum(
		nameNode.Content(sourceCode),
		baseType,
		enumerators,
		treeSitterPoints2Range(nameNode.StartPoint(), nameNode.EndPoint()),
		treeSitterPoints2Range(node.StartPoint(), node.EndPoint()),
		doc.URI,
	)

	return enum
}

func FindIdentifiers(doc *Document) []indexables.Indexable {
	//variableIdentifiers := FindVariableDeclarations(doc, doc.parsedTree.RootNode())
	functionIdentifiers := FindFunctionDeclarations(doc)

	var elements []indexables.Indexable
	//elements = append(elements, variableIdentifiers...)
	elements = append(elements, functionIdentifiers...)

	return elements
}

func FindVariableDeclarations(doc *Document, node *sitter.Node) []indexables.Variable {
	query := VarDeclarationQuery
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, node)

	var identifiers []indexables.Variable
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
				variable := nodeToVariable(doc, c.Node, sourceCode, content)
				identifiers = append(identifiers, variable)
			}
		}
	}

	return identifiers
}

func FindFunctionDeclarations(doc *Document) []indexables.Indexable {
	query := FunctionDeclarationQuery //`(function_declaration name: (identifier) @function_name)`
	q, err := sitter.NewQuery([]byte(query), getLanguage())
	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, doc.parsedTree.RootNode())

	var identifiers []indexables.Indexable
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
				identifier := indexables.NewFunction(
					content,
					doc.URI,
					//protocol.Position{c.Node.StartPoint().Row, c.Node.StartPoint().Column},
					treeSitterPoints2Range(c.Node.StartPoint(), c.Node.EndPoint()),
					treeSitterPoints2Range(c.Node.StartPoint(), c.Node.EndPoint()),
					protocol.CompletionItemKindFunction)

				identifiers = append(identifiers, identifier)
			}
		}
	}

	return identifiers
}
