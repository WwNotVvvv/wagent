package app

import (
	"fmt"
	"time"
)

const maxParseRetries = 3

func (h *Harness) Run(task string) (string, error) {
	h.ctx.AddUser(task)

	parseErrors := 0

	for step := 1; step <= h.cfg.Agent.MaxSteps; step++ {
		stepStart := time.Now()

		action, msg, err := h.llm.Chat(h.ctx.Messages(), task)
		if err != nil {
			return "", fmt.Errorf("LLM error at step %d: %w", step, err)
		}

		guardResult := h.guard.Check(action, h.cfg)
		record := StepRecord{
			Step:     step,
			Message:  msg,
			Action:   action,
			Guard:    &guardResult,
			Duration: time.Since(stepStart),
		}

		if guardResult.Decision == "deny" {
			h.ctx.AddUser(fmt.Sprintf("Action denied: %s", guardResult.Reason))
			record.Error = guardResult.Reason
			h.recordStep(record)
			continue
		}

		if guardResult.Decision == "ask" {
			timeout := parseDuration(h.cfg.Policy.HITL.Timeout, 120)
			stepTimeout := parseDuration(h.cfg.Agent.StepTimeout, 60)
			if stepTimeout < timeout {
				timeout = stepTimeout
			}
			approved := h.hitl.Prompt(action, timeout)
			if !approved {
				h.ctx.AddUser(fmt.Sprintf("Action rejected by user: %s", guardResult.Reason))
				record.Error = "HITL rejected"
				h.recordStep(record)
				continue
			}
		}

		if action.Type == "done" {
			if len(h.cfg.Agent.VerifyCommand) > 0 {
				vResult := h.verif.Verify(h.cfg)
				record.Verifier = &vResult
				if !vResult.Success {
					h.ctx.AddUser(fmt.Sprintf("Verification failed: exit_code=%d\nstdout: %s\nstderr: %s",
						vResult.ExitCode, vResult.Summary, vResult.Stderr))
					h.recordStep(record)
					continue
				}
			}
			h.recordStep(record)
			h.flushTrace()
			return msg, nil
		}

		result, err := h.tools.Execute(action, h.cfg)
		if err != nil {
			record.Error = err.Error()
			h.ctx.AddUser(fmt.Sprintf("Tool error: %s", err.Error()))
			h.recordStep(record)
			parseErrors++
			if parseErrors >= maxParseRetries {
				return "", fmt.Errorf("max parse errors reached at step %d", step)
			}
			continue
		}
		record.ToolResult = result
		parseErrors = 0

		feedback := formatToolResult(action, result)
		h.ctx.AddUser(feedback)
		h.recordStep(record)
	}

	h.flushTrace()
	return "", fmt.Errorf("max steps (%d) reached without completion", h.cfg.Agent.MaxSteps)
}

func (h *Harness) recordStep(step StepRecord) {
	if h.trace != nil {
		h.trace.Record(step)
	}
}

func (h *Harness) flushTrace() {
	if h.trace != nil {
		h.trace.Flush()
	}
}

func formatToolResult(a Action, result map[string]any) string {
	switch a.Type {
	case "run_command":
		stdout, _ := result["stdout"].(string)
		stderr, _ := result["stderr"].(string)
		exitCode, _ := result["exit_code"].(int)
		timeout, _ := result["timeout"].(bool)
		summary := fmt.Sprintf("Command completed: exit_code=%d", exitCode)
		if timeout {
			summary = "Command timed out"
		}
		if stdout != "" {
			summary += "\n" + truncate(stdout, 2000)
		}
		if stderr != "" {
			summary += "\nstderr: " + truncate(stderr, 1000)
		}
		return summary
	case "read_file":
		content, _ := result["content"].(string)
		return fmt.Sprintf("Read file: %d bytes", len(content))
	case "write_file":
		path, _ := result["written"].(string)
		return fmt.Sprintf("Written to: %s", path)
	case "take_note":
		return "Note saved"
	case "search_memory":
		notes, _ := result["notes"].([]string)
		return fmt.Sprintf("Found %d matching notes", len(notes))
	default:
		return fmt.Sprintf("Tool result: %v", result)
	}
}

func parseDuration(s string, fallbackSec int) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Duration(fallbackSec) * time.Second
	}
	return d
}
