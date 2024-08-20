package ast

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

func convert_statement(node *sitter.Node, source []byte) Expression {

	return anyOf([]NodeRule{
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
	}, node, source)
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

func convert_declaration_stmt(node *sitter.Node, source []byte) Expression {
	if node.Type() == "const_declaration" {
		return convert_const_declaration(node, source)
	}

	isStatic := false
	isTlocal := false
	varDecl := VariableDecl{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)

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
			debugNode(colon, source)
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
		value = anyOf([]NodeRule{
			NodeTryConversionFunc("_expr"),
			NodeOfType("type"),
			NodeOfType("default"),
		}, targetNode, source)
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
