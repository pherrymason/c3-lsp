package ast

import "encoding/json"

// ---------------------------------------------------------------
// Visitor that generates a JSON representation of the AST program
//
// Schema ---------
// All objects representing a node have following properties
//    - "node_type": string.
//    - "doc_pos": array[uint,uint]. Represents start and end positions in the file.
//
// FileObject
//  	- "modules": [ModuleObject]
//		- "name": string
// 		- "node_type": "File"

const PNodeType = "node_type"
const PDocPos = "doc_pos"

type JSONObject map[string]interface{}

type JSONVisitor struct {
	Result map[string]interface{}
}

func serialize_pos(node Node) map[string]interface{} {
	return map[string]interface{}{
		"start": []uint{node.StartPosition().Line, node.StartPosition().Column},
		"end":   []uint{node.EndPosition().Line, node.EndPosition().Column},
	}
}

func (v *JSONVisitor) VisitFile(node *File) {
	modulesVisitor := JSONVisitor{}
	jsonModules := []interface{}{}
	for _, mod := range node.Modules {
		Visit(&mod, &modulesVisitor)
		if modulesVisitor.Result != nil {
			jsonModules = append(jsonModules, modulesVisitor.Result)
		}
	}

	v.Result = map[string]interface{}{
		PNodeType: "File",
		"name":    node.Name,
		"modules": jsonModules,
	}
}

func (v *JSONVisitor) VisitModule(node *Module) {
	var declarations []interface{}
	for _, decl := range node.Declarations {
		v.VisitDeclaration(decl)
		declarations = append(declarations, v.Result)
	}

	v.Result = map[string]interface{}{
		PNodeType:      "Module",
		"name":         node.Name,
		PDocPos:        serialize_pos(node),
		"declarations": declarations,
	}
}

func (v *JSONVisitor) VisitImport(node *Import) {

}

func (v *JSONVisitor) VisitDeclaration(node Declaration) {
	switch node.(type) {
	case *VariableDecl:
		v.VisitVariableDeclaration(node.(*VariableDecl))
	case *FunctionDecl:
		v.VisitFunctionDecl(node.(*FunctionDecl))
	}
}

func (v *JSONVisitor) VisitVariableDeclaration(node *VariableDecl) {
	names := []map[string]interface{}{}
	for _, name := range node.Names {
		names = append(names, map[string]interface{}{
			"name":  name.Name,
			PDocPos: serialize_pos(name),
		})
	}

	typeV := JSONVisitor{}
	Visit(&node.Type, &typeV)

	initV := JSONVisitor{}
	Visit(node.Initializer, &initV)

	v.Result = map[string]interface{}{
		PNodeType:        "VariableDeclaration",
		"names":          names,
		"variable_type":  typeV.Result,
		"initialization": initV.Result,
	}
}

func (v *JSONVisitor) VisitConstDeclaration(node *ConstDecl) {

}

func (v *JSONVisitor) VisitEnumDecl(node *EnumDecl) {

}

func (v *JSONVisitor) VisitStructDecl(node *StructDecl) {

}

func (v *JSONVisitor) VisitFaultDecl(node *FaultDecl) {

}

func (v *JSONVisitor) VisitDefDecl(node *DefDecl) {

}

func (v *JSONVisitor) VisitMacroDecl(node *MacroDecl) {

}

func (v *JSONVisitor) VisitLambdaDeclaration(node *LambdaDeclarationExpr) {

}

func (v *JSONVisitor) VisitFunctionDecl(node *FunctionDecl) {
	typeV := JSONVisitor{}
	Visit(&node.Signature.ReturnType, &typeV)
	var returnType interface{}
	if typeV.Result != nil {
		returnType = typeV.Result
	}

	parameters := []interface{}{}
	for _, p := range node.Signature.Parameters {
		parameters = append(parameters, VisitFunctionParameter(&p))
	}

	bodyV := JSONVisitor{}
	Visit(node.Body, &bodyV)

	v.Result = map[string]interface{}{
		PNodeType:    "FunctionDecl",
		"name":       node.Signature.Name.Name,
		"returnType": returnType,
		"parameters": parameters,
		"body":       bodyV.Result,
	}
}

func (v *JSONVisitor) VisitFunctionParameter(node *FunctionParameter) {

}

func VisitFunctionParameter(node *FunctionParameter) JSONObject {
	return map[string]interface{}{
		PNodeType:       "FunctionParameter",
		"name":          node.Name.Name,
		"parameterType": VisitType(&node.Type),
	}
}

func (v *JSONVisitor) VisitFunctionCall(node *FunctionCall) {

}

func (v *JSONVisitor) VisitInterfaceDecl(node *InterfaceDecl) {

}

func (v *JSONVisitor) VisitCompoundStatement(node *CompoundStmt) {
	visitor := JSONVisitor{}
	statements := []JSONObject{}
	for _, s := range node.Statements {
		Visit(s, &visitor)
		statements = append(statements, visitor.Result)
	}
	visitor.Result = JSONObject{
		"statements": statements,
	}
}

func (v *JSONVisitor) VisitType(node *TypeInfo) {
	v.Result = VisitType(node)
}

func VisitType(node *TypeInfo) JSONObject {

	collection := map[string]interface{}{
		"isCollection": false,
	}

	return map[string]interface{}{
		PNodeType:    "Type",
		"name":       node.Identifier.Name,
		"builtin":    node.BuiltIn,
		"optional":   node.Optional,
		"collection": collection,
	}
}

func (v *JSONVisitor) VisitIdentifier(node *Ident) {

}

func (v *JSONVisitor) VisitBinaryExpression(node *BinaryExpression) {

}

func (v *JSONVisitor) VisitIfStatement(node *IfStmt) {

}

func (v *JSONVisitor) VisitIntegerLiteral(node *IntegerLiteral) {
	v.Result = map[string]interface{}{
		PNodeType: "IntegerLiteral",
		"value":   node.Value,
	}
}

func (v *JSONVisitor) ToJSONString() (string, error) {
	// Utilizamos json.MarshalIndent para obtener un JSON formateado con indentación
	jsonBytes, err := json.MarshalIndent(v.Result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
