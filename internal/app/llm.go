package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type LLM interface {
	Chat(context []string, task string) (Action, string, error)
}

type mockResponse struct {
	Action  Action `json:"action"`
	Message string `json:"message"`
}

type MockLLM struct {
	responses []mockResponse
	index     int
}

func NewMockLLM() *MockLLM {
	return &MockLLM{}
}

func (m *MockLLM) AddResponse(a Action, msg string) {
	m.responses = append(m.responses, mockResponse{Action: a, Message: msg})
}

func (m *MockLLM) LoadScript(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read mock script: %w", err)
	}
	var responses []mockResponse
	if err := json.Unmarshal(data, &responses); err != nil {
		return fmt.Errorf("parse mock script: %w", err)
	}
	m.responses = responses
	m.index = 0
	return nil
}

func (m *MockLLM) Chat(context []string, task string) (Action, string, error) {
	if m.index >= len(m.responses) {
		return Action{}, "", fmt.Errorf("mock script exhausted")
	}
	r := m.responses[m.index]
	m.index++
	return r.Action, r.Message, nil
}

type OpenAILLM struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []any         `json:"tools,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewOpenAILLM(apiKey, model, baseURL string) *OpenAILLM {
	return &OpenAILLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (o *OpenAILLM) Chat(context []string, task string) (Action, string, error) {
	req, err := o.buildChatRequest(context, task)
	if err != nil {
		return Action{}, "", err
	}
	return o.doChat(req)
}

func (o *OpenAILLM) doChat(req chatRequest) (Action, string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return Action{}, "", fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(o.baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return Action{}, "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	client := o.client
	resp, err := client.Do(httpReq)
	if err != nil {
		return Action{}, "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Action{}, "", fmt.Errorf("read response: %w", err)
	}

	return o.parseResponse(respBody)
}

func (o *OpenAILLM) buildChatRequest(context []string, task string) (chatRequest, error) {
	messages := []chatMessage{
		{Role: "system", Content: "You are a coding agent. Return a JSON object with 'type' (action type), 'args' (arguments), and optionally 'message'."},
	}
	for _, msg := range context {
		parts := strings.SplitN(msg, ": ", 2)
		if len(parts) == 2 {
			messages = append(messages, chatMessage{Role: parts[0], Content: parts[1]})
		}
	}
	messages = append(messages, chatMessage{Role: "user", Content: task})
	return chatRequest{Model: o.model, Messages: messages}, nil
}

func (o *OpenAILLM) parseResponse(data []byte) (Action, string, error) {
	var resp chatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return Action{}, "", fmt.Errorf("parse LLM response: %w", err)
	}
	if resp.Error != nil {
		return Action{}, "", fmt.Errorf("LLM API error: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return Action{}, "", fmt.Errorf("empty LLM response")
	}
	content := resp.Choices[0].Message.Content
	action, err := ParseAction(content)
	if err != nil {
		return Action{}, "", err
	}
	return action, action.Message, nil
}
