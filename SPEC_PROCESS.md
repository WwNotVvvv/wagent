# SPEC 设计过程记录

## 1. 记录范围与证据来源

本文记录 wagent 从需求讨论、SPEC 形成到 PLAN 生成的主要过程。原始材料来自 OpenCode Desktop 的本地会话数据库副本，筛选范围包括主 brainstorming session、实现阶段的 subagent session 和评审 session。原始数据库不纳入 Git 仓库；本文仅保留与设计决策有关的节选，并在整理前检查和替换疑似 API Key。

本文区分三类内容：

- brainstorming 对话：影响项目范围、架构和机制设计的讨论；
- 设计修订：根据新问题或测试反馈对 SPEC/PLAN 的调整；
- 过程偏差：实际工作流与原计划不一致的地方。

## 2. Brainstorming 第 1 轮：项目范围与交付形式

**时间：** 2026-08-01 09:34，主 session `ses_0450ad46afferthhI41Kd3DEcW`。

**对话节选：**

> 用户：当前初步设想是只做 CLI，针对任意本地 Git 项目，使用 Go，接入 OpenAI-compatible API 和 MockLLM，并通过 GitHub Release 分发。具体方案继续通过 brainstorming 讨论。

**处理决策：**

确定 wagent 是一个面向本地 Git 项目的 CLI Coding Agent Harness，优先交付可下载的跨平台二进制和清晰的使用说明，并将范围限定在 Harness 内核和 CLI 交付。

**对 SPEC 的影响：**

- 增加 CLI 单次任务和后续交互模式；
- 增加 OpenAI 兼容 LLM 接口和可编程 MockLLM；
- 增加 GitHub Release、凭据配置和本地运行说明；
- 将项目重点放在 Agent Loop、治理和反馈闭环，而不是界面开发。

## 3. Brainstorming 第 2 轮：循环结构与实现复杂度

**时间：** 2026-08-04 18:44，主 session `ses_0450ad46afferthhI41Kd3DEcW`。

**对话节选：**

> 用户：希望参考 `agent-loop.txt` 中的 Think → Guard → Act → Observe → Feedback 循环；第一版不实现 Plugin Engine、动态插件和并行执行，重要的是简单实现和完成目标。

**处理决策：**

采用简单的顺序循环：每轮先组织上下文并调用 LLM，再解析 Action、执行 Guardrail、分发工具、收集结果并回灌反馈。没有采用 OpenCode 推荐的完整 Plugin Engine 方案，因为当时的目标是控制项目规模并保证可测试性。

**对 SPEC 的影响：**

- 明确 Agent Loop 的五个阶段；
- 使用简单的工具注册表和函数分发；
- 将 LLM、MockLLM、Guardrail、Verifier 和 Memory 保留为清晰的模块边界；
- 把最大步数、超时、完成条件和连续错误作为退出条件。

## 4. Brainstorming 第 3 轮：治理作为主要贡献

**时间：** 2026-08-04 19:13–21:19，主 session `ses_0450ad46afferthhI41Kd3DEcW`。

**对话节选：**

> 用户：Shell 命令白名单/黑名单 + 文件操作范围限制。

> 用户：治理（护栏/沙箱/HITL）作为主要贡献方向。

> 用户：`run_command` 使用 argv 数组，不经过 shell。

**处理决策：**

确定治理为 wagent 的主要贡献方向，采用三等级策略：`allow` 直接放行，`ask` 进入 HITL，`deny` 直接拒绝。命令以 argv 数组执行，路径先规范化并限制在工作目录范围内；人工确认超时自动拒绝。

**对 SPEC 的影响：**

- 增加命令策略和路径策略；
- 增加 Action Parser/Validator，防止无效动作进入执行阶段；
- 增加 HITL、Trace 审计和路径穿越测试；
- 将 deny/ask/allow、路径边界和反馈回灌列入机制演示。

## 5. Brainstorming 第 4 轮：配置、反馈与记忆边界

**时间：** 2026-08-04 18:53–21:11，主 session `ses_0450ad46afferthhI41Kd3DEcW`。

**对话节选：**

> 用户：配置使用 TOML，包含 LLM、超时、验证命令、工作目录和 allow/ask/deny 策略。

> 用户：Memory 只保留当前会话上下文、Trace 和跨会话的小型 JSONL 笔记；`search_memory` 先使用关键词匹配。

> 用户：JSON 解析失败时不执行动作，把错误回灌给 Agent，达到错误上限后停止。

**处理决策：**

采用 TOML 声明式配置；Verifier 统一返回 exit code、stdout、stderr、timeout 和 success；Memory 使用 `take_note` 与 `search_memory`；LLM 输出错误时重试而不是立即执行或静默退出。

**对 SPEC 的影响：**

- 明确反馈信号和 VerifierResult 数据结构；
- 明确 Memory 与 Trace 的职责分离；
- 增加错误回灌、解析重试和停止条件；
- 将 API Key 从 Config、Trace、Memory、普通日志和子进程环境中隔离。

