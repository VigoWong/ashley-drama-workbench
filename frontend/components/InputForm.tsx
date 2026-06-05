"use client"

import { useState } from "react"
import { Brief } from "@/lib/types"

interface Props {
  onSubmit: (brief: Brief) => void
  disabled?: boolean
  defaults?: Brief
}

// 中文显示标签 + 发送给后端的英文取值(产出仍面向美国市场)
const GENRE_PRESETS: { zh: string; value: string }[] = [
  { zh: "家装改造逆袭", value: "home makeover revenge" },
  { zh: "隐藏富豪翻修", value: "secret-heir renovation" },
  { zh: "离婚后重启人生", value: "fresh start after divorce" },
  { zh: "家庭和解", value: "family reconciliation" },
]

export default function InputForm({ onSubmit, disabled, defaults }: Props) {
  const [genre, setGenre] = useState(defaults?.genre ?? "home makeover revenge")
  const [episodes, setEpisodes] = useState(defaults?.episodes ?? 12)
  const [episodeSecs, setEpisodeSecs] = useState(defaults?.episodeSecs ?? 90)
  const [brandFocus, setBrandFocus] = useState(
    defaults?.brandFocus ?? "living room sofas, bedroom sets"
  )

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    onSubmit({ genre, episodes, episodeSecs, brandFocus })
  }

  const runtime = ((episodes * episodeSecs) / 60).toFixed(1)

  return (
    <form
      onSubmit={handleSubmit}
      className="panel relative overflow-hidden rounded-2xl p-6 sm:p-8"
    >
      {/* clapperboard stripe */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 h-2"
        style={{
          backgroundImage:
            "repeating-linear-gradient(115deg, var(--color-ink-700) 0 16px, var(--color-bone-100) 16px 32px)",
          opacity: 0.5,
        }}
      />

      <div className="mb-6 flex items-end justify-between gap-4">
        <div>
          <span className="label-tech">场记板 · Scene 01</span>
          <h2 className="mt-1 font-display text-2xl font-semibold tracking-tight text-bone-50">
            给编剧室下需求
          </h2>
        </div>
        <div className="hidden text-right sm:block">
          <span className="label-tech">总时长</span>
          <p className="font-mono text-lg text-ember-400">{runtime} 分钟</p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-12">
        {/* Genre */}
        <div className="sm:col-span-7">
          <label htmlFor="genre" className="label-tech mb-2 block">
            题材 / 套路
          </label>
          <input
            id="genre"
            type="text"
            value={genre}
            onChange={(e) => setGenre(e.target.value)}
            disabled={disabled}
            className="w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-4 py-3 font-sans text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-50"
            placeholder="例如:家装改造逆袭"
          />
          <div className="mt-2 flex flex-wrap gap-1.5">
            {GENRE_PRESETS.map((g) => (
              <button
                key={g.value}
                type="button"
                disabled={disabled}
                onClick={() => setGenre(g.value)}
                className={`rounded-full border px-2.5 py-1 font-mono text-[10px] tracking-wider transition disabled:opacity-50 ${
                  genre === g.value
                    ? "border-ember-500/60 bg-ember-500/15 text-ember-200"
                    : "border-bone-500/20 text-bone-500 hover:border-bone-500/40 hover:text-bone-300"
                }`}
              >
                {g.zh}
              </button>
            ))}
          </div>
        </div>

        {/* Brand focus */}
        <div className="sm:col-span-5">
          <label htmlFor="brand" className="label-tech mb-2 block">
            Ashley 品牌重点
          </label>
          <input
            id="brand"
            type="text"
            value={brandFocus}
            onChange={(e) => setBrandFocus(e.target.value)}
            disabled={disabled}
            className="w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-4 py-3 font-sans text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-50"
            placeholder="客厅沙发、卧室套装"
          />
          <p className="mt-2 font-mono text-[10px] text-bone-500">
            影响产品植入与家具道具清单。
          </p>
        </div>

        {/* Episodes */}
        <div className="sm:col-span-3">
          <label htmlFor="episodes" className="label-tech mb-2 block">
            集数
          </label>
          <input
            id="episodes"
            type="number"
            min={1}
            max={60}
            value={episodes}
            onChange={(e) => setEpisodes(Number(e.target.value))}
            disabled={disabled}
            className="w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-4 py-3 font-mono text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-50"
          />
        </div>

        {/* Episode seconds */}
        <div className="sm:col-span-3">
          <label htmlFor="secs" className="label-tech mb-2 block">
            单集秒数
          </label>
          <input
            id="secs"
            type="number"
            min={15}
            max={600}
            value={episodeSecs}
            onChange={(e) => setEpisodeSecs(Number(e.target.value))}
            disabled={disabled}
            className="w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-4 py-3 font-mono text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-50"
          />
        </div>

        {/* Submit */}
        <div className="flex items-end sm:col-span-6">
          <button
            type="submit"
            disabled={disabled}
            className="group relative w-full overflow-hidden rounded-lg bg-ember-500 px-6 py-3.5 font-sans text-sm font-semibold tracking-wide text-ink-900 transition hover:bg-ember-400 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <span className="relative z-10 flex items-center justify-center gap-2">
              {disabled ? (
                <>
                  <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-ink-900/40 border-t-ink-900" />
                  生成中…
                </>
              ) : (
                <>
                  <span className="font-mono text-base leading-none">▸</span>
                  Action · 生成方案
                </>
              )}
            </span>
          </button>
        </div>
      </div>
    </form>
  )
}
