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

func (c *Config) Validate() error {
	return nil
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
