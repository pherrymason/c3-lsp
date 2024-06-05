package symbol_trie

import (
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
)

type Trie struct {
	root *TrieNode
}

func NewTrie() *Trie {
	return &Trie{
		root: NewTrieNode("root"),
	}
}

func (t *Trie) ClearByTag(tag string) {
	clearByTagHelper(t.root, tag)
}

func clearByTagHelper(node *TrieNode, docId string) bool {
	if node == nil {
		return false
	}

	// Recursively clear children
	for key, child := range node.children {
		if clearByTagHelper(child, docId) {
			delete(node.children, key)
		}
	}

	// Clear this node if it matches the tag
	if node.symbol != nil && node.symbol.GetDocumentURI() == docId {
		node.symbol = nil
	}

	// If node has children, just clear the symbol
	if node.symbol == nil && len(node.children) == 0 {
		return true
	}

	return false
}

func (t *Trie) Insert(symbol symbols.Indexable) {
	node := t.root
	fqn := symbol.GetFQN()
	parts := splitFQN(fqn)
	for _, part := range parts {
		if node.children[part] == nil {
			node.children[part] = NewTrieNode(part)
		}
		node = node.children[part]
	}
	node.symbol = symbol
}

func splitFQN(fqn string) []string {
	return strings.FieldsFunc(fqn, func(r rune) bool {
		return r == ':' || r == '.'
	})
}

// accepted queries
//   - mod::path   -> will return path element if found
//   - mod::path.* -> will return all children of path
//   - mod::path.t* -> will return children of path starting with `t`
func (t *Trie) Search(query string) []symbols.Indexable {
	if strings.HasSuffix(query, ".") {
		prefix := strings.TrimSuffix(query, ".")
		node := t.searchExact(prefix)
		if node == nil {
			return nil
		}
		return collectSymbols(node, false)
	} else if strings.Contains(query, "*") {
		parts := splitFQN(query)
		prefix := parts[len(parts)-1]
		trimmedQuery := strings.TrimSuffix(query, prefix)
		node := t.searchExact(trimmedQuery)
		if node == nil {
			return nil
		}
		return collectPrefixedSymbols(node, strings.TrimSuffix(prefix, "*"))
	} else {
		node := t.searchExact(query)
		if node != nil && node.symbol != nil {
			return []symbols.Indexable{node.symbol}
		}
		return nil
	}
}

// Searches an exact node in the trie
func (t *Trie) searchExact(query string) *TrieNode {
	node := t.root
	parts := splitFQN(query)
	for _, part := range parts {
		if node.children[part] == nil {
			return nil
		}
		node = node.children[part]
	}
	return node
}

// collectNodeAndChildrenSymbols collect all symbols from a given node
func collectSymbols(node *TrieNode, includeParent bool) []symbols.Indexable {
	var results []symbols.Indexable
	if includeParent && node.symbol != nil {
		results = append(results, node.symbol)
	}
	for _, child := range node.children {
		results = append(results, collectSymbols(child, true)...)
	}

	return results
}

// collectPrefixedSymbols collects all symbols that start by a prefrix from a given node
func collectPrefixedSymbols(node *TrieNode, prefix string) []symbols.Indexable {
	var results []symbols.Indexable

	for key, child := range node.children {
		if strings.HasPrefix(key, prefix) {
			results = append(results, collectSymbols(child, true)...)
		}
	}

	return results
}

type TrieNode struct {
	children map[string]*TrieNode
	symbol   symbols.Indexable
	name     string
}

func NewTrieNode(name string) *TrieNode {
	return &TrieNode{
		children: make(map[string]*TrieNode),
		name:     name,
	}
}
