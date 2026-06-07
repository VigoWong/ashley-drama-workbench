# 子项目 B:前端对话式 ReAct UI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有前端加一个「对话式 ReAct agent」界面:用户在对话框发自然语言,左栏流式呈现 agent 的思考/工具调用/回复,右栏一个「活文档」画布随 agent 产出自上而下生长出 8 块短剧方案。通过同一页面的「向导 / 对话」模式切换接入,复用现有登录/历史/PlanView/ExportBar。

**Architecture:** 纯 React(`useReducer`,**不引入新状态库**)。一个纯函数 reducer 把后端 `/api/chat` 的 `ChatEvent` 流折叠成「对话 segments + plan + 每块构建状态」。`ChatView` 双栏容器消费 SSE;左栏 `ConversationColumn`(思考块/工具卡/消息气泡),右栏 `PlanCanvas`(构建状态条 + 复用 `PlanView` 渲染内容)。现有四步向导抽成 `WizardView` 组件,`page.tsx` 降为「认证 + 历史 + 模式切换」外壳。

**Tech Stack:** Next.js 16 / React 19 / TypeScript / Tailwind v4。验证:`npm run lint` + `npm run build`(类型检查)+ Playwright 无 key 冒烟(后端 DemoMock 出确定性 ReAct 轨迹)。**无单元测试框架,不要引入。**

依赖契约:`frontend/lib/types.ts` 的 `ChatEvent / ChatMessage / ChatReq`(子项目 A 已交付,镜像 `backend/internal/model/chat.go`)。后端 `POST /api/chat` `{message,history?,plan?}` → SSE of `ChatEvent`。主设计:`docs/superpowers/specs/2026-06-07-conversational-react-agent-design.md`(渲染:布局 B 双栏 / 左栏 C 思考流+工具卡 / 右栏 A 活文档)。

> 说明:主设计 §5.1 写的是 Zustand,但本前端实际未安装任何状态库,故改用 `useReducer`(零新依赖),行为等价。入口决策:**同一页面「向导/对话」模式切换**(非新路由)。

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `frontend/lib/api.ts`(修改) | 新增 `chat(req, onEvent)`;把 `streamSSE` 泛型化以复用给 `ChatEvent` |
| `frontend/lib/chatState.ts`(新建) | 纯函数:`ChatState`、`initialChatState`、`chatReducer(state, ChatEvent)`、segment 类型。无 React 依赖 |
| `frontend/components/chat/ThoughtBlock.tsx`(新建) | 思考段:流式 → 完成塌成一句摘要,可展开 |
| `frontend/components/chat/ToolCard.tsx`(新建) | 工具卡:图标+友好名+状态(运行/✓/✗),可展开真实 I/O |
| `frontend/components/chat/MessageBubble.tsx`(新建) | 用户/assistant 文本气泡 |
| `frontend/components/chat/ConversationColumn.tsx`(新建) | 左栏:按 segment 顺序渲染上面三种 + 输入框 |
| `frontend/components/chat/PlanCanvas.tsx`(新建) | 右栏:8 块构建状态条(pending/writing/done)+ 复用 `PlanView` 渲染已产出内容 + ExportBar |
| `frontend/components/ChatView.tsx`(新建) | 双栏容器:`useReducer(chatReducer)`,wires `chat()`,处理输入/错误/未授权 |
| `frontend/components/WizardView.tsx`(新建) | 把现有四步向导从 `page.tsx` 整体抽出(行为不变) |
| `frontend/app/page.tsx`(修改) | 降为外壳:认证 + 历史 + 「向导/对话」模式切换,渲染 `WizardView` 或 `ChatView` |
| `frontend/e2e/chat.spec.ts`(新建) | Playwright 无 key 冒烟 |

---

## Task 1: API 客户端 `chat()`

**Files:**
- Modify: `frontend/lib/api.ts`

无单测框架;本任务以 `npm run build`(类型检查)为验收。

- [ ] **Step 1: 泛型化 streamSSE 并新增 chat()**

In `frontend/lib/api.ts`: change the import line to also pull in the chat types, generalise `streamSSE` to a type parameter, and append `chat()`.

Change the top import:
```ts
import { Brief, BriefImage, ChatEvent, ChatReq, Concept, ProposeResp, RefineReq, SSEvent } from "./types"
```

Change `streamSSE`'s signature to be generic (only the signature + the cast change; the body stays identical):
```ts
async function streamSSE<T = SSEvent>(
  path: string,
  body: unknown,
  onEvent: (e: T) => void
): Promise<void> {
```
and inside the loop change the cast:
```ts
      if (line) onEvent(JSON.parse(line.slice(6)) as T)
```

Append at the end of the file:
```ts
// chat drives the conversational ReAct agent. It POSTs one user message (plus the
// prior text history and the in-progress plan) to /api/chat and streams ChatEvents
// — thought/tool/message/block/turn — back via onEvent. One call == one agent turn
// (which may loop through many tool calls server-side) and ends with a turn.done.
export function chat(req: ChatReq, onEvent: (e: ChatEvent) => void): Promise<void> {
  return streamSSE<ChatEvent>("/api/chat", req, onEvent)
}
```

- [ ] **Step 2: Verify build + lint**

Run: `cd frontend && npm run build`
Expected: compiles, no type errors. `generate()` / `refine()` still type-check (they default `T = SSEvent`).
Run: `cd frontend && npm run lint`
Expected: no new errors in `lib/api.ts`.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add lib/api.ts
git commit -m "feat(chat-ui): chat() SSE client for /api/chat"
```

---

## Task 2: 对话状态 reducer(纯函数)

**Files:**
- Create: `frontend/lib/chatState.ts`

纯模块,无 React。验收:类型检查 + 被后续组件使用。逻辑正确性由 Task 9 的 e2e 冒烟覆盖。

- [ ] **Step 1: 写实现**

```ts
// frontend/lib/chatState.ts
import { ChatEvent, Plan } from "./types"

