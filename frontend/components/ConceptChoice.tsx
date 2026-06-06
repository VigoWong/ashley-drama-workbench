"use client"

import { useState } from "react"
import { Concept } from "@/lib/types"

interface Props {
  concepts: Concept[]
  onConfirm: (chosen: Concept) => void
  onBack: () => void
}

// shared field styling, matching PlanView / InputForm.
const FIELD =
  "w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-3 py-2 font-sans text-sm text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20"

// ConceptChoice renders the 2-3 candidate 立意方向 as selectable cards. The user
// picks one (single-select, highlighted), may tweak its key fields inline, then
// confirms — sending the (possibly edited) concept upward to drive generation.
export default function ConceptChoice({ concepts, onConfirm, onBack }: Props) {
  const [selected, setSelected] = useState(0)
  // edits holds an editable copy of each concept so tweaking one direction never
  // mutates the originals (the user can switch selection without losing prior text
  // — but only the chosen one's edits are submitted).
  const [edits, setEdits] = useState<Concept[]>(() =>
    concepts.map((c) => ({ ...c }))
  )

  function patch(i: number, p: Partial<Concept>) {
    setEdits((prev) => prev.map((c, j) => (j === i ? { ...c, ...p } : c)))
  }

  if (concepts.length === 0) return null
  const chosen = edits[selected]

  return (
    <section className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <span className="label-tech">第 2 步 · 立意方向</span>
          <h2 className="mt-1 font-display text-2xl font-semibold tracking-tight">
            选择一个立意方向
          </h2>
          <p className="mt-2 max-w-xl font-sans text-sm leading-relaxed text-bone-300">
            AI 围绕你的需求给出 {concepts.length} 个显著不同的方向。选定一个（可就地微调），
            再据此生成完整制作方案。
          </p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {concepts.map((c, i) => {
          const active = i === selected
          return (
            <button
              key={i}
              type="button"
              onClick={() => setSelected(i)}
              className={`panel flex flex-col gap-3 rounded-2xl p-5 text-left transition ${
                active
                  ? "border-ember-500/70 ring-2 ring-ember-500/30 shadow-[0_0_24px_rgba(228,132,47,0.25)]"
                  : "border-bone-500/20 hover:border-ember-400/40"
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-mono text-xs text-ember-500/80">
                  方向 {String(i + 1).padStart(2, "0")}
                </span>
                <span
                  className={`flex h-5 w-5 items-center justify-center rounded-full border text-[10px] ${
                    active
                      ? "border-ember-500 bg-ember-500 text-ink-900"
                      : "border-bone-500/30 text-transparent"
                  }`}
                >
                  ✓
                </span>
              </div>
              <p className="font-display text-base font-medium leading-snug text-bone-50">
                {c.logline}
              </p>
              <div className="mt-auto space-y-1.5">
                <Tag label="爽点引擎" value={c.payoffEngine} />
                <Tag label="基调" value={c.tone} />
                <Tag label="主题" value={c.theme} />
              </div>
            </button>
          )
        })}
      </div>

      {/* inline tweak of the chosen direction */}
      <div className="panel rounded-2xl p-6">
        <div className="mb-4 flex items-center gap-2">
          <span className="label-tech">微调所选方向</span>
          <span className="font-mono text-[10px] text-bone-400">
            · 方向 {String(selected + 1).padStart(2, "0")}
          </span>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="一句话梗概 (logline)" className="sm:col-span-2">
            <textarea
              rows={2}
              className={FIELD}
              value={chosen.logline}
              onChange={(e) => patch(selected, { logline: e.target.value })}
            />
          </Field>
          <Field label="爽点引擎 (payoffEngine)">
            <textarea
              rows={2}
              className={FIELD}
              value={chosen.payoffEngine}
              onChange={(e) => patch(selected, { payoffEngine: e.target.value })}
            />
          </Field>
          <Field label="核心冲突 (coreConflict)">
            <textarea
              rows={2}
              className={FIELD}
              value={chosen.coreConflict}
              onChange={(e) => patch(selected, { coreConflict: e.target.value })}
            />
          </Field>
          <Field label="主题 (theme)">
            <input
              className={FIELD}
              value={chosen.theme}
              onChange={(e) => patch(selected, { theme: e.target.value })}
            />
          </Field>
          <Field label="基调 (tone)">
            <input
              className={FIELD}
              value={chosen.tone}
              onChange={(e) => patch(selected, { tone: e.target.value })}
            />
          </Field>
        </div>
      </div>

      <div className="flex items-center justify-between gap-3">
        <button
          onClick={onBack}
          className="rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700"
        >
          ← 返回修改需求
        </button>
        <button
          onClick={() => onConfirm(chosen)}
          className="rounded-lg bg-ember-500 px-5 py-2.5 font-mono text-xs font-semibold uppercase tracking-wider text-ink-900 shadow-[0_0_18px_rgba(228,132,47,0.45)] transition hover:bg-ember-400"
        >
          确认并生成 →
        </button>
      </div>
    </section>
  )
}

function Tag({ label, value }: { label: string; value: string }) {
  if (!value) return null
  return (
    <p className="font-mono text-[11px] leading-relaxed text-bone-300">
      <span className="text-ember-400/80">{label}:</span> {value}
    </p>
  )
}

function Field({
  label,
  className = "",
  children,
}: {
  label: string
  className?: string
  children: React.ReactNode
}) {
  return (
    <label className={`block ${className}`}>
      <span className="label-tech mb-1.5 block">{label}</span>
      {children}
    </label>
  )
}
