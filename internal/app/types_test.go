package app

import (
	"testing"
	"time"
)

func TestActionValidation(t *testing.T) {
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"ls", "-la"}}}
	if a.Type != "run_command" {
		t.Errorf("expected run_command, got %s", a.Type)
	}
}

func TestGuardResultDecision(t *testing.T) {
	g := GuardResult{Decision: "deny", Reason: "blacklisted command"}
	if g.Decision != "deny" {
		t.Errorf("expected deny, got %s", g.Decision)
	}
}

func TestVerifierResult(t *testing.T) {
	v := VerifierResult{Success: true, ExitCode: 0, Argv: []string{"go", "test"}}
	if !v.Success {
		t.Errorf("expected success")
	}
}

func TestStepRecordDuration(t *testing.T) {
	s := StepRecord{Step: 1, Duration: 50 * time.Millisecond}
	if s.Step != 1 {
		t.Errorf("expected step 1")
	}
}

func TestConfigDefault(t *testing.T) {
	cfg := Config{}
	cfg.SetDefaults()
	if cfg.Policy.Default != "ask" {
		t.Errorf("expected default ask, got %s", cfg.Policy.Default)
	}
	if cfg.Agent.MaxSteps != 25 {
		t.Errorf("expected max_steps 25, got %d", cfg.Agent.MaxSteps)
	}
}
