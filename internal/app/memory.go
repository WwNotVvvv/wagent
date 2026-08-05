package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TraceRecorder struct {
	file  *os.File
	runID string
	steps []StepRecord
}

func NewTraceRecorder(cfg *Config, task string) (*TraceRecorder, error) {
	traceDir := expandPath(cfg.Storage.TraceDir)
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return nil, fmt.Errorf("create trace dir: %w", err)
	}

	runID := generateRunID()
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.jsonl", timestamp, runID)
	path := filepath.Join(traceDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create trace file: %w", err)
	}

	tr := &TraceRecorder{file: f, runID: runID}

	meta := map[string]any{
		"run_id":    runID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"task":      sanitizeTask(task),
	}
	metaData, _ := json.Marshal(meta)
	metaData = append(metaData, '\n')
	f.Write(metaData)

	return tr, nil
}

func (t *TraceRecorder) Record(step StepRecord) {
	t.steps = append(t.steps, step)
	data, err := json.Marshal(step)
	if err != nil {
		return
	}
	data = append(data, '\n')
	t.file.Write(data)
}

func (t *TraceRecorder) Flush() {
	t.file.Close()
}

func (t *TraceRecorder) RunID() string {
	return t.runID
}

func generateRunID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func sanitizeTask(task string) string {
	if len(task) > 100 {
		task = task[:100] + "..."
	}
	task = strings.ReplaceAll(task, "\n", " ")
	return task
}

type Context struct {
	messages []string
}

func NewContext() *Context {
	return &Context{}
}

func (c *Context) AddSystem(msg string) {
	c.messages = append(c.messages, "system: "+msg)
}

func (c *Context) AddUser(msg string) {
	c.messages = append(c.messages, "user: "+msg)
}

func (c *Context) AddAssistant(a Action, msg string) {
	c.messages = append(c.messages, "assistant: "+msg)
}

func (c *Context) Messages() []string {
	return c.messages
}