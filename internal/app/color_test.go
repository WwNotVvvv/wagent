package app

import (
	"os"
	"strings"
	"testing"
)

func TestColorizeEnabled(t *testing.T) {
	result := Colorize(true, "hello", ansiGreen)
	if !strings.Contains(result, ansiGreen) {
		t.Error("enabled Colorize should contain ANSI code")
	}
	if !strings.Contains(result, ansiReset) {
		t.Error("enabled Colorize should contain reset code")
	}
	if !strings.Contains(result, "hello") {
		t.Error("enabled Colorize should contain original text")
	}
}

func TestColorizeDisabled(t *testing.T) {
	result := Colorize(false, "hello", ansiGreen)
	if result != "hello" {
		t.Errorf("disabled Colorize should return plain text, got %q", result)
	}
}

func TestColorizeEmpty(t *testing.T) {
	result := Colorize(true, "", ansiGreen)
	if result != "" {
		t.Errorf("empty text should return empty, got %q", result)
	}
}

func TestGreen(t *testing.T) {
	r := Green(true, "ok")
	if !strings.Contains(r, ansiGreen) {
		t.Error("Green should contain green ANSI code")
	}
	r2 := Green(false, "ok")
	if r2 != "ok" {
		t.Error("Green with enabled=false should return plain text")
	}
}

func TestRed(t *testing.T) {
	r := Red(true, "error")
	if !strings.Contains(r, ansiRed) {
		t.Error("Red should contain red ANSI code")
	}
}

func TestCyan(t *testing.T) {
	r := Cyan(true, "action")
	if !strings.Contains(r, ansiCyan) {
		t.Error("Cyan should contain cyan ANSI code")
	}
}

func TestYellow(t *testing.T) {
	r := Yellow(true, "ask")
	if !strings.Contains(r, ansiYellow) {
		t.Error("Yellow should contain yellow ANSI code")
	}
}

func TestGray(t *testing.T) {
	r := Gray(true, "result")
	if !strings.Contains(r, ansiGray) {
		t.Error("Gray should contain gray ANSI code")
	}
}

func TestBold(t *testing.T) {
	r := Bold(true, "bold")
	if !strings.Contains(r, ansiBold) {
		t.Error("Bold should contain bold ANSI code")
	}
}

func TestColorModeAlways(t *testing.T) {
	if !ColorMode("always") {
		t.Error("ColorMode always should return true")
	}
}

func TestColorModeNever(t *testing.T) {
	if ColorMode("never") {
		t.Error("ColorMode never should return false")
	}
}

func TestColorModeAutoNoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	if ColorMode("auto") {
		t.Error("ColorMode auto should return false when NO_COLOR is set")
	}
}

func TestColorModeAutoNoColorEmpty(t *testing.T) {
	os.Setenv("NO_COLOR", "")
	defer os.Unsetenv("NO_COLOR")
	if ColorMode("auto") {
		t.Error("ColorMode auto should return false when NO_COLOR is empty string")
	}
}

func TestColorModeDefaultIsAuto(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	if ColorMode("") {
		t.Error("ColorMode default (empty) should behave like auto and respect NO_COLOR")
	}
}

func TestColorizeNoANSICodesInPlainOutput(t *testing.T) {
	text := "sk-proj-1234567890abcdef"
	result := Colorize(false, text, ansiGreen)
	if strings.Contains(result, "\033") {
		t.Error("disabled Colorize should not contain ANSI escape codes")
	}
	if result != text {
		t.Errorf("disabled Colorize should return exact input, got %q", result)
	}
}

func TestAllColorFuncsDisabledReturnPlain(t *testing.T) {
	text := "test message"
	funcs := []func(bool, string) string{Green, Red, Cyan, Yellow, Gray, Bold}
	for _, fn := range funcs {
		r := fn(false, text)
		if r != text {
			t.Errorf("disabled color func should return plain text, got %q", r)
		}
		if strings.Contains(r, "\033") {
			t.Error("disabled color func should not contain ANSI codes")
		}
	}
}

func TestAllColorFuncsEnabledContainANSICodes(t *testing.T) {
	text := "test message"
	funcs := []struct {
		fn   func(bool, string) string
		code string
	}{
		{Green, ansiGreen},
		{Red, ansiRed},
		{Cyan, ansiCyan},
		{Yellow, ansiYellow},
		{Gray, ansiGray},
		{Bold, ansiBold},
	}
	for _, f := range funcs {
		r := f.fn(true, text)
		if !strings.Contains(r, f.code) {
			t.Errorf("enabled color func should contain ANSI code")
		}
		if !strings.Contains(r, text) {
			t.Errorf("enabled color func should contain original text")
		}
	}
}