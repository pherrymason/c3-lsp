package factory

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg"
	"log"

	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

func (c *ASTConverter) convert_statement(node *sitter.Node, source []byte) ast.Statement {
	if pkg.SliceContains(ignoreStatements[:], node.Type()) {
		return &ast.EmptyNode{}
	}

	dd := c.anyOf("statement", []NodeRule{
		NodeOfType("compound_stmt"),
		NodeOfType("expr_stmt"),
		NodeOfType("declaration_stmt"),
		NodeOfType("var_stmt"),
		NodeOfType("return_stmt"),
		NodeOfType("continue_stmt"),
		NodeOfType("break_stmt"),
		NodeOfType("switch_stmt"),
		NodeOfType("nextcase_stmt"),
		NodeOfType("if_stmt"),
		NodeOfType("for_stmt"),
		NodeOfType("foreach_stmt"),
		NodeOfType("while_stmt"),
		NodeOfType("do_stmt"),
		NodeOfType("defer_stmt"),
		NodeOfType("assert_stmt"),
		NodeOfType("asm_block_stmt"),
		NodeOfType("ct_echo_stmt"),
		NodeOfType("ct_assert_stmt"),
		NodeOfType("ct_if_stmt"),
		NodeOfType("ct_switch_stmt"),
		NodeOfType("ct_foreach_stmt"),
		NodeOfType("ct_for_stmt"),
	}, node, source, false)

	if dd == nil {
		log.Fatalf("Could not convert_statement. Node TypeDescription: %s. Content: %s\n----- %s\n", node.Type(), node.Content(source), node)
	}

	return dd.(ast.Statement)
}

func (c *ASTConverter) convert_compound_stmt(node *sitter.Node, source []byte) ast.Statement {
	cmpStatement := &ast.CompoundStmt{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Statements:     []ast.Statement{},
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() != "{" && n.Type() != "}" {
			cmpStatement.Statements = append(
				cmpStatement.Statements,
				c.convert_statement(n, source),
			)
		}
	}

	return cmpStatement
}

func (c *ASTConverter) convert_return_stmt(node *sitter.Node, source []byte) ast.Statement {
	expr := option.None[ast.Expression]()

	argNode := node.Child(0).NextSibling()
	if argNode.Type() != ";" {
		expr = option.Some(c.convert_expression(argNode, source).(ast.Expression))
	}

	return &ast.ReturnStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Return:         expr,
	}
}

func (c *ASTConverter) convert_split_declaration_stmt(node *sitter.Node, source []byte) ast.Statement {
	return c.convert_declaration_stmt(node.Parent(), source)
}

func (c *ASTConverter) convert_declaration_stmt(node *sitter.Node, source []byte) ast.Statement {
	if node.Type() == "const_declaration" {
		return &ast.DeclarationStmt{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
			Decl:           c.convert_const_declaration(node, source),
		}
	}

	isStatic := false
	isTlocal := false

	varDecl := &ast.GenDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Token:          ast.VAR,
		//Spec:           &ast.ValueSpec{},
	}
	end := false
	valueSpec := &ast.ValueSpec{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}
	for i := 0; i < int(node.ChildCount()) && !end; i++ {
		n := node.Child(i)
		//debugNode(n, source, "dd")

		switch n.Type() {
		case "local_decl_storage":
			if n.Content(source) == "static" {
				isStatic = true
			} else {
				isTlocal = true
			}

		case "type":
			Type := c.convert_type(n, source)
			Type.Static = isStatic
			Type.TLocal = isTlocal
			valueSpec.Type = Type

		case "local_decl_after_type":
			//valueSpec.Names =
			ident := ast.NewIdentifierBuilder().
				WithId(c.getNextID()).
				WithName(n.ChildByFieldName("name").Content(source)).
				WithSitterPos(n.ChildByFieldName("name")).
				BuildPtr()
			valueSpec.Names = append(valueSpec.Names, ident)

			right := n.ChildByFieldName("right")
			if right != nil {
				valueSpec.Value = c.convert_expression(right, source).(ast.Expression)
			}
			end = true
		}
	}
	varDecl.Spec = valueSpec

	return &ast.DeclarationStmt{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Decl:           varDecl,
	}
}

