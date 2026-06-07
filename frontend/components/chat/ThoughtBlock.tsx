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
