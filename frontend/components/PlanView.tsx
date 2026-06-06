"use client"

import { useState } from "react"
import { Plan } from "@/lib/types"

interface Props {
  plan: Plan
  editable?: boolean
  onChange?: (plan: Plan) => void
  onRefine?: (fromStage: string, only: boolean, note: string) => void
}

// shared input class, matching InputForm's field styling.
const FIELD =
  "w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-3 py-2 font-sans text-sm text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20"

export default function PlanView({ plan, editable = false, onChange, onRefine }: Props) {
  // update merges a partial patch into the plan and bubbles it to the parent.
  function update(patch: Partial<Plan>) {
    onChange?.({ ...plan, ...patch })
  }
  const shared = { plan, editable, update, onRefine }
  return (
    <div className="space-y-10">
      <Masthead plan={plan} />
      <VisualsSection {...shared} />
      <ConceptSection {...shared} />
      <BibleSection {...shared} />
      <CharactersSection {...shared} />
      <EpisodesSection {...shared} />
      <PlacementsSection {...shared} />
      <HeroSection {...shared} />
      <ProductionSection {...shared} />
      <DistributionSection {...shared} />
    </div>
  )
}

interface SectionProps {
  plan: Plan
  editable: boolean
  update: (patch: Partial<Plan>) => void
  onRefine?: (fromStage: string, only: boolean, note: string) => void
}

/* ---------- editable primitives ---------- */

function EditText({
  value,
  onChange,
  className = "",
}: {
  value: string
  onChange: (v: string) => void
  className?: string
}) {
  return (
    <input
      type="text"
      value={value ?? ""}
      onChange={(e) => onChange(e.target.value)}
      className={`${FIELD} ${className}`}
    />
  )
}

function EditArea({
  value,
  onChange,
  rows = 2,
  className = "",
}: {
  value: string
  onChange: (v: string) => void
  rows?: number
  className?: string
}) {
  return (
    <textarea
      value={value ?? ""}
      onChange={(e) => onChange(e.target.value)}
      rows={rows}
      className={`${FIELD} resize-y leading-relaxed ${className}`}
    />
  )
}

/* ---------- refine controls (per section) ---------- */

