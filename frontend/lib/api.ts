import { Brief, SSEvent } from "./types"

const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"

export async function generate(brief: Brief, onEvent: (e: SSEvent) => void): Promise<void> {
  const res = await fetch(`${API}/api/generate`, {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(brief),
  })
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
