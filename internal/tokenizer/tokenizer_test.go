package tokenizer

import "testing"

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"error AND timeout", []string{"error", "AND", "timeout"}},
		{"foo_bar baz", []string{"foo_bar", "baz"}},
		{"", nil},
		{"123 abc", []string{"123", "abc"}},
	}
	for _, tt := range tests {
		tokens := Tokenize(tt.input)
		if len(tokens) != len(tt.expected) {
			t.Errorf("Tokenize(%q) = %v, want %v", tt.input, tokens, tt.expected)
		}
	}
}

func TestNormalizeToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "hello"},
		{"WORLD", "world"},
		{" Foo ", "foo"},
	}
	for _, tt := range tests {
		result := NormalizeToken(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeToken(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsValidToken(t *testing.T) {
	if !IsValidToken("hello") {
		t.Error("IsValidToken(hello) = false")
	}
	if !IsValidToken("foo_bar") {
		t.Error("IsValidToken(foo_bar) = false")
	}
	if IsValidToken("hello world") {
		t.Error("IsValidToken(hello world) = true")
	}
	if IsValidToken("") {
		t.Error("IsValidToken('') = true")
	}
}
