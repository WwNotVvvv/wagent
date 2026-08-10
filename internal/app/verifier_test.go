package app

import (
	"os"
	"strings"
	"testing"
)

func TestVerifierSuccess(t *testing.T) {
	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{VerifyCommand: testCommandEcho("ok")}}
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
	cfg := &Config{Agent: AgentConfig{VerifyCommand: testCommandExit(1)}}
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
	cfg := &Config{Agent: AgentConfig{VerifyCommand: testCommandEcho("hello world")}}
	result := v.Verify(cfg)
	if strings.TrimSpace(result.Stdout) != "hello world" {
		t.Errorf("expected 'hello world\\r\\n', got %q", result.Stdout)
	}
}

func TestVerifierWorkDir(t *testing.T) {
	dir := t.TempDir()
	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{VerifyCommand: testCommandWorkingDirectory(), WorkDir: dir}}
	result := v.Verify(cfg)
	if !result.Success {
		t.Errorf("expected success, got exit_code=%d stderr=%s", result.ExitCode, result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != strings.TrimSpace(dir) {
		t.Errorf("expected work_dir %s in stdout, got %q", dir, result.Stdout)
	}
}

func TestVerifierExecutableNotFound(t *testing.T) {
	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{VerifyCommand: []string{"nonexistent_binary_xyz"}}}
	result := v.Verify(cfg)
	if result.Success {
		t.Error("expected failure for nonexistent executable")
	}
	if result.ExitCode != -1 {
		t.Errorf("expected exit_code -1, got %d", result.ExitCode)
	}
}

func TestVerifierEnvProbe(t *testing.T) {
	if os.Getenv("WAGENT_VERIFIER_ENV_PROBE") != "1" {
		return
	}
	if os.Getenv("WAGENT_API_KEY") != "" {
		t.Fatal("WAGENT_API_KEY was passed to verifier subprocess")
	}
}

func TestVerifierDoesNotPassAPIKey(t *testing.T) {
	t.Setenv("WAGENT_API_KEY", "sk-verifier-environment-secret")
	t.Setenv("WAGENT_VERIFIER_ENV_PROBE", "1")

	v := &Verifier{}
	cfg := &Config{Agent: AgentConfig{
		VerifyCommand: []string{os.Args[0], "-test.run=TestVerifierEnvProbe"},
	}}
	result := v.Verify(cfg)
	if !result.Success {
		t.Fatalf("verifier subprocess received the API key: exit_code=%d stderr=%s", result.ExitCode, result.Stderr)
	}
}

func TestVerifierRedactsOutput(t *testing.T) {
	key := "sk-verifier-output-secret"
	v := &Verifier{}
	v.SetRedactFunc(func(s string) string { return RedactAPIKey(s, key) })
	cfg := &Config{Agent: AgentConfig{VerifyCommand: testCommandEcho(key)}}
	result := v.Verify(cfg)
	if !result.Success {
		t.Fatalf("expected verifier success, got exit_code=%d stderr=%s", result.ExitCode, result.Stderr)
	}
	if strings.Contains(result.Stdout, key) || strings.Contains(result.Stderr, key) || strings.Contains(result.Summary, key) {
		t.Error("verifier result contains unredacted API key")
	}
	if !strings.Contains(result.Stdout, "[REDACTED]") {
		t.Error("verifier stdout should contain a redacted placeholder")
	}
}