func (c *ASTConverter) convert_continue_stmt(node *sitter.Node, source []byte) ast.Statement {
	label := option.None[string]()
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return &ast.ContinueStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Label:          label,
	}
}

func (c *ASTConverter) convert_break_stmt(node *sitter.Node, source []byte) ast.Statement {
	label := option.None[string]()
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return &ast.BreakStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Label:          label,
	}
}

func (c *ASTConverter) convert_switch_stmt(node *sitter.Node, source []byte) ast.Statement {
	label := option.None[string]()
	var cases []ast.SwitchCase
	var defaultStatement []ast.Statement

	body := node.ChildByFieldName("body")
	for i := 0; i < int(body.ChildCount()); i++ {
		n := body.Child(i)
		if n.Type() == "case_stmt" {
			conditionNode := n.ChildByFieldName("value")
			var caseValue ast.Statement
			if conditionNode.Type() == "case_range" {
				caseValue = &ast.SwitchCaseRange{
					NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), conditionNode),
					Start:          c.convert_expression(conditionNode.Child(0), source).(ast.Expression),
					End:            c.convert_expression(conditionNode.Child(2), source).(ast.Expression),
				}
			} else if conditionNode.Type() == "type" {
				caseValue = &ast.ExpressionStmt{
					NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), conditionNode),
					Expr:           c.convert_type(conditionNode, source),
				}
			} else {
				caseValue = &ast.ExpressionStmt{
					NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), conditionNode),
					Expr:           c.convert_expression(conditionNode, source).(ast.Expression),
				}

			}

			colon := conditionNode.NextSibling()
			ns := colon.NextSibling()
			var statements []ast.Statement
			for {
				if ns == nil {
					break
				}
				statements = append(statements, c.convert_statement(ns, source))
				ns = ns.NextSibling()
			}

			cases = append(cases, ast.SwitchCase{
				NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), n),
				Value:          caseValue,
				Statements:     statements,
			})
		} else {
			for d := 0; d < int(n.ChildCount()); d++ {
				dn := n.Child(d)
				if dn.Type() != "default" && dn.Type() != ":" {
					defaultStatement = append(defaultStatement, c.convert_statement(dn, source))
				}
			}
		}
	}

	var conditionExpression []*ast.DeclOrExpr
	conditionNode := node.ChildByFieldName("condition")
	if conditionNode != nil {
		ConvertDebug = false
		conditionExpression = c.convert_paren_conditions(conditionNode, source)
		ConvertDebug = false
	}

	return &ast.SwitchStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Label:          label,
		Condition:      conditionExpression,
		Cases:          cases,
		Default:        defaultStatement,
	}
}

func (c *ASTConverter) convert_nextcase_stmt(node *sitter.Node, source []byte) ast.Statement {
	label := option.None[string]()
	var value ast.Expression
	targetNode := node.ChildByFieldName("target")
	if targetNode != nil {
		found := c.anyOf("nextcase_stmt", []NodeRule{
			NodeTryConversionFunc("_expr"),
			NodeOfType("type"),
			NodeOfType("default"),
		}, targetNode, source, false)

		if found != nil {
			value = found.(ast.Expression)
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return &ast.Nextcase{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Label:          label,
		Value:          value,
	}
}

func (c *ASTConverter) convert_if_stmt(node *sitter.Node, source []byte) ast.Statement {
	conditions := c.convert_paren_conditions(node.ChildByFieldName("condition"), source)

	stmt := &ast.IfStmt{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Label:          option.None[string](),
		Condition:      conditions,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "label":
			stmt.Label = option.Some(n.Child(0).Content(source))
		case "else_part":
			elseStmt := c.convert_statement(n.Child(1), source)
			switch elseStmt.(type) {
			case *ast.CompoundStmt:
				if len(elseStmt.(*ast.CompoundStmt).Statements) == 0 {
					elseStmt = nil
				}
			}
			stmt.Else = ast.ElseStatement{
				NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), n),
				Statement:      elseStmt,
			}
		}
	}
	//ifBody := node.Child(int(node.ChildCount()) - 1)
	ifBody := node.ChildByFieldName("body")
	bodyStmt := c.convert_statement(ifBody, source)
	switch bodyStmt.(type) {
	case *ast.CompoundStmt:
		if len(bodyStmt.(*ast.CompoundStmt).Statements) == 0 {
			bodyStmt = nil
		}
	}
	stmt.Statement = bodyStmt

	return stmt
}

