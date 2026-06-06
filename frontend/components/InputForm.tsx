"use client"

import { useRef, useState } from "react"
import { Brief, BriefImage } from "@/lib/types"
import { PRESET_MATERIALS } from "@/lib/materials"
import { assist } from "@/lib/api"

interface Props {
  onSubmit: (brief: Brief) => void
  disabled?: boolean
  defaults?: Brief
}

const MAX_IMAGES = 3

// 一张已选参考图，记录来源(预设 id 或上传)、缩略图地址与展示标签。base64 在
// 提交时再按需获取，避免在选择阶段就把大字符串塞进 state。
interface SelectedImage {
  key: string
  src: string
  label: string
  preset: boolean
  file?: File
}

// 去掉 data URL 前缀，返回纯 base64。
function stripDataPrefix(dataUrl: string): string {
  const i = dataUrl.indexOf(",")
  return i >= 0 ? dataUrl.slice(i + 1) : dataUrl
}

function fileToImage(file: File): Promise<BriefImage> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () =>
      resolve({
        mimeType: file.type || "image/jpeg",
        data: stripDataPrefix(String(reader.result)),
        label: file.name,
      })
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

async function presetToImage(m: SelectedImage): Promise<BriefImage> {
  const res = await fetch(m.src)
  const blob = await res.blob()
  const dataUrl: string = await new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(blob)
  })
  return {
    mimeType: blob.type || "image/jpeg",
    data: stripDataPrefix(dataUrl),
    label: m.label,
  }
}

// DEFAULT_REQ 是初始/「套用模板」的整段需求骨架，融合了题材 + 爽点 + Ashley 植入。
const DEFAULT_REQ =
  "做一部「家装改造逆袭」题材的竖屏短剧，主打逆袭打脸爽点；重点植入 Ashley 客厅沙发、卧室套装。"

// SCAFFOLD 是「提示词助手」的可点击片段，按组排列。点一下就把 insert 子句追加到需求
// 文本框里(用户可继续手改)。insert 写成可拼接的小句，连点也能读通。
const SCAFFOLD: { group: string; chips: { zh: string; insert: string }[] }[] = [
  {
    group: "题材套路",
    chips: [
      { zh: "家装改造逆袭", insert: "题材：家装改造逆袭。" },
      { zh: "重生造梦想家", insert: "题材：重生之打造梦想之家。" },
      { zh: "离婚爆改出租屋", insert: "题材：离婚后爆改出租屋。" },
      { zh: "婆媳和解之家", insert: "题材：婆媳和解之家。" },
    ],
  },
  {
    group: "爽点引擎",
    chips: [
      { zh: "逆袭打脸", insert: "主打爽点：逆袭打脸。" },
      { zh: "阶层跃升", insert: "主打爽点：阶层跃升。" },
      { zh: "双向奔赴", insert: "主打爽点：双向奔赴。" },
      { zh: "扮猪吃虎", insert: "主打爽点：扮猪吃虎。" },
    ],
  },
  {
    group: "Ashley 植入",
    chips: [
      { zh: "客厅沙发", insert: "重点植入 Ashley 客厅沙发。" },
      { zh: "卧室套装", insert: "重点植入 Ashley 卧室套装。" },
      { zh: "餐桌", insert: "重点植入 Ashley 餐桌。" },
      { zh: "全屋定制", insert: "重点植入 Ashley 全屋定制。" },
    ],
  },
]

