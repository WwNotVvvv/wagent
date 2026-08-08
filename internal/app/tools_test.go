package app

import (
	"os"
	"path/filepath"
	"strings"
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

func TestToolReadFileDirectory(t *testing.T) {
	dir := t.TempDir()
	reg := NewToolRegistry()
	a := Action{Type: "read_file", Args: map[string]any{"path": dir}}
	_, err := reg.Execute(a, nil)
	if err == nil {
		t.Fatal("expected error when reading a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention directory, got: %v", err)
	}
}

func TestToolRunCommandExecutableNotFound(t *testing.T) {
	reg := NewToolRegistry()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"nonexistent_binary_xyz"}}}
	result, err := reg.Execute(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	startErr, ok := result["start_error"].(string)
	if !ok || startErr == "" {
		t.Error("start_error should be set when executable not found")
	}
	t.Logf("start_error: %s", startErr)
	if result["exit_code"] != -1 {
		t.Errorf("expected exit_code -1, got %v", result["exit_code"])
	}
}

func TestToolRunCommandNonZeroExit(t *testing.T) {
	reg := NewToolRegistry()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"cmd.exe", "/c", "exit", "42"}}}
	result, err := reg.Execute(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result["exit_code"] != 42 {
		t.Errorf("expected exit_code 42, got %v", result["exit_code"])
	}
	if result["timeout"] != false {
		t.Error("expected timeout=false for non-zero exit")
	}
}

func TestToolRunCommandStderr(t *testing.T) {
	reg := NewToolRegistry()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"cmd.exe", "/c", "echo", "error message", ">&2"}}}
	result, err := reg.Execute(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	stderr, _ := result["stderr"].(string)
	if !strings.Contains(stderr, "error message") {
		t.Errorf("stderr should contain 'error message', got: %s", stderr)
	}
}

func TestToolRunCommandWorkDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Agent: AgentConfig{WorkDir: dir}}
	reg := NewToolRegistry()
	a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"cmd.exe", "/c", "cd"}}}
	result, err := reg.Execute(a, cfg)
	if err != nil {
		t.Fatal(err)
	}
	stdout, _ := result["stdout"].(string)
	if !strings.Contains(stdout, dir) {
		t.Errorf("command should run in work_dir %s, got stdout: %s", dir, stdout)
	}
}

func TestFormatToolResultRunCommandStartError(t *testing.T) {
	result := map[string]any{
		"stdout":      "",
		"stderr":      "",
		"exit_code":   -1,
		"timeout":     false,
		"start_error": "executable file not found: sh",
	}
	a := Action{Type: "run_command"}
	summary := formatToolResult(a, result)
	if !strings.Contains(summary, "executable file not found") {
		t.Errorf("summary should contain start_error, got: %s", summary)
	}
}

func TestFormatToolResultRunCommandStderr(t *testing.T) {
	result := map[string]any{
		"stdout":    "",
		"stderr":    "warning: deprecated",
		"exit_code": 0,
		"timeout":   false,
	}
	a := Action{Type: "run_command"}
	summary := formatToolResult(a, result)
	if !strings.Contains(summary, "warning: deprecated") {
		t.Errorf("summary should contain stderr, got: %s", summary)
	}
}

func TestFormatToolResultRunCommandTimeout(t *testing.T) {
	result := map[string]any{
		"stdout":    "",
		"stderr":    "",
		"exit_code": -1,
		"timeout":   true,
	}
	a := Action{Type: "run_command"}
	summary := formatToolResult(a, result)
	if !strings.Contains(summary, "timed out") {
		t.Errorf("summary should indicate timeout, got: %s", summary)
	}
}