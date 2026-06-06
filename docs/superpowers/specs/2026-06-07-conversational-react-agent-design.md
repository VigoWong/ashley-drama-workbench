# 对话式 ReAct Agent — 主设计文档

> 短剧生产工作台的第二代交互：把现有的「四步向导 + 8-stage 线性流水线」升级为一个
> **对话式、支持工具调用与 ReAct 循环的 agent**（参考豆包 / deepseek 的当代观感）。
> 用户用自然语言聊，agent 自己「思考 → 调工具 → 看结果 → 再决定」，在一个实时生长的
> 画布里产出完整短剧方案。
>
> 状态：设计已定稿，待拆分为子项目分别实现。
> 日期：2026-06-07

---

## 1. 目标与非目标

### 目标
- 让用户**感受到 agent 在动脑、在调工具**——把现在完全隐藏的推理、工具调用、自我校验过程「演」出来。
- **复用现有全部资产**：8 个 stage、3 个确定性工具、PlanView、ExportBar、多模态贴图、auth、history。不推倒重来。
- 交互从「点四步表单」变成「对话驱动」，但产出仍是那份结构化的 8 块短剧方案。
- 行为**稳定可复现**：无 API key 的 DemoMock 模式下，前端与测试也能看到完整的工具调用轨迹。

### 非目标（本期不做）
- 不做通用聊天机器人；agent 的职责域限定在「短剧方案生产」。
- 不做多会话管理 / 跨会话长期记忆（单会话内的上下文即可）。
- 不替换现有四步向导的底层能力契约——8 stage 仍是真实的生产单元，只是被包装成工具。
- 不追求「全自由」的乱序 agent（见 §3 的自主度决策）。

---

## 2. 核心决策摘要（已与产品对齐）

| 维度 | 决策 | 备选 / 理由 |
|---|---|---|
| 与现有系统关系 | **现有能力包装成工具，agent 调度**（复用全部资产） | 而非替换向导 / 并存双入口 |
| Agent 自主度 | **引导式 ReAct**：系统提示词给推荐流程，agent 默认顺着走、可按用户指令回头改/跳过/重做；工具自带前置校验兜底 | 而非全自由乱序 / Plan-and-Execute 审批流 |
| 整体布局 | **对话 + 画布双栏**（左 ~40% 对话，右 ~60% 方案画布） | 而非纯对话流 / 工作日志抽屉 |
| 左栏 ReAct 渲染 | **思考流 + 工具卡（混合）**：思考流式打字、完成塌成摘要；工具紧凑卡片可展开真实 I/O | 而非全透明展开 / 状态行折叠 |
| 右栏画布 | **活文档·自上而下长出**：待做→写入中→完成，自动滚到当前块，块级重生成 | 而非标签页 / 卡片瀑布锚点 |

---

## 3. 后端架构：引导式 ReAct 引擎

### 3.1 设计立场
保留现有「LLM 创作 · 确定性 Go 校验 · 反馈重写」的内核，在其**上方**加一层 ReAct 调度。
现有 `Orchestrator` 的固定顺序从「死管道」降级为 agent 的「软建议」——写进系统提示词，
而非写死在控制流里。

### 3.2 工具目录（agent 可调用的 Action）
把现有能力暴露为带 JSON schema 的工具。两类：

**A. 生成类工具（内部仍是现有 stage / LLM 调用）**
- `generateConcepts(requirement, episodes, secs)` → 2-3 个立意方向（复用 `Propose`）
- `writeBible(conceptId)` / `writeCharacters()` / `generateEpisodes()` /
  `planPlacements()` / `designHeroScenes()` / `planProductionDistribution()` /
  `renderVisuals()` —— 分别对应现有 8 stage。
- `refineBlock(stage, note)` —— 对某块按用户指令重生成（复用 `RunFrom(only=true)`）。

