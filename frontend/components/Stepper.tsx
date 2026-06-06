"use client"

export type Step = 1 | 2 | 3 | 4

interface Props {
  current: Step
  onStep?: (step: Step) => void
}

const STEPS = [
  { n: 1 as Step, label: "填写需求", sub: "Brief" },
  { n: 2 as Step, label: "选立意", sub: "Direction" },
  { n: 3 as Step, label: "生成中", sub: "Pipeline" },
  { n: 4 as Step, label: "制作方案", sub: "Plan" },
]

export default function Stepper({ current, onStep }: Props) {
  return (
    <nav className="mb-8">
      <ol className="flex items-center">
        {STEPS.map((s, i) => {
          const status = s.n < current ? "done" : s.n === current ? "active" : "todo"
          const clickable = Boolean(onStep) && s.n < current
          return (
            <li key={s.n} className="flex flex-1 items-center last:flex-none">
              <button
                type="button"
                disabled={!clickable}
                onClick={() => clickable && onStep?.(s.n)}
                className={`flex items-center gap-3 ${clickable ? "cursor-pointer" : "cursor-default"}`}
              >
                <span
                  className={`flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full border font-mono text-sm transition ${
                    status === "active"
                      ? "border-ember-500 bg-ember-500 text-ink-900 shadow-[0_0_18px_rgba(228,132,47,0.45)]"
                      : status === "done"
                        ? "border-ember-500/60 bg-ember-500/15 text-ember-300"
                        : "border-bone-500/20 text-bone-400"
                  }`}
                >
                  {status === "done" ? "✓" : s.n.toString().padStart(2, "0")}
                </span>
                <span className="text-left">
                  <span
                    className={`block font-sans text-sm font-medium transition ${
                      status === "todo" ? "text-bone-400" : "text-bone-50"
                    }`}
                  >
                    {s.label}
                  </span>
                  <span className="label-tech">{s.sub}</span>
                </span>
              </button>
              {i < STEPS.length - 1 && (
                <span
                  aria-hidden
                  className={`mx-3 h-px flex-1 transition sm:mx-5 ${
                    s.n < current ? "bg-ember-500/40" : "bg-bone-500/15"
                  }`}
                />
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
