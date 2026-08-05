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
		{Role: "system", Content: systemPrompt},
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

var systemPrompt = `You are a coding agent that operates on a local project. You MUST respond with exactly one JSON object per turn. The JSON object must have:
- "type": one of the following action types
- "args": a JSON object with the required arguments for that action type
- "message": a short description of what you are doing (optional)

## Available Actions

### read_file
Read the contents of a file.
Required args: {"path": "<file path>"}
Example: {"type": "read_file", "args": {"path": "src/main.go"}, "message": "reading main.go"}

### write_file
Write content to a file.
Required args: {"path": "<file path>", "content": "<file content>"}
Example: {"type": "write_file", "args": {"path": "src/main.go", "content": "package main\n..."}, "message": "writing main.go"}

### run_command
Execute a command using argv (no shell).
Required args: {"argv": ["<command>", "<arg1>", ...]}
Example: {"type": "run_command", "args": {"argv": ["go", "test", "./..."]}, "message": "running tests"}

### take_note
Save a note to cross-session memory.
Required args: {"content": "<note text>"}
Example: {"type": "take_note", "args": {"content": "This project uses gorilla/mux for routing"}, "message": "saving note"}

### search_memory
Search cross-session notes by keyword (case-insensitive).
Required args: {"keyword": "<search term>"}
Example: {"type": "search_memory", "args": {"keyword": "routing"}, "message": "searching memory"}

### done
Signal that the task is complete. No args required.
Example: {"type": "done", "message": "task completed successfully"}

## Rules
- Always respond with valid JSON only, no markdown wrapping, no extra text
- Always include all required args for each action type
- Use run_command for executing shell commands, always pass argv as an array of strings
- When done, return {"type": "done"}`

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
