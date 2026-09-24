package logrecord

import (
	"testing"
	"time"
)

func TestLogRecordValidate(t *testing.T) {
	rec := &LogRecord{Service: "test", Message: "hello"}
	if err := rec.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestLogRecordValidateMissingService(t *testing.T) {
	rec := &LogRecord{Message: "hello"}
	if err := rec.Validate(); err == nil {
		t.Error("Validate() should return error for missing service")
	}
}

func TestLogRecordValidateMissingMessage(t *testing.T) {
	rec := &LogRecord{Service: "test"}
	if err := rec.Validate(); err == nil {
		t.Error("Validate() should return error for missing message")
	}
}

func TestLogRecordAutoTimestamp(t *testing.T) {
	rec := &LogRecord{Service: "test", Message: "hello"}
	rec.Validate()
	if rec.Timestamp == 0 {
		t.Error("Timestamp should be auto-set")
	}
	if rec.Level != LevelINFO {
		t.Errorf("Default level should be INFO, got %s", rec.Level)
	}
}

func TestTokenize(t *testing.T) {
	tokens := Tokenize("hello world foo_bar")
	if len(tokens) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0] != "hello" || tokens[1] != "world" || tokens[2] != "foo_bar" {
		t.Errorf("Unexpected tokens: %v", tokens)
	}
}

func TestTokenizeLower(t *testing.T) {
	tokens := Tokenize("HELLO World")
	if tokens[0] != "HELLO" || tokens[1] != "World" {
		t.Errorf("Tokenize preserves case: %v", tokens)
	}
}

func TestErrMissingField(t *testing.T) {
	err := ErrMissingField("service")
	if err.Error() != "missing required field: service" {
		t.Errorf("Error message mismatch: %s", err.Error())
	}
}

func TestLogRecordTokens(t *testing.T) {
	rec := &LogRecord{Message: "error at line 42"}
	tokens := rec.Tokenize()
	if len(tokens) != 4 {
		t.Errorf("Expected 4 tokens, got %d", len(tokens))
	}
}

func TestLogRecordValidateWithTimestamp(t *testing.T) {
	ts := time.Now().UnixMilli()
	rec := &LogRecord{Service: "test", Message: "msg", Timestamp: ts}
	rec.Validate()
	if rec.Timestamp != ts {
		t.Errorf("Timestamp should be preserved: got %d, want %d", rec.Timestamp, ts)
	}
}
