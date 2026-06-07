"use client"
import { useEffect, useState } from "react"
import { clearToken, verifyToken } from "@/lib/auth"
import LoginForm from "@/components/LoginForm"
import HistoryView from "@/components/HistoryView"
import WizardView from "@/components/WizardView"
import ChatView from "@/components/ChatView"

type View = "workbench" | "history"
type Mode = "wizard" | "chat"

export default function Home() {
  const [ready, setReady] = useState(false)
  const [authed, setAuthed] = useState(false)
  const [view, setView] = useState<View>("workbench")
  const [mode, setMode] = useState<Mode>("wizard")

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      const ok = await verifyToken()
      if (!cancelled) { setAuthed(ok); setReady(true) }
    })()
    return () => { cancelled = true }
  }, [])

  function logout() { clearToken(); setAuthed(false); setView("workbench") }
  function handleUnauthorized() { clearToken(); setAuthed(false); setView("workbench") }

  if (!ready) return null
  if (!authed) return <LoginForm onAuthed={() => setAuthed(true)} />

  return (
    <div className="relative min-h-screen">
      <main className="mx-auto max-w-6xl px-5 py-10 sm:px-8 sm:py-16">
        <header className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="mb-4 flex items-center gap-3">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src="/ashley-logo.png" alt="Ashley Furniture Industries" className="h-8 w-auto rounded-md bg-bone-50 px-2.5 py-1.5 shadow-sm ring-1 ring-black/5" />
              <span className="inline-flex items-center gap-2 rounded-full border border-bone-500/20 bg-ink-800/60 px-3 py-1">
                <span className="h-1.5 w-1.5 rounded-full bg-ember-400" />
                <span className="label-tech">品牌内容工作台</span>
              </span>
            </div>
            <h1 className="font-display text-3xl font-semibold leading-[0.95] tracking-tight sm:text-4xl">
              短剧<span className="text-ember-400"> 生产工作台</span>
            </h1>
          </div>
          <div className="flex items-center gap-2">
            {/* mode switch */}
            <div className="flex rounded-lg border border-bone-500/20 bg-ink-800 p-0.5">
              {(["wizard", "chat"] as Mode[]).map((m) => (
                <button key={m} onClick={() => setMode(m)}
                  className={`rounded-md px-3 py-1 font-mono text-[11px] uppercase tracking-wider transition ${mode === m ? "bg-ember-500 text-ink-900" : "text-bone-300 hover:text-ember-400"}`}>
                  {m === "chat" ? "对话" : "向导"}
                </button>
              ))}
            </div>
            <button onClick={() => setView("history")} className="rounded-lg border border-bone-500/20 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-bone-300 transition hover:border-ember-400/50 hover:text-ember-400">历史</button>
            <button onClick={logout} className="rounded-lg border border-bone-500/20 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-bone-300 transition hover:border-bone-500/50 hover:text-bone-100">退出登录</button>
          </div>
        </header>

        {view === "history" ? (
          <HistoryView onBack={() => setView("workbench")} onUnauthorized={handleUnauthorized} />
        ) : mode === "chat" ? (
          <ChatView onUnauthorized={handleUnauthorized} />
        ) : (
          <WizardView onUnauthorized={handleUnauthorized} />
        )}
      </main>

      <footer className="border-t border-bone-500/10 py-6 text-center">
        <span className="label-tech">无 Key 演示模式生成完整示例方案 · 配置 GEMINI_API_KEY 启用真实生成</span>
      </footer>
    </div>
  )
}
