"use client"

import { useMemo, useState } from "react"
import { SSEvent } from "@/lib/types"

interface Props {
  stages: string[]
  events: SSEvent[]
}

type Status = "pending" | "running" | "done"

const STAGE_LABELS: Record<string, string> = {
  concept: "立意",
  bible: "剧集圣经",
  characters: "人物",
  episodes: "分集大纲",
  placements: "品牌植入",
  hero: "英雄场景",
  production_distribution: "制作与分发",
}

const STAGE_SUB: Record<string, string> = {
  concept: "logline · 爽点引擎",
  bible: "标题 · 平台 · 植入策略",
  characters: "卡司 · 弧光 · 关系",
  episodes: "钩子 · 悬念 · 节奏校验",
  placements: "SKU → 场景 → CTA",
  hero: "分镜表",
  production_distribution: "预算 · 格式 · CTA 文案",
}

function deriveStatus(stage: string, events: SSEvent[]): Status {
  let status: Status = "pending"
  for (const e of events) {
    if (e.stage !== stage) continue
    if (e.type === "stage_start") status = "running"
    if (e.type === "stage_done") status = "done"
  }
  return status
}

export default function StageTimeline({ stages, events }: Props) {
  const [open, setOpen] = useState<Record<string, boolean>>({})

  const payloads = useMemo(() => {
    const map: Record<string, unknown> = {}
    for (const e of events) {
      if (e.type === "stage_done" && e.stage) map[e.stage] = e.payload
    }
    return map
  }, [events])

  const errors = useMemo(
    () => events.filter((e) => e.type === "error"),
    [events]
  )

  const doneCount = stages.filter((s) => deriveStatus(s, events) === "done").length
  const started = events.length > 0

  return (
    <section className="panel rounded-2xl p-6 sm:p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <span className="label-tech">流水线 · 样片</span>
          <h2 className="mt-1 font-display text-2xl font-semibold tracking-tight">
            Agent 阶段
          </h2>
        </div>
        <div className="text-right">
          <span className="label-tech">已完成</span>
          <p className="font-mono text-lg text-ember-400">
            {doneCount.toString().padStart(2, "0")}
            <span className="text-bone-500">/{stages.length.toString().padStart(2, "0")}</span>
          </p>
        </div>
      </div>

      <ol className="relative">
        {/* the vertical rail */}
        <div
          aria-hidden
          className="absolute bottom-3 left-[15px] top-3 w-px bg-bone-500/15"
        />

        {stages.map((stage, i) => {
          const status = deriveStatus(stage, events)
          const payload = payloads[stage]
          const isOpen = open[stage]

          return (
            <li
              key={stage}
              className="relative flex gap-4 pb-5 last:pb-0"
              style={
                started
                  ? {
                      animationName: "rise",
                      animationDuration: "0.6s",
                      animationTimingFunction: "cubic-bezier(0.16,1,0.3,1)",
                      animationFillMode: "both",
                      animationDelay: `${i * 50}ms`,
                    }
                  : undefined
              }
            >
              {/* node */}
              <div className="relative z-10 flex-shrink-0">
                <StatusNode status={status} index={i} />
              </div>

              {/* body */}
              <div className="min-w-0 flex-1 pt-0.5">
                <div className="flex items-baseline justify-between gap-3">
                  <h3
                    className={`font-sans text-sm font-medium transition ${
                      status === "pending" ? "text-bone-500" : "text-bone-50"
                    }`}
                  >
                    {STAGE_LABELS[stage] ?? stage}
                  </h3>
                  <span className="label-tech whitespace-nowrap">
                    {status === "running" ? "生成中…" : status === "done" ? "完成 ✓" : "待命"}
                  </span>
                </div>
                <p className="mt-0.5 font-mono text-[10px] text-bone-500">
                  {STAGE_SUB[stage] ?? ""}
                </p>

                {status === "running" && (
                  <div className="mt-2 h-1 overflow-hidden rounded-full bg-ink-700">
                    <div
                      className="h-full w-1/2 rounded-full bg-gradient-to-r from-transparent via-ember-400 to-transparent animate-shimmer"
                      style={{ backgroundSize: "200% 100%" }}
                    />
                  </div>
                )}

                {status === "done" && payload != null && (
                  <div className="mt-2">
                    <button
                      onClick={() => setOpen((o) => ({ ...o, [stage]: !o[stage] }))}
                      className="font-mono text-[10px] uppercase tracking-wider text-ember-400/80 transition hover:text-ember-300"
                    >
                      {isOpen ? "▾ 收起原始输出" : "▸ 查看原始输出"}
                    </button>
                    {isOpen && (
                      <pre className="mt-2 max-h-72 overflow-auto rounded-lg border border-bone-500/10 bg-ink-900/80 p-3 font-mono text-[11px] leading-relaxed text-bone-300">
                        {JSON.stringify(payload, null, 2)}
                      </pre>
                    )}
                  </div>
                )}
              </div>
            </li>
          )
        })}
      </ol>

      {errors.length > 0 && (
        <div className="mt-4 rounded-lg border border-signal-stop/40 bg-signal-stop/10 p-4">
          {errors.map((e, i) => (
            <p key={i} className="font-mono text-xs text-signal-stop">
              <span className="font-semibold">✕ {e.stage ?? "流水线"}:</span> {e.message}
            </p>
          ))}
        </div>
      )}
    </section>
  )
}

function StatusNode({ status, index }: { status: Status; index: number }) {
  if (status === "done") {
    return (
      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-ember-500 text-ink-900 shadow-[0_0_18px_rgba(228,132,47,0.45)]">
        <svg viewBox="0 0 16 16" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={2.5}>
          <path d="M3 8.5l3.2 3.5L13 5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </div>
    )
  }
  if (status === "running") {
    return (
      <div className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-ember-500/60 bg-ink-800">
        <span className="h-4 w-4 animate-spin rounded-full border-2 border-ember-500/30 border-t-ember-400" />
      </div>
    )
  }
  return (
    <div className="flex h-8 w-8 items-center justify-center rounded-full border border-bone-500/20 bg-ink-800">
      <span className="font-mono text-[10px] text-bone-500">
        {(index + 1).toString().padStart(2, "0")}
      </span>
    </div>
  )
}
