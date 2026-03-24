package logger

import (
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
