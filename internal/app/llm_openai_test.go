package app

import (
	"testing"
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