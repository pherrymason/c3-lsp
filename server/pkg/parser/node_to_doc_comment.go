package parser

import (
	"github.com/pherrymason/c3-lsp/pkg/cast"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
// From grammar.js in tree-sitter-c3 v0.2.3:

// Doc comments and contracts
// -------------------------
// NOTE parsed by scanner.c (scan_doc_comment_contract_text)
doc_comment_contract: $ => seq(

	field('name', $.at_ident),
	optional($.doc_comment_contract_text)

),
doc_comment: $ => seq(

	'<*',
	optional($.doc_comment_text), // NOTE parsed by scanner.c (scan_doc_comment_text)
	repeat($.doc_comment_contract),
	'*>',

),

// (...)

at_ident: _ => token(seq('@', IDENT)),
*/
func (p *Parser) nodeToDocComment(node *sitter.Node, sourceCode []byte) idx.DocComment {
	body := ""
	bodyNode := node.Child(1)
	if bodyNode.Type() == "doc_comment_text" {
		body = bodyNode.Content(sourceCode)
	}

	docComment := idx.NewDocComment(body)

	if node.ChildCount() >= 4 {
		for i := 2; i <= int(node.ChildCount())-2; i++ {
			contractNode := node.Child(i)
			if contractNode.Type() == "doc_comment_contract" {
				name := contractNode.ChildByFieldName("name").Content(sourceCode)
				body := ""
				if contractNode.ChildCount() >= 2 {
					body = contractNode.Child(1).Content(sourceCode)
				}

				contract := idx.NewDocCommentContract(name, body)

				docComment.AddContracts([]*idx.DocCommentContract{cast.ToPtr(contract)})
			}
		}
	}

	return docComment
}