function RefineBar({
  stage,
  onRefine,
}: {
  stage: string
  onRefine?: (fromStage: string, only: boolean, note: string) => void
}) {
  const [openOnly, setOpenOnly] = useState<boolean | null>(null) // which mode's note box is open
  const [note, setNote] = useState("")
  if (!onRefine) return null

  function toggle(only: boolean) {
    setOpenOnly((cur) => (cur === only ? null : only))
  }
  function confirm() {
    if (openOnly === null) return
    onRefine!(stage, openOnly, note.trim())
    setOpenOnly(null)
    setNote("")
  }

  return (
    <div className="flex flex-col items-end gap-2">
      <div className="flex gap-2">
        <button
          onClick={() => toggle(true)}
          title="仅重新生成本段"
          className={`rounded-lg border px-2.5 py-1 font-mono text-[10px] uppercase tracking-wider transition ${
            openOnly === true
              ? "border-ember-500/60 bg-ember-500/15 text-ember-200"
              : "border-bone-500/20 text-bone-300 hover:border-ember-400/50 hover:text-ember-400"
          }`}
        >
          ↻ 重生成本段
        </button>
        <button
          onClick={() => toggle(false)}
          title="从本段开始，往下所有阶段重跑"
          className={`rounded-lg border px-2.5 py-1 font-mono text-[10px] uppercase tracking-wider transition ${
            openOnly === false
              ? "border-ember-500/60 bg-ember-500/15 text-ember-200"
              : "border-bone-500/20 text-bone-300 hover:border-ember-400/50 hover:text-ember-400"
          }`}
        >
          ⏬ 从此往下重跑
        </button>
      </div>
      {openOnly !== null && (
        <div className="flex w-full max-w-md flex-col gap-2 rounded-lg border border-ember-500/25 bg-ink-900/60 p-3">
          <span className="label-tech">
            {openOnly ? "仅重生成本段" : "从本段往下重跑"} · 额外要求（可空）
          </span>
          <textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={2}
            placeholder="例如：基调更轻松一点，加入更多职场元素"
            className={`${FIELD} resize-y`}
          />
          <div className="flex justify-end gap-2">
            <button
              onClick={() => {
                setOpenOnly(null)
                setNote("")
              }}
              className="rounded-lg border border-bone-500/20 px-3 py-1 font-mono text-[10px] uppercase tracking-wider text-bone-300 transition hover:border-bone-500/50"
            >
              取消
            </button>
            <button
              onClick={confirm}
              className="rounded-lg bg-ember-500 px-3 py-1 font-mono text-[10px] font-semibold uppercase tracking-wider text-ink-900 transition hover:bg-ember-400"
            >
              确认重跑
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

/* ---------- section chrome ---------- */

function SectionHead({
  no,
  kicker,
  title,
  stage,
  onRefine,
}: {
  no: string
  kicker: string
  title: string
  stage?: string
  onRefine?: (fromStage: string, only: boolean, note: string) => void
}) {
  return (
    <div className="mb-5 flex items-start justify-between gap-4 border-b border-bone-500/15 pb-3">
      <div className="flex items-baseline gap-4">
        <span className="font-mono text-xs text-ember-500/70">{no}</span>
        <div>
          <span className="label-tech">{kicker}</span>
          <h2 className="font-display text-2xl font-semibold tracking-tight">{title}</h2>
        </div>
      </div>
      {stage && <RefineBar stage={stage} onRefine={onRefine} />}
    </div>
  )
}

/* ---------- 00 Masthead ---------- */

function Masthead({ plan }: { plan: Plan }) {
  const { bible, concept, brief } = plan
  return (
    <header className="panel relative overflow-hidden rounded-2xl p-8 sm:p-10">
      <div
        aria-hidden
        className="pointer-events-none absolute -right-10 -top-16 h-64 w-64 rounded-full"
        style={{ background: "radial-gradient(circle, rgba(228,132,47,0.22), transparent 70%)" }}
      />
      <span className="label-tech">剧集 · 9:16 竖屏</span>
      <h1 className="mt-2 max-w-3xl font-display text-4xl font-semibold leading-[1.05] tracking-tight sm:text-6xl">
        {bible.title || "未命名剧集"}
      </h1>
      <p className="mt-4 max-w-2xl font-display text-lg italic leading-relaxed text-bone-300">
        &ldquo;{concept.logline}&rdquo;
      </p>
      <div className="mt-6 flex flex-wrap gap-2">
        {(bible.genreTags?.length ? bible.genreTags : concept.tropesUsed ?? []).map((t) => (
          <span
            key={t}
            className="rounded-full border border-ember-500/30 bg-ember-500/10 px-3 py-1 font-mono text-[10px] uppercase tracking-wider text-ember-200"
          >
            {t}
          </span>
        ))}
      </div>
      <dl className="mt-7 grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-bone-500/10 bg-bone-500/10 sm:grid-cols-4">
        <Stat label="集数" value={String(bible.episodes || brief.episodes)} />
        <Stat label="单集秒数" value={`${bible.episodeSecs || brief.episodeSecs}s`} />
        <Stat label="平台" value={bible.platform || "抖音 / 红果短剧"} />
        <Stat label="市场" value={brief.market || "中国 · 中文"} />
      </dl>
    </header>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-ink-800 px-4 py-3">
      <span className="label-tech">{label}</span>
      <p className="mt-0.5 font-mono text-sm text-bone-50">{value}</p>
    </div>
  )
}

/* ---------- 00b Visuals (AI concept art) ---------- */

function VisualsSection({ plan, onRefine }: SectionProps) {
  const visuals = plan.visuals ?? []
  if (visuals.length === 0) return null
  return (
    <section>
      <SectionHead
        no="00"
        kicker="AI 生成 · Imagen 文生图"
        title="分镜 · 概念图"
        stage="visuals"
        onRefine={onRefine}
      />
      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        {visuals.map((v, i) => (
          <figure key={i} className="panel group overflow-hidden rounded-2xl">
            <div className="relative aspect-[9/16] overflow-hidden bg-ink-900">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={`data:${v.mimeType};base64,${v.data}`}
                alt={v.label}
                className="h-full w-full object-cover transition duration-500 group-hover:scale-[1.03]"
              />
              <div
                aria-hidden
                className="pointer-events-none absolute inset-0 bg-gradient-to-t from-ink-900/70 via-transparent to-transparent"
              />
            </div>
            <figcaption className="flex items-center justify-between gap-2 px-4 py-3">
              <span className="font-sans text-sm text-bone-100">{v.label}</span>
              <span className="rounded border border-ember-500/30 px-1.5 py-0.5 font-mono text-[9px] uppercase tracking-wider text-ember-200">
                9:16
              </span>
            </figcaption>
          </figure>
        ))}
      </div>
    </section>
  )
}

/* ---------- 01 Concept ---------- */

function ConceptSection({ plan, editable, update, onRefine }: SectionProps) {
  const c = plan.concept
  function patch(field: keyof typeof c, v: string) {
    update({ concept: { ...c, [field]: v } })
  }
  const rows: [string, keyof typeof c, string][] = [
    ["主题", "theme", c.theme],
    ["受众", "audience", c.audience],
    ["基调", "tone", c.tone],
    ["核心冲突", "coreConflict", c.coreConflict],
  ]
  return (
    <section>
      <SectionHead no="01" kicker="创意北极星" title="立意" stage="concept" onRefine={onRefine} />
      {editable && (
        <div className="panel mb-5 rounded-xl p-4">
          <span className="label-tech">Logline</span>
          <EditArea value={c.logline} onChange={(v) => patch("logline", v)} className="mt-1" />
        </div>
      )}
      <div className="grid gap-5 lg:grid-cols-3">
        <div className="panel rounded-2xl p-6 lg:col-span-1">
          <span className="label-tech">爽点引擎 · Payoff Engine</span>
          {editable ? (
            <EditArea
              value={c.payoffEngine}
              onChange={(v) => patch("payoffEngine", v)}
              className="mt-2"
              rows={4}
            />
          ) : (
            <p className="mt-2 font-display text-xl leading-snug text-ember-200">{c.payoffEngine}</p>
          )}
          <p className="mt-3 font-sans text-sm text-bone-300">
            驱动观众持续追看的可复用爽感机制。
          </p>
        </div>
        <dl className="grid gap-3 lg:col-span-2 sm:grid-cols-2">
          {rows.map(([k, field, v]) => (
            <div key={k} className="panel rounded-xl p-4">
              <dt className="label-tech">{k}</dt>
              {editable && field !== "audience" ? (
                <EditArea value={v} onChange={(nv) => patch(field, nv)} className="mt-1" />
              ) : (
                <dd className="mt-1 font-sans text-sm leading-relaxed text-bone-100">{v}</dd>
              )}
            </div>
          ))}
        </dl>
      </div>
    </section>
  )
}

/* ---------- 02 Series Bible ---------- */

function BibleSection({ plan, editable, update, onRefine }: SectionProps) {
  const b = plan.bible
  function patch(field: keyof typeof b, v: string) {
    update({ bible: { ...b, [field]: v } })
  }
  return (
    <section>
      <SectionHead no="02" kicker="形态契约" title="剧集圣经" stage="bible" onRefine={onRefine} />
      <div className="panel rounded-2xl p-6">
        {editable && (
          <div className="mb-4">
            <span className="label-tech">标题</span>
            <EditText value={b.title} onChange={(v) => patch("title", v)} className="mt-1" />
          </div>
        )}
        {editable ? (
          <>
            <span className="label-tech">植入策略 · Integration Thesis</span>
            <EditArea
              value={b.integrationThesis}
              onChange={(v) => patch("integrationThesis", v)}
              className="mt-1"
              rows={3}
            />
          </>
        ) : (
          <p className="font-sans text-base leading-relaxed text-bone-100">{b.integrationThesis}</p>
        )}
        {b.genreTags?.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2">
            {b.genreTags.map((t) => (
              <span key={t} className="rounded-md bg-ink-700 px-2.5 py-1 font-mono text-[11px] text-bone-300">
                {t}
              </span>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

/* ---------- 03 Characters ---------- */

const ROLE_TINT: Record<string, string> = {
  protagonist: "border-ember-500/40 text-ember-200",
  antagonist: "border-signal-stop/40 text-signal-stop",
  "love-interest": "border-bone-300/40 text-bone-100",
}

function CharactersSection({ plan, editable, update, onRefine }: SectionProps) {
  function patch(idx: number, field: "name" | "bio" | "arc", v: string) {
    const next = plan.characters.map((c, i) => (i === idx ? { ...c, [field]: v } : c))
    update({ characters: next })
  }
  return (
    <section>
      <SectionHead no="03" kicker="卡司阵容" title="人物" stage="characters" onRefine={onRefine} />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {plan.characters.map((c, idx) => (
          <article key={idx} className="panel group flex flex-col rounded-2xl p-5">
            <div className="flex items-start justify-between gap-2">
              {editable ? (
                <EditText value={c.name} onChange={(v) => patch(idx, "name", v)} className="font-display" />
              ) : (
                <h3 className="font-display text-xl font-semibold tracking-tight">{c.name}</h3>
              )}
              <span
                className={`whitespace-nowrap rounded-full border px-2 py-0.5 font-mono text-[9px] uppercase tracking-wider ${
                  ROLE_TINT[c.role?.toLowerCase()] ?? "border-bone-500/30 text-bone-400"
                }`}
              >
                {c.role}
              </span>
            </div>
            {editable ? (
              <EditArea value={c.bio} onChange={(v) => patch(idx, "bio", v)} className="mt-2" />
            ) : (
              <p className="mt-2 font-sans text-sm leading-relaxed text-bone-300">{c.bio}</p>
            )}
            {editable ? (
              <EditArea value={c.arc} onChange={(v) => patch(idx, "arc", v)} className="mt-3" />
            ) : (
              c.arc && (
                <p className="mt-3 border-l-2 border-ember-500/40 pl-3 font-sans text-sm italic text-bone-100">
                  {c.arc}
                </p>
              )
            )}
            {c.relationships && (
              <p className="mt-auto pt-3 font-mono text-[10px] uppercase tracking-wider text-bone-400">
                {c.relationships}
              </p>
            )}
          </article>
        ))}
      </div>
    </section>
  )
}

/* ---------- 04 Episodes ---------- */

function EpisodesSection({ plan, editable, update, onRefine }: SectionProps) {
  const [openEp, setOpenEp] = useState<number | null>(plan.episodes[0]?.number ?? null)
  function patch(idx: number, field: "title" | "synopsis" | "hook" | "cliffhanger" | "payoff", v: string) {
    const next = plan.episodes.map((e, i) => (i === idx ? { ...e, [field]: v } : e))
    update({ episodes: next })
  }
  return (
    <section>
      <SectionHead no="04" kicker="节拍表" title="分集" stage="episodes" onRefine={onRefine} />
      <div className="space-y-2">
        {plan.episodes.map((e, idx) => {
          const isOpen = openEp === e.number
          return (
            <div
              key={e.number}
              className={`panel overflow-hidden rounded-xl transition ${isOpen ? "border-ember-500/30" : ""}`}
            >
              <button
                onClick={() => setOpenEp(isOpen ? null : e.number)}
                className="flex w-full items-center gap-4 px-5 py-4 text-left transition hover:bg-ink-700/40"
              >
                <span className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-ink-700 font-mono text-sm text-ember-400">
                  {e.number.toString().padStart(2, "0")}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-sans text-sm font-medium text-bone-50">{e.title}</span>
                  <span className="block truncate font-mono text-[10px] text-bone-400">{e.hook}</span>
                </span>
                <span className={`font-mono text-xs text-bone-400 transition ${isOpen ? "rotate-90" : ""}`}>
                  ▸
                </span>
              </button>
              {isOpen && (
                <div className="space-y-4 border-t border-bone-500/10 px-5 py-4">
                  {editable && (
                    <div>
                      <span className="label-tech">标题</span>
                      <EditText value={e.title} onChange={(v) => patch(idx, "title", v)} className="mt-1" />
                    </div>
                  )}
                  {editable ? (
                    <div>
                      <span className="label-tech">梗概</span>
                      <EditArea value={e.synopsis} onChange={(v) => patch(idx, "synopsis", v)} className="mt-1" />
                    </div>
                  ) : (
                    e.synopsis && (
                      <p className="font-sans text-sm leading-relaxed text-bone-200">{e.synopsis}</p>
                    )
                  )}
                  <div className="grid gap-3 sm:grid-cols-3">
                    <Beat
                      tone="open"
                      label="钩子 · 黄金3秒"
                      value={e.hook}
                      editable={editable}
                      onChange={(v) => patch(idx, "hook", v)}
                    />
                    <Beat
                      tone="payoff"
                      label="爽点 · Payoff"
                      value={e.payoff}
                      editable={editable}
                      onChange={(v) => patch(idx, "payoff", v)}
                    />
                    <Beat
                      tone="cliff"
                      label="结尾悬念"
                      value={e.cliffhanger}
                      editable={editable}
                      onChange={(v) => patch(idx, "cliffhanger", v)}
                    />
                  </div>
                  {e.beats?.length > 0 && (
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="label-tech">节拍</span>
                      {e.beats.map((b, i) => (
                        <span key={i} className="rounded bg-ink-700 px-2 py-0.5 font-mono text-[10px] text-bone-300">
                          {b}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </section>
  )
}

function Beat({
  tone,
  label,
  value,
  editable,
  onChange,
}: {
  tone: "open" | "payoff" | "cliff"
  label: string
  value: string
  editable?: boolean
  onChange?: (v: string) => void
}) {
  const tint = {
    open: "border-bone-300/30",
    payoff: "border-ember-500/40",
    cliff: "border-signal-stop/40",
  }[tone]
  if (!editable && !value) return null
  return (
    <div className={`rounded-lg border ${tint} bg-ink-900/50 p-3`}>
      <span className="label-tech">{label}</span>
      {editable && onChange ? (
        <EditArea value={value} onChange={onChange} className="mt-1" />
      ) : (
        <p className="mt-1 font-sans text-sm leading-snug text-bone-100">{value}</p>
      )}
    </div>
  )
}

/* ---------- 05 Placements ---------- */

function PlacementsSection({ plan, editable, update, onRefine }: SectionProps) {
  function patch(idx: number, field: "scene" | "emotionalBeat" | "ctaTiming", v: string) {
    const next = plan.placements.map((p, i) => (i === idx ? { ...p, [field]: v } : p))
    update({ placements: next })
  }
  return (
    <section>
      <SectionHead no="05" kicker="品牌植入" title="Ashley 植入" stage="placements" onRefine={onRefine} />
      <div className="panel overflow-hidden rounded-2xl">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[64rem] border-collapse text-left">
            <colgroup>
              <col className="w-14" />
              <col className="w-36" />
              <col className="w-[26%]" />
              <col className="w-[20%]" />
              <col className="w-[34%]" />
            </colgroup>
            <thead>
              <tr className="border-b border-bone-500/15">
                {["集", "产品", "场景", "情绪节点", "CTA 时机"].map((h) => (
                  <th key={h} className="label-tech whitespace-nowrap px-4 py-3 font-normal">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {plan.placements.map((p, i) => (
                <tr key={i} className="border-b border-bone-500/10 align-top transition last:border-0 hover:bg-ink-700/30">
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-sm text-ember-400">
                    {p.episode.toString().padStart(2, "0")}
                  </td>
                  <td className="px-4 py-3">
                    <span className="block whitespace-nowrap font-mono text-[11px] text-bone-400">{p.productSku}</span>
                    <span className="block font-sans text-sm capitalize text-bone-100">{p.category}</span>
                  </td>
                  <td className="px-4 py-3 font-sans text-sm leading-relaxed text-bone-300">
                    {editable ? (
                      <EditArea value={p.scene} onChange={(v) => patch(i, "scene", v)} />
                    ) : (
                      p.scene
                    )}
                  </td>
                  <td className="px-4 py-3 font-sans text-sm leading-relaxed text-bone-200">
                    {editable ? (
                      <EditArea value={p.emotionalBeat} onChange={(v) => patch(i, "emotionalBeat", v)} />
                    ) : (
                      p.emotionalBeat
                    )}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs leading-relaxed text-bone-300">
                    {editable ? (
                      <EditText value={p.ctaTiming} onChange={(v) => patch(i, "ctaTiming", v)} />
                    ) : (
                      p.ctaTiming
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}

/* ---------- 06 Hero scenes (read-only) ---------- */

function HeroSection({ plan, onRefine }: SectionProps) {
  return (
    <section>
      <SectionHead no="06" kicker="样板分镜" title="英雄场景" stage="hero" onRefine={onRefine} />
      <div className="grid gap-5 lg:grid-cols-2">
        {plan.heroScenes.map((h, i) => (
          <article key={i} className="panel overflow-hidden rounded-2xl">
            <div className="flex items-center justify-between border-b border-bone-500/15 px-5 py-3">
              <h3 className="font-display text-lg font-semibold tracking-tight">{h.title}</h3>
              <span className="rounded bg-ink-700 px-2 py-0.5 font-mono text-[10px] text-ember-400">
                EP {h.episode.toString().padStart(2, "0")}
              </span>
            </div>
            <ol className="divide-y divide-bone-500/10">
              {h.shots.map((s) => (
                <li key={s.number} className="flex gap-4 px-5 py-3">
                  <span className="font-mono text-xs text-bone-400">
                    {s.number.toString().padStart(2, "0")}
                  </span>
                  <span className="flex-shrink-0 rounded border border-ember-500/30 px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wider text-ember-200">
                    {s.shotType}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="font-sans text-sm leading-snug text-bone-100">{s.action}</p>
                    {s.dialogue && (
                      <p className="mt-1 font-display text-sm italic text-bone-300">
                        &ldquo;{s.dialogue}&rdquo;
                      </p>
                    )}
                  </div>
                </li>
              ))}
            </ol>
          </article>
        ))}
      </div>
    </section>
  )
}

/* ---------- 07 Production (read-only; shares stage with distribution) ---------- */

function ProductionSection({ plan, onRefine }: SectionProps) {
  const pr = plan.production
  return (
    <section>
      <SectionHead
        no="07"
        kicker="拍摄现场"
        title="制作"
        stage="production_distribution"
        onRefine={onRefine}
      />
      <div className="grid gap-5 lg:grid-cols-3">
        <dl className="grid grid-cols-2 gap-3 lg:col-span-1">
          <Stat2 label="格式" value={pr.format} />
          <Stat2 label="预算档" value={pr.budgetTier} />
          <Stat2 label="镜头数" value={String(pr.shotCount || "—")} />
          <Stat2 label="卡司数" value={String(pr.castSize || "—")} />
        </dl>
        <div className="panel rounded-2xl p-5 lg:col-span-1">
          <span className="label-tech">场景地点</span>
          <ul className="mt-3 space-y-2">
            {(pr.locations ?? []).map((l) => (
              <li key={l} className="flex items-center gap-2 font-sans text-sm text-bone-100">
                <span className="text-ember-500">▪</span> {l}
              </li>
            ))}
          </ul>
        </div>
        <div className="panel rounded-2xl p-5 lg:col-span-1">
          <span className="label-tech">家具道具 · Ashley</span>
          <ul className="mt-3 space-y-2">
            {(pr.furnitureProps ?? []).map((f) => (
              <li key={f} className="font-sans text-sm text-bone-200">
                {f}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  )
}

function Stat2({ label, value }: { label: string; value: string }) {
  return (
    <div className="panel rounded-xl p-4">
      <span className="label-tech">{label}</span>
      <p className="mt-1 font-sans text-sm leading-snug text-bone-50">{value || "—"}</p>
    </div>
  )
}

/* ---------- 08 Distribution ---------- */

function DistributionSection({ plan, editable, update }: SectionProps) {
  const d = plan.distribution
  function patch(field: keyof typeof d, v: string) {
    update({ distribution: { ...d, [field]: v } })
  }
  return (
    <section>
      {/* Distribution shares the production_distribution stage; the rerun control
          lives on the Production section to avoid two buttons for one stage. */}
      <SectionHead no="08" kicker="走向市场" title="分发" />
      <div className="panel rounded-2xl p-6">
        {editable ? (
          <>
            <span className="label-tech">CTA 文案</span>
            <EditArea value={d.ctaCopy} onChange={(v) => patch("ctaCopy", v)} className="mt-1" rows={2} />
          </>
        ) : (
          <p className="font-display text-2xl leading-snug text-ember-200">&ldquo;{d.ctaCopy}&rdquo;</p>
        )}
        {d.linkPlacement && (
          <p className="mt-3 font-sans text-sm text-bone-300">
            <span className="label-tech mr-2">挂链</span>
            {d.linkPlacement}
          </p>
        )}
        {d.hashtags?.length > 0 && (
          <div className="mt-5 flex flex-wrap gap-2">
            {d.hashtags.map((h) => (
              <span
                key={h}
                className="rounded-full bg-ink-700 px-3 py-1 font-mono text-xs text-bone-300 transition hover:bg-ember-500/15 hover:text-ember-200"
              >
                {h}
              </span>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
