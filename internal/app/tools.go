package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ToolRegistry struct{}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{}
}

func (r *ToolRegistry) Execute(a Action, cfg *Config) (map[string]any, error) {
	switch a.Type {
	case "read_file":
		return r.readFile(a)
	case "write_file":
		return r.writeFile(a)
	case "run_command":
		return r.runCommand(a, cfg)
	case "take_note":
		return r.takeNote(a, cfg)
	case "search_memory":
		return r.searchMemory(a, cfg)
	default:
		return nil, fmt.Errorf("unknown tool: %s", a.Type)
	}
}

func (r *ToolRegistry) readFile(a Action) (map[string]any, error) {
	path, _ := a.Args["path"].(string)
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return nil, fmt.Errorf("read_file target is a directory; please provide a file path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return map[string]any{"content": string(data)}, nil
}

func (r *ToolRegistry) writeFile(a Action) (map[string]any, error) {
	path, _ := a.Args["path"].(string)
	content, _ := a.Args["content"].(string)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	return map[string]any{"written": path}, nil
}

func (r *ToolRegistry) runCommand(a Action, cfg *Config) (map[string]any, error) {
	argvRaw, _ := a.Args["argv"].([]any)
	argv := make([]string, len(argvRaw))
	for i, v := range argvRaw {
		argv[i] = fmt.Sprint(v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if cfg != nil && cfg.Agent.WorkDir != "" {
		cmd.Dir = cfg.Agent.WorkDir
	}

	cmd.Env = os.Environ()
	cmd.Env = filterEnv(cmd.Env, "WAGENT_API_KEY")

	err := cmd.Run()
	exitCode := 0
	timedOut := false
	var startError string
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			timedOut = true
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			startError = err.Error()
		}
	}

	outStr := stdout.String()
	errStr := stderr.String()

	result := map[string]any{
		"stdout":    outStr,
		"stderr":    errStr,
		"exit_code": exitCode,
		"timeout":   timedOut,
	}
	if startError != "" {
		result["start_error"] = startError
	}
	return result, nil
}

func (r *ToolRegistry) takeNote(a Action, cfg *Config) (map[string]any, error) {
	content, _ := a.Args["content"].(string)
	notesDir := expandPath(cfg.Storage.MemoryDir)
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return nil, fmt.Errorf("create notes dir: %w", err)
	}
	notesPath := filepath.Join(notesDir, "notes.jsonl")
	f, err := os.OpenFile(notesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open notes: %w", err)
	}
	defer f.Close()
	entry := map[string]string{"content": content}
	data, _ := json.Marshal(entry)
	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return nil, fmt.Errorf("write note: %w", err)
	}
	return map[string]any{"saved": true}, nil
}

func (r *ToolRegistry) searchMemory(a Action, cfg *Config) (map[string]any, error) {
	keyword, _ := a.Args["keyword"].(string)
	keyword = strings.ToLower(keyword)
	notesDir := expandPath(cfg.Storage.MemoryDir)
	notesPath := filepath.Join(notesDir, "notes.jsonl")

	f, err := os.Open(notesPath)
	if err != nil {
		return map[string]any{"notes": []string{}}, nil
	}
	defer f.Close()

	var matches []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), keyword) {
			matches = append(matches, line)
		}
	}
	if len(matches) > 10 {
		matches = matches[:10]
	}
	return map[string]any{"notes": matches}, nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func filterEnv(env []string, key string) []string {
	var out []string
	for _, e := range env {
		if !strings.HasPrefix(e, key+"=") {
			out = append(out, e)
		}
	}
	return out
}