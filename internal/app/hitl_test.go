package app

import (
	"testing"
	"time"
)

func TestHITLApprove(t *testing.T) {
	h := &HITL{}
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"git", "push"}}}
	result := h.Prompt(a, 1*time.Millisecond)
	if result {
		t.Error("expected timeout to reject")
	}
}

func TestHITLTimeout(t *testing.T) {
	h := &HITL{}
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"git", "push"}}}
	result := h.Prompt(a, 1*time.Millisecond)
	if result {
		t.Error("expected timeout to reject")
	}
}

func TestParseHITLInputApproveUnix(t *testing.T) {
	if !parseHITLInput("y\n") {
		t.Error("expected 'y\\n' to approve")
	}
}

func TestParseHITLInputApproveWindows(t *testing.T) {
	if !parseHITLInput("y\r\n") {
		t.Error("expected 'y\\r\\n' to approve")
	}
}

func TestParseHITLInputApproveUppercase(t *testing.T) {
	if !parseHITLInput("Y\r\n") {
		t.Error("expected 'Y\\r\\n' to approve")
	}
}

func TestParseHITLInputRejectN(t *testing.T) {
	if parseHITLInput("N\n") {
		t.Error("expected 'N' to reject")
	}
}

func TestParseHITLInputRejectNWindows(t *testing.T) {
	if parseHITLInput("N\r\n") {
		t.Error("expected 'N\\r\\n' to reject")
	}
}

func TestParseHITLInputRejectEmpty(t *testing.T) {
	if parseHITLInput("\n") {
		t.Error("expected empty input to reject")
	}
}