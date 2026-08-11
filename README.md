# wagent — Coding Agent Harness

**English | [简体中文](README.zh-CN.md)**

`wagent` is a CLI coding-agent harness for local Git projects. It drives an LLM through a Think → Guard → Act → Observe → Feedback loop while enforcing command and path governance policies.

The project is CLI-only, and the intended distribution is a GitHub Release containing native binaries.

## Quick start

The `wagent` source code does not need to be copied into the project being edited. Download a release binary and place it in any tools directory, add that directory to `PATH`, or invoke the binary with its full path. Then change the current directory to the root of the local Git project before starting a task.

For example, on Windows:

```powershell
# The binary may be stored anywhere; this is only an example location.
cd D:\work\my-project
D:\tools\wagent\wagent.exe key set
D:\tools\wagent\wagent.exe "add unit tests for user.go"
```

If the `wagent` directory is on `PATH`, the same task can be started with:

```powershell
cd D:\work\my-project
wagent "add unit tests for user.go"
```

By default, `wagent` looks for `wagent.toml` in the current directory, which should normally be the root of the Git project being edited. This file configures the LLM endpoint, step limits, verifier command, work directory, storage locations, and governance policies. Copy [`examples/wagent.toml`](examples/wagent.toml) as a starting point and adjust it for your project. If no local configuration exists, built-in defaults are used. A configuration in another location can be supplied explicitly with `--config`.

```bash
# Store an API key in the operating-system keychain.
wagent key set

# Run one coding task.
wagent "add unit tests for user.go"

# Start an interactive session.
wagent --interactive

# Run an offline deterministic mock session.
wagent --mock scripts/demo_mock.json "demonstrate governance"
```

In interactive mode, use `:help`, `:reset`, and `:exit`. The terminal progress display supports `--color=auto|always|never`; `auto` enables ANSI colors only for a terminal and respects `NO_COLOR`.

## Installation and releases

Download the latest Windows, Linux, or macOS binary from the repository's [GitHub Releases](../../releases) page. GoReleaser builds `amd64` and `arm64` artifacts for all three platforms.

The binary is self-contained. No Go installation is required for end users. On Windows, run `wagent.exe`; on Linux and macOS, make the downloaded binary executable before running it.

## Repository layout

```text
wagent/
├── main.go                     CLI entry point and command-line options
├── internal/app/               Agent loop, tools, guardrails, LLM, memory, and tests
├── examples/wagent.toml        Example project configuration
├── scripts/demo_mock.json      Offline governance demonstration script
├── .github/workflows/go.yml    GitHub Actions test and vet workflow
├── .gitlab-ci.yml              GitLab CI unit-test job
├── .goreleaser.yaml            Cross-platform release configuration
├── SPEC.md                     System specification
├── PLAN.md                     Implementation plan
├── AGENT_LOG.md                Development process log
├── SPEC_PROCESS.md             Specification and decision process
├── REFLECTION.md               Project reflection
├── go.mod                      Go module and dependency versions
└── go.sum                      Dependency checksums
```

The `internal/app` package contains the implementation and unit tests. The `examples` and `scripts` directories are supporting files for configuration and deterministic demonstrations; they are not required when using a downloaded release binary.

## Dependencies and licenses

The versions below are pinned in `go.mod`. License names refer to the license files distributed with the corresponding upstream Go modules.

| Module | Version | License | Role |
| --- | --- | --- | --- |
| `github.com/BurntSushi/toml` | v1.6.0 | MIT | TOML configuration parsing |
| `github.com/zalando/go-keyring` | v0.2.8 | MIT | Operating-system keychain access |
| `golang.org/x/term` | v0.30.0 | BSD-3-Clause | Hidden terminal input and terminal detection |
| `github.com/danieljoos/wincred` | v1.2.3 | MIT | Windows Credential Manager support (transitive) |
| `github.com/godbus/dbus/v5` | v5.2.2 | BSD-2-Clause | Linux Secret Service support (transitive) |
| `golang.org/x/sys` | v0.31.0 | BSD-3-Clause | Platform system-call support (transitive) |

The project does not copy or modify these upstream libraries. Their respective license and copyright notices remain applicable when redistributing the binary. The repository currently has no separate project-level `LICENSE` file; the table above documents third-party dependency licensing only.

## Configuration

Configuration discovery follows this order:

1. An explicitly supplied `--config path` (missing or invalid files are errors).
2. `wagent.toml` in the current directory.
3. Built-in defaults when no local configuration exists.

See [`examples/wagent.toml`](examples/wagent.toml) for a complete example. The configuration controls the LLM endpoint, step limits, verifier command, work directory, storage locations, command policy, path policy, and HITL timeout.

## API-key security

- `wagent key set` reads the key with hidden terminal input and stores it in the OS keychain.
- `WAGENT_API_KEY` is a read-only environment-variable fallback and is never written by wagent.
- The key is not written to `wagent.toml`, Memory JSONL, Trace JSONL, ordinary logs, or subprocess environments.
- `run_command` and Verifier subprocesses explicitly filter `WAGENT_API_KEY`.
- Memory notes, Trace task boundaries, Trace records, and Verifier output are redacted before persistence or feedback.
- `wagent key status` reports only whether a key is configured; it never prints the key.

## CLI commands

| Command | Description |
| --- | --- |
| `wagent <task>` | Run one task and exit. |
| `wagent --interactive` | Run multiple tasks in a shared session. |
| `wagent --mock <script> <task>` | Use the deterministic offline MockLLM. |
| `wagent key set` | Store an API key in the OS keychain. |
| `wagent key status` | Check key configuration without revealing it. |
| `wagent key clear` | Remove the stored key. |

## Governance and auditability

- **Guardrail:** command `allow`/`ask`/`deny` rules and path checks with normalized, component-aware work-directory boundaries. Tilde paths, recursive deny patterns, and existing symlink components are handled before file access.
- **HITL:** `ask` actions pause for approval; rejection or timeout prevents execution.
- **Verifier:** runs the configured argv-array verification command and feeds exit code, output, and timeout status back to the agent.
- **Memory:** `take_note` and keyword-based `search_memory` use JSONL storage.
- **Trace:** every step includes `task_id` and `task_index`; interactive sessions write a task boundary before each task to the shared JSONL file.

## Tests and mechanism demonstrations

```bash
go test ./... -v
go vet ./...
go build -o wagent .
```

The deterministic mechanism tests include governance tiers and a feedback-loop demonstration that asserts `done → verifier failure → read_file → done`. They do not require a network connection or a real API key.

CI is configured in [`.gitlab-ci.yml`](.gitlab-ci.yml); the `unit-test` job runs `go test ./...` on pushes. The module targets Go 1.23.

## Known limitations

- MockLLM uses a sequential script and does not support conditional branching.
- Memory search is case-insensitive keyword matching rather than vector search.
- The current release has been tested on Windows and Linux; macOS has not yet been verified.
