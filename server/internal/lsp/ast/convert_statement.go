package ast

import (
	"fmt"
	"log"

	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

func convert_statement(node *sitter.Node, source []byte) Statement {
	dd := anyOf("statement", []NodeRule{
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
		log.Fatalf("Could not conver_statement. Node Type: %s. Content: %s\n----- %s\n", node.Type(), node.Content(source), node)
	}

	return dd.(Statement)
}

func convert_compound_stmt(node *sitter.Node, source []byte) Statement {
	cmpStatement := &CompoundStmt{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Statements:     []Statement{},
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() != "{" && n.Type() != "}" {
			cmpStatement.Statements = append(
				cmpStatement.Statements,
				convert_statement(n, source),
			)
		}
	}

	return cmpStatement
}

func convert_return_stmt(node *sitter.Node, source []byte) Statement {
	expr := option.None[Expression]()

	argNode := node.Child(0).NextSibling()
	if argNode.Type() != ";" {
		expr = option.Some(convert_expression(argNode, source).(Expression))
	}

	return &ReturnStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Return:         expr,
	}
}

func convert_split_declaration_stmt(node *sitter.Node, source []byte) Statement {
	return convert_declaration_stmt(node.Parent(), source)
}

func convert_declaration_stmt(node *sitter.Node, source []byte) Statement {
	if node.Type() == "const_declaration" {
		return &DeclarationStmt{
			Decl: convert_const_declaration(node, source),
		}
	}

	isStatic := false
	isTlocal := false
	varDecl := VariableDecl{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
	}
	end := false
	for i := 0; i < int(node.ChildCount()) && !end; i++ {
		n := node.Child(i)
		debugNode(n, source, "dd")

		switch n.Type() {
		case "local_decl_storage":
			if n.Content(source) == "static" {
				isStatic = true
			} else {
				isTlocal = true
			}

		case "type":
			varDecl.Type = convert_type(n, source)
			varDecl.Type.Static = isStatic
			varDecl.Type.TLocal = isTlocal

		case "local_decl_after_type":
			varDecl.Names = append(varDecl.Names, NewIdentifierBuilder().
				WithName(n.ChildByFieldName("name").Content(source)).
				WithSitterPos(n.ChildByFieldName("name")).
				BuildPtr(),
			)

			right := n.ChildByFieldName("right")
			if right != nil {
				varDecl.Initializer = convert_expression(right, source).(Expression)
			}
			end = true
		}
	}

	return &DeclarationStmt{
		Decl: &varDecl,
	}
}

func convert_continue_stmt(node *sitter.Node, source []byte) Statement {
	label := option.None[string]()
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return &ContinueStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Label:          label,
	}
}

func convert_break_stmt(node *sitter.Node, source []byte) Statement {
	label := option.None[string]()
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return &BreakStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Label:          label,
	}
}

func convert_switch_stmt(node *sitter.Node, source []byte) Statement {
	label := option.None[string]()
	var cases []SwitchCase
	var defaultStatement []Statement

	body := node.ChildByFieldName("body")
	for i := 0; i < int(body.ChildCount()); i++ {
		n := body.Child(i)
		if n.Type() == "case_stmt" {
			conditionNode := n.ChildByFieldName("value")
			var caseValue Statement
			if conditionNode.Type() == "case_range" {
				caseValue = &SwitchCaseRange{
					NodeAttributes: NewBaseNodeFromSitterNode(conditionNode),
					Start:          convert_expression(conditionNode.Child(0), source).(Expression),
					End:            convert_expression(conditionNode.Child(2), source).(Expression),
				}
			} else if conditionNode.Type() == "type" {
				caseValue = &ExpressionStmt{
					Expr: convert_type(conditionNode, source),
				}
			} else {
				caseValue = &ExpressionStmt{
					Expr: convert_expression(conditionNode, source).(Expression),
				}

			}

			colon := conditionNode.NextSibling()
			ns := colon.NextSibling()
			var statements []Statement
			for {
				if ns == nil {
					break
				}
				statements = append(statements, convert_statement(ns, source))
				ns = ns.NextSibling()
			}

			cases = append(cases, SwitchCase{
				NodeAttributes: NewBaseNodeFromSitterNode(n),
				Value:          caseValue,
				Statements:     statements,
			})
		} else {
			for d := 0; d < int(n.ChildCount()); d++ {
				dn := n.Child(d)
				if dn.Type() != "default" && dn.Type() != ":" {
					defaultStatement = append(defaultStatement, convert_statement(dn, source))
				}
			}
		}
	}

	var conditionExpression []*DeclOrExpr
	conditionNode := node.ChildByFieldName("condition")
	if conditionNode != nil {
		ConvertDebug = false
		conditionExpression = convert_paren_conditions(conditionNode, source)
		ConvertDebug = false
	}

	return &SwitchStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Label:          label,
		Condition:      conditionExpression,
		Cases:          cases,
		Default:        defaultStatement,
	}
}

