# Task 1: Project scaffolding + core types

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/app/types.go`
- Create: `internal/app/types_test.go`

**Interfaces:**
- Consumes: (none — first task)
- Produces: `Config`, `Action`, `GuardResult`, `VerifierResult`, `StepRecord` structs; `ConfigValidate()` function

## Steps

### Step 1: Initialize the Go module

```bash
cd wagent
go mod init wagent
mkdir -p internal/app
```

### Step 2: Write the failing test for types

```go
// internal/app/types_test.go
package app

import (
    "testing"
    "time"
)

func TestActionValidation(t *testing.T) {
    a := Action{Type: "run_command", Args: map[string]any{"argv": []any{"ls", "-la"}}}
    if a.Type != "run_command" {
        t.Errorf("expected run_command, got %s", a.Type)
    }
}

func TestGuardResultDecision(t *testing.T) {
    g := GuardResult{Decision: "deny", Reason: "blacklisted command"}
    if g.Decision != "deny" {
        t.Errorf("expected deny, got %s", g.Decision)
    }
}

func TestVerifierResult(t *testing.T) {
    v := VerifierResult{Success: true, ExitCode: 0, Argv: []string{"go", "test"}}
    if !v.Success {
        t.Errorf("expected success")
    }
}

func TestStepRecordDuration(t *testing.T) {
    s := StepRecord{Step: 1, Duration: 50 * time.Millisecond}
    if s.Step != 1 {
        t.Errorf("expected step 1")
    }
}

func TestConfigDefault(t *testing.T) {
    cfg := Config{}
    cfg.SetDefaults()
    if cfg.Policy.Default != "ask" {
        t.Errorf("expected default ask, got %s", cfg.Policy.Default)
    }
    if cfg.Agent.MaxSteps != 25 {
        t.Errorf("expected max_steps 25, got %d", cfg.Agent.MaxSteps)
    }
}
```

### Step 3: Run test to verify it fails

Run: `go test ./internal/app/ -run TestConfigDefault -v`
Expected: compile error — `Config` undefined, `SetDefaults` undefined

### Step 4: Write minimal types implementation

```go
// internal/app/types.go
package app

import "time"

type Action struct {
    Type    string         `json:"type"`
    Args    map[string]any `json:"args,omitempty"`
    Message string         `json:"message,omitempty"`
}

type GuardResult struct {
    Decision string `json:"decision"` // "allow" | "ask" | "deny"
    Reason   string `json:"reason"`
}

type VerifierResult struct {
    Success  bool     `json:"success"`
    ExitCode int      `json:"exit_code"`
    Stdout   string   `json:"stdout"`
    Stderr   string   `json:"stderr"`
    Timeout  bool     `json:"timeout"`
    Argv     []string `json:"argv"`
    Summary  string   `json:"summary"`
}

type StepRecord struct {
    Step       int              `json:"step"`
    Message    string           `json:"message"`
    Action     Action           `json:"action"`
    Guard      *GuardResult     `json:"guard,omitempty"`
    ToolResult map[string]any   `json:"tool_result,omitempty"`
    Verifier   *VerifierResult  `json:"verifier,omitempty"`
    Error      string           `json:"error,omitempty"`
    Duration   time.Duration    `json:"duration_ns"`
}

type LLMConfig struct {
    Provider string `toml:"provider"`
    Model    string `toml:"model"`
    BaseURL  string `toml:"base_url"`
}

type AgentConfig struct {
    MaxSteps       int      `toml:"max_steps"`
    StepTimeout    string   `toml:"step_timeout"`
    TotalTimeout   string   `toml:"total_timeout"`
    VerifyCommand  []string `toml:"verify_command"`
    WorkDir        string   `toml:"work_dir"`
}

type StorageConfig struct {
    TraceDir  string `toml:"trace_dir"`
    MemoryDir string `toml:"memory_dir"`
}

type CommandPolicy struct {
    Deny  []string `toml:"deny"`
    Ask   []string `toml:"ask"`
    Allow []string `toml:"allow"`
}

type PathPolicy struct {
    Deny []string `toml:"deny"`
}

type HITLConfig struct {
    Timeout string `toml:"timeout"`
}

type PolicyConfig struct {
    Default  string        `toml:"default"`
    Commands CommandPolicy `toml:"commands"`
    Paths    PathPolicy    `toml:"paths"`
    HITL     HITLConfig    `toml:"hitl"`
}

type Config struct {
    LLM     LLMConfig     `toml:"llm"`
    Agent   AgentConfig   `toml:"agent"`
    Storage StorageConfig `toml:"storage"`
    Policy  PolicyConfig  `toml:"policy"`
}

func (c *Config) SetDefaults() {
    if c.Policy.Default == "" {
        c.Policy.Default = "ask"
    }
    if c.Agent.MaxSteps == 0 {
        c.Agent.MaxSteps = 25
    }
    if c.Agent.StepTimeout == "" {
        c.Agent.StepTimeout = "60s"
    }
    if c.Agent.TotalTimeout == "" {
        c.Agent.TotalTimeout = "600s"
    }
    if c.Agent.WorkDir == "" {
        c.Agent.WorkDir = "."
    }
    if c.Storage.TraceDir == "" {
        c.Storage.TraceDir = "~/.wagent/traces"
    }
    if c.Storage.MemoryDir == "" {
        c.Storage.MemoryDir = "~/.wagent/memory"
    }
    if c.LLM.BaseURL == "" {
        c.LLM.BaseURL = "https://api.openai.com/v1"
    }
    if c.Policy.HITL.Timeout == "" {
        c.Policy.HITL.Timeout = "120s"
    }
}
```

### Step 5: Run test to verify it passes

Run: `go test ./internal/app/ -v`
Expected: PASS for all 5 tests

### Step 6: Write minimal main.go skeleton

```go
// main.go
package main

func main() {}
```

### Step 7: Commit

```bash
git add -A
git commit -m "feat: project scaffolding and core types"
```