// A Segment is one rendered unit in the conversation column, in arrival order.
export type Segment =
  | { kind: "user"; text: string }
  | { kind: "thought"; text: string; done: boolean }
  | { kind: "tool"; id: string; name: string; friendlyName: string; input?: unknown; output?: unknown; status: "running" | "ok" | "fail"; affectsStage?: string }
  | { kind: "message"; text: string; done: boolean }

// The 8 plan blocks, in pipeline order. "pending" = not built, "writing" = being
// produced now, "done" = produced, "error" = failed this turn.
export type BlockStatus = "pending" | "writing" | "done" | "error"
export const BLOCK_ORDER = [
  "concept", "bible", "characters", "episodes",
  "placements", "hero", "production_distribution", "visuals",
] as const
export type BlockKey = (typeof BLOCK_ORDER)[number]

export interface ChatState {
  segments: Segment[]
  blocks: Record<BlockKey, BlockStatus>
  plan: Plan | null
  running: boolean       // a turn is in flight
  error: string | null   // turn-level error
}

function freshBlocks(): Record<BlockKey, BlockStatus> {
  return {
    concept: "pending", bible: "pending", characters: "pending", episodes: "pending",
    placements: "pending", hero: "pending", production_distribution: "pending", visuals: "pending",
  }
}

export function initialChatState(plan: Plan | null = null): ChatState {
  // If we already have a plan (e.g. resumed from history), mark present blocks done.
  const blocks = freshBlocks()
  if (plan) {
    if (plan.concept?.logline) blocks.concept = "done"
    if (plan.bible?.title) blocks.bible = "done"
    if (plan.characters?.length) blocks.characters = "done"
    if (plan.episodes?.length) blocks.episodes = "done"
    if (plan.placements?.length) blocks.placements = "done"
    if (plan.heroScenes?.length) blocks.hero = "done"
    if (plan.production?.format) blocks.production_distribution = "done"
    if (plan.visuals?.length) blocks.visuals = "done"
  }
  return { segments: [], blocks, plan, running: false, error: null }
}

// pushUser appends the user's message and marks the turn running. Call this when
// sending, before the stream starts.
export function pushUser(state: ChatState, text: string): ChatState {
  return { ...state, segments: [...state.segments, { kind: "user", text }], running: true, error: null }
}

function isBlock(stage?: string): stage is BlockKey {
  return !!stage && (BLOCK_ORDER as readonly string[]).includes(stage)
}

// chatReducer folds one ChatEvent into the state. It is a pure function (no I/O).
export function chatReducer(state: ChatState, e: ChatEvent): ChatState {
  switch (e.type) {
    case "thought.delta": {
      const last = state.segments[state.segments.length - 1]
      if (last && last.kind === "thought" && !last.done) {
        const segments = state.segments.slice(0, -1).concat({ ...last, text: last.text + (e.text ?? "") })
        return { ...state, segments }
      }
      return { ...state, segments: [...state.segments, { kind: "thought", text: e.text ?? "", done: false }] }
    }
    case "thought.done": {
      const segments = [...state.segments]
      for (let i = segments.length - 1; i >= 0; i--) {
        const s = segments[i]
        if (s.kind === "thought" && !s.done) { segments[i] = { ...s, text: e.text ?? s.text, done: true }; break }
      }
      return { ...state, segments }
    }
    case "tool.start":
      return {
        ...state,
        segments: [...state.segments, {
          kind: "tool", id: e.toolId ?? "", name: e.toolName ?? "", friendlyName: e.friendlyName ?? e.toolName ?? "",
          input: e.input, status: "running", affectsStage: e.affectsStage,
        }],
      }
    case "tool.result": {
      const segments = state.segments.map((s) =>
        s.kind === "tool" && s.id === e.toolId
          ? { ...s, status: (e.status === "ok" ? "ok" : "fail") as "ok" | "fail", output: e.output, affectsStage: e.affectsStage ?? s.affectsStage }
          : s
      )
      return { ...state, segments }
    }
    case "message.delta": {
      const last = state.segments[state.segments.length - 1]
      if (last && last.kind === "message" && !last.done) {
        const segments = state.segments.slice(0, -1).concat({ ...last, text: last.text + (e.text ?? "") })
        return { ...state, segments }
      }
      return { ...state, segments: [...state.segments, { kind: "message", text: e.text ?? "", done: false }] }
    }
    case "message.done": {
      const segments = [...state.segments]
      for (let i = segments.length - 1; i >= 0; i--) {
        const s = segments[i]
        if (s.kind === "message" && !s.done) { segments[i] = { ...s, text: e.text ?? s.text, done: true }; break }
      }
      return { ...state, segments }
    }
    case "block.start":
      if (!isBlock(e.stage)) return state
      return { ...state, blocks: { ...state.blocks, [e.stage]: "writing" } }
    case "block.done":
      if (!isBlock(e.stage)) return state
      return { ...state, blocks: { ...state.blocks, [e.stage]: "done" } }
    case "turn.done":
      return { ...state, running: false, plan: e.plan ?? state.plan }
    case "error":
      // A stage-scoped error resolves that block; a turn-level error ends the turn.
      if (isBlock(e.stage)) {
        return { ...state, blocks: { ...state.blocks, [e.stage]: "error" } }
      }
      return { ...state, running: false, error: e.message ?? "出错了" }
    default:
      return state
  }
}
```

- [ ] **Step 2: Verify build**

Run: `cd frontend && npm run build`
Expected: compiles, no type errors.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add lib/chatState.ts
git commit -m "feat(chat-ui): pure ChatEvent→state reducer"
```

