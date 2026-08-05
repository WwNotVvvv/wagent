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