"use client"

import { useState } from "react"
import { Plan } from "@/lib/types"

interface Props {
  plan: Plan
}

export default function PlanView({ plan }: Props) {
  return (
    <div className="space-y-10">
      <Masthead plan={plan} />
      <ConceptSection plan={plan} />
      <BibleSection plan={plan} />
      <CharactersSection plan={plan} />
      <EpisodesSection plan={plan} />
      <PlacementsSection plan={plan} />
      <HeroSection plan={plan} />
      <ProductionSection plan={plan} />
      <DistributionSection plan={plan} />
    </div>
  )
}

/* ---------- section chrome ---------- */

function SectionHead({ no, kicker, title }: { no: string; kicker: string; title: string }) {
  return (
    <div className="mb-5 flex items-baseline gap-4 border-b border-bone-500/12 pb-3">
      <span className="font-mono text-xs text-ember-500/70">{no}</span>
      <div>
        <span className="label-tech">{kicker}</span>
        <h2 className="font-display text-2xl font-semibold tracking-tight">{title}</h2>
      </div>
    </div>
  )
}

/* ---------- 00 Masthead ---------- */

function Masthead({ plan }: Props) {
  const { bible, concept, brief } = plan
  return (
    <header className="panel relative overflow-hidden rounded-2xl p-8 sm:p-10">
      <div
        aria-hidden
        className="pointer-events-none absolute -right-10 -top-16 h-64 w-64 rounded-full"
        style={{ background: "radial-gradient(circle, rgba(228,132,47,0.22), transparent 70%)" }}
      />
      <span className="label-tech">The Series · 9:16 Vertical</span>
      <h1 className="mt-2 max-w-3xl font-display text-4xl font-semibold leading-[1.05] tracking-tight sm:text-6xl">
        {bible.title || "Untitled Series"}
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
        <Stat label="Episodes" value={String(bible.episodes || brief.episodes)} />
        <Stat label="Secs / Ep" value={`${bible.episodeSecs || brief.episodeSecs}s`} />
        <Stat label="Platform" value={bible.platform || "ReelShort"} />
        <Stat label="Market" value={brief.market || "US · English"} />
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

/* ---------- 01 Concept ---------- */

function ConceptSection({ plan }: Props) {
  const c = plan.concept
  const rows: [string, string][] = [
    ["Theme", c.theme],
    ["Audience", c.audience],
    ["Tone", c.tone],
    ["Core conflict", c.coreConflict],
  ]
  return (
    <section>
      <SectionHead no="01" kicker="Creative North Star" title="Concept" />
      <div className="grid gap-5 lg:grid-cols-3">
        <div className="panel rounded-2xl p-6 lg:col-span-1">
          <span className="label-tech">Payoff Engine · 爽点</span>
          <p className="mt-2 font-display text-xl leading-snug text-ember-200">
            {c.payoffEngine}
          </p>
          <p className="mt-3 font-sans text-sm text-bone-300">
            The repeatable satisfaction mechanism that keeps viewers binging.
          </p>
        </div>
        <dl className="grid gap-3 lg:col-span-2 sm:grid-cols-2">
          {rows.map(([k, v]) => (
            <div key={k} className="panel rounded-xl p-4">
              <dt className="label-tech">{k}</dt>
              <dd className="mt-1 font-sans text-sm leading-relaxed text-bone-100">{v}</dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  )
}

/* ---------- 02 Series Bible ---------- */

function BibleSection({ plan }: Props) {
  const b = plan.bible
  return (
    <section>
      <SectionHead no="02" kicker="Format Contract" title="Series Bible" />
      <div className="panel rounded-2xl p-6">
        <p className="font-sans text-base leading-relaxed text-bone-100">
          {b.integrationThesis}
        </p>
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

function CharactersSection({ plan }: Props) {
  return (
    <section>
      <SectionHead no="03" kicker="The Cast" title="Characters" />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {plan.characters.map((c) => (
          <article key={c.name} className="panel group flex flex-col rounded-2xl p-5">
            <div className="flex items-start justify-between gap-2">
              <h3 className="font-display text-xl font-semibold tracking-tight">{c.name}</h3>
              <span
                className={`rounded-full border px-2 py-0.5 font-mono text-[9px] uppercase tracking-wider ${
                  ROLE_TINT[c.role?.toLowerCase()] ?? "border-bone-500/30 text-bone-500"
                }`}
              >
                {c.role}
              </span>
            </div>
            <p className="mt-2 font-sans text-sm leading-relaxed text-bone-300">{c.bio}</p>
            {c.arc && (
              <p className="mt-3 border-l-2 border-ember-500/40 pl-3 font-sans text-sm italic text-bone-100">
                {c.arc}
              </p>
            )}
            {c.relationships && (
              <p className="mt-auto pt-3 font-mono text-[10px] uppercase tracking-wider text-bone-500">
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

function EpisodesSection({ plan }: Props) {
  const [openEp, setOpenEp] = useState<number | null>(plan.episodes[0]?.number ?? null)
  return (
    <section>
      <SectionHead no="04" kicker="Beat Sheet" title="Episodes" />
      <div className="space-y-2">
        {plan.episodes.map((e) => {
          const isOpen = openEp === e.number
          return (
            <div
              key={e.number}
              className={`panel overflow-hidden rounded-xl transition ${
                isOpen ? "border-ember-500/30" : ""
              }`}
            >
              <button
                onClick={() => setOpenEp(isOpen ? null : e.number)}
                className="flex w-full items-center gap-4 px-5 py-4 text-left transition hover:bg-ink-700/40"
              >
                <span className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-ink-700 font-mono text-sm text-ember-400">
                  {e.number.toString().padStart(2, "0")}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-sans text-sm font-medium text-bone-50">
                    {e.title}
                  </span>
                  <span className="block truncate font-mono text-[10px] text-bone-500">
                    {e.hook}
                  </span>
                </span>
                <span className={`font-mono text-xs text-bone-500 transition ${isOpen ? "rotate-90" : ""}`}>
                  ▸
                </span>
              </button>
              {isOpen && (
                <div className="space-y-4 border-t border-bone-500/10 px-5 py-4">
                  {e.synopsis && (
                    <p className="font-sans text-sm leading-relaxed text-bone-200">{e.synopsis}</p>
                  )}
                  <div className="grid gap-3 sm:grid-cols-3">
                    <Beat tone="open" label="Hook · golden 3s" value={e.hook} />
                    <Beat tone="payoff" label="Payoff · 爽点" value={e.payoff} />
                    <Beat tone="cliff" label="Cliffhanger" value={e.cliffhanger} />
                  </div>
                  {e.beats?.length > 0 && (
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="label-tech">Beats</span>
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

function Beat({ tone, label, value }: { tone: "open" | "payoff" | "cliff"; label: string; value: string }) {
  const tint = {
    open: "border-bone-300/30",
    payoff: "border-ember-500/40",
    cliff: "border-signal-stop/40",
  }[tone]
  if (!value) return null
  return (
    <div className={`rounded-lg border ${tint} bg-ink-900/50 p-3`}>
      <span className="label-tech">{label}</span>
      <p className="mt-1 font-sans text-sm leading-snug text-bone-100">{value}</p>
    </div>
  )
}

/* ---------- 05 Placements ---------- */

function PlacementsSection({ plan }: Props) {
  return (
    <section>
      <SectionHead no="05" kicker="Branded Integration" title="Ashley Placements" />
      <div className="panel overflow-hidden rounded-2xl">
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-bone-500/15">
                {["Ep", "Product", "Scene", "Emotional Beat", "CTA Timing"].map((h) => (
                  <th key={h} className="label-tech px-4 py-3 font-normal">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {plan.placements.map((p, i) => (
                <tr
                  key={i}
                  className="border-b border-bone-500/8 transition last:border-0 hover:bg-ink-700/30"
                >
                  <td className="px-4 py-3 font-mono text-sm text-ember-400">
                    {p.episode.toString().padStart(2, "0")}
                  </td>
                  <td className="px-4 py-3">
                    <span className="block font-mono text-[11px] text-bone-500">{p.productSku}</span>
                    <span className="block font-sans text-sm capitalize text-bone-100">{p.category}</span>
                  </td>
                  <td className="max-w-xs px-4 py-3 font-sans text-sm text-bone-300">{p.scene}</td>
                  <td className="max-w-xs px-4 py-3 font-sans text-sm text-bone-200">{p.emotionalBeat}</td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-bone-300">{p.ctaTiming}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}

/* ---------- 06 Hero scenes ---------- */

function HeroSection({ plan }: Props) {
  return (
    <section>
      <SectionHead no="06" kicker="Showcase Shot Lists" title="Hero Scenes" />
      <div className="grid gap-5 lg:grid-cols-2">
        {plan.heroScenes.map((h, i) => (
          <article key={i} className="panel overflow-hidden rounded-2xl">
            <div className="flex items-center justify-between border-b border-bone-500/12 px-5 py-3">
              <h3 className="font-display text-lg font-semibold tracking-tight">{h.title}</h3>
              <span className="rounded bg-ink-700 px-2 py-0.5 font-mono text-[10px] text-ember-400">
                EP {h.episode.toString().padStart(2, "0")}
              </span>
            </div>
            <ol className="divide-y divide-bone-500/8">
              {h.shots.map((s) => (
                <li key={s.number} className="flex gap-4 px-5 py-3">
                  <span className="font-mono text-xs text-bone-500">
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

/* ---------- 07 Production ---------- */

function ProductionSection({ plan }: Props) {
  const pr = plan.production
  return (
    <section>
      <SectionHead no="07" kicker="On The Floor" title="Production" />
      <div className="grid gap-5 lg:grid-cols-3">
        <dl className="grid grid-cols-2 gap-3 lg:col-span-1">
          <Stat2 label="Format" value={pr.format} />
          <Stat2 label="Budget" value={pr.budgetTier} />
          <Stat2 label="Shots" value={String(pr.shotCount || "—")} />
          <Stat2 label="Cast" value={String(pr.castSize || "—")} />
        </dl>
        <div className="panel rounded-2xl p-5 lg:col-span-1">
          <span className="label-tech">Locations</span>
          <ul className="mt-3 space-y-2">
            {(pr.locations ?? []).map((l) => (
              <li key={l} className="flex items-center gap-2 font-sans text-sm text-bone-100">
                <span className="text-ember-500">▪</span> {l}
              </li>
            ))}
          </ul>
        </div>
        <div className="panel rounded-2xl p-5 lg:col-span-1">
          <span className="label-tech">Furniture Props · Ashley</span>
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

function DistributionSection({ plan }: Props) {
  const d = plan.distribution
  return (
    <section>
      <SectionHead no="08" kicker="Go To Market" title="Distribution" />
      <div className="panel rounded-2xl p-6">
        <p className="font-display text-2xl leading-snug text-ember-200">
          &ldquo;{d.ctaCopy}&rdquo;
        </p>
        {d.linkPlacement && (
          <p className="mt-3 font-sans text-sm text-bone-300">
            <span className="label-tech mr-2">Link</span>
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
