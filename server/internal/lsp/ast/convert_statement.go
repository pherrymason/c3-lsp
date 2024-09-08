package ast

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

func convert_statement(node *sitter.Node, source []byte) Expression {
	return anyOf("statement", []NodeRule{
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
}

func convert_compound_stmt(node *sitter.Node, source []byte) Expression {
	cmpStatement := CompoundStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Statements:  []Expression{},
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

func convert_return_stmt(node *sitter.Node, source []byte) Expression {
	expr := option.None[Expression]()

	argNode := node.Child(0).NextSibling()
	if argNode.Type() != ";" {
		expr = option.Some(convert_expression(argNode, source))
	}

	return ReturnStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Return:      expr,
	}
}

func convert_split_declaration_stmt(node *sitter.Node, source []byte) Expression {
	return convert_declaration_stmt(node.Parent(), source)
}

func convert_declaration_stmt(node *sitter.Node, source []byte) Expression {
	if node.Type() == "const_declaration" {
		return convert_const_declaration(node, source)
	}

	isStatic := false
	isTlocal := false
	varDecl := VariableDecl{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
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
			varDecl.Type = convert_type(n, source).(TypeInfo)
			varDecl.Type.Static = isStatic
			varDecl.Type.TLocal = isTlocal

		case "local_decl_after_type":
			varDecl.Names = append(varDecl.Names, NewIdentifierBuilder().
				WithName(n.ChildByFieldName("name").Content(source)).
				WithSitterPos(n.ChildByFieldName("name")).
				Build(),
			)

			right := n.ChildByFieldName("right")
			if right != nil {
				varDecl.Initializer = convert_expression(right, source)
			}
			end = true
		}
	}

	return varDecl
}

func convert_continue_stmt(node *sitter.Node, source []byte) Expression {
	label := option.None[string]()
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return ContinueStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Label:       label,
	}
}

func convert_break_stmt(node *sitter.Node, source []byte) Expression {
	label := option.None[string]()
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return BreakStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Label:       label,
	}
}

