// frontend/components/chat/PlanCanvas.tsx
"use client"
import { Plan } from "@/lib/types"
import { BLOCK_ORDER, BlockKey, BlockStatus } from "@/lib/chatState"
import PlanView from "@/components/PlanView"
import ExportBar from "@/components/ExportBar"

const LABEL: Record<BlockKey, string> = {
  concept: "立意", bible: "圣经", characters: "人物", episodes: "分集",
  placements: "植入", hero: "英雄场景", production_distribution: "制作分发", visuals: "概念图",
}

interface Props {
  plan: Plan | null
  blocks: Record<BlockKey, BlockStatus>
  onChange: (p: Plan) => void
  onRefine: (fromStage: string, only: boolean, note: string) => void
}

// PlanCanvas is the live "document" pane: a build-status rail (pending → writing →
// done) over the reused PlanView. As the agent finishes each block the status chip
// flips and (once a plan exists) PlanView shows the content; ExportBar appears when
// there's something to export.
export default function PlanCanvas({ plan, blocks, onChange, onRefine }: Props) {
  const doneCount = BLOCK_ORDER.filter((b) => blocks[b] === "done").length
  return (
    <div className="flex h-full flex-col">
      <div className="mb-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="label-tech">方案画布 · 实时</span>
          <span className="font-mono text-xs text-ember-400">{doneCount.toString().padStart(2, "0")}<span className="text-bone-400">/08</span></span>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {BLOCK_ORDER.map((b) => <StatusChip key={b} label={LABEL[b]} status={blocks[b]} />)}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto pr-1">
        {plan ? (
          <div className="space-y-6">
            {doneCount > 0 && <ExportBar plan={plan} />}
            <PlanView plan={plan} onChange={onChange} onRefine={onRefine} />
          </div>
        ) : (
          <div className="flex h-full items-center justify-center">
            <p className="font-sans text-sm text-bone-400">方案会在这里随 agent 一步步生长出来。</p>
          </div>
        )}
      </div>
    </div>
  )
}

function StatusChip({ label, status }: { label: string; status: BlockStatus }) {
  const cls =
    status === "done" ? "border-signal-go/50 bg-signal-go/10 text-bone-100"
    : status === "writing" ? "border-ember-400/70 bg-ember-400/10 text-ember-300 animate-pulse"
    : status === "error" ? "border-signal-stop/50 bg-signal-stop/10 text-signal-stop"
    : "border-bone-500/20 text-bone-500"
  const mark = status === "done" ? "✓" : status === "writing" ? "✍" : status === "error" ? "✕" : "·"
  return (
    <span className={`rounded-md border px-2 py-1 font-mono text-[10px] ${cls}`}>{mark} {label}</span>
  )
}
