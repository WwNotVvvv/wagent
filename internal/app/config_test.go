package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := []byte(`
[llm]
provider = "openai"
model = "gpt-4o"

[agent]
max_steps = 10
verify_command = ["go", "test", "./..."]

[policy.commands]
deny = ["rm -rf"]
ask = ["git push"]
allow = ["git status"]

[policy.paths]
deny = ["/etc/"]
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "wagent.toml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected openai, got %s", cfg.LLM.Provider)
	}
	if cfg.Policy.Default != "ask" {
		t.Errorf("expected default ask, got %s", cfg.Policy.Default)
	}
	if len(cfg.Policy.Commands.Deny) != 1 || cfg.Policy.Commands.Deny[0] != "rm -rf" {
		t.Errorf("deny list wrong: %v", cfg.Policy.Commands.Deny)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path.toml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfigMissingFileWrappedError(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path.toml")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("errors.Is(err, os.ErrNotExist) should be true, got false: %v", err)
	}
}

func TestLoadConfigOrDefaultMissingFile(t *testing.T) {
	cfg, err := LoadConfigOrDefault("/nonexistent/path.toml")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if cfg.Agent.MaxSteps != 25 {
		t.Errorf("expected default max_steps 25, got %d", cfg.Agent.MaxSteps)
	}
	if cfg.Policy.Default != "ask" {
		t.Errorf("expected default policy ask, got %s", cfg.Policy.Default)
	}
	if cfg.Agent.StepTimeout != "60s" {
		t.Errorf("expected default step_timeout 60s, got %s", cfg.Agent.StepTimeout)
	}
}

func TestLoadConfigInvalidDefault(t *testing.T) {
	content := []byte(`
[policy]
default = "invalid"
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	os.WriteFile(path, content, 0644)
	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid default value")
	}
}