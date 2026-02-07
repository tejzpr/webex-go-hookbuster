package display

import (
	"strings"
	"testing"
)

func TestQuestion(t *testing.T) {
	result := Question("hello")
	if !strings.Contains(result, "hello") {
		t.Errorf("Question() should contain the text, got %q", result)
	}
	if !strings.Contains(result, bold) {
		t.Error("Question() should contain bold ANSI code")
	}
	if !strings.HasSuffix(result, " ") {
		t.Error("Question() should end with a trailing space")
	}
}

func TestAnswer(t *testing.T) {
	result := Answer("ok")
	if !strings.Contains(result, "ok") {
		t.Errorf("Answer() should contain the text, got %q", result)
	}
	if !strings.Contains(result, green) {
		t.Error("Answer() should contain green ANSI code")
	}
}

func TestInfo(t *testing.T) {
	result := Info("info msg")
	if !strings.Contains(result, "info msg") {
		t.Errorf("Info() should contain the text, got %q", result)
	}
	if !strings.Contains(result, blue) {
		t.Error("Info() should contain blue ANSI code")
	}
}

func TestError(t *testing.T) {
	result := Error("fail")
	if !strings.Contains(result, "fail") {
		t.Errorf("Error() should contain the text, got %q", result)
	}
	if !strings.Contains(result, red) {
		t.Error("Error() should contain red ANSI code")
	}
}

func TestHighlight(t *testing.T) {
	result := Highlight("important")
	if !strings.Contains(result, "important") {
		t.Errorf("Highlight() should contain the text, got %q", result)
	}
	if !strings.Contains(result, magenta) {
		t.Error("Highlight() should contain magenta ANSI code")
	}
}

func TestAllFormattedStringsEndWithReset(t *testing.T) {
	funcs := []struct {
		name string
		fn   func(string) string
	}{
		{"Answer", Answer},
		{"Info", Info},
		{"Error", Error},
		{"Highlight", Highlight},
	}

	for _, f := range funcs {
		result := f.fn("test")
		if !strings.HasSuffix(result, reset) {
			t.Errorf("%s() should end with ANSI reset code, got %q", f.name, result)
		}
	}
}
