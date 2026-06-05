import { Brief, SSEvent } from "./types"
import { getToken } from "./auth"

const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"

// Thrown when the backend rejects the session token; the UI should re-login.
export class UnauthorizedError extends Error {
  constructor() {
    super("UNAUTHORIZED")
    this.name = "UnauthorizedError"
  }
}

export async function generate(brief: Brief, onEvent: (e: SSEvent) => void): Promise<void> {
  const token = getToken()
  const res = await fetch(`${API}/api/generate`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(brief),
  })
  if (res.status === 401) throw new UnauthorizedError()
  if (!res.body) throw new Error("no stream")
  const reader = res.body.getReader()
  const dec = new TextDecoder()
  let buf = ""
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buf += dec.decode(value, { stream: true })
    const chunks = buf.split("\n\n")
    buf = chunks.pop() ?? ""
    for (const c of chunks) {
      const line = c.split("\n").find((l) => l.startsWith("data: "))
      if (line) onEvent(JSON.parse(line.slice(6)) as SSEvent)
    }
  }
}
