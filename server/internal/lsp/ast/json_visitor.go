package ast

import "encoding/json"

type JSONVisitor struct {
	Result map[string]interface{}
}

func serialize_pos(node ASTNode) map[string]interface{} {
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
		"type":    "File",
		"name":    node.Name,
		"modules": jsonModules,
	}
}

func (v *JSONVisitor) VisitModule(node *Module) {
	declarationsV := JSONVisitor{}
	declarations := []interface{}{}
	for _, decl := range node.Declarations {
		Visit(decl, &declarationsV)
		if declarationsV.Result != nil {
			declarations = append(declarations, declarationsV.Result)
		}
	}

	v.Result = map[string]interface{}{
		"type":         "Module",
		"name":         node.Name,
		"pos":          serialize_pos(node),
		"declarations": declarations,
	}
}

func (v *JSONVisitor) VisitImport(node *Import) {

}

func (v *JSONVisitor) VisitVariableDeclaration(node *VariableDecl) {
	names := []map[string]interface{}{}
	for _, name := range node.Names {
		names = append(names, map[string]interface{}{
			"name": name.Name,
			"pos":  serialize_pos(name),
		})
	}

	typeV := JSONVisitor{}
	Visit(node.Type, &typeV)

	initV := JSONVisitor{}
	Visit(node.Initializer, &initV)

	v.Result = map[string]interface{}{
		"type":           "VariableDeclaration",
		"names":          names,
		"kind":           typeV.Result,
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

func (v *JSONVisitor) VisitLambdaDeclaration(node *LambdaDeclaration) {

}

func (v *JSONVisitor) VisitFunctionDecl(node *FunctionDecl) {

}

func (v *JSONVisitor) VisitFunctionCall(node *FunctionCall) {

}

func (v *JSONVisitor) VisitInterfaceDecl(node *InterfaceDecl) {

}

func (v *JSONVisitor) VisitType(node *TypeInfo) {

	collection := map[string]interface{}{
		"isCollection": false,
	}

	v.Result = map[string]interface{}{
		"type":       "Type",
		"name":       node.Identifier.Name,
		"builtin":    node.BuiltIn,
		"optional":   node.Optional,
		"collection": collection,
	}
}

func (v *JSONVisitor) VisitIdentifier(node *Identifier) {

}

func (v *JSONVisitor) VisitBinaryExpression(node *BinaryExpression) {

}

func (v *JSONVisitor) VisitIfStatement(node *IfStatement) {

}

func (v *JSONVisitor) VisitIntegerLiteral(node *IntegerLiteral) {
	v.Result = map[string]interface{}{
		"type":  "IntegerLiteral",
		"value": node.Value,
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
