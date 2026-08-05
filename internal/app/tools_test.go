package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	reg := NewToolRegistry()
	a := Action{Type: "read_file", Args: map[string]any{"path": path}}
	result, err := reg.Execute(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result["content"] != "hello world" {
		t.Errorf("expected 'hello world', got %v", result["content"])
	}
}

func TestToolReadFileNotFound(t *testing.T) {
	reg := NewToolRegistry()
	a := Action{Type: "read_file", Args: map[string]any{"path": "/nonexistent/file.txt"}}
	_, err := reg.Execute(a, nil)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestToolWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")

	reg := NewToolRegistry()
	a := Action{Type: "write_file", Args: map[string]any{"path": path, "content": "hello"}}
	_, err := reg.Execute(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %s", string(data))
	}
}

func TestToolRunCommand(t *testing.T) {
	reg := NewToolRegistry()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"cmd.exe", "/c", "echo", "hello"}}}
	result, err := reg.Execute(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result["stdout"] != "hello\r\n" && result["stdout"] != "hello\n" {
		t.Errorf("expected 'hello\\n', got %v", result["stdout"])
	}
	if result["exit_code"] != 0 {
		t.Errorf("expected exit_code 0, got %v", result["exit_code"])
	}
}

func TestToolTakeNote(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Storage: StorageConfig{MemoryDir: dir}}
	reg := NewToolRegistry()
	a := Action{Type: "take_note", Args: map[string]any{"content": "important note"}}
	_, err := reg.Execute(a, cfg)
	if err != nil {
		t.Fatal(err)
	}
	notesPath := filepath.Join(dir, "notes.jsonl")
	data, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty notes file")
	}
}

func TestToolSearchMemory(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Storage: StorageConfig{MemoryDir: dir}}
	reg := NewToolRegistry()

	reg.Execute(Action{Type: "take_note", Args: map[string]any{"content": "using logger library"}}, cfg)

	a := Action{Type: "search_memory", Args: map[string]any{"keyword": "logger"}}
	result, err := reg.Execute(a, cfg)
	if err != nil {
		t.Fatal(err)
	}
	notes, ok := result["notes"].([]string)
	if !ok || len(notes) == 0 {
		t.Error("expected to find matching notes")
	}
}

func TestToolUnknownType(t *testing.T) {
	reg := NewToolRegistry()
	a := Action{Type: "unknown_type", Args: map[string]any{}}
	_, err := reg.Execute(a, nil)
	if err == nil {
		t.Error("expected error for unknown tool type")
	}
}