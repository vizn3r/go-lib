package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestColorString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"keyword", "GET /api", string(Green) + "GET" + string(Reset)},
		{"number", "port 8080", string(Cyan) + "8080" + string(Reset)},
		{"hex number", "0xFF", string(Cyan) + "0xFF" + string(Reset)},
		{"float", "latency 3.14ms", string(Cyan) + "3.14" + string(Reset)},
		{"double quoted string", `said "hello world"`, string(Yellow) + `"hello world"` + string(Reset)},
		{"single quoted string", "key 'value'", string(Yellow) + "'value'" + string(Reset)},
		{"error keyword", "error occurred", string(Red) + "error" + string(Reset)},
		{"no match", "plain text", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorString(tt.input)
			if len(tt.contains) > 0 && !containsStr(result, tt.contains) {
				t.Errorf("colorString(%q) = %q, want it to contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAddHighlighter(t *testing.T) {
	before := len(highlighters)
	AddHighlighter(`TODO`, Magenta)
	if len(highlighters) != before+1 {
		t.Fatal("AddHighlighter did not append")
	}

	result := colorString("fix TODO later")
	expected := string(Magenta) + "TODO" + string(Reset)
	if !containsStr(result, expected) {
		t.Errorf("custom highlighter not applied: got %q", result)
	}
}

func TestAddHighlight(t *testing.T) {
	before := len(highlighters)
	AddHighlight("WARN", Yellow)
	if len(highlighters) != before+1 {
		t.Fatal("AddHighlight did not append")
	}

	result := colorString("WARN something")
	expected := string(Yellow) + "WARN" + string(Reset)
	if !containsStr(result, expected) {
		t.Errorf("keyword highlighter not applied: got %q", result)
	}
}

func newTestLogger(buf *bytes.Buffer) *Logger {
	lg := New("TEST", Reset, buf)
	lg.l.SetFlags(0)
	lg.l.SetPrefix("[TEST] ")
	lg.colorOutput = false
	lg.printTime = false
	lg.sync = true
	lg.closed = true
	return lg
}

func TestStackLogs_Consecutive_Identical(t *testing.T) {
	var buf bytes.Buffer
	lg := newTestLogger(&buf)
	lg.stackLogs = true

	lg.Info("Checking guild limits")
	lg.Info("Checking guild limits")
	lg.Info("Checking guild limits")

	out := buf.String()
	if !strings.Contains(out, "[x2]") {
		t.Errorf("expected [x2] in output, got: %q", out)
	}
	if !strings.Contains(out, "[x3]") {
		t.Errorf("expected [x3] in output, got: %q", out)
	}
	if strings.Contains(out, "[x1]") {
		t.Errorf("should not contain [x1], got: %q", out)
	}
	escape := "\033[A\033[K"
	if c := strings.Count(out, escape); c != 2 {
		t.Errorf("expected 2 escape sequences, got %d", c)
	}
}

func TestStackLogs_Different_Breaks(t *testing.T) {
	var buf bytes.Buffer
	lg := newTestLogger(&buf)
	lg.stackLogs = true

	lg.Info("Checking guild limits")
	lg.Info("Checking guild limits")
	lg.Info("Different message")
	lg.Info("Checking guild limits")

	out := buf.String()
	if !strings.Contains(out, "[x2]") {
		t.Errorf("expected [x2] in output, got: %q", out)
	}
	if strings.Contains(out, "[x3]") {
		t.Errorf("should not contain [x3], got: %q", out)
	}
}

func TestStackLogs_Different_Levels(t *testing.T) {
	var buf bytes.Buffer
	lg := newTestLogger(&buf)
	lg.stackLogs = true

	lg.Info("Checking guild limits")
	lg.Warn("Checking guild limits")
	lg.Info("Checking guild limits")

	out := buf.String()
	if strings.Contains(out, "[x2]") {
		t.Errorf("different levels should not stack, got: %q", out)
	}
}

func TestStackLogs_Disabled(t *testing.T) {
	var buf bytes.Buffer
	lg := newTestLogger(&buf)
	lg.stackLogs = false

	lg.Info("Checking guild limits")
	lg.Info("Checking guild limits")
	lg.Info("Checking guild limits")

	out := buf.String()
	times := strings.Count(out, "Checking guild limits")
	if times != 3 {
		t.Errorf("expected 3 lines, got %d", times)
	}
	if strings.Contains(out, "[x") {
		t.Errorf("should not contain stack counter, got: %q", out)
	}
}

func TestStackLogs_Toggle_Resets(t *testing.T) {
	var buf bytes.Buffer
	lg := newTestLogger(&buf)
	lg.stackLogs = true

	lg.Info("Checking guild limits")
	lg.Info("Checking guild limits")

	lg.SetStackLogs(false)
	lg.SetStackLogs(true)

	lg.Info("Checking guild limits")
	lg.Info("Checking guild limits")

	out := buf.String()
	if c := strings.Count(out, "[x2]"); c != 2 {
		t.Errorf("expected two separate [x2] groups after toggle, got %d", c)
	}
}