/*
	    choice(
		  choice($._try_unwrap_chain, $.catch_unwrap),
		  seq(
		    commaSep1($._decl_or_expr),
		    optional(seq(',', choice($._try_unwrap_chain, $.catch_unwrap))),
		  ),
		)
*/
func (c *ASTConverter) convert_paren_conditions(node *sitter.Node, source []byte) []*ast.DeclOrExpr {
	return c.convert_conditions(node.Child(1), source)
}

func (c *ASTConverter) convert_conditions(node *sitter.Node, source []byte) []*ast.DeclOrExpr {
	var conditions []*ast.DeclOrExpr

	// Option 1: try_unwrap_chain
	if node.Type() == "try_unwrap_chain" {

	} else if node.Type() == "catch_unwrap" { // Option 2: catch_unwrap

	} else {
		// Option 3:
		//		(decl_or_expr),* + optional(, try_unwrap | catch_unwrap)
		nodes := commaSep(c.convert_decl_or_expression, node, source)
		for _, n := range nodes {
			conditions = append(conditions, n.(*ast.DeclOrExpr))
		}
	}

	return conditions
}

func (c *ASTConverter) convert_local_declaration_after_type(node *sitter.Node, source []byte) ast.Declaration {
	var init ast.Expression

	if node.Child(1).Type() == "attributes" {
		// TODO Get attributes
	}

	right := node.ChildByFieldName("right")
	if right != nil {
		init = c.convert_expression(right, source).(ast.Expression)
	}
	/*
		varDec2l := &ast.VariableDecl{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(node),
			Names: []*ast.Ident{
				ast.NewIdentifierBuilder().
					WithName(node.ChildByFieldName("name").Content(source)).
					WithSitterPos(node.ChildByFieldName("name")).
					BuildPtr(),
			},
			Initializer: init,
		}*/
	varDecl := &ast.GenDecl{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Token:          ast.VAR,
		Spec: &ast.ValueSpec{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
			Names: []*ast.Ident{
				ast.NewIdentifierBuilder().
					WithId(c.getNextID()).
					WithName(node.ChildByFieldName("name").Content(source)).
					WithSitterPos(node.ChildByFieldName("name")).
					BuildPtr(),
			},
			Value: init,
			// TODO Isn't this missing the Type field???
		},
	}

	return varDecl
}

func (c *ASTConverter) convert_for_stmt(node *sitter.Node, source []byte) ast.Statement {
	forStmt := &ast.ForStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "for_cond" {
			initNode := n.ChildByFieldName("initializer")
			if initNode != nil {
				forStmt.Initializer = c.convert_comma_decl_or_expression(initNode, source)
			}

			condNode := n.ChildByFieldName("condition")
			if condNode != nil {
				forStmt.Condition = c.convert_conditions(condNode, source)[0]
			}

			updateNode := n.ChildByFieldName("update")
			if updateNode != nil {
				forStmt.Update = c.convert_comma_decl_or_expression(updateNode, source)
			}
		}
	}

	nodeBody := node.ChildByFieldName("body")
	forStmt.Body = c.convert_statement(nodeBody, source)

	return forStmt
}

func (c *ASTConverter) convert_comma_decl_or_expression(node *sitter.Node, source []byte) []*ast.DeclOrExpr {
	nodes := commaSep(c.convert_decl_or_expression, node.Child(0), source)

	var stmts []*ast.DeclOrExpr
	for _, n := range nodes {
		stmts = append(stmts, n.(*ast.DeclOrExpr))
	}
	return stmts
}

