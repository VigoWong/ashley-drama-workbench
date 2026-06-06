import { HistoryRecord, HistorySummary } from "./types"
import { UnauthorizedError } from "./api"
import { getToken } from "./auth"

const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"

function authHeaders(): HeadersInit {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export async function listHistory(): Promise<HistorySummary[]> {
  const res = await fetch(`${API}/api/history`, { headers: authHeaders() })
  if (res.status === 401) throw new UnauthorizedError()
  if (!res.ok) throw new Error(`加载历史列表失败 (${res.status})`)
  return (await res.json()) as HistorySummary[]
}

export async function getHistory(id: string): Promise<HistoryRecord> {
  const res = await fetch(`${API}/api/history/${id}`, { headers: authHeaders() })
  if (res.status === 401) throw new UnauthorizedError()
  if (res.status === 404) throw new Error("该方案不存在或已被删除")
  if (!res.ok) throw new Error(`加载方案详情失败 (${res.status})`)
  return (await res.json()) as HistoryRecord
}

export async function deleteHistory(id: string): Promise<void> {
  const res = await fetch(`${API}/api/history/${id}`, {
    method: "DELETE",
    headers: authHeaders(),
  })
  if (res.status === 401) throw new UnauthorizedError()
  if (!res.ok && res.status !== 204) throw new Error(`删除失败 (${res.status})`)
}

// updateHistory 把编辑/重生成后的方案「存回历史」(覆盖原记录的 plan)。
export async function updateHistory(id: string, plan: unknown): Promise<void> {
  const res = await fetch(`${API}/api/history/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify({ plan }),
  })
  if (res.status === 401) throw new UnauthorizedError()
  if (res.status === 404) throw new Error("该方案不存在或已被删除")
  if (!res.ok && res.status !== 204) throw new Error(`保存失败 (${res.status})`)
}
