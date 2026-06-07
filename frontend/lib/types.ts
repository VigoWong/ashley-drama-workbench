export interface BriefImage { mimeType: string; data: string; label?: string }
export interface Brief { requirement: string; episodes: number; episodeSecs: number; market?: string; extra?: string; images?: BriefImage[] }
export interface Concept { logline: string; theme: string; audience: string; tone: string; payoffEngine: string; coreConflict: string; tropesUsed: string[] }
export interface SeriesBible { title: string; genreTags: string[]; episodes: number; episodeSecs: number; platform: string; integrationThesis: string }
export interface Character { name: string; role: string; bio: string; arc: string; relationships: string }
export interface Episode { number: number; title: string; synopsis: string; beats: string[]; hook: string; cliffhanger: string; payoff: string }
export interface Placement { episode: number; scene: string; productSku: string; category: string; emotionalBeat: string; ctaTiming: string }
export interface Shot { number: number; shotType: string; action: string; dialogue: string }
export interface HeroScene { episode: number; title: string; shots: Shot[] }
export interface Production { format: string; budgetTier: string; shotCount: number; castSize: number; locations: string[]; furnitureProps: string[] }
export interface Distribution { ctaCopy: string; linkPlacement: string; hashtags: string[] }
export interface Visual { label: string; mimeType: string; data: string }
export interface Plan { brief: Brief; concept: Concept; bible: SeriesBible; characters: Character[]; episodes: Episode[]; placements: Placement[]; heroScenes: HeroScene[]; production: Production; distribution: Distribution; visuals?: Visual[] }
export type EventType = "stage_start" | "stage_done" | "error" | "complete"
export interface SSEvent { type: EventType; stage?: string; index?: number; total?: number; message?: string; payload?: unknown; plan?: Plan }
export interface RefineReq { plan: Plan; fromStage: string; only: boolean; note?: string }
export interface ProposeResp { concepts: Concept[] }
export interface HistorySummary { id: string; title: string; genre: string; episodes: number; createdAt: string }
export interface HistoryRecord { id: string; createdAt: string; brief: Brief; plan: Plan }

// ---- conversational ReAct agent (mirrors backend/internal/model/chat.go) ----
export type ChatEventType =
  | "thought.delta" | "thought.done"
  | "tool.start" | "tool.result"
  | "message.delta" | "message.done"
  | "block.start" | "block.done"
  | "turn.done" | "error"
export interface ChatEvent {
  type: ChatEventType
  text?: string; message?: string
  toolId?: string; toolName?: string; friendlyName?: string
  input?: unknown; output?: unknown; status?: "ok" | "fail"; affectsStage?: string
  stage?: string; payload?: unknown
  plan?: Plan
}
// Prior text history sent back on /api/chat (user/assistant bubbles only).
export interface ChatMessage { role: "user" | "assistant" | "tool"; text?: string }
export interface ChatReq { message: string; history?: ChatMessage[]; plan?: Plan }
