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
