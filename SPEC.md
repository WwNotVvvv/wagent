# SPEC: wagent — Coding Agent Harness

## 1. 问题陈述

**wagent** 是一个 CLI 工具，接收自然语言任务描述，在本地 Git 项目目录中驱动一个 LLM agent 自主完成编码任务——规划、执行、验证、自我修正——同时通过可配置的治理策略防止危险操作。

**目标用户**：日常使用 LLM 辅助编码的开发者，希望在一个受控、可审计的环境下让 agent 自主执行编码任务，而不是在聊天框中手动复制粘贴。

**为什么值得做**：当前的编码 agent（Claude Code、Codex CLI 等）各有各的治理策略和供应商锁定。wagent 提供：
- 供应商无关的 OpenAI 兼容接口
- 可编程、可离线测试的 MockLLM 模式
- 代码实现的 Shell 命令和文件范围治理
- allow / ask / deny 与 HITL 人工审批
- 测试结果反馈和有限自我修正
- 不包含 API Key 的审计追溯
- 面向 Windows、Linux 和 macOS 的 CLI Release

## 2. 用户故事

### US1：开发者让 agent 自动完成一个编码任务
作为 Go 项目的开发者，我在项目根目录运行 `wagent "为 user.go 添加单元测试"`，agent 读取文件、编写测试、运行 `go test`，看到测试通过后输出最终结果。整个过程不超过 25 步。

### US2：治理护栏拦截危险命令
我在 `wagent.toml` 中配置 `deny = ["rm -rf"]`。agent 尝试执行 `rm -rf /tmp/cache`，护栏识别到匹配，直接拒绝并不执行，agent 收到拦截消息后改变策略。

### US3：HITL 人工审批中等风险动作
agent 尝试 `git push`（配置为 ask），harness 暂停并打印：`[HITL] 允许执行? (y/N, 120s 超时自动拒绝):`。我输入 y 放行，或超时自动拒绝，agent 继续执行或被拒绝后调整。

### US4：MockLLM 离线确定性测试
作为开发者，我在没有 API Key 的 CI 环境中用 `wagent --mock script.json "修复 bug"` 运行 harness。MockLLM 按脚本顺序返回预设响应，验证护栏和反馈循环的确定性行为。

### US5：追溯审计过去的运行
我查看 `~/.wagent/traces/20260804-123456-abc123.jsonl`，看到每步的 Message、action、guard 决策、工具结果。Trace 中不包含任何 API Key，可供审查，但分享前仍需检查项目敏感内容。

## 3. 功能规约

### 3.1 CLI 入口

- `wagent <task>` — 单次请求模式，加载 `wagent.toml`，执行任务
- `wagent --mock <script.json> <task>` — 使用 MockLLM 模式运行
- `wagent key set` — 交互式引导录入 API Key 到 OS Keychain
- `wagent key status` — 显示是否已配置 Key（不回显明文）
- `wagent key clear` — 从 OS Keychain 中删除 Key
- 输入：`os.Args` 手动分派子命令，不使用第三方 CLI 框架

### 3.2 Agent 主循环

```
while steps < max_steps and not done:
    1. 组织上下文（system prompt + 规则 + 记忆 + 对话历史 + 本轮任务）
    2. 调用 LLM，获取结构化响应
    3. Action Parser 校验 JSON 格式、类型、参数
    4. Guardrail 检查 action 是否允许
    5. 若 ask 决策 → HITL 等待用户输入
    6. 执行工具（或 action 为 done 时执行 Verifier）
    7. 收集结果，回灌给 agent
    8. 写入 Trace，更新上下文
    9. 判断退出条件
```

退出条件（任一满足即停止）：
- 最大步数达到
- Verifier 验证成功（配置了验证命令且 agent 返回 done 时）
- 用户拒绝（HITL 拒绝后）
- 连续失败次数达到上限（后续扩展）
- LLM 解析失败超过重试次数

### 3.3 Action Parser / Validator

- 解析 LLM 返回的 JSON 为 `Action` 结构体
- 校验：type 是否已知、args 中必填字段是否存在、参数类型是否正确
- 失败处理：记录结构化错误，将错误信息回灌给 agent，允许有限次重试，超过上限后停止并返回错误
- 未知 action 类型、参数缺失、参数类型错误均按同样方式处理

### 3.4 Guardrail 治理

优先级：`deny > ask > allow`

**命令守卫**：将 argv 拼接为命令字符串，按配置的 deny/ask/allow 字符串前缀匹配：
- `deny` 列表 → 直接拒绝，不执行，记录原因
- `allow` 列表 → 跳过 HITL，直接执行
- `ask` 列表 → 进入 HITL
- 未匹配 → 按 `default = "ask"` 处理

