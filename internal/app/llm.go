package app

import (
	"encoding/json"
	"fmt"
	"os"
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