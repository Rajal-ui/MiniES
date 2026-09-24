package logrecord

import (
	"time"
)

type Level string

const (
	LevelDEBUG Level = "DEBUG"
	LevelINFO  Level = "INFO"
	LevelWARN  Level = "WARN"
	LevelERROR Level = "ERROR"
	LevelFATAL Level = "FATAL"
)

type LogRecord struct {
	Timestamp  int64  `json:"timestamp"`
	Service    string `json:"service"`
	Level      Level  `json:"level"`
	Message    string `json:"message"`
	SourceFile string `json:"source_file,omitempty"`
	LineNo     int    `json:"line_no,omitempty"`
}

func (r *LogRecord) Validate() error {
	if r.Service == "" {
		return ErrMissingField("service")
	}
	if r.Message == "" {
		return ErrMissingField("message")
	}
	if r.Level == "" {
		r.Level = LevelINFO
	}
	if r.Timestamp == 0 {
		r.Timestamp = time.Now().UnixMilli()
	}
	return nil
}

type ErrMissingField string

func (e ErrMissingField) Error() string {
	return "missing required field: " + string(e)
}

func (r *LogRecord) Tokenize() []string {
	return Tokenize(r.Message)
}

func Tokenize(text string) []string {
	var tokens []string
	var current []rune
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = nil
			}
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	return tokens
}
