import { Plan } from "./types"

// planToMarkdown mirrors the Go backend's render.Markdown so the client can
// produce an identical Markdown export without a server round-trip.
const nz = (s: string, d: string) => (s && s.trim() !== "" ? s : d)

export function planToMarkdown(p: Plan): string {
  const out: string[] = []
  const w = (s: string) => out.push(s)

  w(`# ${nz(p.bible?.title ?? "", "Untitled Series")}\n`)

  const c = p.concept ?? ({} as Plan["concept"])
  w("## Concept")
  w(`- **Logline:** ${c.logline ?? ""}`)
  w(`- **Theme:** ${c.theme ?? ""}`)
  w(`- **Audience:** ${c.audience ?? ""}`)
  w(`- **Tone:** ${c.tone ?? ""}`)
  w(`- **Payoff engine:** ${c.payoffEngine ?? ""}`)
  w(`- **Core conflict:** ${c.coreConflict ?? ""}\n`)

  const b = p.bible ?? ({} as Plan["bible"])
  w("## Series Bible")
  w(`- Episodes: ${b.episodes ?? 0} x ${b.episodeSecs ?? 0}s | Platform: ${b.platform ?? ""}`)
  w(`- Integration: ${b.integrationThesis ?? ""}\n`)

  w("## Characters")
  for (const ch of p.characters ?? []) {
    w(`- **${ch.name}** (${ch.role}): ${ch.bio} _Arc:_ ${ch.arc}`)
  }
  w("")

  w("## Episodes")
  for (const e of p.episodes ?? []) {
    w(`### Ep ${e.number} — ${e.title}`)
    w(e.synopsis ?? "")
    w(`- **Hook:** ${e.hook ?? ""}`)
    w(`- **Cliffhanger:** ${e.cliffhanger ?? ""}`)
    w(`- **Payoff:** ${e.payoff ?? ""}\n`)
  }

  w("## Brand Integration")
  for (const pl of p.placements ?? []) {
    w(`- Ep ${pl.episode} — ${pl.category} (${pl.productSku}): ${pl.emotionalBeat} | CTA: ${pl.ctaTiming}`)
  }
  w("")

  w("## Hero Scenes")
  for (const h of p.heroScenes ?? []) {
    w(`### Ep ${h.episode} — ${h.title}`)
    for (const s of h.shots ?? []) {
      w(`${s.number}. [${s.shotType}] ${s.action} — "${s.dialogue}"`)
    }
  }
  w("")

  const pr = p.production ?? ({} as Plan["production"])
  w("## Production")
  w(`- Format: ${pr.format ?? ""} | Budget: ${pr.budgetTier ?? ""} | Shots: ${pr.shotCount ?? 0} | Cast: ${pr.castSize ?? 0}`)
  w(`- Furniture: ${(pr.furnitureProps ?? []).join(", ")}\n`)

  const d = p.distribution ?? ({} as Plan["distribution"])
  w("## Distribution")
  w(`- CTA: ${d.ctaCopy ?? ""}`)
  w(`- Hashtags: ${(d.hashtags ?? []).join(" ")}`)

  return out.join("\n")
}
