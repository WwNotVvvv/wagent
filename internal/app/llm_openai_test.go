package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAILLMBuildRequest(t *testing.T) {
	llm := NewOpenAILLM("sk-test", "gpt-4o", "https://api.openai.com/v1")
	body, err := llm.buildChatRequest([]string{"user: hello"}, "do something")
	if err != nil {
		t.Fatal(err)
	}
	if len(body.Model) == 0 {
		t.Error("expected model name in request")
	}
	if len(body.Messages) == 0 {
		t.Error("expected messages in request")
	}
}

func TestOpenAILLMResponseParse(t *testing.T) {
	llm := NewOpenAILLM("sk-test", "gpt-4o", "https://api.openai.com/v1")
	jsonStr := `{"choices":[{"message":{"content":"{\"type\":\"done\",\"message\":\"ok\"}"}}]}`
	action, msg, err := llm.parseResponse([]byte(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "done" {
		t.Errorf("expected done, got %s", action.Type)
	}
	if msg != "ok" {
		t.Errorf("expected 'ok', got %s", msg)
	}
}

func TestOpenAILLMChatSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer sk-test" {
			t.Errorf("expected 'Bearer sk-test', got %s", auth)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected application/json, got %s", contentType)
		}

		var reqBody chatRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.Model != "gpt-4o" {
			t.Errorf("expected gpt-4o, got %s", reqBody.Model)
		}
		if len(reqBody.Messages) == 0 {
			t.Error("expected non-empty messages")
		}

		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: `{"type": "done", "message": "task complete"}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("sk-test", "gpt-4o", server.URL)
	llm.client = server.Client()

	action, msg, err := llm.Chat([]string{"user: hello"}, "do task")
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "done" {
		t.Errorf("expected done, got %s", action.Type)
	}
	if msg != "task complete" {
		t.Errorf("expected 'task complete', got %s", msg)
	}
}

func TestOpenAILLMChatAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		resp := chatResponse{
			Error: &struct {
				Message string `json:"message"`
			}{Message: "rate limit exceeded"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("sk-test", "gpt-4o", server.URL)
	llm.client = server.Client()

	_, _, err := llm.Chat([]string{}, "test")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestOpenAILLMChatEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("sk-test", "gpt-4o", server.URL)
	llm.client = server.Client()

	_, _, err := llm.Chat([]string{}, "test")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestOpenAILLMChatTimeout(t *testing.T) {
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-done
	}))
	defer func() {
		close(done)
		server.Close()
	}()

	llm := NewOpenAILLM("sk-test", "gpt-4o", server.URL)
	llm.client = server.Client()
	llm.client.Timeout = 1 * time.Millisecond

	_, _, err := llm.Chat([]string{}, "test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestSystemPromptCoversAllActions(t *testing.T) {
	for _, actionType := range []string{
		"read_file", "write_file", "run_command", "take_note", "search_memory", "done",
	} {
		if !strings.Contains(systemPrompt, actionType) {
			t.Errorf("system prompt missing action type: %s", actionType)
		}
	}
}

func TestParseRealResponseReadFile(t *testing.T) {
	llm := NewOpenAILLM("sk-test", "gpt-4o", "https://api.openai.com/v1")
	raw := `{"type": "read_file", "args": {"path": "main.go"}, "message": "reading main.go"}`
	action, msg, err := llm.parseResponse([]byte(`{"choices":[{"message":{"content":"` + escapeJSON(raw) + `"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "read_file" {
		t.Errorf("expected read_file, got %s", action.Type)
	}
	path, ok := action.Args["path"].(string)
	if !ok || path != "main.go" {
		t.Errorf("expected path=main.go, got args=%v", action.Args)
	}
	if msg != "reading main.go" {
		t.Errorf("expected 'reading main.go', got %s", msg)
	}
}

func TestParseRealResponseWriteFile(t *testing.T) {
	raw := `{"type": "write_file", "args": {"path": "test.go", "content": "package main"}, "message": "writing test.go"}`
	action, err := ParseAction(raw)
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "write_file" {
		t.Errorf("expected write_file, got %s", action.Type)
	}
	if action.Args["content"] != "package main" {
		t.Errorf("expected content, got %v", action.Args["content"])
	}
}

func TestParseRealResponseRunCommand(t *testing.T) {
	raw := `{"type": "run_command", "args": {"argv": ["go", "test", "./..."]}, "message": "running tests"}`
	action, err := ParseAction(raw)
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "run_command" {
		t.Errorf("expected run_command, got %s", action.Type)
	}
}

func TestParseRealResponseSearchMemory(t *testing.T) {
	raw := `{"type": "search_memory", "args": {"keyword": "logger"}, "message": "searching"}`
	action, err := ParseAction(raw)
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "search_memory" {
		t.Errorf("expected search_memory, got %s", action.Type)
	}
	if action.Args["keyword"] != "logger" {
		t.Errorf("expected keyword=logger, got %v", action.Args["keyword"])
	}
}

func TestParseRealResponseMissingArgsRejected(t *testing.T) {
	raw := `{"type": "read_file", "args": {}}`
	_, err := ParseAction(raw)
	if err == nil {
		t.Fatal("expected error for missing required args")
	}
}

func TestParseRealResponseNilArgsRejected(t *testing.T) {
	raw := `{"type": "read_file"}`
	_, err := ParseAction(raw)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestParseRealResponseDone(t *testing.T) {
	raw := `{"type": "done", "message": "all tests pass"}`
	action, err := ParseAction(raw); msg := action.Message
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != "done" {
		t.Errorf("expected done, got %s", action.Type)
	}
	if msg != "all tests pass" {
		t.Errorf("expected 'all tests pass', got %s", msg)
	}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
