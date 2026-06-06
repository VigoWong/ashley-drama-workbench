"use client"

import { useEffect, useState } from "react"
import { HistoryRecord, HistorySummary } from "@/lib/types"
import { listHistory, getHistory, deleteHistory } from "@/lib/history"
import { refine, UnauthorizedError } from "@/lib/api"
import PlanView from "@/components/PlanView"

interface Props {
  onBack: () => void
  onUnauthorized: () => void
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export default function HistoryView({ onBack, onUnauthorized }: Props) {
  const [items, setItems] = useState<HistorySummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [detail, setDetail] = useState<HistoryRecord | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [refining, setRefining] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    listHistory()
      .then((list) => {
        if (!cancelled) setItems(list)
      })
      .catch((err) => {
        if (cancelled) return
        if (err instanceof UnauthorizedError) {
          onUnauthorized()
          return
        }
        setError(err instanceof Error ? err.message : "加载失败")
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [onUnauthorized])

  async function openDetail(id: string) {
    setDetailLoading(true)
    setError(null)
    try {
      const rec = await getHistory(id)
      setDetail(rec)
    } catch (err) {
      if (err instanceof UnauthorizedError) {
        onUnauthorized()
        return
      }
      setError(err instanceof Error ? err.message : "加载方案详情失败")
    } finally {
      setDetailLoading(false)
    }
  }

  async function removeItem(id: string, title: string) {
    if (!window.confirm(`确定删除方案「${title || "未命名方案"}」?此操作不可恢复。`)) return
    setDeletingId(id)
    setError(null)
    try {
      await deleteHistory(id)
      setItems((prev) => (prev ? prev.filter((x) => x.id !== id) : prev))
    } catch (err) {
      if (err instanceof UnauthorizedError) {
        onUnauthorized()
        return
      }
      setError(err instanceof Error ? err.message : "删除失败")
    } finally {
      setDeletingId(null)
    }
  }

  // onRefine 对历史方案的某一段做重生成(复用 /api/refine)。注意:refine 不入库,
  // 因此这里只更新当前查看的方案(交互草稿),不会改写已保存的历史记录。
  async function onRefine(fromStage: string, only: boolean, note: string) {
    if (!detail || refining) return
    setRefining(true)
    setError(null)
    try {
      await refine({ plan: detail.plan, fromStage, only, note }, (e) => {
        const np = e.plan
        if (e.type === "complete" && np) {
          setDetail((d) => (d ? { ...d, plan: np } : d))
        }
      })
    } catch (err) {
      if (err instanceof UnauthorizedError) {
        onUnauthorized()
        return
      }
      setError(err instanceof Error ? err.message : "重跑失败")
    } finally {
      setRefining(false)
    }
  }

  /* ---- Detail view ---- */
  if (detail) {
    return (
      <div className="space-y-8">
        <div className="flex items-center justify-between">
          <p className="font-mono text-xs text-bone-400">
            历史方案 · 共 {detail.plan.episodes?.length ?? 0} 集 · 生成于{" "}
            {formatTime(detail.createdAt)}
          </p>
          <div className="flex items-center gap-2">
            {refining && (
              <span className="font-mono text-xs text-ember-300">正在重跑该段…</span>
            )}
            <button
              onClick={() => setDetail(null)}
              disabled={refining}
              className="rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700 disabled:opacity-50"
            >
              ← 返回列表
            </button>
          </div>
        </div>
        {error && (
          <div className="rounded-xl border border-signal-stop/40 bg-signal-stop/10 p-4">
            <p className="font-mono text-sm text-signal-stop">✕ {error}</p>
          </div>
        )}
        <PlanView
          plan={detail.plan}
          onChange={(p) => setDetail((d) => (d ? { ...d, plan: p } : d))}
          onRefine={onRefine}
        />
      </div>
    )
  }

  /* ---- List view ---- */
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <span className="label-tech">历史方案</span>
          <p className="mt-1 font-mono text-xs text-bone-400">
            过往生成的全部方案,点击查看完整内容。
          </p>
        </div>
        <button
          onClick={onBack}
          className="rounded-lg border border-bone-500/20 bg-ink-800 px-4 py-2 font-mono text-xs uppercase tracking-wider text-bone-100 transition hover:border-bone-500/50 hover:bg-ink-700"
        >
          ← 返回工作台
        </button>
      </div>

      {loading && (
        <p className="font-mono text-sm text-bone-400">加载中…</p>
      )}

      {error && (
        <div className="rounded-xl border border-signal-stop/40 bg-signal-stop/10 p-4">
          <p className="font-mono text-sm text-signal-stop">✕ {error}</p>
        </div>
      )}

      {detailLoading && (
        <p className="font-mono text-sm text-bone-400">正在加载方案详情…</p>
      )}

      {!loading && !error && items && items.length === 0 && (
        <div className="rounded-xl border border-bone-500/15 bg-ink-800/40 p-10 text-center">
          <p className="font-sans text-sm text-bone-300">
            暂无历史方案。
          </p>
          <p className="mt-2 font-mono text-xs text-bone-400">
            生成第一份方案后,会自动保存到这里。
          </p>
        </div>
      )}

      {!loading && !error && items && items.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2">
          {items.map((it) => (
            <div
              key={it.id}
              role="button"
              tabIndex={0}
              onClick={() => !detailLoading && deletingId !== it.id && openDetail(it.id)}
              onKeyDown={(e) => {
                if ((e.key === "Enter" || e.key === " ") && !detailLoading) openDetail(it.id)
              }}
              className={`panel group relative flex cursor-pointer flex-col gap-3 rounded-xl border border-bone-500/15 bg-ink-800/60 p-5 text-left transition hover:border-ember-400/50 ${
                deletingId === it.id ? "opacity-50" : ""
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <h3 className="font-display text-lg font-semibold leading-tight text-bone-100 group-hover:text-ember-400">
                  {it.title || "未命名方案"}
                </h3>
                <span className="shrink-0 rounded-full border border-ember-400/30 bg-ember-400/10 px-2.5 py-0.5 font-mono text-[10px] text-ember-400">
                  {it.episodes} 集
                </span>
              </div>
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                {it.genre && (
                  <span className="line-clamp-1 max-w-[60ch] font-mono text-xs text-bone-300">
                    需求 · {it.genre}
                  </span>
                )}
                <span className="font-mono text-[11px] text-bone-400">
                  {formatTime(it.createdAt)}
                </span>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  removeItem(it.id, it.title)
                }}
                disabled={deletingId === it.id}
                aria-label="删除该方案"
                className="absolute bottom-3 right-3 rounded-md border border-bone-500/20 bg-ink-900/70 px-2 py-1 font-mono text-[10px] uppercase tracking-wider text-bone-400 opacity-0 transition hover:border-signal-stop/50 hover:text-signal-stop group-hover:opacity-100 disabled:opacity-60"
              >
                {deletingId === it.id ? "删除中…" : "🗑 删除"}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