**B. 确定性工具（现有 3 个，纯 Go，瞬时）**
- `getWinningTropes(market, vertical)` → 候选爽点库
- `getProductCatalog(category)` → Ashley SKU 目录
- `validatePacing(episodes)` → 通过/不通过 + 问题清单 + 评分单

**前置校验兜底**：每个生成类工具在入口检查依赖（如 `generateEpisodes` 要求 concept/bible/
characters 已就绪），缺失则返回结构化 observation（`{error:"需先生成立意"}`）而非直接报错，
让 agent 从反馈中自我纠正。这是「引导式」稳健性的关键。

### 3.3 Provider 扩展（function-calling）
现有 `Provider.GenerateJSON(ctx, stage, prompt, images, schema)` 是单发结构化输出，
无法支撑多轮工具循环。需新增能力：

```
type ToolProvider interface {
    // 一轮 ReAct：传入对话历史 + 可用工具定义，返回 either 一个文本回复，
    // or 一个/多个 tool-call 请求（供引擎执行后把 observation 再喂回来）。
    GenerateWithTools(ctx, messages []Message, tools []ToolDef) (ToolTurn, error)
}
```
- **Gemini（Vertex / AI Studio）**：映射到原生 function-calling。
- **DemoMock**：实现一段**剧本化**的固定 ReAct 轨迹（见 §6 测试策略）——保证无 key 可跑。
- 保留 `GenerateJSON` 不动（生成类工具内部仍可复用它产结构化块）。

### 3.4 ReAct 引擎（新 `internal/agent/react` 或扩展 orchestrator）
循环：`agent 产出(思考 + tool-call) → 引擎执行工具 → observation 回灌 → 重复`，
直到 agent 产出「终态文本回复」或达到步数上限。每一步通过 `Emitter` 发**细粒度事件**
（§4）。系统提示词注入：职责域、推荐流程（8 步顺序）、工具清单、「可按用户话回头改某块」的规则。

### 3.5 会话状态
服务端持有单会话上下文：消息历史 + 当前 `*model.Plan`（画布的真相源）。
工具对 Plan 做增量更新；引擎据此发 `block.*` 事件驱动画布。

---

## 4. 流式契约（前后端接缝，新 SSE 事件词汇）

替换现有 `stage_start/done/error/complete` 的粗粒度词汇。前端据此驱动左栏对话与右栏画布。

| 事件 | 载荷 | 驱动 |
|---|---|---|
| `thought.delta` / `thought.done` | 流式思考 token / 收尾摘要 | 左栏思考段（流式→塌成一句） |
| `tool.start` | `{id, name, friendlyName, input}` | 左栏工具卡出现（⟳ 运行中） |
| `tool.result` | `{id, status:ok/fail, output}` | 工具卡定格 ✓/✗ + 可展开 I/O |
| `message.delta` / `message.done` | agent 文本回复 token | 左栏文本段 |
| `block.start` | `{stage}` | 右栏对应块进入「写入中」（高亮+微光），自动滚动 |
| `block.delta` | `{stage, partial}` | 块内增量内容 |
| `block.done` | `{stage, payload}` | 块定格「完成」 |
| `turn.done` | — | 一个 agent turn 结束，可输入 |
| `error` | `{message, scope}` | 错误展示 |

- 工具卡与画布块的联动：`tool.result` 若产生/更新某块，载荷带 `affectsStage`，前端在卡片里
  渲染 `→ 已更新右侧「分集」↗`，点击滚动+闪烁画布块。
- 传输沿用现有 SSE（`text/event-stream` + flush），无需 WebSocket。

---

## 5. 前端渲染设计

### 5.1 整体（布局 B · 双栏）
- 左 ~40% 对话列，右 ~60% 方案画布。窄屏画布收为抽屉，顶部按钮切换。
- 状态：`conversationStore`（messages + 每个 turn 的有序 segments）+ `planStore`
  （8 块内容 + 各自状态 pending/live/done），均由 SSE 流喂。技术栈用已在用的 Zustand。

