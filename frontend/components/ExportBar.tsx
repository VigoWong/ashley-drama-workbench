"use client"

import { useState } from "react"
import { Plan } from "@/lib/types"
import { planToMarkdown } from "@/lib/markdown"

interface Props {
  plan: Plan
}

function slugify(s: string): string {
  return (
    s
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "production-plan"
  )
}

function download(filename: string, content: string, mime: string) {
  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

export default function ExportBar({ plan }: Props) {
  const [flash, setFlash] = useState<string | null>(null)
  const base = slugify(plan.bible?.title ?? "production-plan")

  function ping(msg: string) {
    setFlash(msg)
    window.setTimeout(() => setFlash(null), 1600)
  }

  return (
    <div className="panel sticky top-4 z-30 flex flex-wrap items-center justify-between gap-3 rounded-xl px-5 py-3">
      <div className="flex items-center gap-3">
        <span className="flex h-2 w-2 animate-pulse-glow rounded-full bg-ember-400" />
        <span className="label-tech">Plan locked · ready to ship</span>
        {flash && (
          <span className="font-mono text-[10px] text-signal-go">↓ {flash}</span>
        )}
      </div>
      <div className="flex gap-2">
        <button
          onClick={() => {
            download(`${base}.json`, JSON.stringify(plan, null, 2), "application/json")
            ping("JSON exported")
          }}
          className="rounded-lg border border-bone-500/25 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700"
        >
          ⤓ JSON
        </button>
        <button
          onClick={() => {
            download(`${base}.md`, planToMarkdown(plan), "text/markdown")
            ping("Markdown exported")
          }}
          className="rounded-lg bg-ember-500 px-4 py-2 font-mono text-xs uppercase tracking-wider text-ink-900 transition hover:bg-ember-400"
        >
          ⤓ Markdown
        </button>
      </div>
    </div>
  )
}