func convert_switch_stmt(node *sitter.Node, source []byte) Expression {
	label := option.None[string]()
	cases := []SwitchCase{}
	var defaultStatement []Statement

	body := node.ChildByFieldName("body")
	for i := 0; i < int(body.ChildCount()); i++ {
		n := body.Child(i)
		if n.Type() == "case_stmt" {
			conditionNode := n.ChildByFieldName("value")
			var caseValue Expression
			if conditionNode.Type() == "case_range" {
				caseValue = SwitchCaseRange{
					ASTBaseNode: NewBaseNodeFromSitterNode(conditionNode),
					Start:       convert_expression(conditionNode.Child(0), source),
					End:         convert_expression(conditionNode.Child(2), source),
				}
			} else if conditionNode.Type() == "type" {
				caseValue = convert_type(conditionNode, source)
			} else {
				caseValue = convert_expression(conditionNode, source)
			}

			colon := conditionNode.NextSibling()
			ns := colon.NextSibling()
			statements := []Statement{}
			for {
				if ns == nil {
					break
				}
				statements = append(statements, convert_statement(ns, source))
				ns = ns.NextSibling()
			}

			cases = append(cases, SwitchCase{
				ASTBaseNode: NewBaseNodeFromSitterNode(n),
				Value:       caseValue,
				Statements:  statements,
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

	return SwitchStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Label:       label,
		Condition:   convert_expression(node.ChildByFieldName("condition"), source),
		Cases:       cases,
		Default:     defaultStatement,
	}
}

func convert_nextcase_stmt(node *sitter.Node, source []byte) Expression {
	label := option.None[string]()
	var value Expression
	targetNode := node.ChildByFieldName("target")
	if targetNode != nil {
		value = anyOf("nextcase_stmt", []NodeRule{
			NodeTryConversionFunc("_expr"),
			NodeOfType("type"),
			NodeOfType("default"),
		}, targetNode, source, false)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "label_target" {
			label = option.Some(n.Content(source))
		}
	}

	return Nextcase{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Label:       label,
		Value:       value,
	}
}

func convert_if_stmt(node *sitter.Node, source []byte) Expression {
	conditions := convert_paren_conditions(node.ChildByFieldName("condition"), source)
	//fmt.Printf("%s", reflect.TypeOf(conditions).String())
	stmt := IfStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Label:       option.None[string](),
		Condition:   conditions,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "label":
			stmt.Label = option.Some(n.Child(0).Content(source))
		case "else_part":
			elseStmt := convert_statement(n.Child(1), source)
			switch elseStmt.(type) {
			case CompoundStatement:
				if len(elseStmt.(CompoundStatement).Statements) == 0 {
					elseStmt = nil
				}
			}
			stmt.Else = ElseStatement{
				ASTBaseNode: NewBaseNodeFromSitterNode(n),
				Statement:   elseStmt,
			}
		}
	}
	//ifBody := node.Child(int(node.ChildCount()) - 1)
	ifBody := node.ChildByFieldName("body")
	bodyStmt := convert_statement(ifBody, source)
	switch bodyStmt.(type) {
	case CompoundStatement:
		if len(bodyStmt.(CompoundStatement).Statements) == 0 {
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
func convert_paren_conditions(node *sitter.Node, source []byte) []Expression {
	return convert_conditions(node.Child(1), source)
}

func convert_conditions(node *sitter.Node, source []byte) []Expression {
	conditions := []Expression{}

	// Option 1: try_unwrap_chain
	if node.Type() == "try_unwrap_chain" {

	} else if node.Type() == "catch_unwrap" { // Option 2: catch_unwrap

	} else {
		// Option 3:
		//		(decl_or_expr),* + optional(, try_unwrap | catch_unwrap)
		conditions = commaSep(convert_decl_or_expression, node, source)
	}

	return conditions
}

func convert_local_declaration_after_type(node *sitter.Node, source []byte) Expression {
	var init Expression

	if node.Child(1).Type() == "attributes" {
		// TODO Get attributes
	}

	right := node.ChildByFieldName("right")
	if right != nil {
		init = convert_expression(right, source)
	}

	varDecl := VariableDecl{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Names: []Identifier{
			NewIdentifierBuilder().
				WithName(node.ChildByFieldName("name").Content(source)).
				WithSitterPos(node.ChildByFieldName("name")).
				Build(),
		},
		Initializer: init,
	}

	return varDecl
}

func convert_for_stmt(node *sitter.Node, source []byte) Expression {
	forStmt := ForStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
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

func convert_comma_decl_or_expression(node *sitter.Node, source []byte) []Expression {
	return commaSep(convert_decl_or_expression, node.Child(0), source)
}

// This takes anon nodes
func convert_decl_or_expression(node *sitter.Node, source []byte) Expression {
	return anyOf("decl_or_expression", []NodeRule{
		NodeOfType("var_decl"),
		NodeSiblingsWithSequenceOf([]NodeRule{
			NodeOfType("type"), NodeOfType("local_decl_after_type"),
		}, "split_declaration_stmt"),
		NodeAnonymous("_expr"),
	}, node, source, true)
}

func convert_foreach_stmt(node *sitter.Node, source []byte) Expression {
	/*
		foreach_stmt: $ => seq(
		      choice('foreach', 'foreach_r'),
		      optional($.label),
		      $.foreach_cond,
		      field('body', $._statement)
		    )

		foreach_cond: $ => seq(
			'(',
			optional(seq(field('index', $.foreach_var), ',')),
			field('value', $.foreach_var),
			':',
			field('collection', $._expr),
			')',
		),

		foreach_var: $ => choice(
			seq(optional($._type_or_optional_type), optional('&'), $.ident),
		),
	*/
	stmt := ForeachStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
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
					stmt.Collection = convert_expression(cn, source)
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
			value.Type = convert_type(n, source).(TypeInfo)

		case "&":
			// ??
			value.Type.Reference = true

		case "ident":
			value.Identifier = convert_ident(n, source).(Identifier)
		}
	}

	return value
}

func convert_while_stmt(node *sitter.Node, source []byte) Expression {
	stmt := WhileStatement{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
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
