package search

import (
	"testing"
)

func TestParser(t *testing.T) {
	p := NewParser()

	ast, err := p.Parse("error AND timeout")
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if ast.Type != QueryBoolean || ast.Operator != "AND" {
		t.Errorf("Expected boolean AND, got %v", ast)
	}

	ast, err = p.Parse("auth*")
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if ast.Type != QueryWildcard {
		t.Errorf("Expected wildcard, got %v", ast.Type)
	}

	ast, err = p.Parse("tiemout~1")
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if ast.Type != QueryFuzzy {
		t.Errorf("Expected fuzzy, got %v", ast.Type)
	}

	_, err = p.Parse("")
	if err == nil {
		t.Error("Parse('') should return error")
	}
}

func TestWildcardMatch(t *testing.T) {
	if !WildcardMatch("auth*", "authenticate") {
		t.Error("WildcardMatch(auth*, authenticate) = false")
	}
	if WildcardMatch("auth*", "xyz") {
		t.Error("WildcardMatch(auth*, xyz) = true")
	}
}

func TestParseQueryTerm(t *testing.T) {
	result := ParseQueryTerm("Error")
	if result != "error" {
		t.Errorf("Expected 'error', got %q", result)
	}
}

func TestKMP(t *testing.T) {
	tests := []struct {
		text    string
		pattern string
		want    bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"hello world", "", true},
		{"short", "longerpattern", false},
	}
	for _, tt := range tests {
		got := KMP(tt.text, tt.pattern)
		if got != tt.want {
			t.Errorf("KMP(%q, %q) = %v, want %v", tt.text, tt.pattern, got, tt.want)
		}
	}
}

func TestKMPSearch(t *testing.T) {
	positions := KMPSearch("abababab", "aba")
	if len(positions) != 3 {
		t.Errorf("Expected 3 positions, got %d", len(positions))
	}
	if positions[0] != 0 || positions[1] != 2 || positions[2] != 4 {
		t.Errorf("Unexpected positions: %v", positions)
	}
}

func TestBitapSearch(t *testing.T) {
	tests := []struct {
		text        string
		pattern     string
		maxDistance int
		want        bool
	}{
		{"hello", "hello", 1, true},
		{"hello", "hallo", 1, true},
		{"hello", "world", 1, false},
		{"hello", "helol", 2, true},
		{"hello", "helol", 0, false},
	}
	for _, tt := range tests {
		got := BitapSearch(tt.text, tt.pattern, tt.maxDistance)
		if got != tt.want {
			t.Errorf("BitapSearch(%q, %q, %d) = %v, want %v", tt.text, tt.pattern, tt.maxDistance, got, tt.want)
		}
	}
}

func TestBitapScore(t *testing.T) {
	score := BitapScore("hello", "hallo", 1)
	if score != 1 {
		t.Errorf("Expected score 1, got %d", score)
	}
	score = BitapScore("hello", "world", 1)
	if score <= 1 {
		t.Errorf("Expected score > 1, got %d", score)
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"kitten", "sitting", 3},
		{"hello", "hello", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"flaw", "lawn", 2},
	}
	for _, tt := range tests {
		got := LevenshteinDistance(tt.s1, tt.s2)
		if got != tt.expected {
			t.Errorf("Levenshtein(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.expected)
		}
	}
}
