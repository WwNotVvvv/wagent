package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteConfigTemplateCreatesAcademyDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wagent.toml")

	if err := WriteConfigTemplate(path); err != nil {
		t.Fatalf("WriteConfigTemplate() error = %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.LLM.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "deepseek-v4-flash" {
		t.Fatalf("model = %q, want deepseek-v4-flash", cfg.LLM.Model)
	}
	if cfg.LLM.BaseURL != "https://njusehub.info/v1" {
		t.Fatalf("base_url = %q, want academy endpoint", cfg.LLM.BaseURL)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(strings.ToLower(string(data)), "api_key") {
		t.Fatal("generated config must not contain an API key field")
	}
}

func TestWriteConfigTemplateDoesNotOverwriteExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wagent.toml")
	original := []byte("existing configuration\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := WriteConfigTemplate(path)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("WriteConfigTemplate() error = %v, want os.ErrExist", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("existing configuration was modified: got %q", string(data))
	}
}
