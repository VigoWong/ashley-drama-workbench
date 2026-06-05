# Ashley Short-Drama Production Workbench

An AI-agent-driven workbench that turns a one-line brief into a complete,
**furniture-brand-marketing short-drama production plan** for [Ashley](https://www.ashleyfurniture.com/).
You give it a genre, episode count, and a brand focus; an orchestrated pipeline
of LLM "agent" stages — grounded and gated by deterministic Go tools — returns a
structured, 8-block plan ready to hand to a writers' room and a production crew.

Two surfaces sit over the same orchestrator: a **CLI** (Markdown/JSON output) and
an **HTTP server with live SSE streaming**, driven by a **Next.js + Tailwind** UI.

> Design doc and implementation plan live under [`docs/superpowers/`](docs/superpowers/).

---

## 1. What & why

Vertical micro-dramas (ReelShort / DramaBox style) are one of the highest-ROI
content formats for reaching US consumers right now. Ashley is a furniture
*manufacturer* — so why a short-drama tool?

**Because a furniture brand's product lives inside exactly the scenes short-drama
is made of.** Makeover revenge, fresh-start-after-divorce, building the dream
home — every one of these arcs is staged in a living room, a bedroom, around a
dining table. The set *is* the showroom. A sofa is not a prop in a reconciliation
scene; it is the emotional anchor of it. That insight is the product:

> **US-market, English, vertical (9:16) branded short dramas that market Ashley
> furniture** — home-makeover / fresh-start / dream-home tropes with natural
> product placement, built for brand awareness plus DTC / in-store conversion.

The deliverable is a structured **production plan** with 8 blocks:

| # | Block | Contents |
|---|-------|----------|
| 1 | **Concept** | logline, theme, audience, tone, core "payoff engine", core conflict |
| 2 | **Series Bible** | bingeable title, genre tags, episode/runtime math, platform, brand-integration thesis |
| 3 | **Characters** | protagonist / antagonist / love-interest bios, arcs, relationships |
| 4 | **Episodes** | per-episode synopsis, beats, golden-3-second hook, cliffhanger, payoff/reversal |
| 5 | **Brand Integration** | which Ashley SKU appears in which scene, on which emotional beat, with CTA timing |
| 6 | **Hero Script** | full shot list + sample dialogue for the 1–2 highest-payoff episodes |
| 7 | **Production** | format, budget tier, shot/cast counts, locations, furniture prop list |
| 8 | **Distribution** | CTA copy, link placement, hashtags |

Defaults: **12 episodes × 90 seconds**. Hero shot lists are only generated for
the 1–2 highest-payoff episodes (to control token spend).

---

## 2. Business understanding

The pipeline encodes real short-drama craft rather than generic "write me a
screenplay" prompting:

- **Golden 3 seconds.** Every episode opens on a scroll-stopping hook; episode 1
  in particular has to earn the first swipe. This is enforced, not hoped for (see
  the pacing validator below).
- **Per-episode cliffhangers.** Each episode ends on an open loop so the viewer
  taps "next". Every episode must carry a non-empty cliffhanger.
- **Payoff / reversal density.** The format thrives on frequent satisfaction
  beats (a reveal, a comeuppance, a status jump). We require payoffs in **≥ 60%**
  of episodes across the season.
- **CTA replaces the paywall.** Where a paid drama gates episodes behind coins, a
  *branded* drama gates nothing — it converts. The plan places a shopping CTA at
  the emotional peak instead.

**Home/furniture tropes** (curated in `backend/internal/tools/tropes.go`) are the
genre engine, each chosen because furniture naturally earns screen time:
*From-broke-to-dream-home*, *Fresh-start-after-divorce*, *Secret-heir-renovation*,
*Family-reconciliation*.

**Placement strategy** ties it together: the Brand Integration stage maps Ashley
SKUs (`backend/internal/tools/catalog.go`, e.g. the *Maeford Sectional*,
*Realyn Queen Bed*, *Haddigan Dining Set*) to scenes by **emotional beat → SKU →
CTA timing**. A reunion lands on the sectional; a fresh-start morning lands on the
bed. The scene sells the feeling; the CTA sells the SKU.

---

## 3. Architecture

An **orchestrator-driven, multi-stage pipeline**. Each stage is one LLM agent
(an embedded prompt template + Gemini structured-JSON output) that reads and
writes a shared `PlanState`. Deterministic Go tools ground the LLM with real data
and **gate** the episodes stage with a one-shot self-correction loop. A pluggable
`Provider` makes the whole chain runnable and testable with no API key.

```
                          Brief {genre, episodes, episodeSecs, brandFocus}
                                            │
                                            ▼
   ┌──────────────────────────  ORCHESTRATOR  ───────────────────────────┐
   │   (runs stages in order, threads *PlanState, emits SSE events)        │
   │                                                                       │
   │  [1] concept ───────────────◄── GetWinningTropes()   (curated tropes) │
   │       │                                                               │
   │  [2] bible                                                            │
   │       │                                                               │
   │  [3] characters                                                       │
   │       │                                                               │
   │  [4] episodes                                                         │
   │       │                                                               │
   │       ▼                                                               │
   │   ValidatePacing()  ── pure Go ──┐  hook + cliffhanger every ep?      │
   │       │                          │  payoff density ≥ 60%?             │
   │       │ pass                fail │                                    │
   │       │                          ▼                                    │
   │       │            [4b] episodes_refine  (one-shot LLM rewrite, fed   │
   │       │◄─────────────────────────┘        the failed-rule report)    │
   │       ▼                                                               │
   │  [5] placements ────────────◄── GetProductCatalog()  (Ashley SKUs)    │
   │       │                                                               │
   │  [6] hero  (shot lists + dialogue for top 1–2 episodes)               │
   │       │                                                               │
   │  [7] production_distribution                                          │
   │       │                                                               │
   │       ▼                                                               │
   │   complete → full Plan (8 blocks)                                     │
   └───────────────────────────────────────────────────────────────────┘
                                            │
              every stage calls:  Provider.GenerateJSON(ctx, stage, prompt, schema)
                                            │
                         ┌──────────────────┴───────────────────┐
                         ▼                                       ▼
                 GeminiProvider                          MockProvider / DemoMock
        (REST, responseMimeType: JSON,                (deterministic fixtures —
         3-attempt retry + JSON-repair)                no API key, full demo plan)
```

- **7 pipeline stages** (`AllStages()` in `backend/internal/agent/orchestrator.go`):
  `concept → bible → characters → episodes → placements → hero →
  production_distribution`. `episodes_refine` is an internal one-shot rewrite
  inside the episodes stage, not a top-level stage.
- **3 deterministic tools** (`backend/internal/tools/`): `GetWinningTropes`
  (RAG-lite trope grounding), `GetProductCatalog` (Ashley SKU grounding), and
  `ValidatePacing` (the gate).
- **2 surfaces, 1 orchestrator:** `backend/cmd/server` (HTTP + SSE) and
  `backend/cmd/cli` both call `agent.New(provider, emit).Run(ctx, brief)`.

---

## 4. Prompt & AI orchestration notes

**Why a staged pipeline instead of free-running ReAct.** A fixed sequence of
single-purpose prompts is *controllable* (we know exactly what runs and in what
order), *observable* (each stage emits a discrete SSE event with its partial
output), and *testable* (each stage and the whole chain run against a Mock
provider). An autonomous function-calling agent would be unpredictable, hard to
test, and token-expensive — and a single mega-prompt isn't orchestration at all.
Both alternatives were considered and rejected (see the design doc).

**Why `ValidatePacing` is deterministic Go, not an LLM.** Pacing rules are exact
and verifiable — *does every episode have a non-empty hook and cliffhanger? is
payoff density ≥ 60%?* — so judging them with another stochastic LLM call would
be slower, costlier, and *less* reliable than a `for` loop. This is the core
design stance: **the LLM creates, deterministic code verifies, and a feedback
loop re-writes.** When `ValidatePacing` fails, its structured report is rendered
to a feedback string (`PacingReport.FormatIssues()`) and injected, along with the
rejected draft, into the `episodes_refine` prompt for exactly one corrective pass
(`EpisodeStage.Run` in `backend/internal/agent/stages.go`). That is the
genuinely *agentic* part of the system, and it's grounded in code, not vibes.

**Structured output.** The Gemini provider requests
`responseMimeType: "application/json"` so the model returns raw JSON, and each
stage prompt ends with the exact JSON shape of its target `model` struct.

**Retry + JSON-repair.** `GeminiProvider.GenerateJSON`
(`backend/internal/llm/gemini.go`) makes up to **3 attempts**: transport/HTTP
errors back off and retry; if a response comes back but isn't valid JSON
(after stripping stray ``` fences), the next attempt re-prompts the model with its
own bad output and an instruction to return strict JSON only.

---

## 5. Run it

**Prerequisites:** Go 1.22+ and Node 18+.

### Keyless demo first (no API key needed)

With no `GEMINI_API_KEY`, the backend falls back to a `DemoMock` provider that
produces a full, sensible sample plan — so you can see the whole system end-to-end
immediately.

```bash
cd backend

# CLI — prints stage progress to stderr, a Markdown plan to stdout
make cli
# or pass flags directly:
go run ./cmd/cli -episodes 3 -format markdown
go run ./cmd/cli -episodes 3 -format json -out plan.json

# Server — HTTP + SSE on :8080
make server
```

Hit the streaming endpoint:

```bash
curl -N -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"genre":"home makeover revenge","episodes":3,"episodeSecs":90,"brandFocus":"living room sofas, bedroom sets"}'
```

You'll see a stream of `stage_start` / `stage_done` SSE events ending in a single
`complete` event carrying the full plan.

> Note: in keyless demo mode the `DemoMock` returns a fixed sample plan and does
> not vary by the requested episode count.

### With a Gemini key

```bash
cd backend
export GEMINI_API_KEY=your-key-here
export GEMINI_MODEL=gemini-2.0-flash   # optional; this is the default
make server   # or: make cli
```

(See `backend/.env.example`.)

### Frontend

```bash
cd frontend
npm install
npm run dev     # http://localhost:3000
```

The UI talks to `http://localhost:8080` by default. Point it elsewhere with
`NEXT_PUBLIC_API`, e.g. `NEXT_PUBLIC_API=http://localhost:9090 npm run dev`.

---

## 6. Testing

The entire suite runs **with no API key** — the Mock provider stands in for
Gemini across stage, orchestrator, and integration tests:

```bash
cd backend && go test ./...
```

Of note are the **deterministic pacing-validator tests**
(`backend/internal/tools/pacing_test.go`): `TestPacingPerfectScoresHigh`,
`TestPacingMissingHookFails`, and `TestPacingLowPayoffDensityFails` pin the exact
gate rules that drive the self-correction loop.

---

## 7. Project layout

```
backend/
  cmd/server/main.go        HTTP + SSE server (POST /api/generate)
  cmd/cli/main.go           CLI over the same orchestrator (markdown/json)
  internal/
    model/                  domain types: Brief, Plan, Episode, ... + SSE events
    llm/                    Provider interface; Gemini (REST) + Mock/DemoMock; env factory
    tools/                  GetWinningTropes, GetProductCatalog, ValidatePacing (pure Go)
    prompts/                one embedded .tmpl per stage + render helper
    agent/                  Stage interface, PlanState, orchestrator, stage implementations
    render/                 Plan -> Markdown
  Makefile                  test / server / cli / build
  .env.example              GEMINI_API_KEY, GEMINI_MODEL, PORT
frontend/
  app/page.tsx              workbench page (form, timeline, plan view)
  components/               InputForm, StageTimeline, PlanView, ExportBar
  lib/api.ts                POST + SSE stream parser
  lib/types.ts              TypeScript mirror of the Go model
  lib/markdown.ts           client-side Plan -> Markdown for export
docs/superpowers/           design doc + implementation plan
```

---

## 8. Design decisions, trade-offs & future work

**Scope was deliberately bounded (YAGNI).** This is an MVP focused on the agentic
generation core, so it intentionally omits:

- **Auth, a database, and persistence** — generation is stateless; plans are
  returned/exported, not stored.
- **Per-stage human editing** — the flow is one-shot generate + live streaming,
  not an interactive editor with regenerate-this-stage controls.
- **Multi-market / multi-language** — locked to US / English / home vertical.
- The **keyless demo mock** returns a fixed sample and ignores the requested
  episode count (it exists to demonstrate the chain, not to vary output).

**Future work**, roughly in priority order:

1. **Human-in-the-loop** stage editing — approve/edit a stage's output, then
   regenerate downstream stages from that edit.
2. **Multi-market / language switching** — the trope tool already reserves a
   `market` parameter for this.
3. **Persistence** — save, list, and revisit generated plans.
4. **Video-gen handoff** — feed hero shot lists into a text-to-video pipeline.
5. **Real RAG** for the trope and catalog tools — back them with a live product
   feed and a trends/performance index instead of curated static data.
