package app

import (
	"encoding/json"
	"fmt"
	"strings"
)

var knownTypes = map[string][]string{
	"read_file":     {"path"},
	"write_file":    {"path", "content"},
	"run_command":   {"argv"},
	"take_note":     {"content"},
	"search_memory": {"keyword"},
	"done":          {},
}

func ParseAction(raw string) (Action, error) {
	cleaned := cleanJSON(raw)
	var a Action
	if err := json.Unmarshal([]byte(cleaned), &a); err != nil {
		return Action{}, fmt.Errorf("parse action json: %w", err)
	}
	if err := ValidateAction(a); err != nil {
		return Action{}, err
	}
	return a, nil
}

func ValidateAction(a Action) error {
	required, ok := knownTypes[a.Type]
	if !ok {
		return fmt.Errorf("unknown action type: %s", a.Type)
	}
	if a.Args == nil {
		a.Args = make(map[string]any)
	}
	for _, field := range required {
		if _, exists := a.Args[field]; !exists {
			return fmt.Errorf("action %s missing required arg: %s", a.Type, field)
		}
	}
	return nil
}

func cleanJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.SplitN(raw, "\n", 2)
		if len(lines) == 2 {
			raw = strings.TrimSuffix(lines[1], "```")
			raw = strings.TrimSpace(raw)
		}
	}
	return raw
}