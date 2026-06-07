// frontend/components/ChatView.tsx
"use client"
import { useEffect, useReducer, useRef } from "react"
import { ChatEvent, ChatMessage, Plan } from "@/lib/types"
import { chat, UnauthorizedError } from "@/lib/api"
import { chatReducer, initialChatState, pushUser, ChatState } from "@/lib/chatState"
import ConversationColumn from "@/components/chat/ConversationColumn"
import PlanCanvas from "@/components/chat/PlanCanvas"

type Action =
  | { type: "event"; e: ChatEvent }
  | { type: "user"; text: string }
  | { type: "setPlan"; plan: Plan }

// ChatView is the two-pane conversational agent surface. Left: the ReAct
// conversation; right: the live plan canvas. It owns the reduced ChatState and
// drives /api/chat, feeding every ChatEvent through the pure reducer.
export default function ChatView({ onUnauthorized }: { onUnauthorized: () => void }) {
  const [state, dispatch] = useReducer(
    (s: ChatState, a: Action) => {
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
  const historyRef = useRef<ChatMessage[]>([])
  // Sync planRef after each render so onSend always sees the latest plan.
  useEffect(() => { planRef.current = state.plan }, [state.plan])

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