func convert_nextcase_stmt(node *sitter.Node, source []byte) Statement {
	label := option.None[string]()
	var value Expression
	targetNode := node.ChildByFieldName("target")
	if targetNode != nil {
		found := anyOf("nextcase_stmt", []NodeRule{
			NodeTryConversionFunc("_expr"),
			NodeOfType("type"),
			NodeOfType("default"),
		}, targetNode, source, true)

		if found != nil {
			value = found.(Expression)
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return &Nextcase{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Label:          label,
		Value:          value,
	}
}

func convert_if_stmt(node *sitter.Node, source []byte) Statement {
	conditions := convert_paren_conditions(node.ChildByFieldName("condition"), source)
	//fmt.Printf("%s", reflect.TypeOf(conditions).String())
	stmt := &IfStmt{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Label:          option.None[string](),
		Condition:      conditions,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "label":
			stmt.Label = option.Some(n.Child(0).Content(source))
		case "else_part":
			elseStmt := convert_statement(n.Child(1), source)
			switch elseStmt.(type) {
			case *CompoundStmt:
				if len(elseStmt.(*CompoundStmt).Statements) == 0 {
					elseStmt = nil
				}
			}
			stmt.Else = ElseStatement{
				NodeAttributes: NewBaseNodeFromSitterNode(n),
				Statement:      elseStmt,
			}
		}
	}
	//ifBody := node.Child(int(node.ChildCount()) - 1)
	ifBody := node.ChildByFieldName("body")
	bodyStmt := convert_statement(ifBody, source)
	switch bodyStmt.(type) {
	case *CompoundStmt:
		if len(bodyStmt.(*CompoundStmt).Statements) == 0 {
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
func convert_paren_conditions(node *sitter.Node, source []byte) []*DeclOrExpr {
	return convert_conditions(node.Child(1), source)
}

func convert_conditions(node *sitter.Node, source []byte) []*DeclOrExpr {
	var conditions []*DeclOrExpr

	// Option 1: try_unwrap_chain
	if node.Type() == "try_unwrap_chain" {

	} else if node.Type() == "catch_unwrap" { // Option 2: catch_unwrap

	} else {
		// Option 3:
		//		(decl_or_expr),* + optional(, try_unwrap | catch_unwrap)
		nodes := commaSep(convert_decl_or_expression, node, source)
		for _, n := range nodes {
			conditions = append(conditions, n.(*DeclOrExpr))
		}
	}

	return conditions
}

func convert_local_declaration_after_type(node *sitter.Node, source []byte) Declaration {
	var init Expression

	if node.Child(1).Type() == "attributes" {
		// TODO Get attributes
	}

	right := node.ChildByFieldName("right")
	if right != nil {
		init = convert_expression(right, source).(Expression)
	}

	varDecl := &VariableDecl{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
		Names: []*Ident{
			NewIdentifierBuilder().
				WithName(node.ChildByFieldName("name").Content(source)).
				WithSitterPos(node.ChildByFieldName("name")).
				BuildPtr(),
		},
		Initializer: init,
	}

	return varDecl
}

func convert_for_stmt(node *sitter.Node, source []byte) Statement {
	forStmt := &ForStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "for_cond" {
			initNode := n.ChildByFieldName("initializer")
			if initNode != nil {
				forStmt.Initializer = convert_comma_decl_or_expression(initNode, source)
			}

			condNode := n.ChildByFieldName("condition")
			if condNode != nil {
				forStmt.Condition = convert_conditions(condNode, source)[0]
			}

			updateNode := n.ChildByFieldName("update")
			if updateNode != nil {
				forStmt.Update = convert_comma_decl_or_expression(updateNode, source)
			}
		}
	}

	nodeBody := node.ChildByFieldName("body")
	forStmt.Body = convert_statement(nodeBody, source)

	return forStmt
}

func convert_comma_decl_or_expression(node *sitter.Node, source []byte) []*DeclOrExpr {
	nodes := commaSep(convert_decl_or_expression, node.Child(0), source)

	var stmts []*DeclOrExpr
	for _, n := range nodes {
		stmts = append(stmts, n.(*DeclOrExpr))
	}
	return stmts
}

// This takes anon nodes
func convert_decl_or_expression(node *sitter.Node, source []byte) Node {
	found := anyOf("decl_or_expression", []NodeRule{
		NodeOfType("var_decl"),
		NodeSiblingsWithSequenceOf([]NodeRule{
			NodeOfType("type"), NodeOfType("local_decl_after_type"),
		}, "split_declaration_stmt"),
		NodeAnonymous("_expr"),
	}, node, source, false)

	switch found.(type) {
	case Expression:
		return &DeclOrExpr{
			Expr: found.(Expression),
		}

	case Declaration:
		return &DeclOrExpr{
			Decl: found.(Declaration),
		}

	case Statement:
		return &DeclOrExpr{
			Stmt: found.(Statement),
		}

	default:
		panic("decl_or_expression: did not find found type.")
	}
}

func convert_foreach_stmt(node *sitter.Node, source []byte) Statement {
	stmt := &ForeachStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
	}
	foreachVar := ForeachValue{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "foreach_cond" {
			for c := 0; c < int(n.ChildCount()); c++ {
				cn := n.Child(c)
				switch cn.Type() {
				case ",":
					stmt.Index = foreachVar

				case ":":
					stmt.Value = foreachVar

				case "foreach_var":
					foreachVar = convert_foreach_var(cn, source)

				case ")", "(":

				default:
					stmt.Collection = convert_expression(cn, source).(Expression)
				}
			}
		}
	}

	bodyN := node.ChildByFieldName("body")
	if bodyN != nil {
		stmt.Body = convert_statement(bodyN, source)
	}

	return stmt
}

func convert_foreach_var(node *sitter.Node, source []byte) ForeachValue {
	value := ForeachValue{}

	debugNode(node, source, fmt.Sprint(node.ChildCount()))
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type":
			value.Type = convert_type(n, source)

		case "&":
			// ??
			value.Type.Reference = true

		case "ident":
			value.Identifier = convert_ident(n, source).(*Ident)
		}
	}

	return value
}