---

## Task 3: 展示组件 ThoughtBlock / ToolCard / MessageBubble

**Files:**
- Create: `frontend/components/chat/ThoughtBlock.tsx`
- Create: `frontend/components/chat/ToolCard.tsx`
- Create: `frontend/components/chat/MessageBubble.tsx`

Match the existing visual vocabulary used across the app (`panel`, `label-tech`, `font-mono`, `text-ember-400`, `bg-ink-800/900`, `text-bone-300/400`, `border-bone-500/20`).

- [ ] **Step 1: ThoughtBlock**

```tsx
// frontend/components/chat/ThoughtBlock.tsx
"use client"
import { useState } from "react"

// ThoughtBlock renders the agent's reasoning. While streaming (done=false) it shows
// the live text; once done it collapses to a one-line summary the user can expand.
export default function ThoughtBlock({ text, done }: { text: string; done: boolean }) {
  const [open, setOpen] = useState(false)
  if (!done) {
    return (
      <div className="my-1.5 border-l-2 border-bone-500/40 pl-3 font-sans text-xs italic leading-relaxed text-bone-400">
        💭 {text}
        <span className="ml-1 inline-block h-3 w-1 animate-pulse bg-ember-400 align-middle" />
      </div>
    )
  }
  const summary = text.length > 40 ? text.slice(0, 40) + "…" : text
  return (
    <button
      onClick={() => setOpen((o) => !o)}
      className="my-1.5 block w-full rounded-md bg-ink-800/60 px-3 py-1.5 text-left font-sans text-xs leading-relaxed text-bone-400 transition hover:text-bone-300"
    >
      <span className="mr-1 font-mono text-[10px] text-bone-500">{open ? "▾" : "▸"}</span>
      💭 {open ? text : `思考完成:${summary}`}
    </button>
  )
}
```

- [ ] **Step 2: ToolCard**

```tsx
// frontend/components/chat/ToolCard.tsx
"use client"
import { useState } from "react"

interface Props {
  name: string
  friendlyName: string
  status: "running" | "ok" | "fail"
  input?: unknown
  output?: unknown
  affectsStage?: string
}

const STAGE_LABEL: Record<string, string> = {
  concept: "立意", bible: "剧集圣经", characters: "人物", episodes: "分集",
  placements: "品牌植入", hero: "英雄场景", production_distribution: "制作与分发", visuals: "概念图",
}

// ToolCard shows one tool call: friendly name + status pill, expandable to reveal
// the real input/output (the agent-feeling evidence). On a result that touched a
// plan block it shows a "→ 已更新右侧「X」" affordance.
export default function ToolCard({ name, friendlyName, status, input, output, affectsStage }: Props) {
  const [open, setOpen] = useState(false)
  const dot =
    status === "running" ? "bg-ember-400 animate-pulse"
    : status === "ok" ? "bg-signal-go"
    : "bg-signal-stop"
  const statusText = status === "running" ? "运行中…" : status === "ok" ? "完成 ✓" : "失败 ✗"
  return (
    <div className="my-2 rounded-lg border border-sky-400/30 bg-sky-400/[0.06] px-3 py-2">
      <button onClick={() => setOpen((o) => !o)} className="flex w-full items-center gap-2 text-left">
        <span className={`h-2 w-2 flex-shrink-0 rounded-full ${dot}`} />
        <span className="font-mono text-[11px] font-semibold text-sky-300">🔧 {friendlyName}</span>
        <span className="font-mono text-[10px] text-bone-400">{name}</span>
        <span className="ml-auto font-mono text-[10px] text-bone-400">{statusText}</span>
        <span className="font-mono text-[10px] text-bone-500">{open ? "▾" : "▸"}</span>
      </button>
      {affectsStage && status === "ok" && STAGE_LABEL[affectsStage] && (
        <p className="mt-1 font-mono text-[10px] text-ember-400/80">→ 已更新右侧「{STAGE_LABEL[affectsStage]}」↗</p>
      )}
      {open && (
        <div className="mt-2 space-y-1.5">
          {input != null && (
            <pre className="max-h-40 overflow-auto rounded bg-ink-900/80 p-2 font-mono text-[10px] leading-relaxed text-bone-300">in: {safe(input)}</pre>
          )}
          {output != null && (
            <pre className="max-h-56 overflow-auto rounded bg-ink-900/80 p-2 font-mono text-[10px] leading-relaxed text-bone-300">out: {safe(output)}</pre>
          )}
        </div>
      )}
    </div>
  )
}

function safe(v: unknown): string {
  try { return JSON.stringify(v, null, 2) } catch { return String(v) }
}
```

- [ ] **Step 3: MessageBubble**

```tsx
// frontend/components/chat/MessageBubble.tsx
"use client"

// MessageBubble renders a user or assistant text turn. User bubbles sit right
// (ember), assistant left (neutral).
export default function MessageBubble({ role, text }: { role: "user" | "assistant"; text: string }) {
  if (role === "user") {
    return (
      <div className="my-2 ml-10 rounded-2xl rounded-br-sm bg-ember-500/20 px-3.5 py-2 font-sans text-sm leading-relaxed text-bone-50">
        {text}
      </div>
    )
  }
  return (
    <div className="my-2 mr-10 rounded-2xl rounded-bl-sm bg-ink-800/70 px-3.5 py-2 font-sans text-sm leading-relaxed text-bone-100 whitespace-pre-wrap">
      {text}
    </div>
  )
}
```

