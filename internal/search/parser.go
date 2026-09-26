package search

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type QueryType int

const (
	QueryExact QueryType = iota
	QueryWildcard
	QueryFuzzy
	QueryBoolean
)

type QueryAST struct {
	Type     QueryType
	Token    string
	Left     *QueryAST
	Right    *QueryAST
	Operator string
}

// Parser parses a query string into a QueryAST. A Parser keeps per-call
// state, so calls must not be made concurrently.
type Parser struct {
	tokens []string
	pos    int
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(query string) (*QueryAST, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	tokens := tokenizeQuery(query)
	p.tokens = tokens
	p.pos = 0
	ast, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.tokens) {
		return nil, &QueryError{"unexpected token: " + p.tokens[p.pos]}
	}
	return ast, nil
}

// parseOr handles the lowest-precedence boolean operator.
func (p *Parser) parseOr() (*QueryAST, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek() == "OR" {
		p.pos++
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &QueryAST{Type: QueryBoolean, Left: left, Right: right, Operator: "OR"}
	}
	return left, nil
}

func (p *Parser) parseAnd() (*QueryAST, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek() == "AND" {
		p.pos++
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &QueryAST{Type: QueryBoolean, Left: left, Right: right, Operator: "AND"}
	}
	return left, nil
}

func (p *Parser) parseNot() (*QueryAST, error) {
	if p.peek() == "NOT" {
		p.pos++
		operand, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		empty := &QueryAST{Type: QueryExact, Token: ""}
		return &QueryAST{Type: QueryBoolean, Left: empty, Right: operand, Operator: "NOT"}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (*QueryAST, error) {
	tok := p.peek()
	if tok == "" {
		return nil, &QueryError{"unexpected end of query"}
	}
	if tok == "(" {
		p.pos++
		ast, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek() != ")" {
			return nil, &QueryError{"missing closing parenthesis"}
		}
		p.pos++
		return ast, nil
	}
	if tok == ")" || tok == "AND" || tok == "OR" {
		return nil, &QueryError{"unexpected token: " + tok}
	}
	p.pos++
	return buildTermAST(tok), nil
}

func (p *Parser) peek() string {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return ""
}

// buildTermAST turns a single term into an exact/wildcard/fuzzy leaf node.
func buildTermAST(term string) *QueryAST {
	if idx := strings.Index(term, "~"); idx >= 0 {
		base := term[:idx]
		k := 1
		if n, err := strconv.Atoi(term[idx+1:]); err == nil {
			k = n
		}
		if k < 1 {
			k = 1
		}
		if k > 3 {
			k = 3
		}
		if strings.Contains(base, "*") {
			return &QueryAST{Type: QueryWildcard, Token: base}
		}
		return &QueryAST{Type: QueryFuzzy, Token: base, Operator: strconv.Itoa(k)}
	}
	if strings.Contains(term, "*") {
		return &QueryAST{Type: QueryWildcard, Token: term}
	}
	return &QueryAST{Type: QueryExact, Token: term}
}

var termRe = regexp.MustCompile(`[^\s()]+`)

// tokenizeQuery splits a query into boolean keywords, parentheses and terms.
// Keywords keep their original case (AND/OR/NOT); terms are lowercased.
func tokenizeQuery(query string) []string {
	var tokens []string
	for _, raw := range termRe.FindAllString(query, -1) {
		switch raw {
		case "AND", "OR", "NOT":
			tokens = append(tokens, raw)
		default:
			tokens = append(tokens, strings.ToLower(raw))
		}
	}
	return tokens
}

var ErrEmptyQuery = &QueryError{"empty query"}

type QueryError struct {
	msg string
}

func (e *QueryError) Error() string { return e.msg }

func ParseQueryTerm(term string) string {
	return strings.ToLower(strings.TrimFunc(term, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}))
}

func WildcardMatch(pattern string, text string) bool {
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	matched, _ := regexp.MatchString("^"+regexPattern+"$", text)
	return matched
}

func TokenizeQuery(query string) []string {
	re := regexp.MustCompile(`[a-zA-Z0-9_*~]+`)
	return re.FindAllString(query, -1)
}
