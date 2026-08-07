package app

import (
	"testing"
)

func TestMechanismGuardrailDeny(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{
			Default: "ask",
			Commands: CommandPolicy{
				Deny:  []string{"rm -rf"},
				Allow: []string{"git diff"},
				Ask:   []string{"git status"},
			},
		},
	}
	guard := &Guardrail{}

	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"rm", "-rf", "/"}}}
	result := guard.Check(a, cfg)
	if result.Decision != "deny" {
		t.Fatalf("expected deny, got %s: %s", result.Decision, result.Reason)
	}
	t.Logf("DENY: %s -- %s", result.Decision, result.Reason)

	a2 := Action{Type: "run_command", Args: map[string]any{"argv": []any{"git", "diff"}}}
	result2 := guard.Check(a2, cfg)
	if result2.Decision != "allow" {
		t.Fatalf("expected allow, got %s", result2.Decision)
	}
	t.Logf("ALLOW: %s -- %s", result2.Decision, result2.Reason)

	a3 := Action{Type: "run_command", Args: map[string]any{"argv": []any{"git", "status"}}}
	result3 := guard.Check(a3, cfg)
	if result3.Decision != "ask" {
		t.Fatalf("expected ask, got %s", result3.Decision)
	}
	t.Logf("ASK: %s -- %s", result3.Decision, result3.Reason)
}

func TestMechanismFeedbackLoop(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{MaxSteps: 10, VerifyCommand: []string{"git", "diff", "--stat"}},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "done"}, "first try")
	llm.AddResponse(Action{Type: "read_file", Args: map[string]any{"path": "test_fail.go"}}, "checking failure")
	llm.AddResponse(Action{Type: "done"}, "fixed")

	h := NewHarness(cfg, llm)
	_, err := h.Run("fix the test")
	if err != nil {
		t.Logf("Note: loop exited with: %v", err)
	}
}

func TestMechanismGovernanceTiers(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{
			Default: "ask",
			Commands: CommandPolicy{
				Deny:  []string{"rm -rf"},
				Allow: []string{"git diff"},
			},
		},
	}
	guard := &Guardrail{}

	actions := []struct {
		name     string
		argv     []any
		expected string
	}{
		{"git status (default ask)", []any{"git", "status"}, "ask"},
		{"rm -rf (deny)", []any{"rm", "-rf", "node_modules"}, "deny"},
		{"git diff (allow)", []any{"git", "diff"}, "allow"},
	}

	for _, tc := range actions {
		a := Action{Type: "run_command", Args: map[string]any{"argv": tc.argv}}
		result := guard.Check(a, cfg)
		if result.Decision != tc.expected {
			t.Errorf("%s: expected %s, got %s", tc.name, tc.expected, result.Decision)
		}
		t.Logf("%s -> %s (%s)", tc.name, result.Decision, result.Reason)
	}
}