- [ ] **Step 4: Verify build + lint**

Run: `cd frontend && npm run build && npm run lint`
Expected: compiles; no new lint errors. (If `signal-go`/`signal-stop` colors aren't defined, use `bg-emerald-500`/`bg-red-500` — check `app/globals.css`/Tailwind theme first; `signal-stop` is already used in `page.tsx` so it exists.)

- [ ] **Step 5: Commit**

```bash
cd frontend && git add components/chat/ThoughtBlock.tsx components/chat/ToolCard.tsx components/chat/MessageBubble.tsx
git commit -m "feat(chat-ui): thought / tool-card / message presentational components"
```

---

## Task 4: 左栏 ConversationColumn

**Files:**
- Create: `frontend/components/chat/ConversationColumn.tsx`

- [ ] **Step 1: 写实现**

```tsx
// frontend/components/chat/ConversationColumn.tsx
"use client"
import { useEffect, useRef, useState } from "react"
import { Segment } from "@/lib/chatState"
import ThoughtBlock from "./ThoughtBlock"
import ToolCard from "./ToolCard"
import MessageBubble from "./MessageBubble"

interface Props {
  segments: Segment[]
  running: boolean
  error: string | null
  onSend: (text: string) => void
}

// ConversationColumn renders the agent turn segments in arrival order and a
// composer at the bottom. It auto-scrolls to the newest segment.
export default function ConversationColumn({ segments, running, error, onSend }: Props) {
  const [draft, setDraft] = useState("")
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => { endRef.current?.scrollIntoView({ behavior: "smooth" }) }, [segments])

  function submit() {
    const t = draft.trim()
    if (!t || running) return
    onSend(t)
    setDraft("")
  }

  return (
    <div className="flex h-full flex-col">
      <div className="min-h-0 flex-1 overflow-y-auto pr-1">
        {segments.length === 0 && (
          <p className="mt-10 text-center font-sans text-sm text-bone-400">
            告诉我你想做的短剧,例如:<br />「家装改造逆袭,主打逆袭打脸,植入 Ashley 客厅沙发」
          </p>
        )}
        {segments.map((s, i) => {
          switch (s.kind) {
            case "user": return <MessageBubble key={i} role="user" text={s.text} />
            case "message": return <MessageBubble key={i} role="assistant" text={s.text} />
            case "thought": return <ThoughtBlock key={i} text={s.text} done={s.done} />
            case "tool": return (
              <ToolCard key={s.id || i} name={s.name} friendlyName={s.friendlyName}
                status={s.status} input={s.input} output={s.output} affectsStage={s.affectsStage} />
            )
          }
        })}
        {error && <p className="my-2 rounded-lg border border-signal-stop/40 bg-signal-stop/10 px-3 py-2 font-mono text-xs text-signal-stop">✕ {error}</p>}
        <div ref={endRef} />
      </div>

      <div className="mt-3 border-t border-bone-500/10 pt-3">
        <div className="flex items-end gap-2">
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); submit() } }}
            rows={2}
            disabled={running}
            placeholder={running ? "agent 正在工作…" : "输入需求或修改指令,Enter 发送 / Shift+Enter 换行"}
            className="min-h-[44px] flex-1 resize-none rounded-lg border border-bone-500/20 bg-ink-900/60 px-3 py-2 font-sans text-sm text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-60"
          />
          <button
            onClick={submit}
            disabled={running || !draft.trim()}
            className="rounded-lg bg-ember-500 px-4 py-2.5 font-mono text-xs uppercase tracking-wider text-ink-900 transition hover:bg-ember-400 disabled:cursor-not-allowed disabled:opacity-40"
          >
            发送
          </button>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Verify build + lint**

Run: `cd frontend && npm run build && npm run lint`
Expected: compiles; no new lint errors.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add components/chat/ConversationColumn.tsx
git commit -m "feat(chat-ui): left conversation column with composer + autoscroll"
```

---

## Task 5: 右栏 PlanCanvas

**Files:**
- Create: `frontend/components/chat/PlanCanvas.tsx`

复用现有 `PlanView`(渲染已产出内容)+ `ExportBar`,上方加一条 8 块构建状态条体现「活文档生长」。

- [ ] **Step 1: 写实现**

```tsx
// frontend/components/chat/PlanCanvas.tsx
"use client"
import { Plan } from "@/lib/types"
import { BLOCK_ORDER, BlockKey, BlockStatus } from "@/lib/chatState"
import PlanView from "@/components/PlanView"
import ExportBar from "@/components/ExportBar"

const LABEL: Record<BlockKey, string> = {
  concept: "立意", bible: "圣经", characters: "人物", episodes: "分集",
  placements: "植入", hero: "英雄场景", production_distribution: "制作分发", visuals: "概念图",
}

interface Props {
  plan: Plan | null
  blocks: Record<BlockKey, BlockStatus>
  onChange: (p: Plan) => void
  onRefine: (fromStage: string, only: boolean, note: string) => void
}

// PlanCanvas is the live "document" pane: a build-status rail (pending → writing →
// done) over the reused PlanView. As the agent finishes each block the status chip
// flips and (once a plan exists) PlanView shows the content; ExportBar appears when
// there's something to export.
export default function PlanCanvas({ plan, blocks, onChange, onRefine }: Props) {
  const doneCount = BLOCK_ORDER.filter((b) => blocks[b] === "done").length
  return (
    <div className="flex h-full flex-col">
      <div className="mb-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="label-tech">方案画布 · 实时</span>
          <span className="font-mono text-xs text-ember-400">{doneCount.toString().padStart(2, "0")}<span className="text-bone-400">/08</span></span>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {BLOCK_ORDER.map((b) => <StatusChip key={b} label={LABEL[b]} status={blocks[b]} />)}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto pr-1">
        {plan ? (
          <div className="space-y-6">
            {doneCount > 0 && <ExportBar plan={plan} />}
            <PlanView plan={plan} onChange={onChange} onRefine={onRefine} />
          </div>
        ) : (
          <div className="flex h-full items-center justify-center">
            <p className="font-sans text-sm text-bone-400">方案会在这里随 agent 一步步生长出来。</p>
          </div>
        )}
      </div>
    </div>
  )
}

function StatusChip({ label, status }: { label: string; status: BlockStatus }) {
  const cls =
    status === "done" ? "border-signal-go/50 bg-signal-go/10 text-bone-100"
    : status === "writing" ? "border-ember-400/70 bg-ember-400/10 text-ember-300 animate-pulse"
    : status === "error" ? "border-signal-stop/50 bg-signal-stop/10 text-signal-stop"
    : "border-bone-500/20 text-bone-500"
  const mark = status === "done" ? "✓" : status === "writing" ? "✍" : status === "error" ? "✕" : "·"
  return (
    <span className={`rounded-md border px-2 py-1 font-mono text-[10px] ${cls}`}>{mark} {label}</span>
  )
}
```

> Note on `PlanView` with a partial plan: `PlanView` renders all sections. Before a section's data exists the fields render empty — acceptable for v1 (the status rail communicates progress). Do NOT modify `PlanView` to hide empty sections in this task (would risk the wizard that shares it); a polished per-section skeleton is a follow-up.

- [ ] **Step 2: Verify build + lint**

Run: `cd frontend && npm run build && npm run lint`
Expected: compiles; no new lint errors. (Confirm `signal-go` exists in the theme; if not, use `emerald-500`. `signal-stop` is already in use.)

- [ ] **Step 3: Commit**

```bash
cd frontend && git add components/chat/PlanCanvas.tsx
git commit -m "feat(chat-ui): right plan canvas (build-status rail + reused PlanView)"
```

---

## Task 6: ChatView 双栏容器(wires SSE)

**Files:**
- Create: `frontend/components/ChatView.tsx`

- [ ] **Step 1: 写实现**

```tsx
// frontend/components/ChatView.tsx
"use client"
import { useReducer, useRef } from "react"
import { ChatMessage, Plan } from "@/lib/types"
import { chat, UnauthorizedError } from "@/lib/api"
import { chatReducer, initialChatState, pushUser, ChatState } from "@/lib/chatState"
import ConversationColumn from "@/components/chat/ConversationColumn"
import PlanCanvas from "@/components/chat/PlanCanvas"

// ChatView is the two-pane conversational agent surface. Left: the ReAct
// conversation; right: the live plan canvas. It owns the reduced ChatState and
// drives /api/chat, feeding every ChatEvent through the pure reducer.
export default function ChatView({ onUnauthorized }: { onUnauthorized: () => void }) {
  const [state, dispatch] = useReducer(
    (s: ChatState, a: { type: "event"; e: import("@/lib/types").ChatEvent } | { type: "user"; text: string } | { type: "setPlan"; plan: Plan }) => {
      if (a.type === "user") return pushUser(s, a.text)
      if (a.type === "setPlan") return { ...s, plan: a.plan }
      return chatReducer(s, a.e)
    },
    null,
    () => initialChatState(null)
  )
  // Keep a ref to the latest plan + text history so each send posts current context
  // without stale closures.
  const planRef = useRef<Plan | null>(state.plan)
  planRef.current = state.plan
  const historyRef = useRef<ChatMessage[]>([])

  async function onSend(text: string) {
    dispatch({ type: "user", text })
    historyRef.current = [...historyRef.current, { role: "user", text }]
    try {
      await chat(
        { message: text, history: historyRef.current.slice(0, -1), plan: planRef.current ?? undefined },
        (e) => {
          dispatch({ type: "event", e })
          if (e.type === "turn.done" && e.plan) planRef.current = e.plan
          if (e.type === "message.done" && e.text) {
            historyRef.current = [...historyRef.current, { role: "assistant", text: e.text }]
          }
        }
      )
    } catch (err) {
      if (err instanceof UnauthorizedError) { onUnauthorized(); return }
      dispatch({ type: "event", e: { type: "error", message: err instanceof Error ? err.message : "对话失败" } })
    }
  }

  function onChangePlan(p: Plan) { dispatch({ type: "setPlan", plan: p }); planRef.current = p }

  // For now, in-canvas refine flows through a normal user message so the agent
  // performs the refine via its refineBlock tool (keeps one orchestration path).
  function onRefine(fromStage: string, _only: boolean, note: string) {
    onSend(`请用 refineBlock 重做「${fromStage}」这一块:${note}`)
  }

  return (
    <div className="grid h-[calc(100vh-220px)] grid-cols-1 gap-5 lg:grid-cols-[2fr_3fr]">
      <section className="panel flex flex-col rounded-2xl p-4 sm:p-5">
        <ConversationColumn segments={state.segments} running={state.running} error={state.error} onSend={onSend} />
      </section>
      <section className="panel flex flex-col rounded-2xl p-4 sm:p-5">
        <PlanCanvas plan={state.plan} blocks={state.blocks} onChange={onChangePlan} onRefine={onRefine} />
      </section>
    </div>
  )
}
```

- [ ] **Step 2: Verify build + lint**

Run: `cd frontend && npm run build && npm run lint`
Expected: compiles; no new lint errors.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add components/ChatView.tsx
git commit -m "feat(chat-ui): ChatView two-pane container wiring SSE through reducer"
```

---

## Task 7: 把向导抽成 WizardView(行为不变的重构)

**Files:**
- Create: `frontend/components/WizardView.tsx`
- Modify: `frontend/app/page.tsx`

Goal: move the entire four-step wizard (state + step 1-4 JSX) out of `page.tsx` into `WizardView`, so `page.tsx` becomes a thin shell. **No behavior change** — pure relocation.

- [ ] **Step 1: Create WizardView with the moved logic**

Create `frontend/components/WizardView.tsx`. Move from `app/page.tsx`: the `STAGES` const, the wizard state (`step, events, plan, failed, lastBrief, concepts, proposing, timelineStages`), the functions (`run, confirmConcept, onRefine, restart`), and the Stepper + step 1-4 JSX block. It takes `onUnauthorized` as a prop (replacing the local `handleUnauthorized` for the wizard's own 401s) and renders only the wizard body (NOT the masthead/header — that stays in the shell).

```tsx
// frontend/components/WizardView.tsx
"use client"
import { useState } from "react"
import { Brief, Concept, Plan, SSEvent } from "@/lib/types"
import { generate, propose, refine, UnauthorizedError } from "@/lib/api"
import InputForm from "@/components/InputForm"
import ConceptChoice from "@/components/ConceptChoice"
import StageTimeline from "@/components/StageTimeline"
import PlanView from "@/components/PlanView"
import ExportBar from "@/components/ExportBar"
import Stepper, { Step } from "@/components/Stepper"

const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"
const STAGES = ["concept", "bible", "characters", "episodes", "placements", "hero", "production_distribution", "visuals"]

// WizardView is the original four-step deterministic flow (填需求 → 选立意 → 生成 →
// 方案), extracted verbatim from page.tsx so the page can host a mode switch.
export default function WizardView({ onUnauthorized }: { onUnauthorized: () => void }) {
  const [step, setStep] = useState<Step>(1)
  const [events, setEvents] = useState<SSEvent[]>([])
  const [plan, setPlan] = useState<Plan | null>(null)
  const [failed, setFailed] = useState<string | null>(null)
  const [lastBrief, setLastBrief] = useState<Brief | undefined>(undefined)
  const [concepts, setConcepts] = useState<Concept[]>([])
  const [proposing, setProposing] = useState(false)
  const [timelineStages, setTimelineStages] = useState<string[]>(STAGES)

  async function run(brief: Brief) {
    setLastBrief(brief); setEvents([]); setPlan(null); setConcepts([])
    setTimelineStages(STAGES.slice(1)); setFailed(null); setProposing(true); setStep(2)
    try {
      setConcepts(await propose(brief))
    } catch (err) {
      if (err instanceof UnauthorizedError) { onUnauthorized(); return }
      setFailed(err instanceof Error ? `${err.message} — 后端是否在 ${API} 运行?` : "立意提案失败。")
    } finally { setProposing(false) }
  }

  async function confirmConcept(chosen: Concept) {
    if (!lastBrief) return
    setEvents([]); setPlan(null); setTimelineStages(STAGES.slice(1)); setFailed(null); setStep(3)
    try {
      await generate(lastBrief, (e) => {
        setEvents((prev) => [...prev, e])
        if (e.type === "complete" && e.plan) { setPlan(e.plan); setStep(4) }
      }, chosen)
    } catch (err) {
      if (err instanceof UnauthorizedError) { onUnauthorized(); return }
      setFailed(err instanceof Error ? `${err.message} — 后端是否在 ${API} 运行?` : "生成失败。")
    }
  }

  async function onRefine(fromStage: string, only: boolean, note: string) {
    if (!plan) return
    const idx = STAGES.indexOf(fromStage)
    const subset = only ? [fromStage] : STAGES.slice(idx >= 0 ? idx : 0)
    setTimelineStages(subset); setEvents([]); setFailed(null); setStep(3)
    try {
      await refine({ plan, fromStage, only, note }, (e) => {
        setEvents((prev) => [...prev, e])
        if (e.type === "complete" && e.plan) { setPlan(e.plan); setStep(4) }
      })
    } catch (err) {
      if (err instanceof UnauthorizedError) { onUnauthorized(); return }
      setFailed(err instanceof Error ? `${err.message} — 后端是否在 ${API} 运行?` : "重跑失败。")
    }
  }

  function restart() {
    setStep(1); setEvents([]); setPlan(null); setConcepts([]); setTimelineStages(STAGES); setFailed(null)
  }

  return (
    <>
      <Stepper current={step} onStep={(s) => s === 1 && restart()} />
      {step === 1 && <InputForm onSubmit={run} disabled={proposing} defaults={lastBrief} />}
      {step === 2 && (
        <div className="space-y-6">
          {proposing && (
            <div className="panel rounded-2xl p-10 text-center">
              <span className="mx-auto mb-4 block h-6 w-6 animate-spin rounded-full border-2 border-ember-500/30 border-t-ember-400" />
              <p className="font-mono text-sm text-bone-300">正在生成立意方向…</p>
            </div>
          )}
          {!proposing && failed && (
            <div className="rounded-xl border border-signal-stop/40 bg-signal-stop/10 p-4">
              <p className="font-mono text-sm text-signal-stop">✕ {failed}</p>
              <button onClick={restart} className="mt-3 rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700">← 返回修改需求</button>
            </div>
          )}
          {!proposing && !failed && concepts.length > 0 && (
            <ConceptChoice concepts={concepts} onConfirm={confirmConcept} onBack={restart} />
          )}
        </div>
      )}
      {step === 3 && (
        <div className="space-y-6">
          <StageTimeline stages={timelineStages} events={events} />
          {failed && (
            <div className="rounded-xl border border-signal-stop/40 bg-signal-stop/10 p-4">
              <p className="font-mono text-sm text-signal-stop">✕ {failed}</p>
              <button onClick={restart} className="mt-3 rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700">← 返回修改需求</button>
            </div>
          )}
        </div>
      )}
      {step === 4 && plan && (
        <div className="space-y-8">
          <div className="flex items-center justify-between gap-3">
            <p className="font-mono text-xs text-bone-400">方案已生成 · 共 {plan.episodes?.length ?? 0} 集</p>
            <button onClick={restart} className="rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700">＋ 新方案</button>
          </div>
          <ExportBar plan={plan} />
          <PlanView plan={plan} onChange={setPlan} onRefine={onRefine} />
        </div>
      )}
    </>
  )
}
```

- [ ] **Step 2: Verify the new component builds (page.tsx still has old code — expect a duplicate-ish but compiling state)**

Run: `cd frontend && npm run build`
Expected: compiles. (page.tsx is trimmed in Task 8; right now WizardView duplicates logic but both compile.)

- [ ] **Step 3: Commit**

```bash
cd frontend && git add components/WizardView.tsx
git commit -m "refactor(chat-ui): extract four-step wizard into WizardView"
```

---

## Task 8: page.tsx 降为外壳 + 「向导/对话」模式切换

**Files:**
- Modify: `frontend/app/page.tsx`

Replace the wizard guts in `page.tsx` with a `mode` switch that renders `WizardView` or `ChatView`. Keep auth gating, the masthead/header, history toggle, logout, footer.

- [ ] **Step 1: Rewrite page.tsx as a shell**

Replace the entire contents of `frontend/app/page.tsx` with:

```tsx
"use client"
import { useEffect, useState } from "react"
import { clearToken, verifyToken } from "@/lib/auth"
import LoginForm from "@/components/LoginForm"
import HistoryView from "@/components/HistoryView"
import WizardView from "@/components/WizardView"
import ChatView from "@/components/ChatView"

type View = "workbench" | "history"
type Mode = "wizard" | "chat"

export default function Home() {
  const [ready, setReady] = useState(false)
  const [authed, setAuthed] = useState(false)
  const [view, setView] = useState<View>("workbench")
  const [mode, setMode] = useState<Mode>("chat")

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      const ok = await verifyToken()
      if (!cancelled) { setAuthed(ok); setReady(true) }
    })()
    return () => { cancelled = true }
  }, [])

  function logout() { clearToken(); setAuthed(false); setView("workbench") }
  function handleUnauthorized() { clearToken(); setAuthed(false); setView("workbench") }

  if (!ready) return null
  if (!authed) return <LoginForm onAuthed={() => setAuthed(true)} />

  return (
    <div className="relative min-h-screen">
      <main className="mx-auto max-w-6xl px-5 py-10 sm:px-8 sm:py-16">
        <header className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="mb-4 flex items-center gap-3">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src="/ashley-logo.png" alt="Ashley Furniture Industries" className="h-8 w-auto rounded-md bg-bone-50 px-2.5 py-1.5 shadow-sm ring-1 ring-black/5" />
              <span className="inline-flex items-center gap-2 rounded-full border border-bone-500/20 bg-ink-800/60 px-3 py-1">
                <span className="h-1.5 w-1.5 rounded-full bg-ember-400" />
                <span className="label-tech">品牌内容工作台</span>
              </span>
            </div>
            <h1 className="font-display text-3xl font-semibold leading-[0.95] tracking-tight sm:text-4xl">
              短剧<span className="text-ember-400"> 生产工作台</span>
            </h1>
          </div>
          <div className="flex items-center gap-2">
            {/* mode switch */}
            <div className="flex rounded-lg border border-bone-500/20 bg-ink-800 p-0.5">
              {(["chat", "wizard"] as Mode[]).map((m) => (
                <button key={m} onClick={() => setMode(m)}
                  className={`rounded-md px-3 py-1 font-mono text-[11px] uppercase tracking-wider transition ${mode === m ? "bg-ember-500 text-ink-900" : "text-bone-300 hover:text-ember-400"}`}>
                  {m === "chat" ? "对话" : "向导"}
                </button>
              ))}
            </div>
            <button onClick={() => setView("history")} className="rounded-lg border border-bone-500/20 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-bone-300 transition hover:border-ember-400/50 hover:text-ember-400">历史</button>
            <button onClick={logout} className="rounded-lg border border-bone-500/20 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-bone-300 transition hover:border-bone-500/50 hover:text-bone-100">退出登录</button>
          </div>
        </header>

        {view === "history" ? (
          <HistoryView onBack={() => setView("workbench")} onUnauthorized={handleUnauthorized} />
        ) : mode === "chat" ? (
          <ChatView onUnauthorized={handleUnauthorized} />
        ) : (
          <WizardView onUnauthorized={handleUnauthorized} />
        )}
      </main>

      <footer className="border-t border-bone-500/10 py-6 text-center">
        <span className="label-tech">无 Key 演示模式生成完整示例方案 · 配置 GEMINI_API_KEY 启用真实生成</span>
      </footer>
    </div>
  )
}
```

- [ ] **Step 2: Verify build + lint**

Run: `cd frontend && npm run build && npm run lint`
Expected: compiles; no NEW lint errors (the pre-existing `HistoryView.tsx` set-state-in-effect error may remain — it is out of scope; do not fix it here).

- [ ] **Step 3: Commit**

```bash
cd frontend && git add app/page.tsx
git commit -m "feat(chat-ui): page shell with 向导/对话 mode switch (chat default)"
```

---

## Task 9: Playwright 无 key 冒烟

**Files:**
- Create: `frontend/e2e/chat.spec.ts`

A Playwright MCP/browser is available in this environment. This is a manual-run smoke (we do NOT add a test runner to package.json). Drive a real browser against the dev server with the backend in DemoMock (no key) so the ReAct trace is deterministic.

- [ ] **Step 1: Start backend (mock) + frontend dev**

```bash
# terminal A
cd backend && PORT=8080 go run ./cmd/server
# terminal B
cd frontend && npm run dev   # http://localhost:3000
```

- [ ] **Step 2: Write the smoke spec (for documentation / future CI)**

```ts
// frontend/e2e/chat.spec.ts
// Run manually with Playwright against a dev server + mock backend. Not wired into
// package.json (no test runner is installed). Documents the expected happy path.
import { test, expect } from "@playwright/test"

