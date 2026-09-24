package search

import (
	"regexp"
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

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(query string) (*QueryAST, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	return p.parse(query)
}

func (p *Parser) parse(query string) (*QueryAST, error) {
	if strings.Contains(query, " AND ") {
		parts := strings.SplitN(query, " AND ", 2)
		left, err := p.parse(parts[0])
		if err != nil {
			return nil, err
		}
		right, err := p.parse(parts[1])
		if err != nil {
			return nil, err
		}
		return &QueryAST{Type: QueryBoolean, Left: left, Right: right, Operator: "AND"}, nil
	}
	if strings.Contains(query, " OR ") {
		parts := strings.SplitN(query, " OR ", 2)
		left, err := p.parse(parts[0])
		if err != nil {
			return nil, err
		}
		right, err := p.parse(parts[1])
		if err != nil {
			return nil, err
		}
		return &QueryAST{Type: QueryBoolean, Left: left, Right: right, Operator: "OR"}, nil
	}
	if strings.Contains(query, " NOT ") {
		parts := strings.SplitN(query, " NOT ", 2)
		left, err := p.parse(parts[0])
		if err != nil {
			return nil, err
		}
		right, err := p.parse(parts[1])
		if err != nil {
			return nil, err
		}
		return &QueryAST{Type: QueryBoolean, Left: left, Right: right, Operator: "NOT"}, nil
	}
	if strings.Contains(query, "*") {
		return &QueryAST{Type: QueryWildcard, Token: strings.TrimSuffix(strings.TrimPrefix(query, "*"), "*")}, nil
	}
	if strings.Contains(query, "~") {
		parts := strings.Split(query, "~")
		k := 1
		if len(parts) > 1 {
			k = int(parts[1][0] - '0')
			if k > 3 {
				k = 3
			}
		}
		return &QueryAST{Type: QueryFuzzy, Token: parts[0], Operator: string(rune(k))}, nil
	}
	return &QueryAST{Type: QueryExact, Token: query}, nil
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
