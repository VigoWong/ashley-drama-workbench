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

// ConversationColumn renders a composer at the top and the agent turn segments
// below it in arrival order. It auto-scrolls to the newest segment.
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
      <div className="border-b border-bone-500/10 pb-3">
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

      <div className="mt-3 min-h-0 flex-1 overflow-y-auto pr-1">
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
    </div>
  )
}
