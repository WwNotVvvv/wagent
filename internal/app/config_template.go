package app

import (
	"fmt"
	"os"
)

const configTemplate = `[llm]
provider = "openai"
model = "deepseek-v4-flash"
base_url = "https://njusehub.info/v1"

[agent]
max_steps = 25
step_timeout = "60s"
total_timeout = "600s"
verify_command = ["go", "test", "./..."]
work_dir = "."

[storage]
trace_dir = "~/.wagent/traces"
memory_dir = "~/.wagent/memory"

[policy]
default = "ask"

[policy.commands]
deny = ["rm -rf", "dd if=", "> /dev/", "format", "del /f /s"]
ask = ["git push", "git commit --amend", "drop table", "npm publish"]
allow = ["git status", "git diff", "ls", "go test", "go build", "go vet", "go fmt"]

[policy.paths]
deny = ["/etc/", "~/.ssh/", "**/node_modules/**"]

[policy.hitl]
timeout = "120s"
`

// WriteConfigTemplate creates a project configuration without overwriting an
// existing file.
func WriteConfigTemplate(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	if _, err := file.WriteString(configTemplate); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	return nil
}
