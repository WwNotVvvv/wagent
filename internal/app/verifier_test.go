package app

import (
	"testing"
)

func TestVerifierSuccess(t *testing.T) {
	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{VerifyCommand: []string{"cmd", "/c", "echo ok"}}}
	result := v.Verify(cfg)
	if !result.Success {
		t.Errorf("expected success, got exit_code=%d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit_code 0, got %d", result.ExitCode)
	}
}

func TestVerifierFailure(t *testing.T) {
	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{VerifyCommand: []string{"cmd", "/c", "exit 1"}}}
	result := v.Verify(cfg)
	if result.Success {
		t.Error("expected failure")
	}
}

func TestVerifierNoCommand(t *testing.T) {
	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{VerifyCommand: nil}}
	result := v.Verify(cfg)
	if !result.Success {
		t.Error("expected success when no verify command configured")
	}
}

func TestVerifierStdoutCapture(t *testing.T) {
	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{VerifyCommand: []string{"cmd", "/c", "echo hello world"}}}
	result := v.Verify(cfg)
	if result.Stdout != "hello world\r\n" {
		t.Errorf("expected 'hello world\\r\\n', got %q", result.Stdout)
	}
}