**路径守卫**：覆盖所有文件操作工具及可识别的命令路径参数：
- `deny` 路径列表 → 访问即拒绝
- 超出 `work_dir` 范围 → 拒绝
- 路径使用 `filepath.Clean` + `filepath.Abs` 规范化后做范围判断

**HITL 状态机**：
- `step_timeout` 和 `hitl.timeout` 取较早截止时间，任一超时均拒绝当前 action
- 在 Trace 中记录拒绝原因
- 用户输入 y/Y → 放行，其他输入/超时 → 拒绝

### 3.5 工具

| 工具 | 输入 | 输出 | 边界条件 |
|------|------|------|---------|
| `read_file` | path | content, error | 路径越界拒绝；文件不存在返回错误 |
| `write_file` | path, content | error | 路径越界拒绝；目录不存在返回错误 |
| `run_command` | argv[] | stdout, stderr, exit_code, timeout | 不经过 shell；超时终止进程 |
| `take_note` | content | error | 追加到 `notes.jsonl` |
| `search_memory` | keyword | matching_notes[] | 大小写不敏感关键词匹配；返回少量条目 |

### 3.6 Verifier 反馈

- 执行配置的验证命令（argv 数组），不经过 shell
- 收集：exit code、stdout、stderr、timeout、success/failed 状态
- 输出截断，避免敏感信息写入日志
- stdout/stderr 写入前做 API Key 脱敏
- 结果回灌给 agent，agent 据此自我修正

### 3.7 Memory

- **对话历史**：运行中累积的上下文，每轮追加 assistant 消息和 user 结果消息
- **跨会话笔记**：`take_note` 写入 `memory_dir/notes.jsonl`，`search_memory` 按关键词检索
- 不把所有历史记忆一次性塞入上下文，只返回匹配的少量笔记

### 3.8 Trace

- 格式：JSONL，每轮一个 JSON 对象，追加写入
- 文件名：`timestamp-run_id.jsonl`（如 `20260804-123456-a1b2c3.jsonl`）
- 内容：每步的 Message、action、Guard.Decision、工具结果、验证结果、错误、耗时
- 任务摘要放在 Trace 元数据中，限制长度、过滤换行和敏感内容
- 不包含 API Key，可回放

## 4. 非功能性需求

### 4.1 安全

- API Key 可出现在 OS Keychain（主存储）和环境变量 `WAGENT_API_KEY`（只读来源）
- 不写入 `wagent.toml`、Trace、Memory、普通日志
- 不传递给 `run_command` 子进程
- HTTP 调试信息中脱敏
- 脱敏优先替换当前实际 API Key

### 4.2 可移植性

- 单文件二进制
- 交叉编译：`GOOS=windows/linux/darwin` + `GOARCH=amd64/arm64`
- GoReleaser 构建，GitHub Release 分发

### 4.3 可观测性

- Trace JSONL 完整记录每步决策与结果，可回放追溯

### 4.4 性能

- Harness 本地核心循环（Action Parser → Guardrail → Tool Router → Verifier + 上下文拼接）单步开销 ≤ 5ms
- 不含 LLM 网络调用和工具执行时间

## 5. 系统架构

```
┌──────────────────────────────────────────────┐
│                   CLI (main)                  │
│       解析参数 + 加载配置 + 装配 Harness       │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│              Agent Loop                      │
│  Think → Parse/Validate → Guard → Act →      │
│  Observe → Feedback(Verifier) → 循环/退出     │
└───────┬──────────┬──────────┬────────────────┘
        │          │          │
┌───────▼──┐ ┌─────▼─────┐ ┌─▼───────────────┐
│  LLM     │ │ Action    │ │   Tool Router   │
│ Interface│ │ Parser/   │ │  read_file      │
│(MockLLM) │ │ Validator │ │  write_file     │
└──────────┘ └─────┬─────┘ │  run_command    │
                   │       │  take_note      │
             ┌─────▼─────┐ │  search_memory  │
             │ Guardrail │ └───────┬─────────┘
             │ Policy    │         │
             │ Engine    │ ┌───────▼─────────┐
             │ allow/ask │ │   Verifier      │
             │ /deny     │ │  exit code      │
             │ PathCheck │ │  stdout/stderr  │
             │ HITL      │ │  timeout        │
             └───────────┘ │  status         │
                           └─────────────────┘
┌───────────┐  ┌───────────┐  ┌───────────────┐
│ Credential│  │  Memory   │  │    Trace      │
│ Store     │  │ 对话历史   │  │ 逐步记录      │
│ OS Keychain│  │ JSONL    │  │ JSONL         │
│ env fallbk│  │ take_note │  │ 不含 API Key  │
└───────────┘  │ search_mem│  └───────────────┘
               └───────────┘
```

