package search

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rajal-ui/minies/internal/index"
	"github.com/rajal-ui/minies/internal/storage"
	"github.com/rajal-ui/minies/pkg/logrecord"
)

type SearchResult struct {
	Service    string  `json:"service"`
	Level      string  `json:"level"`
	Message    string  `json:"message"`
	SourceFile string  `json:"source_file"`
	LineNo     int     `json:"line_no"`
	Timestamp  int64   `json:"timestamp"`
	Score      float64 `json:"score"`
}

type Executor struct {
	idx  *index.InvertedIndex
	trie *index.Trie
	segs *storage.SegmentManager
}

func NewExecutor(idx *index.InvertedIndex, trie *index.Trie, segs *storage.SegmentManager) *Executor {
	return &Executor{
		idx:  idx,
		trie: trie,
		segs: segs,
	}
}

func (e *Executor) Search(query string) ([]SearchResult, error) {
	ast, err := NewParser().Parse(query)
	if err != nil {
		return nil, err
	}
	return e.executeAST(ast), nil
}

func (e *Executor) executeAST(node *QueryAST) []SearchResult {
	var results []SearchResult
	switch node.Type {
	case QueryExact:
		results = e.searchExact(node.Token)
	case QueryWildcard:
		results = e.searchWildcard(node.Token)
	case QueryFuzzy:
		k := 1
		if v, err := strconv.Atoi(node.Operator); err == nil {
			k = v
		}
		if k < 1 {
			k = 1
		}
		if k > 3 {
			k = 3
		}
		results = dedup(e.searchFuzzy(node.Token, k))
	case QueryBoolean:
		left := e.executeAST(node.Left)
		right := e.executeAST(node.Right)
		switch node.Operator {
		case "AND":
			results = intersect(left, right)
		case "OR":
			results = union(left, right)
		case "NOT":
			results = difference(left, right)
		}
	}
	return results
}

func (e *Executor) searchExact(token string) []SearchResult {
	postings := e.idx.Get(token)
	return e.resultsFromPostings(postings, token)
}

func (e *Executor) searchWildcard(pattern string) []SearchResult {
	// The trie lookup uses the literal prefix before the first '*';
	// WildcardMatch then enforces the full pattern.
	prefix := pattern
	if i := strings.IndexByte(pattern, '*'); i >= 0 {
		prefix = pattern[:i]
	}
	tokens := e.trie.SearchPrefix(prefix)
	var allResults []SearchResult
	for _, t := range tokens {
		if !WildcardMatch(pattern, t) {
			continue
		}
		postings := e.idx.Get(t)
		allResults = append(allResults, e.resultsFromPostings(postings, t)...)
	}
	return dedup(allResults)
}

func (e *Executor) searchFuzzy(token string, k int) []SearchResult {
	var results []SearchResult
	for _, t := range e.idx.GetAllTokens() {
		if BitapScore(t, token, k) <= k {
			postings := e.idx.Get(t)
			results = append(results, e.resultsFromPostings(postings, t)...)
		}
	}
	return results
}

func (e *Executor) resultsFromPostings(postings []index.Posting, token string) []SearchResult {
	var results []SearchResult
	cache := make(map[int][][]byte)
	seen := make(map[string]bool)
	for _, p := range postings {
		if p.SegmentID < 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", p.SegmentID, p.Offset)
		if seen[key] {
			continue
		}
		seen[key] = true

		records, ok := cache[p.SegmentID]
		if !ok {
			segPath := fmt.Sprintf("%s/segment_%06d.bin", e.segs.Dir(), p.SegmentID)
			records, _ = storage.ReadSegment(segPath)
			cache[p.SegmentID] = records
		}
		if p.Offset < 0 || p.Offset >= len(records) {
			continue
		}
		decodeAndAddResult(&results, records[p.Offset])
	}
	return results
}

func decodeAndAddResult(results *[]SearchResult, recordBytes []byte) {
	var record logrecord.LogRecord
	err := json.Unmarshal(recordBytes, &record)
	if err != nil {
		return
	}

	*results = append(*results, SearchResult{
		Service:    record.Service,
		Level:      string(record.Level),
		Message:    record.Message,
		SourceFile: record.SourceFile,
		LineNo:     record.LineNo,
		Timestamp:  record.Timestamp,
		Score:      1.0,
	})
}

func resultKey(r SearchResult) string {
	return strings.ToLower(r.Service) + ":" + r.SourceFile + ":" + strconv.Itoa(r.LineNo) + ":" +
		r.Message + ":" + strconv.FormatInt(r.Timestamp, 10)
}

func intersect(a, b []SearchResult) []SearchResult {
	result := []SearchResult{}
	seen := make(map[string]bool)
	for _, r := range b {
		seen[resultKey(r)] = true
	}
	for _, r := range a {
		key := resultKey(r)
		if seen[key] {
			result = append(result, r)
			delete(seen, key)
		}
	}
	return result
}

func union(a, b []SearchResult) []SearchResult {
	result := []SearchResult{}
	seen := make(map[string]bool)
	for _, r := range append(append([]SearchResult{}, a...), b...) {
		key := resultKey(r)
		if !seen[key] {
			seen[key] = true
			result = append(result, r)
		}
	}
	return result
}

func difference(a, b []SearchResult) []SearchResult {
	bset := make(map[string]bool)
	for _, r := range b {
		bset[resultKey(r)] = true
	}
	var result []SearchResult
	for _, r := range a {
		key := resultKey(r)
		if !bset[key] {
			result = append(result, r)
		}
	}
	return result
}

func dedup(results []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	var out []SearchResult
	for _, r := range results {
		key := strings.ToLower(resultKey(r))
		if !seen[key] {
			seen[key] = true
			out = append(out, r)
		}
	}
	return out
}
