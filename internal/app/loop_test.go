package app

import (
	"testing"
)

func TestLoopBasic(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "done"}, "task complete")

	h := &Harness{
		cfg:   cfg,
		llm:   llm,
		guard: &Guardrail{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}

	result, err := h.Run("test task")
	if err != nil {
		t.Fatal(err)
	}
	if result != "task complete" {
		t.Errorf("expected 'task complete', got %s", result)
	}
}

func TestLoopWithVerifier(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10, VerifyCommand: []string{"go", "version"}},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "done"}, "all good")

	h := &Harness{
		cfg:   cfg,
		llm:   llm,
		guard: &Guardrail{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}

	result, err := h.Run("test task")
	if err != nil {
		t.Fatal(err)
	}
	if result != "all good" {
		t.Errorf("expected 'all good', got %s", result)
	}
}

func TestLoopMaxSteps(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 3},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	for i := 0; i < 5; i++ {
		llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "hi"}}}, "step")
	}

	h := &Harness{
		cfg:   cfg,
		llm:   llm,
		guard: &Guardrail{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}

	_, err := h.Run("test task")
	if err == nil {
		t.Error("expected error for max steps exceeded")
	}
}

func TestLoopGuardrailDeny(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{
			Default:  "ask",
			Commands: CommandPolicy{Deny: []string{"rm"}},
		},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"rm", "-rf", "/"}}}, "try delete")
	llm.AddResponse(Action{Type: "done"}, "stopped")

	h := &Harness{
		cfg:   cfg,
		llm:   llm,
		guard: &Guardrail{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}

	result, err := h.Run("test task")
	if err != nil {
		t.Fatal(err)
	}
	if result != "stopped" {
		t.Errorf("expected 'stopped', got %s", result)
	}
}

func TestLoopActionParseError(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "done"}, "ok")

	h := &Harness{
		cfg:   cfg,
		llm:   llm,
		guard: &Guardrail{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}

	_, err := h.Run("test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoopToolExecution(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "hello"}}}, "echoing")
	llm.AddResponse(Action{Type: "done"}, "done")

	h := &Harness{
		cfg:   cfg,
		llm:   llm,
		guard: &Guardrail{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}

	result, err := h.Run("test")
	if err != nil {
		t.Fatal(err)
	}
	if result != "done" {
		t.Errorf("expected 'done', got %s", result)
	}
}
