const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"
const KEY = "ashley_token"

export function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(KEY)
}

export function setToken(t: string) {
  localStorage.setItem(KEY, t)
}

export function clearToken() {
  localStorage.removeItem(KEY)
}

// verifyToken checks the stored token against the backend (tokens are minted per
// server process, so a token from a previous run is stale). Returns true only if
// a valid token exists; clears an explicitly-rejected (401) token. On a network
// error it keeps the token (assume valid; the next real call will surface auth).
export async function verifyToken(): Promise<boolean> {
  const token = getToken()
  if (!token) return false
  try {
    const res = await fetch(`${API}/api/history`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.status === 401) {
      clearToken()
      return false
    }
    return true
  } catch {
    return true
  }
}

export async function login(username: string, password: string): Promise<void> {
  const res = await fetch(`${API}/api/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  })
  if (!res.ok) {
    let msg = "登录失败"
    try {
      const j = await res.json()
      if (j?.error) msg = j.error
    } catch {
      /* keep default */
    }
    throw new Error(msg)
  }
  const j = await res.json()
  if (!j?.token) throw new Error("登录响应缺少 token")
  setToken(j.token)
}
