package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTraceRecorder(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Storage: StorageConfig{TraceDir: dir}}
	tr, err := NewTraceRecorder(cfg, "test task")
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
	tr, err := NewTraceRecorder(cfg, "test redaction")
	if err != nil {
		t.Fatal(err)
	}

	testKey := "sk-test-trace-redaction-key"
	tr.SetRedactFunc(func(s string) string {
		return RedactAPIKey(s, testKey)
	})

	tr.Record(StepRecord{
		Step:    1,
		Message: "using key " + testKey,
		Action:  Action{Type: "read_file", Args: map[string]any{"path": "test.txt"}},
		ToolResult: map[string]any{
			"content": "file content with " + testKey + " embedded",
			"written": "test.txt",
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