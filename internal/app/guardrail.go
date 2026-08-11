package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
	absPath, err := absoluteGuardPath(pathStr)
	if err != nil {
		return GuardResult{Decision: "deny", Reason: fmt.Sprintf("cannot resolve path: %v", err)}
	}

	workDir := cfg.Agent.WorkDir
	if workDir == "" {
		workDir = "."
	}
	workDir, err = absoluteGuardPath(workDir)
	if err != nil {
		return GuardResult{Decision: "deny", Reason: fmt.Sprintf("cannot resolve work directory: %v", err)}
	}

	// Check the lexical path first so a path that does not exist yet is still
	// constrained to the configured work directory.
	if !pathWithin(workDir, absPath) {
		return GuardResult{Decision: "deny", Reason: "path outside work directory"}
	}

	// Resolve existing path components to prevent a symlink inside workDir
	// from reaching a file outside it. This also covers write_file paths whose
	// final file does not exist but whose parent directory does.
	resolvedWorkDir, err := resolveExistingPrefix(workDir)
	if err != nil {
		return GuardResult{Decision: "deny", Reason: fmt.Sprintf("cannot resolve work directory: %v", err)}
	}
	resolvedPath, err := resolveExistingPrefix(absPath)
	if err != nil {
		return GuardResult{Decision: "deny", Reason: fmt.Sprintf("cannot resolve path: %v", err)}
	}
	if !pathWithin(resolvedWorkDir, resolvedPath) {
		return GuardResult{Decision: "deny", Reason: "path resolves outside work directory"}
	}

	for _, denied := range cfg.Policy.Paths.Deny {
		if matchesDeniedPath(absPath, denied) || matchesDeniedPath(resolvedPath, denied) {
			return GuardResult{Decision: "deny", Reason: fmt.Sprintf("path denied: %s", denied)}
		}
	}

	return GuardResult{Decision: "allow", Reason: "path allowed"}
}

func absoluteGuardPath(path string) (string, error) {
	path = expandHomePath(path)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func expandHomePath(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func resolveExistingPrefix(path string) (string, error) {
	path = filepath.Clean(path)
	current := path
	var suffix []string

	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for _, part := range suffix {
				resolved = filepath.Join(resolved, part)
			}
			return filepath.Clean(resolved), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return path, nil
		}
		suffix = append([]string{filepath.Base(current)}, suffix...)
		current = parent
	}
}

func matchesDeniedPath(candidate, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}

	if strings.ContainsAny(pattern, "*?") {
		return matchPathGlob(candidate, pattern)
	}

	deniedPath, err := absoluteGuardPath(pattern)
	if err != nil {
		return false
	}
	return pathWithin(deniedPath, candidate)
}

func matchPathGlob(candidate, pattern string) bool {
	candidate = normalizeMatchPath(candidate)
	pattern = normalizeMatchPath(expandHomePath(pattern))

	var expression strings.Builder
	expression.WriteString("^")
	for i := 0; i < len(pattern); {
		switch {
		case i+2 < len(pattern) && pattern[i:i+3] == "**/":
			expression.WriteString(`(?:.*/)?`)
			i += 3
		case i+2 < len(pattern) && pattern[i:i+3] == "/**":
			expression.WriteString(`(?:/.*)?`)
			i += 3
		case i+1 < len(pattern) && pattern[i:i+2] == "**":
			expression.WriteString(`.*`)
			i += 2
		case pattern[i] == '*':
			expression.WriteString(`[^/]*`)
			i++
		case pattern[i] == '?':
			expression.WriteString(`[^/]`)
			i++
		default:
			expression.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}
	expression.WriteString("$")

	matched, err := regexp.MatchString(expression.String(), candidate)
	return err == nil && matched
}

func normalizeMatchPath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
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
