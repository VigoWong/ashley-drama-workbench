"use client"

import { useState } from "react"
import { login } from "@/lib/auth"

interface Props {
  onAuthed: () => void
}

export default function LoginForm({ onAuthed }: Props) {
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await login(username, password)
      onAuthed()
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败")
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-5">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex flex-col items-center gap-3 text-center">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src="/ashley-logo.png"
            alt="Ashley Furniture Industries"
            className="h-9 w-auto rounded-md bg-bone-50 px-3 py-1.5 shadow-sm ring-1 ring-black/5"
          />
          <h1 className="font-display text-2xl font-semibold tracking-tight text-bone-50">
            短剧生产工作台
          </h1>
          <p className="font-mono text-[11px] text-bone-400">登录以进入工作台</p>
        </div>

        <form onSubmit={handleSubmit} className="panel rounded-2xl p-6 sm:p-8">
          <div className="space-y-4">
            <div>
              <label htmlFor="username" className="label-tech mb-2 block">
                用户名
              </label>
              <input
                id="username"
                type="text"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                disabled={busy}
                placeholder="admin"
                className="w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-4 py-3 font-sans text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-50"
              />
            </div>
            <div>
              <label htmlFor="password" className="label-tech mb-2 block">
                密码
              </label>
              <input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={busy}
                placeholder="admin"
                className="w-full rounded-lg border border-bone-500/20 bg-ink-900/60 px-4 py-3 font-sans text-bone-50 outline-none transition focus:border-ember-500/70 focus:ring-2 focus:ring-ember-500/20 disabled:opacity-50"
              />
            </div>

            {error && (
              <p className="font-mono text-xs text-signal-stop">✕ {error}</p>
            )}

            <button
              type="submit"
              disabled={busy}
              className="w-full rounded-lg bg-ember-500 px-6 py-3.5 font-sans text-sm font-semibold tracking-wide text-ink-900 transition hover:bg-ember-400 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {busy ? "登录中…" : "登 录"}
            </button>

            <p className="text-center font-mono text-[10px] text-bone-400">
              默认账号 admin / admin
            </p>
          </div>
        </form>
      </div>
    </div>
  )
}
