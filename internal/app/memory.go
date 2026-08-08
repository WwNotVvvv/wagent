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
	file     *os.File
	runID    string
	fileName string
	steps    []StepRecord
	redactFn func(string) string
}

func NewTraceRecorder(cfg *Config, task string, redactFn func(string) string) (*TraceRecorder, error) {
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

	tr := &TraceRecorder{file: f, runID: runID, fileName: filename, redactFn: redactFn}

	metaTask := sanitizeTask(task)
	if redactFn != nil {
		metaTask = redactFn(metaTask)
	}
	meta := map[string]any{
		"run_id":    runID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"task":      metaTask,
	}
	metaData, _ := json.Marshal(meta)
	metaData = append(metaData, '\n')
	f.Write(metaData)

	return tr, nil
}

func (t *TraceRecorder) SetRedactFunc(fn func(string) string) {
	t.redactFn = fn
}

func (t *TraceRecorder) Record(step StepRecord) {
	if t.redactFn != nil {
		step = redactStepRecord(step, t.redactFn)
	}
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

func (t *TraceRecorder) FileName() string {
	return t.fileName
}

func (t *TraceRecorder) WriteTaskBoundary(taskID string, taskIndex int, task string) {
	task = sanitizeTask(task)
	if t.redactFn != nil {
		task = t.redactFn(task)
	}
	boundary := map[string]any{
		"type":       "task_start",
		"task_id":    taskID,
		"task_index": taskIndex,
		"task":       task,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(boundary)
	data = append(data, '\n')
	t.file.Write(data)
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

func redactStepRecord(s StepRecord, fn func(string) string) StepRecord {
	s.Message = fn(s.Message)
	s.Error = fn(s.Error)
	s.Action.Message = fn(s.Action.Message)
	if s.Guard != nil {
		g := *s.Guard
		g.Reason = fn(g.Reason)
		s.Guard = &g
	}
	if s.Action.Args != nil {
		s.Action.Args = redactAny(s.Action.Args, fn).(map[string]any)
	}
	if s.Verifier != nil {
		v := *s.Verifier
		v.Stdout = fn(v.Stdout)
		v.Stderr = fn(v.Stderr)
		v.Summary = fn(v.Summary)
		s.Verifier = &v
	}
	if s.ToolResult != nil {
		s.ToolResult = redactAny(s.ToolResult, fn).(map[string]any)
	}
	return s
}

func redactAny(v any, fn func(string) string) any {
	switch val := v.(type) {
	case string:
		return fn(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = redactAny(vv, fn)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = redactAny(vv, fn)
		}
		return out
	case []string:
		out := make([]string, len(val))
		for i, s := range val {
			out[i] = fn(s)
		}
		return out
	default:
		return v
	}
}