// This takes anon nodes
func (c *ASTConverter) convert_decl_or_expression(node *sitter.Node, source []byte) ast.Node {
	rules := []NodeRule{
		NodeOfType("var_decl"),
		NodeSiblingsWithSequenceOf([]NodeRule{
			NodeOfType("type"), NodeOfType("local_decl_after_type"),
		}, "split_declaration_stmt"),
		NodeAnonymous("_expr"),
	}
	found := c.anyOf("decl_or_expression", rules, node, source, false)

	wrapperNode := node
	if node.Type() == "type" {
		wrapperNode = node.Parent()
	}

	switch found.(type) {
	case ast.Expression:
		return &ast.DeclOrExpr{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), wrapperNode),
			Expr:           found.(ast.Expression),
		}

	case ast.Declaration:
		return &ast.DeclOrExpr{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), wrapperNode),
			Decl:           found.(ast.Declaration),
		}

	case ast.Statement:
		return &ast.DeclOrExpr{
			NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), wrapperNode),
			Stmt:           found.(ast.Statement),
		}

	default:
		panic("decl_or_expression: did not find found type.")
	}
}

func (c *ASTConverter) convert_foreach_stmt(node *sitter.Node, source []byte) ast.Statement {
	stmt := &ast.ForeachStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}
	foreachVar := ast.ForeachValue{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "foreach_cond" {
			for x := 0; x < int(n.ChildCount()); x++ {
				cn := n.Child(x)
				switch cn.Type() {
				case ",":
					stmt.Index = foreachVar

				case ":":
					stmt.Value = foreachVar

				case "foreach_var":
					foreachVar = c.convert_foreach_var(cn, source)

				case ")", "(":

				default:
					stmt.Collection = c.convert_expression(cn, source).(ast.Expression)
				}
			}
		}
	}

	bodyN := node.ChildByFieldName("body")
	if bodyN != nil {
		stmt.Body = c.convert_statement(bodyN, source)
	}

	return stmt
}

func (c *ASTConverter) convert_foreach_var(node *sitter.Node, source []byte) ast.ForeachValue {
	value := ast.ForeachValue{}

	//debugNode(node, source, fmt.Sprint(node.ChildCount()))
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type":
			value.Type = c.convert_type(n, source)

		case "&":
			// ??
			value.Type.Reference = true

		case "ident":
			value.Identifier = c.convert_ident(n, source).(*ast.Ident)
		}
	}

	return value
}

func (c *ASTConverter) convert_while_stmt(node *sitter.Node, source []byte) ast.Statement {
	stmt := &ast.WhileStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "paren_cond":
			stmt.Condition = c.convert_paren_conditions(n, source)
		}
	}

	bodyN := node.ChildByFieldName("body")
	if bodyN != nil {
		stmt.Body = c.convert_statement(bodyN, source)
	}

	return stmt
}

func (c *ASTConverter) convert_do_stmt(node *sitter.Node, source []byte) ast.Statement {
	stmt := &ast.DoStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "compound_stmt":
			stmt.Body = c.convert_statement(n, source)
		case "paren_expr":
			stmt.Condition = c.convert_expression(n.Child(1), source).(ast.Expression)
		}
	}

	return stmt
}

func (c *ASTConverter) convert_defer_stmt(node *sitter.Node, source []byte) ast.Statement {
	stmt := &ast.DeferStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	stmt.Statement = c.convert_statement(
		node.Child(int(node.ChildCount()-1)),
		source,
	)

	return stmt
}

func (c *ASTConverter) convert_assert_stmt(node *sitter.Node, source []byte) ast.Statement {
	stmt := &ast.AssertStatement{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
	}

	nodes := commaSep(
		cv_expr_fn(c.convert_expression),
		node.Child(2),
		source,
	)
	for _, node := range nodes {
		stmt.Assertions = append(stmt.Assertions, node.(ast.Expression))
	}

	return stmt
}
