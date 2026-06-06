import { Brief, BriefImage, Concept, ProposeResp, RefineReq, SSEvent } from "./types"
import { getToken } from "./auth"

const API = process.env.NEXT_PUBLIC_API ?? "http://localhost:8080"

// Thrown when the backend rejects the session token; the UI should re-login.
export class UnauthorizedError extends Error {
  constructor() {
    super("UNAUTHORIZED")
    this.name = "UnauthorizedError"
  }
}

// streamSSE POSTs a JSON body and parses the `data:`-framed SSE response,
// invoking onEvent per event. Shared by generate() and refine().
async function streamSSE(
  path: string,
  body: unknown,
  onEvent: (e: SSEvent) => void
): Promise<void> {
  const token = getToken()
  const res = await fetch(`${API}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
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

// assist powers the 提示词助手「AI 扩写/优化」button: it sends the user's rough
// requirement (plus pacing context) and gets back one complete, polished 生成需求
// string to pre-fill the textarea. Plain JSON, not SSE.
export async function assist(
  requirement: string,
  episodes: number,
  episodeSecs: number,
  images?: BriefImage[]
): Promise<string> {
  const token = getToken()
  const res = await fetch(`${API}/api/assist`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ requirement, episodes, episodeSecs, images }),
  })
  if (res.status === 401) throw new UnauthorizedError()
  if (!res.ok) throw new Error(`AI 扩写失败 (${res.status})`)
  const data = (await res.json()) as { requirement: string }
  return data.requirement ?? ""
}

// propose asks the backend for 2-3 candidate 立意方向 for a brief (plain JSON, not
// SSE). The user picks/tweaks one before generate() runs the rest of the pipeline.
export async function propose(brief: Brief): Promise<Concept[]> {
  const token = getToken()
  const res = await fetch(`${API}/api/propose`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(brief),
  })
  if (res.status === 401) throw new UnauthorizedError()
  if (!res.ok) throw new Error(`提案失败 (${res.status})`)
  const data = (await res.json()) as ProposeResp
  return data.concepts ?? []
}

// generate runs the full pipeline. When `concept` is provided the user has already
// chosen a 立意方向, so it is sent at the top level alongside the brief and the
// backend skips concept generation (starts from the bible stage).
export function generate(
  brief: Brief,
  onEvent: (e: SSEvent) => void,
  concept?: Concept
): Promise<void> {
  const body = { ...brief, ...(concept ? { concept } : {}) }
  return streamSSE("/api/generate", body, onEvent)
}

export function refine(req: RefineReq, onEvent: (e: SSEvent) => void): Promise<void> {
  return streamSSE("/api/refine", req, onEvent)
}
