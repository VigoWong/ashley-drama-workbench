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
