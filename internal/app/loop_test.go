package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestLoopReadFileFeedback(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello from test file"), 0644)

	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10, WorkDir: dir},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "read_file", Args: map[string]any{"path": testFile}}, "reading file")
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

	messages := h.ctx.Messages()
	foundContent := false
	for _, msg := range messages {
		if strContains(msg, "hello from test file") {
			foundContent = true
			break
		}
	}
	if !foundContent {
		t.Error("read_file feedback should contain file content")
	}
}

func strContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type errorMockLLM struct {
	responses []struct {
		action  Action
		message string
		err     error
	}
	index int
}

func (m *errorMockLLM) Chat(context []string, task string) (Action, string, error) {
	if m.index >= len(m.responses) {
		return Action{}, "", fmt.Errorf("mock exhausted")
	}
	r := m.responses[m.index]
	m.index++
	return r.action, r.message, r.err
}

func TestLoopRecoverFromParseError(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0644)

	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10, WorkDir: dir},
		Policy: PolicyConfig{Default: "allow"},
	}

	mock := &errorMockLLM{}
	mock.responses = []struct {
		action  Action
		message string
		err     error
	}{
		{action: Action{Type: "read_file", Args: map[string]any{"path": testFile}}, message: "", err: nil},
		{action: Action{}, message: "", err: fmt.Errorf("response is not JSON: expected '{' or '[' but got \"# wagent\"")},
		{action: Action{Type: "done"}, message: "recovered", err: nil},
	}

	h := &Harness{
		cfg:   cfg,
		llm:   mock,
		guard: &Guardrail{},
		tools: NewToolRegistry(),
		verif: &Verifier{},
		ctx:   NewContext(),
	}

	result, err := h.Run("test recovery")
	if err != nil {
		t.Fatal(err)
	}
	if result != "recovered" {
		t.Errorf("expected 'recovered', got %s", result)
	}
}

func TestLoopOnStepEvents(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "hello"}}}, "echoing")
	llm.AddResponse(Action{Type: "done"}, "finished")

	h := NewHarness(cfg, llm)

	var events []StepEvent
	h.SetOnStep(func(ev StepEvent) {
		events = append(events, ev)
	})

	_, err := h.Run("test task")
	if err != nil {
		t.Fatal(err)
	}

	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}

	if events[0].Phase != StepEventAction {
		t.Errorf("first event phase: expected action, got %s", events[0].Phase)
	}
	if events[0].Action.Type != "run_command" {
		t.Errorf("first event action type: expected run_command, got %s", events[0].Action.Type)
	}

	if events[1].Phase != StepEventGuard {
		t.Errorf("second event phase: expected guard, got %s", events[1].Phase)
	}
	if events[1].Decision != "allow" {
		t.Errorf("second event decision: expected allow, got %s", events[1].Decision)
	}

	if events[2].Phase != StepEventResult {
		t.Errorf("third event phase: expected result, got %s", events[2].Phase)
	}

	foundDone := false
	for _, ev := range events {
		if ev.Action.Type == "done" {
			foundDone = true
			break
		}
	}
	if !foundDone {
		t.Error("expected a done action event")
	}

	t.Logf("Received %d events:", len(events))
	for i, ev := range events {
		t.Logf("  [%d] phase=%s action=%s decision=%s summary=%s", i, ev.Phase, ev.Action.Type, ev.Decision, ev.Summary)
	}
}

func TestLoopOnStepDenyEvent(t *testing.T) {
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

	h := NewHarness(cfg, llm)

	var events []StepEvent
	h.SetOnStep(func(ev StepEvent) {
		events = append(events, ev)
	})

	_, err := h.Run("test task")
	if err != nil {
		t.Fatal(err)
	}

	foundDeny := false
	for _, ev := range events {
		if ev.Phase == StepEventGuard && ev.Decision == "deny" {
			foundDeny = true
			break
		}
	}
	if !foundDeny {
		t.Error("expected a deny guard event")
	}
}

func TestLoopOnStepNilCallback(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "done"}, "ok")

	h := NewHarness(cfg, llm)

	result, err := h.Run("test")
	if err != nil {
		t.Fatal(err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %s", result)
	}
}

func TestLoopMultiTaskWithReset(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "task1"}}}, "task1 step")
	llm.AddResponse(Action{Type: "done"}, "task1 done")
	llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "task2"}}}, "task2 step")
	llm.AddResponse(Action{Type: "done"}, "task2 done")

	h := NewHarness(cfg, llm)

	var events []StepEvent
	h.SetOnStep(func(ev StepEvent) {
		events = append(events, ev)
	})

	result1, err := h.Run("task 1")
	if err != nil {
		t.Fatal(err)
	}
	if result1 != "task1 done" {
		t.Errorf("task1: expected 'task1 done', got %s", result1)
	}

	h.Reset()

	result2, err := h.Run("task 2")
	if err != nil {
		t.Fatal(err)
	}
	if result2 != "task2 done" {
		t.Errorf("task2: expected 'task2 done', got %s", result2)
	}

	msgs := h.ctx.Messages()
	if len(msgs) == 0 {
		t.Fatal("context should have messages from task 2")
	}
	foundTask2 := false
	for _, msg := range msgs {
		if strings.Contains(msg, "task 2") {
			foundTask2 = true
			break
		}
	}
	if !foundTask2 {
		t.Errorf("context should contain task 2, messages: %v", msgs)
	}
}

func TestLoopMultiTaskWithoutResetContextAccumulates(t *testing.T) {
	cfg := &Config{
		Agent:  AgentConfig{MaxSteps: 10},
		Policy: PolicyConfig{Default: "allow"},
	}
	llm := NewMockLLM()
	llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "task1"}}}, "task1 step")
	llm.AddResponse(Action{Type: "done"}, "task1 done")
	llm.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "task2"}}}, "task2 step")
	llm.AddResponse(Action{Type: "done"}, "task2 done")

	h := NewHarness(cfg, llm)

	_, err := h.Run("task 1")
	if err != nil {
		t.Fatal(err)
	}

	ctxBefore := len(h.ctx.Messages())

	_, err = h.Run("task 2")
	if err != nil {
		t.Fatal(err)
	}

	ctxAfter := len(h.ctx.Messages())
	if ctxAfter <= ctxBefore {
		t.Errorf("context should accumulate: before=%d, after=%d", ctxBefore, ctxAfter)
	}
}