test("conversational agent streams a ReAct trace and builds the plan", async ({ page }) => {
  await page.goto("http://localhost:3000")
  // log in with default mock creds admin/admin
  await page.getByLabel(/用户名|username/i).fill("admin")
  await page.getByLabel(/密码|password/i).fill("admin")
  await page.getByRole("button", { name: /登录|login/i }).click()

  // chat is the default mode; send a brief
  await page.getByPlaceholder(/输入需求/).fill("做个家装逆袭短剧,植入 Ashley 沙发")
  await page.getByRole("button", { name: "发送" }).click()

  // a tool card appears (agent calling tools)
  await expect(page.getByText("🔧").first()).toBeVisible({ timeout: 15000 })
  // the canvas builds: at least the 立意 chip reaches done (✓)
  await expect(page.getByText(/✓ 立意/)).toBeVisible({ timeout: 30000 })
  // turn completes: composer re-enabled
  await expect(page.getByPlaceholder(/输入需求/)).toBeEnabled({ timeout: 30000 })
})
```

- [ ] **Step 3: Drive the smoke via the Playwright browser tool (controller does this)**

Using the available Playwright browser tool: navigate to `http://localhost:3000`, log in admin/admin, confirm 对话 mode is default, type the brief, click 发送, and OBSERVE:
- left column shows 💭 thought blocks and 🔧 tool cards streaming in,
- right canvas status chips flip pending → ✍ writing → ✓ done across the 8 blocks,
- an assistant closing message appears and the composer re-enables.
Capture a screenshot for the record.
Expected: full ReAct trace renders end-to-end with no console errors.

