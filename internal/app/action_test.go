package app

import "testing"

func TestParseActionValid(t *testing.T) {
    input := `{"type": "run_command", "args": {"argv": ["ls", "-la"]}, "message": "checking files"}`
    a, err := ParseAction(input)
    if err != nil { t.Fatal(err) }
    if a.Type != "run_command" { t.Errorf("expected run_command, got %s", a.Type) }
    if a.Message != "checking files" { t.Errorf("expected message, got %s", a.Message) }
}

func TestParseActionInvalidJSON(t *testing.T) {
    _, err := ParseAction("{invalid}")
    if err == nil { t.Error("expected error for invalid JSON") }
}

func TestParseActionUnknownType(t *testing.T) {
    _, err := ParseAction(`{"type": "fly_to_moon", "args": {}}`)
    if err == nil { t.Error("expected error for unknown type") }
}

func TestParseActionMissingArgs(t *testing.T) {
    _, err := ParseAction(`{"type": "read_file", "args": {}}`)
    if err == nil { t.Error("expected error for missing args") }
}

func TestParseActionDone(t *testing.T) {
    a, err := ParseAction(`{"type": "done", "message": "all done"}`)
    if err != nil { t.Fatal(err) }
    if a.Type != "done" { t.Errorf("expected done, got %s", a.Type) }
}

func TestParseActionMarkdownWrapper(t *testing.T) {
    a, err := ParseAction("```json\n{\"type\": \"done\"}\n```")
    if err != nil { t.Fatal(err) }
    if a.Type != "done" { t.Errorf("expected done, got %s", a.Type) }
}