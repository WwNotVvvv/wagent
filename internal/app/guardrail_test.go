package app

import (
	"testing"
)

func newTestConfig() *Config {
	return &Config{
		Policy: PolicyConfig{
			Default: "ask",
			Commands: CommandPolicy{
				Deny:  []string{"rm -rf", "dd if="},
				Ask:   []string{"git push"},
				Allow: []string{"git status", "ls", "go test"},
			},
			Paths: PathPolicy{
				Deny: []string{"/etc/"},
			},
		},
		Agent: AgentConfig{
			WorkDir: "/home/user/project",
		},
	}
}

func TestGuardrailDenyCommand(t *testing.T) {
	g := &Guardrail{}
	cfg := newTestConfig()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"rm", "-rf", "/tmp"}}}
	result := g.Check(a, cfg)
	if result.Decision != "deny" {
		t.Errorf("expected deny, got %s", result.Decision)
	}
}

func TestGuardrailAllowCommand(t *testing.T) {
	g := &Guardrail{}
	cfg := newTestConfig()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"git", "status"}}}
	result := g.Check(a, cfg)
	if result.Decision != "allow" {
		t.Errorf("expected allow, got %s", result.Decision)
	}
}

func TestGuardrailAskCommand(t *testing.T) {
	g := &Guardrail{}
	cfg := newTestConfig()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"git", "push"}}}
	result := g.Check(a, cfg)
	if result.Decision != "ask" {
		t.Errorf("expected ask, got %s", result.Decision)
	}
}

func TestGuardrailDefaultAsk(t *testing.T) {
	g := &Guardrail{}
	cfg := newTestConfig()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"unknown", "cmd"}}}
	result := g.Check(a, cfg)
	if result.Decision != "ask" {
		t.Errorf("expected ask (default), got %s", result.Decision)
	}
}

func TestGuardrailDenyPath(t *testing.T) {
	g := &Guardrail{}
	cfg := newTestConfig()
	a := Action{Type: "write_file", Args: map[string]any{"path": "/etc/passwd", "content": "hack"}}
	result := g.Check(a, cfg)
	if result.Decision != "deny" {
		t.Errorf("expected deny for /etc/ path, got %s", result.Decision)
	}
}

func TestGuardrailDenyPathTraversal(t *testing.T) {
	g := &Guardrail{}
	cfg := newTestConfig()
	a := Action{Type: "read_file", Args: map[string]any{"path": "../../../etc/passwd"}}
	result := g.Check(a, cfg)
	if result.Decision != "deny" {
		t.Errorf("expected deny for path traversal, got %s", result.Decision)
	}
}

func TestGuardrailNonFileAction(t *testing.T) {
	g := &Guardrail{}
	cfg := newTestConfig()
	a := Action{Type: "take_note", Args: map[string]any{"content": "hello"}}
	result := g.Check(a, cfg)
	if result.Decision == "" {
		result.Decision = "allow"
	}
}