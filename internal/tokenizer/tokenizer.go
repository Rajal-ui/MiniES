package tokenizer

import (
	"regexp"
	"strings"
	"unicode"
)

var tokenPattern = regexp.MustCompile(`[a-zA-Z0-9_]+`)

func Tokenize(text string) []string {
	tokens := tokenPattern.FindAllString(text, -1)
	for i, t := range tokens {
		tokens[i] = strings.ToLower(t)
	}
	return tokens
}

func TokenizeLower(text string) []string {
	return Tokenize(text)
}

func NormalizeToken(token string) string {
	return strings.ToLower(strings.TrimSpace(token))
}

func IsValidToken(token string) bool {
	if len(token) == 0 {
		return false
	}
	for _, r := range token {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}
