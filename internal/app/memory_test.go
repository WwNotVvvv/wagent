package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceRecorder(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Storage: StorageConfig{TraceDir: dir}}
	tr, err := NewTraceRecorder(cfg, "test task", nil)
	if err != nil {
		t.Fatal(err)
	}

	tr.Record(StepRecord{Step: 1, Message: "hello", Action: Action{Type: "done"}})
	tr.Flush()

	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		t.Fatal("expected trace file")
	}
	data, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if len(data) == 0 {
		t.Error("expected non-empty trace")
	}
	content := string(data)
	if containsAPIKey(content) {
		t.Error("trace contains potential API Key pattern")
	}
}

func TestTraceRedaction(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Storage: StorageConfig{TraceDir: dir}}
	testKey := "sk-test-trace-redaction-key"
	redactFn := func(s string) string { return RedactAPIKey(s, testKey) }

	tr, err := NewTraceRecorder(cfg, "task with "+testKey+" embedded", redactFn)
	if err != nil {
		t.Fatal(err)
	}

	tr.Record(StepRecord{
		Step:    1,
		Message: "using key " + testKey,
		Action: Action{
			Type:    "write_file",
			Message: "writing content with " + testKey,
			Args: map[string]any{
				"path":    "test.txt",
				"content": "file content with " + testKey + " embedded",
			},
		},
		ToolResult: map[string]any{
			"content": "result has " + testKey,
			"written": "test.txt",
			"nested": map[string]any{
				"inner": "nested " + testKey,
			},
			"list": []string{"item with " + testKey},
		},
		Error: "error: " + testKey + " not found",
		Verifier: &VerifierResult{
			Stdout:  "stdout has " + testKey,
			Stderr:  "stderr has " + testKey,
			Summary: "summary: " + testKey,
		},
	})
	tr.Flush()

	entries, _ := os.ReadDir(dir)
	data, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	content := string(data)

	if containsStr(content, testKey) {
		t.Error("trace file contains unredacted API key")
	}
	if !containsStr(content, "[REDACTED]") {
		t.Error("trace file should contain [REDACTED] placeholders")
	}
	// verify at least 10 redactions (message, error, action.message, action.args.content,
	// toolresult.content, toolresult.nested.inner, toolresult.list[0],
	// verifier.stdout, verifier.stderr, verifier.summary, metadata task)
	redactedCount := 0
	for i := 0; i <= len(content)-len("[REDACTED]"); i++ {
		if content[i:i+len("[REDACTED]")] == "[REDACTED]" {
			redactedCount++
		}
	}
	if redactedCount < 10 {
		t.Errorf("expected at least 10 [REDACTED] occurrences, got %d", redactedCount)
	}
}

func TestContextAppend(t *testing.T) {
	ctx := NewContext()
	ctx.AddUser("hello agent")
	ctx.AddAssistant(Action{Type: "done"}, "done")
	msgs := ctx.Messages()
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func containsAPIKey(s string) bool {
	patterns := []string{"sk-", "api_key", "api-key", "apiKey", "WAGENT_API_KEY"}
	for _, p := range patterns {
		if len(s) > 0 && contains(s, p) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTraceTaskBoundary(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Storage: StorageConfig{TraceDir: dir},
		Agent:   AgentConfig{MaxSteps: 10},
	}
	cfg.SetDefaults()

	tr, err := NewTraceRecorder(cfg, "task 1", nil)
	if err != nil {
		t.Fatal(err)
	}

	step := StepRecord{
		Step:      1,
		TaskID:    "test-task-id",
		TaskIndex: 1,
		Action:    Action{Type: "done"},
		Message:   "done",
	}
	tr.Record(step)
	tr.Flush()

	data, err := os.ReadFile(filepath.Join(dir, tr.FileName()))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"task_id"`) {
		t.Error("trace should contain task_id field")
	}
	if !strings.Contains(content, `"task_index"`) {
		t.Error("trace should contain task_index field")
	}
}

func TestTraceMultipleTasks(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Storage: StorageConfig{TraceDir: dir},
		Agent:   AgentConfig{MaxSteps: 10},
	}
	cfg.SetDefaults()

	tr, err := NewTraceRecorder(cfg, "first task", nil)
	if err != nil {
		t.Fatal(err)
	}

	tr.Record(StepRecord{
		Step:      1,
		TaskID:    "task-aaa",
		TaskIndex: 1,
		Action:    Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "hello"}}},
		Message:   "step 1",
	})

	tr.Record(StepRecord{
		Step:      2,
		TaskID:    "task-aaa",
		TaskIndex: 1,
		Action:    Action{Type: "done"},
		Message:   "done",
	})

	tr.Record(StepRecord{
		Step:      1,
		TaskID:    "task-bbb",
		TaskIndex: 2,
		Action:    Action{Type: "run_command", Args: map[string]any{"argv": []any{"echo", "world"}}},
		Message:   "step 1",
	})

	tr.Flush()

	data, err := os.ReadFile(filepath.Join(dir, tr.FileName()))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"task_id"`) {
		t.Error("trace should contain task_id")
	}
	if strings.Count(content, `"task_index":1`) < 1 {
		t.Error("trace should contain task_index 1")
	}
	if strings.Count(content, `"task_index":2`) < 1 {
		t.Error("trace should contain task_index 2")
	}
}