**外部依赖**：
- LLM 供应商 OpenAI 兼容 API（HTTP 调用）
- OS Keychain（go-keyring 库）
- 本地文件系统（读写文件、执行命令、JSONL 持久化）

## 6. 数据模型

### 6.1 TOML 配置 (`wagent.toml`)

```toml
[llm]
provider = "openai"
model = "gpt-4o"
base_url = "https://api.openai.com/v1"

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
```

### 6.2 Go 结构体

```go
type Action struct {
    Type      string         // "read_file" | "write_file" | "run_command" | "take_note" | "search_memory" | "done"
    Args      map[string]any
    Message   string         // LLM 输出的附带消息（可选）
}

type GuardResult struct {
    Decision  string         // "allow" | "ask" | "deny"
    Reason    string
}

type VerifierResult struct {
    Success   bool
    ExitCode  int
    Stdout    string         // 截断后
    Stderr    string         // 截断后
    Timeout   bool
    Argv      []string
    Summary   string         // 输出摘要
}

type StepRecord struct {
    Step      int
    Message   string
    Action    Action
    Guard     *GuardResult
    ToolResult map[string]any // 通用工具结果
    Verifier  *VerifierResult
    Error     string
    Duration  time.Duration
}
```

## 7. 凭据与分发设计

### 7.1 凭据方案

- **主存储**：OS Keychain，通过 `go-keyring` 库访问
  - Windows: `wincred`
  - macOS: Keychain
  - Linux: Secret Service (DBus)
- **备用来源**：环境变量 `WAGENT_API_KEY`（只读，不写入持久化文件）
- **首次引导**：Keychain 和环境变量均未找到时，交互式提示隐藏输入，读取后写入 Keychain
- **管理命令**：`wagent key set`、`wagent key status`（不回显明文）、`wagent key clear`
- **非交互环境**：无法获取凭据时直接报错退出

### 7.2 安全边界

| 位置 | 是否可出现 API Key |
|------|-------------------|
| OS Keychain | ✅ 主存储 |
| 环境变量 `WAGENT_API_KEY` | ✅ 只读来源 |
| `wagent.toml` | ❌ |
| Trace (JSONL) | ❌（写入前脱敏） |
| Memory (JSONL) | ❌（写入前脱敏） |
| 普通日志 | ❌ |
| run_command 子进程 | ❌ |
| HTTP 调试信息 | ❌（脱敏） |

### 7.3 分发方案

- **形态**：原生可执行二进制，单文件
- **平台**：Windows (amd64/arm64)、Linux (amd64/arm64)、macOS (amd64/arm64)
- **工具**：GoReleaser 自动化构建
- **渠道**：GitHub Release 发布
- **README 说明**：获取方式、运行命令、Key 安全配置方式、已知限制

## 8. 技术选型与理由

| 维度 | 选择 | 理由 |
|------|------|------|
| 语言 | Go 1.23+ | 单文件二进制、交叉编译、跨平台钥匙串支持 |
| LLM 接口 | OpenAI 兼容 API | 供应商无关，可接入 OpenAI/Anthropic/Azure/本地 Ollama 等。Anthropic 需兼容层或适配端点 |
| MockLLM | 可编程 JSON 脚本 | 确定性离线测试 |
| 凭据 | go-keyring | 跨平台钥匙串统一接口 |
| 配置 | BurntSushi/toml | Go 社区标准 TOML 库 |
| CLI | 标准库 flag + 手动子命令分派 | 最小依赖 |
| 分发 | GoReleaser | 自动化 GitHub Release 多平台构建 |

## 9. 领域与机制设计（§A.5）

### 9.1 动作/工具（代码实现）

| 工具 | 实现方式 |
|------|---------|
| `read_file` | `os.ReadFile`，路径经守卫检查 |
| `write_file` | `os.WriteFile`，路径经守卫检查 |
| `run_command` | `os/exec`，argv 数组，不经过 shell |
| `take_note` | 追加到 `memory_dir/notes.jsonl` |
| `search_memory` | 大小写不敏感关键词匹配 |

### 9.2 客观反馈信号（代码实现）