- [ ] **Step 4: Commit the spec**

```bash
cd frontend && git add e2e/chat.spec.ts
git commit -m "test(chat-ui): playwright no-key smoke for conversational agent"
```

---

## 收尾验证

- [ ] `cd frontend && npm run build` → 通过(类型检查 + 编译)
- [ ] `cd frontend && npm run lint` → 无新增错误(HistoryView 既有告警不在范围内)
- [ ] Playwright 冒烟:对话模式发一句 → 左栏思考/工具卡流式出现 → 右栏 8 块 pending→writing→done → 收尾消息 + 输入框恢复
- [ ] 「向导」模式仍按原四步工作(回归:抽取未改变行为)
- [ ] 历史、登录、登出仍正常

---

## Self-Review(作者已核对)

**1. Spec coverage(对照主设计 §5 渲染 + 入口决策)**
- 布局 B 双栏 → Task 6 ChatView grid `lg:grid-cols-[2fr_3fr]` ✓
- 左栏 C 思考流+工具卡 → Task 3/4(ThoughtBlock 流式→塌、ToolCard 可展开 I/O、ConversationColumn 顺序渲染+输入框)✓
- 右栏 A 活文档生长 → Task 5 PlanCanvas(状态条 pending/writing/done + 复用 PlanView);说明:逐 section shimmer 简化为顶部状态条,已注明为 v1 取舍 ✓
- 联动钩子「→ 已更新右侧「X」」→ ToolCard 用 `affectsStage` 渲染 ✓
- 复用 PlanView/ExportBar/types/auth ✓
- 多轮:ChatView 用 historyRef + planRef 维持单会话上下文,每次 send 带 plan + 文本 history(契合后端 `/api/chat` 请求体)✓
- 入口=同一页面模式切换 → Task 7(抽 WizardView)+ Task 8(page 外壳 + 切换,对话为默认)✓
- 贴图多模态:主设计 §5.2 提到对话支持贴图;本期 MVP **未做**(后端 `/api/chat` 当前请求体不含 images)——记为 follow-up,不在本计划范围(避免与后端契约不一致)。

**2. Placeholder scan**:无 TBD/TODO;每个代码步骤给出完整文件内容或精确改动。颜色类(`signal-go`)在 Task 3/5 注明「不存在则用 emerald-500」并要求先查主题——`signal-stop` 已在 page.tsx 使用,确证该命名族存在。

**3. Type consistency**:`Segment/BlockKey/BlockStatus/ChatState/chatReducer/initialChatState/pushUser`(Task 2)在 Task 4/5/6 的调用与 props 前后一致;`chat()/ChatReq/ChatEvent/ChatMessage`(Task 1 + 契约)在 Task 6 使用一致;`WizardView`/`ChatView` 的 `onUnauthorized` prop 在 Task 8 提供一致。

**4. 状态库更正**:主设计写 Zustand,本前端无该依赖,改用 `useReducer`,已在头部与本节标注,零新依赖。