## 6. AI 建议的采纳、修正与拒绝

| AI 建议 | 处理结果 | 原因 |
|---|---|---|
| 使用 Plugin Engine 和大量可插拔接口 | 修正为简单顺序循环 | 当前项目优先保证可完成性和确定性测试，避免过度抽象 |
| 未匹配命令默认 allow 或默认 deny | 采用默认 ask | 在灵活性和安全性之间取中间方案，未知命令必须经过人工确认 |
| MockLLM 使用可编程脚本 | 采纳 | 能够离线复现多轮动作、治理决策和反馈修正 |
| 使用 OS Keychain 保存 API Key | 采纳 | 减少凭据进入配置文件、Trace 和日志的风险 |
| 先实现复杂 Memory 或向量检索 | 拒绝 | MVP 只需要 JSONL 笔记和关键词检索，避免超出作业范围 |
| 交互模式和终端输出全部内联到循环 | 修正为 CLI 注入 `OnStep` 回调 | 保持循环可测试，同时改善 CLI 使用体验 |

## 7. 设计修订与过程反馈

实现阶段的实际运行暴露了若干 SPEC 中未充分明确的问题，随后进行了修订：

1. MockLLM 运行时缺少 `wagent.toml`，暴露出错误包装导致默认配置回退失效的问题；随后增加了严格配置和默认配置的区分。
2. Windows 下输入 `y\r\n` 被错误拒绝；随后使用 `strings.TrimSpace` 统一处理终端换行符。
3. 真实 LLM 读取文件后返回 Markdown，导致 JSON 解析失败；随后增加解析错误回灌和重试机制，而不是简单提高最大步数。
4. Windows 下 `ls`、`sh` 和目录读取行为与预期不同；随后改进错误摘要、工作目录处理和跨平台测试命令。
5. 颜色输出只放在 CLI 层，避免 ANSI 转义码进入 Loop、Trace 和 LLM 上下文。

这些修订说明 SPEC 不只是一次性设计文档，也需要通过真实运行和测试反馈持续校正。

## 8. 阶段确认偏差

2026-08-05，我在 SPEC 审阅过程中调整了 `wagent` 项目目录，并将目录变化通知给 OpenCode。该消息的本意只是同步工作目录，并不表示已经确认 SPEC，也不表示允许进入下一阶段；OpenCode 将其理解为 SPEC 没有意见，提前开始了 PLAN 生成流程。

这次偏差暴露出两个问题：

- 工作目录变化通知和设计批准没有明确区分；
- 阶段转换缺少可验证的确认语句。

之后采用以下改进：目录、文件和环境变化消息明确标注为“仅同步信息”；进入 PLAN 前使用明确的“确认 SPEC，可以进入 PLAN”语句；Agent 在阶段转换前复述当前阶段、目标文件和确认依据。

## 9. 冷启动验证记录

作业要求是在正式实现前，使用不同类型的 Agent、新 session，且只提供 SPEC.md 和 PLAN.md，尝试执行 1–2 个 task，并记录它暴露出的歧义。

本项目没有执行作业要求的冷启动验证。在正式实现前，没有使用一个不带主开发会话历史、且只获得 SPEC.md 和 PLAN.md 的不同类型 Agent 来尝试完成 1–2 个 task。这是本项目工作流程中的明确缺失，不能用普通 subagent 的实现或 review session 代替。

现有记录中虽然有 `explore` session、多个 `general` subagent 的实现和 review session，但它们都处于项目开发上下文中，不符合严格的冷启动条件。因此，本项目不将冷启动验证描述为已完成。

现有记录可以证明：实现阶段使用了多个独立 subagent，并对 Task 1–14 进行了实现和 review；但这些 session 已经处于项目开发上下文中，不等同于严格的 SPEC/PLAN 冷启动验证。

如果补做该验证，应在新的、无历史上下文的 Agent session 中只提供 SPEC.md 和 PLAN.md，指定执行 1–2 个 task，并记录：Agent 的提问、错误解读、与预期的差距，以及由此产生的 SPEC/PLAN diff。补做时应明确标注为“事后补充验证”，不能伪装成实现前已经完成。

## 10. 对 brainstorming 的反思

brainstorming 对本项目最有帮助的地方，是把原本较宽泛的“做一个 Coding Agent”逐步收缩成可实现、可测试的 CLI Harness，并迫使我明确治理等级、凭据边界、反馈信号和退出条件。它也帮助我识别了“能够运行”与“能够审计、拒绝和恢复”之间的差别。

不足之处是，部分阶段的问题过多，容易让设计逐渐膨胀；另外，Agent 对工作目录变化的误解说明，仅依靠上下文推断阶段状态并不可靠。之后应把阶段批准、工作目录变更和范围变更分成不同类型的显式事件，并在每次进入实现前进行冷启动检查。
