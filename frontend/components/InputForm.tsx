"use client"

import { useRef, useState } from "react"
import { Brief, BriefImage } from "@/lib/types"
import { PRESET_MATERIALS } from "@/lib/materials"

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

// 国内市场中文题材预设(直接作为发往后端的题材输入)
const GENRE_PRESETS: { zh: string; value: string }[] = [
  { zh: "家装改造逆袭", value: "家装改造逆袭" },
  { zh: "重生之打造梦想之家", value: "重生之打造梦想之家" },
  { zh: "离婚后爆改出租屋", value: "离婚后爆改出租屋" },
  { zh: "婆媳和解之家", value: "婆媳和解之家" },
]

export default function InputForm({ onSubmit, disabled, defaults }: Props) {
  const [genre, setGenre] = useState(defaults?.genre ?? "家装改造逆袭")
  const [episodes, setEpisodes] = useState(defaults?.episodes ?? 5)
  const [episodeSecs, setEpisodeSecs] = useState(defaults?.episodeSecs ?? 30)
  const [brandFocus, setBrandFocus] = useState(
    defaults?.brandFocus ?? "客厅沙发、卧室套装"
  )

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

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (disabled || converting) return
    let imgPayload: BriefImage[] | undefined
    if (images.length > 0) {
      setConverting(true)
      try {
        imgPayload = await Promise.all(
          images.map((i) => (i.preset ? presetToImage(i) : fileToImage(i.file!)))
        )
      } catch {
        setImgNote("参考图处理失败，请重试或移除后再生成")
        setConverting(false)
        return
      }
      setConverting(false)
    }
    onSubmit({ genre, episodes, episodeSecs, brandFocus, images: imgPayload })
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

        {/* Reference materials (multimodal) */}
        <div className="sm:col-span-12">
          <div className="mb-2 flex items-end justify-between gap-2">
            <label className="label-tech block">
              参考素材（可选）
            </label>
            <span className="font-mono text-[10px] text-bone-500">
              已选 {images.length} / {MAX_IMAGES} · 喂给 Gemini 影响立意 / 植入 / 分镜
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

        {/* Submit */}
        <div className="flex items-end sm:col-span-6">
          <button
            type="submit"
            disabled={disabled || converting}
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
