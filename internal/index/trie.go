package index

import (
	"sync"

	"github.com/rajal-ui/minies/internal/tokenizer"
)

type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
	tokens   []string
}

type Trie struct {
	root *TrieNode
	mu   sync.RWMutex
}

func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{children: make(map[rune]*TrieNode)},
	}
}

func (t *Trie) Insert(token string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	node := t.root
	normalized := tokenizer.NormalizeToken(token)
	for _, ch := range normalized {
		if _, ok := node.children[ch]; !ok {
			node.children[ch] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[ch]
	}
	if !node.isEnd {
		node.isEnd = true
		node.tokens = append(node.tokens, normalized)
	}
}

func (t *Trie) SearchPrefix(prefix string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	node := t.root
	normalized := tokenizer.NormalizeToken(prefix)
	for _, ch := range normalized {
		if _, ok := node.children[ch]; !ok {
			return nil
		}
		node = node.children[ch]
	}
	var results []string
	collectTokens(node, &results)
	return results
}

func collectTokens(node *TrieNode, results *[]string) {
	if node.isEnd {
		*results = append(*results, node.tokens...)
	}
	for _, child := range node.children {
		collectTokens(child, results)
	}
}

func (t *Trie) Contains(token string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	node := t.root
	normalized := tokenizer.NormalizeToken(token)
	for _, ch := range normalized {
		if _, ok := node.children[ch]; !ok {
			return false
		}
		node = node.children[ch]
	}
	return node.isEnd
}

func (t *Trie) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return countNodes(t.root)
}

func countNodes(node *TrieNode) int {
	count := 0
	if node.isEnd {
		count++
	}
	for _, child := range node.children {
		count += countNodes(child)
	}
	return count
}
