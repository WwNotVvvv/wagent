# wagent — Coding Agent Harness

**[English](README.md) | 简体中文**

`wagent` 是一个面向本地 Git 项目的 CLI 编码代理 harness。它驱动 LLM 按照 Think → Guard → Act → Observe → Feedback 循环完成编码任务，同时通过命令和路径治理策略限制危险操作。

本项目只提供 CLI，最终分发方式是通过 GitHub Release 提供适用于不同平台的原生二进制文件。

## 快速开始

不需要把 `wagent` 源代码复制到待操作的项目中。下载 Release 中的二进制文件，将它放在任意工具目录中，并将该目录加入 `PATH`；也可以直接使用二进制文件的完整路径运行。执行任务前，将当前目录切换到本地 Git 项目的根目录。

例如，在 Windows PowerShell 中：

```powershell
# 二进制文件可以放在任意位置，这里只是示例路径。
cd D:\work\my-project
D:\tools\wagent\wagent.exe key set
D:\tools\wagent\wagent.exe "为 user.go 添加单元测试"
```

如果已经将 `wagent` 所在目录加入 `PATH`，可以直接运行：

```powershell
cd D:\work\my-project
wagent "为 user.go 添加单元测试"
```

默认情况下，`wagent` 会在当前目录查找 `wagent.toml`；这个目录通常应当是待操作 Git 项目的根目录。该文件用于配置 LLM 服务地址、步数限制、验证命令、工作目录、存储位置和治理策略。可以复制 [`examples/wagent.toml`](examples/wagent.toml) 作为配置起点，再根据项目需要修改。如果当前目录没有这个文件，wagent 会使用内置默认值；如果配置文件位于其他位置，可以通过 `--config` 显式指定。

```bash
# 将 API Key 保存到操作系统钥匙串。
wagent key set

# 执行一次编码任务。
wagent "为 user.go 添加单元测试"

# 启动交互式会话。
wagent --interactive

# 执行离线、确定性的 MockLLM 会话。
wagent --mock scripts/demo_mock.json "演示治理策略"
```

交互模式支持 `:help`、`:reset` 和 `:exit`。终端进度显示支持 `--color=auto|always|never`；`auto` 只在终端输出时启用 ANSI 颜色，并遵守 `NO_COLOR` 环境变量。

## 安装与发布

从仓库的 [GitHub Releases](../../releases) 页面下载最新的 Windows、Linux 或 macOS 二进制文件。GoReleaser 会为三个平台构建 `amd64` 和 `arm64` 版本。

二进制文件自包含，普通用户不需要安装 Go。Windows 下运行 `wagent.exe`；Linux 和 macOS 下运行前需要为下载的二进制文件添加可执行权限。

## 配置

配置文件按以下顺序发现：

1. 显式传入的 `--config path`（文件不存在或内容无效时直接报错）。
2. 当前目录下的 `wagent.toml`。
3. 当前目录没有本地配置文件时，使用内置默认值。

完整示例见 [`examples/wagent.toml`](examples/wagent.toml)。配置可以控制 LLM 端点、步数限制、验证命令、工作目录、存储位置、命令策略、路径策略和 HITL 超时时间。

## API Key 安全

- `wagent key set` 使用隐藏输入读取 Key，并将其保存到操作系统钥匙串。
- `WAGENT_API_KEY` 是只读的环境变量备用来源，wagent 不会写入它。
- API Key 不会写入 `wagent.toml`、Memory JSONL、Trace JSONL、普通日志或子进程环境。
- `run_command` 和 Verifier 启动的子进程会显式过滤 `WAGENT_API_KEY`。
- Memory 笔记、Trace 任务边界、Trace 记录和 Verifier 输出在持久化或回灌前都会脱敏。
- `wagent key status` 只显示是否已配置，不会打印 Key 内容。

## CLI 命令

| 命令 | 说明 |
| --- | --- |
| `wagent <task>` | 执行一次任务后退出。 |
| `wagent --interactive` | 在共享会话中连续执行多个任务。 |
| `wagent --mock <script> <task>` | 使用确定性的离线 MockLLM。 |
| `wagent key set` | 将 API Key 保存到操作系统钥匙串。 |
| `wagent key status` | 检查 API Key 是否已配置，不显示明文。 |
| `wagent key clear` | 删除已保存的 API Key。 |

## 治理与审计

- **Guardrail：**支持命令 `allow`/`ask`/`deny` 规则和工作目录路径检查。
- **HITL：**遇到 `ask` 决策时暂停等待人工确认；拒绝或超时都会阻止动作执行。
- **Verifier：**运行配置中的 argv 数组验证命令，并将退出码、输出和超时状态回灌给 agent。
- **Memory：**通过 JSONL 存储 `take_note` 笔记，并支持基于关键词的 `search_memory`。
- **Trace：**每一步都包含 `task_id` 和 `task_index`；交互会话会在共享 JSONL 文件中为每个任务写入任务边界。

## 测试与机制演示

```bash
go test ./... -v
go vet ./...
go build -o wagent .
```

确定性机制测试包括治理分级和反馈闭环演示，并断言 `done → verifier failure → read_file → done` 的动作序列。它们不需要网络连接或真实 API Key。

CI 配置位于 [`.gitlab-ci.yml`](.gitlab-ci.yml)；其中的 `unit-test` job 会在 push 时运行 `go test ./...`。项目模块使用 Go 1.23。

## 已知限制

- MockLLM 使用顺序脚本，不支持条件分支。
- Memory 检索使用不区分大小写的关键词匹配，不是向量检索。
- 当前版本已在 Windows 和 Linux 上测试，macOS 尚未验证。
