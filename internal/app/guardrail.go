package app

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Guardrail struct{}

func (g *Guardrail) Check(a Action, cfg *Config) GuardResult {
	switch a.Type {
	case "run_command":
		return g.checkCommand(a, cfg)
	case "read_file", "write_file":
		return g.checkPath(a, cfg)
	default:
		return GuardResult{Decision: "allow", Reason: "no guard needed"}
	}
}

func (g *Guardrail) checkCommand(a Action, cfg *Config) GuardResult {
	argvRaw, ok := a.Args["argv"]
	if !ok {
		return GuardResult{Decision: "deny", Reason: "missing argv"}
	}
	argv := toStringSlice(argvRaw)
	cmdStr := strings.Join(argv, " ")

	for _, pattern := range cfg.Policy.Commands.Deny {
		if strings.HasPrefix(cmdStr, pattern) {
			return GuardResult{Decision: "deny", Reason: fmt.Sprintf("denied by policy: %s", pattern)}
		}
	}
	for _, pattern := range cfg.Policy.Commands.Allow {
		if strings.HasPrefix(cmdStr, pattern) {
			return GuardResult{Decision: "allow", Reason: fmt.Sprintf("allowed by policy: %s", pattern)}
		}
	}
	for _, pattern := range cfg.Policy.Commands.Ask {
		if strings.HasPrefix(cmdStr, pattern) {
			return GuardResult{Decision: "ask", Reason: fmt.Sprintf("ask by policy: %s", pattern)}
		}
	}
	return GuardResult{Decision: cfg.Policy.Default, Reason: fmt.Sprintf("default policy: %s", cfg.Policy.Default)}
}

func (g *Guardrail) checkPath(a Action, cfg *Config) GuardResult {
	pathRaw, ok := a.Args["path"]
	if !ok {
		return GuardResult{Decision: "deny", Reason: "missing path"}
	}
	pathStr, ok := pathRaw.(string)
	if !ok {
		return GuardResult{Decision: "deny", Reason: "path not a string"}
	}
	absPath, err := filepath.Abs(pathStr)
	if err != nil {
		return GuardResult{Decision: "deny", Reason: fmt.Sprintf("cannot resolve path: %v", err)}
	}

	for _, denied := range cfg.Policy.Paths.Deny {
		denied = filepath.Clean(denied)
		if strings.HasPrefix(absPath, denied) {
			return GuardResult{Decision: "deny", Reason: fmt.Sprintf("path denied: %s", denied)}
		}
	}

	workDir, _ := filepath.Abs(cfg.Agent.WorkDir)
	if !strings.HasPrefix(absPath, workDir) {
		return GuardResult{Decision: "deny", Reason: "path outside work directory"}
	}

	return GuardResult{Decision: "allow", Reason: "path allowed"}
}

func toStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, len(raw))
	for i, item := range raw {
		out[i] = fmt.Sprint(item)
	}
	return out
}