export default function InputForm({ onSubmit, disabled, defaults }: Props) {
  const [requirement, setRequirement] = useState(defaults?.requirement ?? DEFAULT_REQ)
  const [episodes, setEpisodes] = useState(defaults?.episodes ?? 5)
  const [episodeSecs, setEpisodeSecs] = useState(defaults?.episodeSecs ?? 30)
  const [assisting, setAssisting] = useState(false)
  const [assistErr, setAssistErr] = useState("")

  const [images, setImages] = useState<SelectedImage[]>([])
  const [imgNote, setImgNote] = useState("")
  const [converting, setConverting] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)

  function togglePreset(id: string) {
    setImgNote("")
    setImages((prev) => {
      const existing = prev.find((i) => i.key === `preset:${id}`)
      if (existing) return prev.filter((i) => i.key !== `preset:${id}`)
      if (prev.length >= MAX_IMAGES) {
        setImgNote(`最多选择 ${MAX_IMAGES} 张参考图`)
        return prev
      }
      const m = PRESET_MATERIALS.find((x) => x.id === id)!
      return [
        ...prev,
        { key: `preset:${id}`, src: m.src, label: m.label, preset: true },
      ]
    })
  }

  function addUploads(files: FileList | null) {
    if (!files || files.length === 0) return
    setImgNote("")
    setImages((prev) => {
      const next = [...prev]
      for (const f of Array.from(files)) {
        if (next.length >= MAX_IMAGES) {
          setImgNote(`最多选择 ${MAX_IMAGES} 张参考图，多余的已忽略`)
          break
        }
        next.push({
          key: `upload:${f.name}:${f.size}:${f.lastModified}`,
          src: URL.createObjectURL(f),
          label: f.name,
          preset: false,
          file: f,
        })
      }
      return next
    })
    if (fileInput.current) fileInput.current.value = ""
  }

  function removeImage(key: string) {
    setImgNote("")
    setImages((prev) => prev.filter((i) => i.key !== key))
  }

  // appendFragment 把助手片段追加到需求末尾(已有内容则换行衔接)。
  function appendFragment(insert: string) {
    setRequirement((prev) => {
      const base = prev.trimEnd()
      return base ? `${base}\n${insert}` : insert
    })
  }

  // buildImagePayload 把已选参考素材(预设/上传)转成 base64 的 BriefImage。submit
  // 与 AI 扩写共用,确保两条路都拿同一份素材。无素材时返回 undefined。
  async function buildImagePayload(): Promise<BriefImage[] | undefined> {
    if (images.length === 0) return undefined
    return Promise.all(
      images.map((i) => (i.preset ? presetToImage(i) : fileToImage(i.file!)))
    )
  }

  // runAssist 调后端 /api/assist，把当前(粗略)需求 + 已选参考素材一起喂给模型，
  // 扩写/优化成完整一段需求并回填文本框——让上传的图直接影响生成内容。
  async function runAssist() {
    if (assisting || disabled || converting) return
    setAssistErr("")
    setAssisting(true)
    try {
      const imgPayload = await buildImagePayload()
      const out = await assist(requirement, episodes, episodeSecs, imgPayload)
      if (out) setRequirement(out)
    } catch (e) {
      setAssistErr(e instanceof Error ? e.message : "AI 扩写失败")
    } finally {
      setAssisting(false)
    }
  }

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (disabled || converting || assisting) return
    let imgPayload: BriefImage[] | undefined
    if (images.length > 0) {
      setConverting(true)
      try {
        imgPayload = await buildImagePayload()
      } catch {
        setImgNote("参考图处理失败，请重试或移除后再生成")
        setConverting(false)
        return
      }
      setConverting(false)
    }
    onSubmit({ requirement, episodes, episodeSecs, images: imgPayload })
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

      <div className="mb-6">
        <span className="label-tech">场记板 · Scene 01</span>
        <h2 className="mt-1 font-display text-2xl font-semibold tracking-tight text-bone-50">
          给编剧室下需求
        </h2>
      </div>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-12">
        {/* Requirement — merged 题材/套路 + Ashley 品牌重点 into one prompt */}
        <div className="sm:col-span-12">
          <div className="mb-2 flex items-end justify-between gap-2">
            <label htmlFor="requirement" className="label-tech block">
              短剧生成需求
            </label>
            <span className="font-mono text-[10px] text-bone-400">
              一段话写清:题材 / 套路 · 爽点 · Ashley 植入重点
            </span>
          </div>
          <textarea
            id="requirement"
            rows={4}
            value={requirement}
            onChange={(e) => setRequirement(e.target.value)}
            disabled={disabled || assisting}
            className="w-full resize-y rounded-lg border border-bone-500/20 bg-ink-900/60 px-4 py-3 font-sans leading-relaxed text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-50"
            placeholder="例如:做一部「家装改造逆袭」题材的竖屏短剧，主打逆袭打脸爽点；重点植入 Ashley 客厅沙发、卧室套装。"
          />

          {/* prompt assistant */}
          <div className="mt-3 rounded-lg border border-bone-500/15 bg-ink-900/30 p-3">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <span className="label-tech text-bone-400">提示词助手</span>
              <button
                type="button"
                disabled={disabled || assisting}
                onClick={() => setRequirement(DEFAULT_REQ)}
                className="rounded-md border border-bone-500/20 px-2.5 py-1 font-mono text-[10px] tracking-wider text-bone-300 transition hover:border-bone-500/40 hover:text-bone-100 disabled:opacity-50"
              >
                套用模板
              </button>
              <button
                type="button"
                disabled={disabled || assisting}
                onClick={runAssist}
                className="ml-auto inline-flex items-center gap-1.5 rounded-md border border-ember-500/50 bg-ember-500/10 px-3 py-1 font-mono text-[10px] tracking-wider text-ember-200 transition hover:bg-ember-500/20 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {assisting ? (
                  <>
                    <span className="inline-block h-3 w-3 animate-spin rounded-full border-2 border-ember-300/40 border-t-ember-300" />
                    扩写中…
                  </>
                ) : (
                  <>✨ AI 扩写 / 优化</>
                )}
              </button>
            </div>

            <div className="space-y-1.5">
              {SCAFFOLD.map((row) => (
                <div key={row.group} className="flex flex-wrap items-center gap-1.5">
                  <span className="w-[68px] shrink-0 font-mono text-[10px] text-bone-500">
                    {row.group}
                  </span>
                  {row.chips.map((c) => (
                    <button
                      key={c.zh}
                      type="button"
                      disabled={disabled || assisting}
                      onClick={() => appendFragment(c.insert)}
                      className="rounded-full border border-bone-500/20 px-2.5 py-1 font-mono text-[10px] tracking-wider text-bone-400 transition hover:border-ember-500/50 hover:text-ember-200 disabled:opacity-50"
                    >
                      + {c.zh}
                    </button>
                  ))}
                </div>
              ))}
            </div>
            <p className="mt-2 font-mono text-[10px] text-bone-500">
              点标签即追加到需求里;「AI 扩写」与「生成方案」都会结合你的草稿与下方参考素材一起分析,可继续手改。
            </p>
            {assistErr && (
              <p className="mt-1 font-mono text-[10px] text-ember-300">{assistErr}</p>
            )}
          </div>

          {/* Reference materials — sits right under the requirement/assistant so it
              reads as input to 扩写/生成 above, not a detached field far below. */}
          <div className="mt-3 rounded-lg border border-bone-500/15 bg-ink-900/30 p-3">
            <div className="mb-2 flex items-end justify-between gap-2">
              <span className="label-tech text-bone-400">↑ 参考素材（可选 · 喂给上方扩写 / 生成）</span>
              <span className="font-mono text-[10px] text-bone-500">
                已选 {images.length} / {MAX_IMAGES} · 影响立意 / 植入 / 分镜
              </span>
            </div>

            {/* preset thumbnails */}
            <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
              {PRESET_MATERIALS.map((m) => {
                const active = images.some((i) => i.key === `preset:${m.id}`)
                return (
                  <button
                    key={m.id}
                    type="button"
                    disabled={disabled}
                    onClick={() => togglePreset(m.id)}
                    title={m.label}
                    className={`group relative aspect-[4/3] overflow-hidden rounded-lg border transition disabled:opacity-50 ${
                      active
                        ? "border-ember-500/80 ring-2 ring-ember-500/30"
                        : "border-bone-500/20 hover:border-bone-500/50"
                    }`}
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img
                      src={m.src}
                      alt={m.label}
                      className="h-full w-full object-cover transition group-hover:scale-105"
                    />
                    <span className="absolute inset-x-0 bottom-0 truncate bg-ink-900/70 px-1.5 py-0.5 text-[9px] text-bone-200">
                      {m.label}
                    </span>
                    {active && (
                      <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-ember-500 text-[10px] font-bold text-ink-900">
                        ✓
                      </span>
                    )}
                  </button>
                )
              })}
            </div>

            {/* upload */}
            <div className="mt-2 flex items-center gap-2">
              <button
                type="button"
                disabled={disabled || images.length >= MAX_IMAGES}
                onClick={() => fileInput.current?.click()}
                className="rounded-lg border border-bone-500/20 px-3 py-1.5 font-mono text-[10px] tracking-wider text-bone-300 transition hover:border-bone-500/40 hover:text-bone-100 disabled:cursor-not-allowed disabled:opacity-50"
              >
                + 上传图片
              </button>
              <input
                ref={fileInput}
                type="file"
                accept="image/*"
                multiple
                className="hidden"
                onChange={(e) => addUploads(e.target.files)}
              />
              {imgNote && (
                <span className="font-mono text-[10px] text-ember-300">{imgNote}</span>
              )}
            </div>

            {/* selected list */}
            {images.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-2">
                {images.map((i) => (
                  <div
                    key={i.key}
                    className="relative h-14 w-14 overflow-hidden rounded-md border border-bone-500/20"
                    title={i.label}
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={i.src} alt={i.label} className="h-full w-full object-cover" />
                    <button
                      type="button"
                      disabled={disabled}
                      onClick={() => removeImage(i.key)}
                      className="absolute right-0.5 top-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-ink-900/80 text-[10px] leading-none text-bone-100 transition hover:bg-ember-500 hover:text-ink-900 disabled:opacity-50"
                      aria-label={`移除 ${i.label}`}
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Episodes */}
        <div className="sm:col-span-4">
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
        <div className="sm:col-span-4">
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

        {/* Total runtime (derived from episodes × seconds) */}
        <div className="sm:col-span-4">
          <span className="label-tech mb-2 block">总时长</span>
          <div className="flex w-full items-center rounded-lg border border-bone-500/20 bg-ink-900/40 px-4 py-3 font-mono">
            <span className="text-lg leading-none text-ember-400">{runtime}</span>
            <span className="ml-1.5 text-xs text-bone-400">分钟</span>
          </div>
        </div>

        {/* Submit */}
        <div className="flex items-end sm:col-span-12">
          <button
            type="submit"
            disabled={disabled || converting || assisting}
            className="group relative w-full overflow-hidden rounded-lg bg-ember-500 px-6 py-3.5 font-sans text-sm font-semibold tracking-wide text-ink-900 transition hover:bg-ember-400 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <span className="relative z-10 flex items-center justify-center gap-2">
              {disabled || converting ? (
                <>
                  <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-ink-900/40 border-t-ink-900" />
                  {converting ? "处理参考图…" : "生成中…"}
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
