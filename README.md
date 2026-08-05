# wagent — Coding Agent Harness

A CLI tool that drives an LLM agent through a Think → Guard → Act → Observe → Feedback loop for local Git projects, with configurable governance policies.

## Quick Start

```bash
# Set your API Key
wagent key set

# Run a task
wagent "add unit tests for user.go"

# Run with mock LLM (offline, deterministic)
wagent --mock scripts/demo_mock.json "test task"
```

## Installation

Download the latest binary from [GitHub Releases](https://github.com/your-username/wagent/releases).

### Platforms

- Windows (amd64, arm64)
- Linux (amd64, arm64)
- macOS (amd64, arm64)

## Configuration

Create a `wagent.toml` in your project root (see `examples/wagent.toml`).

## API Key Security

- **Storage**: OS keychain (Windows Credential Manager, macOS Keychain, Linux Secret Service)
- **Fallback**: `WAGENT_API_KEY` environment variable (read-only)
- **Never stored in**: config file, traces, memory, logs, or subprocess environment

## Commands

| Command | Description |
|---------|-------------|
| `wagent <task>` | Run a coding task |
| `wagent --mock <script> <task>` | Run with mock LLM |
| `wagent key set` | Store API Key in keychain |
| `wagent key status` | Check if API Key is configured |
| `wagent key clear` | Remove API Key from keychain |

## Architecture

```
Agent Loop: Think → Guard → Act → Observe → Feedback
```

- **Guardrail**: Command and path policy with deny/ask/allow tiers
- **HITL**: Human-in-the-loop for risky operations
- **Verifier**: Run test commands and validate results
- **Trace**: Full audit trail in JSONL format

## Building from Source

```bash
go build -o wagent .
```

## Testing

```bash
go test ./... -v
```

## Known Limitations

- MockLLM uses sequential script (no conditional branching)
- Memory search is keyword-based (no vector search)
- Windows path handling may differ from Linux/macOS