func convert_while_stmt(node *sitter.Node, source []byte) Statement {
	stmt := &WhileStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "paren_cond":
			stmt.Condition = convert_paren_conditions(n, source)
		}
	}

	bodyN := node.ChildByFieldName("body")
	if bodyN != nil {
		stmt.Body = convert_statement(bodyN, source)
	}

	return stmt
}

func convert_do_stmt(node *sitter.Node, source []byte) Statement {
	stmt := &DoStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "compound_stmt":
			stmt.Body = convert_statement(n, source)
		case "paren_expr":
			stmt.Condition = convert_expression(n.Child(1), source).(Expression)
		}
	}

	return stmt
}

func convert_defer_stmt(node *sitter.Node, source []byte) Statement {
	stmt := &DeferStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
	}

	stmt.Statement = convert_statement(
		node.Child(int(node.ChildCount()-1)),
		source,
	)

	return stmt
}

func convert_assert_stmt(node *sitter.Node, source []byte) Statement {
	stmt := &AssertStatement{
		NodeAttributes: NewBaseNodeFromSitterNode(node),
	}

	nodes := commaSep(
		cv_expr_fn(convert_expression),
		node.Child(2),
		source,
	)
	for _, node := range nodes {
		stmt.Assertions = append(stmt.Assertions, node.(Expression))
	}

	return stmt
}
