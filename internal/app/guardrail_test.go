package app

import (
	"os"
	"path/filepath"
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

func TestGuardrailRejectsSiblingOfWorkDir(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "project")
	sibling := filepath.Join(root, "project-sibling", "secret.txt")
	cfg := &Config{
		Policy: PolicyConfig{Paths: PathPolicy{}},
		Agent:  AgentConfig{WorkDir: workDir},
	}

	result := (&Guardrail{}).Check(Action{
		Type: "read_file",
		Args: map[string]any{"path": sibling},
	}, cfg)
	if result.Decision != "deny" {
		t.Fatalf("expected sibling path to be denied, got %s (%s)", result.Decision, result.Reason)
	}
}

func TestGuardrailAllowsChildOfWorkDir(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "project")
	child := filepath.Join(workDir, "src", "main.go")
	cfg := &Config{
		Policy: PolicyConfig{Paths: PathPolicy{}},
		Agent:  AgentConfig{WorkDir: workDir},
	}

	result := (&Guardrail{}).Check(Action{
		Type: "write_file",
		Args: map[string]any{"path": child, "content": "package main"},
	}, cfg)
	if result.Decision != "allow" {
		t.Fatalf("expected child path to be allowed, got %s (%s)", result.Decision, result.Reason)
	}
}

func TestGuardrailDenyPatternRespectsPathBoundary(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	nearby := filepath.Join(root, "blocked-sibling", "file.txt")
	cfg := &Config{
		Policy: PolicyConfig{Paths: PathPolicy{Deny: []string{blocked}}},
		Agent:  AgentConfig{WorkDir: root},
	}

	result := (&Guardrail{}).Check(Action{
		Type: "read_file",
		Args: map[string]any{"path": nearby},
	}, cfg)
	if result.Decision != "allow" {
		t.Fatalf("expected nearby path not to match deny pattern, got %s (%s)", result.Decision, result.Reason)
	}
}

func TestGuardrailExpandsHomeDenyPattern(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home directory: %v", err)
	}
	cfg := &Config{
		Policy: PolicyConfig{Paths: PathPolicy{Deny: []string{"~/.ssh/"}}},
		Agent:  AgentConfig{WorkDir: home},
	}

	result := (&Guardrail{}).Check(Action{
		Type: "read_file",
		Args: map[string]any{"path": filepath.Join(home, ".ssh", "config")},
	}, cfg)
	if result.Decision != "deny" {
		t.Fatalf("expected ~/.ssh path to be denied, got %s (%s)", result.Decision, result.Reason)
	}
}

func TestGuardrailMatchesRecursiveDenyPattern(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{
		Policy: PolicyConfig{Paths: PathPolicy{Deny: []string{"**/node_modules/**"}}},
		Agent:  AgentConfig{WorkDir: root},
	}

	result := (&Guardrail{}).Check(Action{
		Type: "read_file",
		Args: map[string]any{"path": filepath.Join(root, "src", "node_modules", "package.json")},
	}, cfg)
	if result.Decision != "deny" {
		t.Fatalf("expected recursive deny pattern to match, got %s (%s)", result.Decision, result.Reason)
	}
}

func TestGuardrailRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	cfg := &Config{
		Policy: PolicyConfig{Paths: PathPolicy{}},
		Agent:  AgentConfig{WorkDir: root},
	}
	result := (&Guardrail{}).Check(Action{
		Type: "write_file",
		Args: map[string]any{"path": filepath.Join(link, "secret.txt"), "content": "secret"},
	}, cfg)
	if result.Decision != "deny" {
		t.Fatalf("expected symlink escape to be denied, got %s (%s)", result.Decision, result.Reason)
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
