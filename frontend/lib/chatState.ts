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
