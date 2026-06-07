// frontend/components/WizardView.tsx
"use client"
import { useState } from "react"
import { Brief, Concept, Plan, SSEvent } from "@/lib/types"
import { generate, propose, refine, UnauthorizedError } from "@/lib/api"
import InputForm from "@/components/InputForm"
import ConceptChoice from "@/components/ConceptChoice"
import StageTimeline from "@/components/StageTimeline"
import PlanView from "@/components/PlanView"
import ExportBar from "@/components/ExportBar"
import Stepper, { Step } from "@/components/Stepper"

const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"
const STAGES = ["concept", "bible", "characters", "episodes", "placements", "hero", "production_distribution", "visuals"]

// WizardView is the original four-step deterministic flow (填需求 → 选立意 → 生成 →
// 方案), extracted verbatim from page.tsx so the page can host a mode switch.
export default function WizardView({ onUnauthorized }: { onUnauthorized: () => void }) {
  const [step, setStep] = useState<Step>(1)
  const [events, setEvents] = useState<SSEvent[]>([])
  const [plan, setPlan] = useState<Plan | null>(null)
  const [failed, setFailed] = useState<string | null>(null)
  const [lastBrief, setLastBrief] = useState<Brief | undefined>(undefined)
  const [concepts, setConcepts] = useState<Concept[]>([])
  const [proposing, setProposing] = useState(false)
  const [timelineStages, setTimelineStages] = useState<string[]>(STAGES)

  async function run(brief: Brief) {
    setLastBrief(brief); setEvents([]); setPlan(null); setConcepts([])
    setTimelineStages(STAGES.slice(1)); setFailed(null); setProposing(true); setStep(2)
    try {
      setConcepts(await propose(brief))
    } catch (err) {
      if (err instanceof UnauthorizedError) { onUnauthorized(); return }
      setFailed(err instanceof Error ? `${err.message} — 后端是否在 ${API} 运行?` : "立意提案失败。")
    } finally { setProposing(false) }
  }

  async function confirmConcept(chosen: Concept) {
    if (!lastBrief) return
    setEvents([]); setPlan(null); setTimelineStages(STAGES.slice(1)); setFailed(null); setStep(3)
    try {
      await generate(lastBrief, (e) => {
        setEvents((prev) => [...prev, e])
        if (e.type === "complete" && e.plan) { setPlan(e.plan); setStep(4) }
      }, chosen)
    } catch (err) {
      if (err instanceof UnauthorizedError) { onUnauthorized(); return }
      setFailed(err instanceof Error ? `${err.message} — 后端是否在 ${API} 运行?` : "生成失败。")
    }
  }

  async function onRefine(fromStage: string, only: boolean, note: string) {
    if (!plan) return
    const idx = STAGES.indexOf(fromStage)
    const subset = only ? [fromStage] : STAGES.slice(idx >= 0 ? idx : 0)
    setTimelineStages(subset); setEvents([]); setFailed(null); setStep(3)
    try {
      await refine({ plan, fromStage, only, note }, (e) => {
        setEvents((prev) => [...prev, e])
        if (e.type === "complete" && e.plan) { setPlan(e.plan); setStep(4) }
      })
    } catch (err) {
      if (err instanceof UnauthorizedError) { onUnauthorized(); return }
      setFailed(err instanceof Error ? `${err.message} — 后端是否在 ${API} 运行?` : "重跑失败。")
    }
  }

  function restart() {
    setStep(1); setEvents([]); setPlan(null); setConcepts([]); setTimelineStages(STAGES); setFailed(null)
  }

  return (
    <>
      <Stepper current={step} onStep={(s) => s === 1 && restart()} />
      {step === 1 && <InputForm onSubmit={run} disabled={proposing} defaults={lastBrief} />}
      {step === 2 && (
        <div className="space-y-6">
          {proposing && (
            <div className="panel rounded-2xl p-10 text-center">
              <span className="mx-auto mb-4 block h-6 w-6 animate-spin rounded-full border-2 border-ember-500/30 border-t-ember-400" />
              <p className="font-mono text-sm text-bone-300">正在生成立意方向…</p>
            </div>
          )}
          {!proposing && failed && (
            <div className="rounded-xl border border-signal-stop/40 bg-signal-stop/10 p-4">
              <p className="font-mono text-sm text-signal-stop">✕ {failed}</p>
              <button onClick={restart} className="mt-3 rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700">← 返回修改需求</button>
            </div>
          )}
          {!proposing && !failed && concepts.length > 0 && (
            <ConceptChoice concepts={concepts} onConfirm={confirmConcept} onBack={restart} />
          )}
        </div>
      )}
      {step === 3 && (
        <div className="space-y-6">
          <StageTimeline stages={timelineStages} events={events} />
          {failed && (
            <div className="rounded-xl border border-signal-stop/40 bg-signal-stop/10 p-4">
              <p className="font-mono text-sm text-signal-stop">✕ {failed}</p>
              <button onClick={restart} className="mt-3 rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700">← 返回修改需求</button>
            </div>
          )}
        </div>
      )}
      {step === 4 && plan && (
        <div className="space-y-8">
          <div className="flex items-center justify-between gap-3">
            <p className="font-mono text-xs text-bone-400">方案已生成 · 共 {plan.episodes?.length ?? 0} 集</p>
            <button onClick={restart} className="rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700">＋ 新方案</button>
          </div>
          <ExportBar plan={plan} />
          <PlanView plan={plan} onChange={setPlan} onRefine={onRefine} />
        </div>
      )}
    </>
  )
}