- **Verifier**：执行验证命令 → 收集 exit code / stdout / stderr / timeout → 输出 `VerifierResult`
- 退出码 ≠ 0 时，agent 收到失败信号及输出摘要，可据此自我修正
- 确定性测试：mock LLM 返回动作序列，Verifier 返回失败，断言 agent 收到反馈后改变下一步

### 9.3 危险动作（代码实现）

- **Guardrail**：命令守卫（deny/ask/allow 三表 + 默认 ask）+ 路径守卫（work_dir 范围 + deny 路径）
- **HITL**：ask 决策时暂停等待输入，step_timeout 和 hitl.timeout 取较早截止时间，任一超时拒绝
- 确定性测试：mock LLM 返回 `run_command("rm -rf /")` → 断言 `Decision == "deny"`

### 9.4 记忆（代码实现）

- 对话历史：循环内 context 累积
- 跨会话笔记：`take_note` 写入 JSONL，`search_memory` 关键词检索
- 确定性测试：mock LLM 调用 `take_note` 后调用 `search_memory`，断言返回之前写入的笔记

### 9.5 主要贡献：治理

选择治理作为深入维度，因为：
1. 天然由代码构成（`Guardrail.Check(action) → Decision`）
2. 可完全脱离 LLM 做确定性测试
3. 分级策略（deny/ask/allow）与 HITL 状态机组合，覆盖真实场景
4. 策略配置 + 路径安全 + 审计追溯，工程深度足够

## 10. 验收标准

### 10.1 功能验收

| 功能 | 验收标准 |
|------|---------|
| Agent 主循环 | mock LLM 返回 `done` 且无验证命令 → 循环终止；有验证命令时先执行 Verifier，success=true 才终止 |
| Action Parser | LLM 返回无效 JSON → 断言 parsed=false，错误回灌，agent 重试达到上限后停止 |
| Guardrail(deny) | Guardrail.Check(`rm -rf`) → Decision=deny，不执行，agent 收到拦截消息 |
| Guardrail(ask) | Guardrail.Check(`git push`) → Decision=ask，HITL 等待输入，超时自动拒绝 |
| Guardrail(路径) | write_file 目标为 `/etc/passwd` → Decision=deny；`../../outside` → Decision=deny |
| Verifier | 验证命令 exit 0 → success=true；exit 1 → success=false，stdout/stderr 被捕获 |
| Memory | take_note("已使用 Logger 库") → search_memory("logger") 返回该笔记 |
| MockLLM | 按脚本顺序返回预设响应 → 断言每步 action 符合预期 |
| 凭据 | 环境变量未设置 + keychain 为空 → 引导输入；key status 不回显明文 |
| Trace | 运行后 JSONL 文件存在，不含 API Key，可回放每步记录 |
| CI | 每次 push 自动运行 `go test ./...`，`.gitlab-ci.yml` 中 `unit-test` job 为绿色 |
| GitHub Release | GoReleaser 构建 Windows/Linux/macOS 三平台二进制，上传至 Release |

### 10.2 机制演示

mock LLM 下确定性复现：
1. **治理护栏拦截危险动作**：mock LLM 返回 `run_command("rm -rf /")`，Guardrail 返回 deny，断言不执行
2. **验证失败后 agent 改变行为**：mock LLM 先返回 `run_command("go test ./...")`，Verifier 返回失败，LLM 下一步改为 `read_file("test_fail.go")`，断言 agent 收到反馈后改变了动作
3. **治理维度确定性行为**：mock LLM 依次返回三个动作：`run_command("git status")`（未匹配规则，default=ask）→ `run_command("rm -rf node_modules")`（deny 匹配）→ `run_command("ls")`（allow 匹配）。断言：第一个暂停等待 HITL（超时自动拒绝），第二个直接拒绝，第三个直接放行。三个动作的 Guard.Decision 在 Trace 中可查

## 11. 风险与未决问题

1. **LLM 输出格式不稳定**：不同模型输出的 JSON 格式可能不一致，Action Parser 需要容忍常见变体（如 markdown 包裹、多余字段）
2. **MockLLM 脚本复杂度**：可编程脚本需要支持条件分支或多轮对话，MVP 用顺序脚本，后续扩展
3. **Windows 路径处理**：`filepath.Abs` 在不同平台行为不同，路径守卫需要在 Windows 上正确处理 `C:\` 和 `\\` 路径
4. **跨会话记忆膨胀**：`search_memory` 用关键词匹配，笔记数量增大后需要分页或时间范围过滤，MVP 暂不处理
5. **HITL 超时竞态**：`step_timeout` 和 `hitl.timeout` 取较早截止时间，任一超时均拒绝当前 action，在 Trace 中记录拒绝原因