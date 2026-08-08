package app

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

type Verifier struct{}

func (v *Verifier) Verify(cfg *Config) VerifierResult {
	if len(cfg.Agent.VerifyCommand) == 0 {
		return VerifierResult{Success: true, Argv: nil}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cfg.Agent.VerifyCommand[0], cfg.Agent.VerifyCommand[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if cfg.Agent.WorkDir != "" {
		cmd.Dir = cfg.Agent.WorkDir
	}

	cmd.Env = filterEnv(nil, "")

	err := cmd.Run()
	result := VerifierResult{
		Argv: cfg.Agent.VerifyCommand,
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Timeout = true
		result.Success = false
		return result
	}

	outStr := stdout.String()
	errStr := stderr.String()
	result.Stdout = truncate(outStr, 4096)
	result.Stderr = truncate(errStr, 4096)
	result.Summary = summarizeOutput(outStr, errStr)

	if err != nil {
		result.Success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "\n... [truncated]"
	}
	return s
}

func summarizeOutput(stdout, stderr string) string {
	lines := strings.Split(stdout, "\n")
	if len(lines) > 10 {
		lines = lines[:10]
	}
	summary := strings.Join(lines, "\n")
	if stderr != "" {
		summary += "\n--- stderr ---\n" + stderr
	}
	return truncate(summary, 2048)
}