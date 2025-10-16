package parser

import (
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/dedent"
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
	hasBody := false
	bodyNode := node.Child(1)
	if bodyNode.Type() == "doc_comment_text" {
		// Dedent to accept indented doc strings.
		body = dedent.Dedent(bodyNode.Content(sourceCode))
		hasBody = true
	}

	docComment := idx.NewDocComment(body)

	if (hasBody && node.ChildCount() >= 4) || (!hasBody && node.ChildCount() >= 3) {
		for i := 1; i <= int(node.ChildCount())-2; i++ {
			contractNode := node.Child(i)

			// Skip the body
			// (We already skip '<*' and '*>' since we skip first and last indices above)
			if contractNode.Type() == "doc_comment_contract" {
				name := contractNode.ChildByFieldName("name").Content(sourceCode)
				body := ""
				if contractNode.ChildCount() >= 2 {
					// Right now, contracts can only have a single line, so we don't dedent.
					// They can also be arbitrary expressions, so it's best to not modify them
					// at the moment.
					start := contractNode.Child(1).StartByte()
					end := contractNode.Child(int(contractNode.ChildCount()) - 1).EndByte()
					body = string(sourceCode[start:end])
				}

				contract := idx.NewDocCommentContract(name, body)

				docComment.AddContracts([]*idx.DocCommentContract{cast.ToPtr(contract)})
			}
		}
	}

	return docComment
}
