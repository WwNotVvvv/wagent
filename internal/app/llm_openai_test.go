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