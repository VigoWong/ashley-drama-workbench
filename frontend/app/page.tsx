"use client"

import { useEffect, useState } from "react"
import { Brief, Plan, SSEvent } from "@/lib/types"
import { generate, refine, UnauthorizedError } from "@/lib/api"
import { getToken, clearToken } from "@/lib/auth"
import InputForm from "@/components/InputForm"
import StageTimeline from "@/components/StageTimeline"
import PlanView from "@/components/PlanView"
import ExportBar from "@/components/ExportBar"
import Stepper, { Step } from "@/components/Stepper"
import LoginForm from "@/components/LoginForm"
import HistoryView from "@/components/HistoryView"

type View = "workbench" | "history"

const STAGES = [
  "concept",
  "bible",
  "characters",
  "episodes",
  "placements",
  "hero",
  "production_distribution",
  "visuals",
]

const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"

export default function Home() {
  const [ready, setReady] = useState(false)
  const [authed, setAuthed] = useState(false)
  const [view, setView] = useState<View>("workbench")
  const [step, setStep] = useState<Step>(1)
  const [events, setEvents] = useState<SSEvent[]>([])
  const [plan, setPlan] = useState<Plan | null>(null)
  const [running, setRunning] = useState(false)
  const [failed, setFailed] = useState<string | null>(null)
  const [lastBrief, setLastBrief] = useState<Brief | undefined>(undefined)
  // editing: whether step-3 fields are editable (draft mode).
  const [editing, setEditing] = useState(false)
  // timelineStages: which stages the StageTimeline shows. Full set for a fresh
  // generate; only the reran subset during a refine.
  const [timelineStages, setTimelineStages] = useState<string[]>(STAGES)

  useEffect(() => {
    setAuthed(Boolean(getToken()))
    setReady(true)
  }, [])

  function logout() {
    clearToken()
    setAuthed(false)
    setView("workbench")
    setStep(1)
    setEvents([])
    setPlan(null)
    setFailed(null)
  }

  function handleUnauthorized() {
    clearToken()
    setAuthed(false)
    setView("workbench")
  }

  async function run(brief: Brief) {
    setLastBrief(brief)
    setEvents([])
    setPlan(null)
    setEditing(false)
    setTimelineStages(STAGES)
    setFailed(null)
    setRunning(true)
    setStep(2)
    try {
      await generate(brief, (e) => {
        setEvents((prev) => [...prev, e])
        if (e.type === "complete" && e.plan) {
          setPlan(e.plan)
          setStep(3)
        }
      })
    } catch (err) {
      if (err instanceof UnauthorizedError) {
        clearToken()
        setAuthed(false)
        return
      }
      setFailed(
        err instanceof Error
          ? `${err.message} — 后端是否在 ${API} 运行?`
          : "生成失败。"
      )
    } finally {
      setRunning(false)
    }
  }

  // onRefine reruns part of the pipeline against the current (possibly edited)
  // draft plan. It reuses the step-2 timeline, showing only the stages that will
  // actually run, then folds the returned plan back into the draft.
  async function onRefine(fromStage: string, only: boolean, note: string) {
    if (!plan) return
    const idx = STAGES.indexOf(fromStage)
    const subset = only ? [fromStage] : STAGES.slice(idx >= 0 ? idx : 0)
    setTimelineStages(subset)
    setEvents([])
    setFailed(null)
    setEditing(false)
    setRunning(true)
    setStep(2)
    try {
      await refine({ plan, fromStage, only, note }, (e) => {
        setEvents((prev) => [...prev, e])
        if (e.type === "complete" && e.plan) {
          setPlan(e.plan)
          setStep(3)
        }
      })
    } catch (err) {
      if (err instanceof UnauthorizedError) {
        handleUnauthorized()
        return
      }
      setFailed(
        err instanceof Error ? `${err.message} — 后端是否在 ${API} 运行?` : "重跑失败。"
      )
    } finally {
      setRunning(false)
    }
  }

  function restart() {
    setStep(1)
    setEvents([])
    setPlan(null)
    setEditing(false)
    setTimelineStages(STAGES)
    setFailed(null)
  }

  // Avoid SSR/first-paint flash before we know auth state.
  if (!ready) return null
  if (!authed) return <LoginForm onAuthed={() => setAuthed(true)} />

  return (
    <div className="relative min-h-screen">
      <main className="mx-auto max-w-6xl px-5 py-10 sm:px-8 sm:py-16">
        {/* ---- Masthead ---- */}
        <header className="mb-10 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="mb-4 flex items-center gap-3">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src="/ashley-logo.png"
                alt="Ashley Furniture Industries"
                className="h-8 w-auto rounded-md bg-bone-50 px-2.5 py-1.5 shadow-sm ring-1 ring-black/5"
              />
              <span className="inline-flex items-center gap-2 rounded-full border border-bone-500/20 bg-ink-800/60 px-3 py-1">
                <span className="h-1.5 w-1.5 rounded-full bg-ember-400" />
                <span className="label-tech">品牌内容工作台</span>
              </span>
            </div>
            <h1 className="font-display text-4xl font-semibold leading-[0.95] tracking-tight sm:text-5xl">
              短剧
              <br />
              <span className="text-ember-400">生产工作台</span>
            </h1>
            <p className="mt-4 max-w-md font-sans text-sm leading-relaxed text-bone-300">
              一句需求,即可生成面向国内市场、为 Ashley 家具带货的竖屏短剧 ——
              从立意到通告单,由多 Agent 流水线自动产出。
            </p>
          </div>
          <div className="flex items-center gap-4 sm:flex-col sm:items-end">
            <div className="hidden text-right sm:block">
              <span className="label-tech">画幅</span>
              <div className="ml-auto mt-1 flex h-20 w-12 items-center justify-center rounded-md border border-bone-500/25 bg-ink-800">
                <span className="font-mono text-[10px] text-bone-500">9:16</span>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setView("history")}
                className="rounded-lg border border-bone-500/25 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-bone-300 transition hover:border-ember-400/50 hover:text-ember-400"
              >
                历史
              </button>
              <button
                onClick={logout}
                className="rounded-lg border border-bone-500/25 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-bone-300 transition hover:border-bone-500/50 hover:text-bone-100"
              >
                退出登录
              </button>
            </div>
          </div>
        </header>

        {view === "history" ? (
          <HistoryView
            onBack={() => setView("workbench")}
            onUnauthorized={handleUnauthorized}
          />
        ) : (
          <>
        {/* ---- Stepper ---- */}
        <Stepper current={step} onStep={(s) => s === 1 && restart()} />

        {/* ---- Step 1 · Brief ---- */}
        {step === 1 && (
          <InputForm onSubmit={run} disabled={running} defaults={lastBrief} />
        )}

        {/* ---- Step 2 · Pipeline ---- */}
        {step === 2 && (
          <div className="space-y-6">
            <StageTimeline stages={timelineStages} events={events} />
            {failed && (
              <div className="rounded-xl border border-signal-stop/40 bg-signal-stop/10 p-4">
                <p className="font-mono text-sm text-signal-stop">✕ {failed}</p>
                <button
                  onClick={restart}
                  className="mt-3 rounded-lg border border-bone-500/25 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700"
                >
                  ← 返回修改需求
                </button>
              </div>
            )}
          </div>
        )}

        {/* ---- Step 3 · Plan ---- */}
        {step === 3 && plan && (
          <div className="space-y-8">
            <div className="flex items-center justify-between gap-3">
              <p className="font-mono text-xs text-bone-500">
                方案已生成 · 共 {plan.episodes?.length ?? 0} 集
                {editing && <span className="ml-2 text-ember-400">· 编辑中（改动仅本地草稿）</span>}
              </p>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setEditing((v) => !v)}
                  className={`rounded-lg border px-4 py-2 font-mono text-xs uppercase tracking-wider transition ${
                    editing
                      ? "border-ember-500/60 bg-ember-500/15 text-ember-200"
                      : "border-bone-500/25 bg-ink-800 text-bone-100 hover:border-ember-400/50 hover:text-ember-400"
                  }`}
                >
                  {editing ? "✓ 完成" : "✎ 编辑"}
                </button>
                <button
                  onClick={restart}
                  className="rounded-lg border border-bone-500/25 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700"
                >
                  ＋ 新方案
                </button>
              </div>
            </div>
            <ExportBar plan={plan} />
            <PlanView
              plan={plan}
              editable={editing}
              onChange={setPlan}
              onRefine={onRefine}
            />
          </div>
        )}
          </>
        )}
      </main>

      <footer className="border-t border-bone-500/10 py-6 text-center">
        <span className="label-tech">
          无 Key 演示模式生成完整示例方案 · 配置 GEMINI_API_KEY 启用真实生成
        </span>
      </footer>
    </div>
  )
}