### 5.2 左栏 · 对话 + ReAct（渲染 C）
一个 agent turn 内按到达顺序串「段」：
- **思考段**：灰斜体流式打字（`thought.delta`）；`thought.done` 后塌成
  `💭 思考完成：…`，`▾` 可重新展开。
- **工具卡**：图标 + 友好名 + 状态（`⟳`/`✓`/`✗`），可展开看真实 I/O
  （命中的爽点、SKU 列表、节奏评分单）。
- **文本段**：markdown 回复（`message.delta`）。
- 底部输入框 textarea + 发送，支持贴图（复用现有多模态 `Brief.Images`）。
- 联动钩子：工具卡内 `→ 已更新右侧「X」↗`。

### 5.3 右栏 · 方案画布（渲染 A，复用 PlanView）
- 8 块自上而下：`待做`(灰虚线/暗) → `写入中`(琥珀边+微光) → `完成`(实线)，
  `block.start` 时自动滚到当前块。
- 顶部：标题 + `进度 n/8` + 导出栏（复用 `ExportBar`）。
- 每块头部 `⟳ 重生成此块` → 触发 `refineBlock(stage)`。
- 对话指令定向某块（"立意换狗血点"）→ agent 跑 `refineBlock("concept", note)` →
  对应块原地更新 + 闪烁。

### 5.4 复用清单
PlanView 各块渲染器、ExportBar、`lib/types.ts` 镜像（需扩展消息/段/事件类型）、
登录 token、贴图多模态、history（对话产出的 Plan 仍可入库）。

---

## 6. 测试与 Mock 策略（关键约束）
现有「整个测试套件 + CLI + server 无 key 即可跑」的性质必须保住。
- **DemoMock 实现剧本化 ReAct 轨迹**：固定输出一串 `thought → tool.start/result → block.*`，
  覆盖「查爽点库 → 生成立意 → 生成分集 → 节奏校验失败 → 重写 → … → 完成」的完整剧情，
  让前端、demo、测试无需真实模型即可看到工具卡与画布生长。
- 工具前置校验、引擎循环上限、observation 回灌均有 Go 单测。
- 节奏校验门控的既有测试（`pacing_test.go`）保持不变——它现在是 `validatePacing` 工具的内核。

---

## 7. 拆分为子项目

整件事过大，按依赖拆两个子项目，各自走 spec → plan → 实现：

### 子项目 A：后端 ReAct / 工具调用引擎 + 流式契约（地基，先做）
- ToolProvider 扩展（Gemini function-calling + Mock 剧本轨迹）
- 工具目录 + 前置校验
- ReAct 引擎 + 系统提示词（引导式）
- 新 SSE 事件词汇 + 新对话端点（如 `POST /api/chat`，SSE）
- 会话状态管理
- 测试：无 key 全绿

### 子项目 B：前端对话式 UI（依赖 A 的契约）
- 双栏布局 + 响应式
- conversationStore / planStore（消费 SSE）
- 左栏 ReAct 渲染（思考流 + 工具卡）
- 右栏活文档画布（复用 PlanView + 生长动画 + 块级重生成）
- 联动、贴图、导出、history 接入

> 落地顺序：A 的「流式契约」一旦钉死，B 可基于 DemoMock 的剧本轨迹并行开发。

---

## 8. 风险与开放问题
- **Gemini function-calling 与现有结构化输出的耦合**：生成类工具内部仍要产结构化块，
  需确认 function-calling 模式下能稳定拿到 JSON（可能工具实现内部再调一次 `GenerateJSON`）。
- **引导式的"软建议"会不会被 agent 忽略导致乱序**：靠工具前置校验兜底，但需 prompt 调优。
- **Mock 剧本轨迹的维护成本**：stage/工具变化时要同步剧本。
- **旧四步向导是否保留**：本设计默认对话式为新主入口；是否下线向导留待产品决定（非本设计范围）。
- **会话过长的 token 成本**：单会话上下文增长，可能需要阶段性裁剪（本期非目标，先记录）。
