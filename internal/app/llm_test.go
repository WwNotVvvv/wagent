package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMockLLM(t *testing.T) {
	m := NewMockLLM()
	m.AddResponse(Action{Type: "run_command", Args: map[string]any{"argv": []any{"ls"}}}, "listing files")
	m.AddResponse(Action{Type: "done", Args: nil}, "task complete")

	action, msg, err := m.Chat(nil, "test task")
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "run_command" {
		t.Errorf("expected run_command, got %s", action.Type)
	}
	if msg != "listing files" {
		t.Errorf("expected 'listing files', got %s", msg)
	}

	action, msg, err = m.Chat(nil, "next")
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "done" {
		t.Errorf("expected done, got %s", action.Type)
	}
}

func TestMockLLMExhausted(t *testing.T) {
	m := NewMockLLM()
	_, _, err := m.Chat(nil, "anything")
	if err == nil {
		t.Error("expected error when script exhausted")
	}
}

func TestMockLLMLoadScript(t *testing.T) {
	script := `[
		{"action": {"type": "read_file", "args": {"path": "test.txt"}}, "message": "reading file"},
		{"action": {"type": "done", "args": {}}, "message": "done"}
	]`
	dir := t.TempDir()
	path := filepath.Join(dir, "mock.json")
	os.WriteFile(path, []byte(script), 0644)

	m := NewMockLLM()
	if err := m.LoadScript(path); err != nil {
		t.Fatal(err)
	}
	action, _, _ := m.Chat(nil, "x")
	if action.Type != "read_file" {
		t.Errorf("expected read_file, got %s", action.Type)
	}
}