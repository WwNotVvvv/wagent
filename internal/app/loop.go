package app

import (
	"fmt"
	"time"
)

const maxParseRetries = 5

func (h *Harness) Run(task string) (string, error) {
	taskID := h.nextTaskID()
	h.ctx.AddUser(task)

	parseErrors := 0

	for step := 1; step <= h.cfg.Agent.MaxSteps; step++ {
		stepStart := time.Now()

		action, msg, err := h.llm.Chat(h.ctx.Messages(), task)
		if err != nil {
			h.onStep(StepEvent{
				Step:     step,
				MaxSteps: h.cfg.Agent.MaxSteps,
				Phase:    StepEventError,
				Error:    truncate(err.Error(), 200),
			})
			record := StepRecord{
				Step:      step,
				TaskID:    taskID,
				TaskIndex: h.taskIdx,
				Error:     err.Error(),
				Duration:  time.Since(stepStart),
			}
			h.ctx.AddUser(fmt.Sprintf("Action parse error: %s. Response must be valid JSON only. Please retry with a valid action.", err.Error()))
			h.recordStep(record)
			parseErrors++
			if parseErrors >= maxParseRetries {
				return "", fmt.Errorf("max parse errors reached at step %d: %w", step, err)
			}
			continue
		}

		h.onStep(StepEvent{
			Step:     step,
			MaxSteps: h.cfg.Agent.MaxSteps,
			Phase:    StepEventAction,
			Action:   action,
		})

		guardResult := h.guard.Check(action, h.cfg)

		h.onStep(StepEvent{
			Step:     step,
			MaxSteps: h.cfg.Agent.MaxSteps,
			Phase:    StepEventGuard,
			Action:   action,
			Decision: guardResult.Decision,
			Reason:   guardResult.Reason,
		})

		record := StepRecord{
			Step:      step,
			TaskID:    taskID,
			TaskIndex: h.taskIdx,
			Message:   msg,
			Action:    action,
			Guard:     &guardResult,
			Duration:  time.Since(stepStart),
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
			h.onStep(StepEvent{
				Step:     step,
				MaxSteps: h.cfg.Agent.MaxSteps,
				Phase:    StepEventError,
				Action:   action,
				Error:    truncate(err.Error(), 200),
			})
			record.Error = err.Error()
			h.ctx.AddUser(fmt.Sprintf("Tool error: %s", err.Error()))
			h.recordStep(record)
			parseErrors++
			if parseErrors >= maxParseRetries {
				return "", fmt.Errorf("max parse errors reached at step %d", step)
			}
			continue
		}
		record.ToolResult = truncateToolResult(action, result)
		parseErrors = 0

		summary := formatToolResult(action, result)
		h.onStep(StepEvent{
			Step:     step,
			MaxSteps: h.cfg.Agent.MaxSteps,
			Phase:    StepEventResult,
			Action:   action,
			Summary:  truncate(summary, 200),
		})

		feedback := summary
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
		if content == "" {
			return "Read file: empty"
		}
		return fmt.Sprintf("Read file: %d bytes\n%s", len(content), truncate(content, 4000))
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

func truncateToolResult(a Action, result map[string]any) map[string]any {
	if a.Type != "read_file" {
		return result
	}
	content, ok := result["content"].(string)
	if !ok {
		return result
	}
	out := make(map[string]any, len(result))
	for k, v := range result {
		if k == "content" {
			out[k] = truncate(content, 4000)
		} else {
			out[k] = v
		}
	}
	return out
}