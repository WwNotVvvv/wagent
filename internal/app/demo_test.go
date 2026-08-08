package app

import (
	"os"
	"path/filepath"
	"strings"
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
	dir := t.TempDir()
	failureFile := filepath.Join(dir, "test_fail.go")
	if err := os.WriteFile(failureFile, []byte("failure details"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10, WorkDir: dir, VerifyCommand: []string{"go", "test", "./..."}},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "done"}, "first try")
	llm.AddResponse(Action{Type: "read_file", Args: map[string]any{"path": failureFile}}, "checking failure")
	llm.AddResponse(Action{Type: "done"}, "fixed")

	h := NewHarness(cfg, llm)
	verifier := &scriptedVerifier{results: []VerifierResult{
		{Success: false, ExitCode: 1, Summary: "test failure: expected value mismatch", Stderr: "failure details"},
		{Success: true, ExitCode: 0, Summary: "ok"},
	}}
	h.verif = verifier
	var actionTypes []string
	h.SetOnStep(func(ev StepEvent) {
		if ev.Phase == StepEventAction {
			actionTypes = append(actionTypes, ev.Action.Type)
		}
	})

	result, err := h.Run("fix the test")
	if err != nil {
		t.Fatal(err)
	}
	if result != "fixed" {
		t.Fatalf("expected final response 'fixed', got %q", result)
	}
	if verifier.calls != 2 {
		t.Fatalf("expected verifier to run twice, got %d calls", verifier.calls)
	}
	wantActions := []string{"done", "read_file", "done"}
	if len(actionTypes) != len(wantActions) {
		t.Fatalf("expected action sequence %v, got %v", wantActions, actionTypes)
	}
	for i := range wantActions {
		if actionTypes[i] != wantActions[i] {
			t.Fatalf("expected action sequence %v, got %v", wantActions, actionTypes)
		}
	}
	joinedContext := strings.Join(h.ctx.Messages(), "\n")
	if !strings.Contains(joinedContext, "Verification failed: exit_code=1") {
		t.Error("agent context should contain verifier failure feedback")
	}
	if !strings.Contains(joinedContext, "failure details") {
		t.Error("agent context should contain verifier failure details")
	}
}

type scriptedVerifier struct {
	results []VerifierResult
	calls   int
}

func (v *scriptedVerifier) Verify(_ *Config) VerifierResult {
	result := v.results[v.calls]
	v.calls++
	return result
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
