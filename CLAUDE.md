# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

短剧生产工作台 (Short-Drama Production Workbench): a one-line brief becomes an 8-block,
Chinese-language short-drama production plan that markets Ashley furniture for the **China
domestic market** (抖音 / 快手 / 红果短剧, vertical 9:16). A Go backend runs the orchestration;
a Next.js + Tailwind frontend drives it. The README is the authoritative deep reference
(architecture, business rationale, run guide, design trade-offs) — read it for the "why".
This file covers commands and the structural facts you need before editing.

## Commands

Backend commands run from `backend/` (Go 1.23, module `github.com/ashley/drama-workbench`):

```bash
make test                                  # go test ./...  (entire suite runs with NO API key / NO DB)
go test ./internal/tools/ -run TestPacingMissingHookFails   # single test
go test ./internal/agent/ -v               # one package, verbose
make server                                # HTTP + SSE on :8080 (PORT overrides)
make cli ARGS="-episodes 5 -format json"   # CLI; pass flags via ARGS
go run ./cmd/cli -genre "家装改造逆袭" -episodes 5 -secs 30   # run CLI directly (CLI flag defaults are still English placeholders)
make build                                 # builds bin/server and bin/cli
```

History persistence (Postgres) via docker-compose:

```bash
DB_PORT=5433 docker compose up -d          # default port 5432; override with DB_PORT
# then point the backend at it:
export DATABASE_URL="postgres://drama:drama@localhost:5433/drama?sslmode=disable"
# store integration test (skipped when DATABASE_URL unset):
DATABASE_URL=... go test ./internal/store
```

Frontend (from `frontend/`, Next.js 16, npm):

```bash
npm install && npm run dev   # http://localhost:3000, calls :8080 by default
npm run build                # production build
npm run lint                 # eslint
```

`NEXT_PUBLIC_API` overrides the backend URL the UI calls (e.g. 8091 when 8080 is busy).

## Environment / Provider selection (important for dev & tests)

`llm.FromEnv()` (`backend/internal/llm/factory.go`) picks a provider in priority order:

1. **Vertex AI** — when `VERTEX_CREDENTIALS_FILE` points at a service-account JSON.
   `VERTEX_LOCATION` (default us-central1), `VERTEX_PROJECT` (default = SA's project_id),
   `GEMINI_MODEL` (default gemini-2.5-flash). Mints OAuth tokens, hits the regional endpoint.
2. **AI Studio** — when `GEMINI_API_KEY` is set. `GEMINI_MODEL` (default gemini-2.0-flash),
   optional `GEMINI_BASE_URL` proxy.
3. **DemoMock** — no credentials: emits a fixed, complete Chinese sample plan. The CLI, server,
   and the **entire test suite** run this way; you never need a real key to develop or test.
   The demo mock ignores the requested episode count (fixed ~12-ep season).

Gemini provider (both modes, `internal/llm/gemini.go`): structured JSON output, 3-attempt
retry + JSON-repair; `thinkingBudget=0` for 2.5 models. `vertex-sa.json` / `run-dev.sh` /
`.env.local` are gitignored — **never commit secrets**.

Auth env (`internal/auth/auth.go`): `AUTH_USERNAME` / `AUTH_PASSWORD` (default admin/admin).
`DATABASE_URL` enables history persistence; unset/unreachable → graceful degradation.
`PORT` (default 8080).

## Architecture essentials

A linear, orchestrator-driven pipeline of single-purpose LLM stages, grounded and gated
by deterministic Go. Core design stance: **the LLM creates, deterministic Go verifies,
a feedback loop rewrites.**

- **Orchestrator** (`internal/agent/orchestrator.go`): runs `AllStages()` in order, threads a
  shared `*PlanState` (`internal/agent/stage.go` — holds `*model.Plan` + the `Provider`), emits
  an `Event` per stage via an `Emitter` (server → SSE, CLI → stdout). **Retries a single stage's
  transient failures up to `maxStageAttempts` (3)** with linear backoff, honoring ctx cancel.
- **7 top-level stages** (`internal/agent/stages.go`): concept → bible → characters → episodes →
  placements → hero → production_distribution.
- **The genuinely agentic part is the episodes gate.** `EpisodeStage.Run` generates episodes,
  then calls `tools.ValidatePacing` (pure Go: every episode needs a non-empty hook + cliffhanger;
  payoff density ≥ 60%). On failure it renders `PacingReport.FormatIssues()` and feeds it plus the
  rejected draft into the `episodes_refine` prompt for **exactly one** corrective rewrite.
  `episodes_refine` is internal to this stage, not a top-level stage.
- **3 deterministic tools** (`internal/tools/`): `GetWinningTropes` (Chinese home tropes,
  `tropes.go`), `GetProductCatalog` (Ashley SKUs, `catalog.go`), `ValidatePacing` (the gate,
  `pacing.go`).
- **Multimodal**: `Brief.Images` (`model.Image{mimeType, data=raw base64, label}`) are attached
  only to the **concept / placements / hero** stages (the `withImages` flag in `call`) to limit
  token cost.
- **Provider abstraction** (`internal/llm/provider.go`):
  `Provider.GenerateJSON(ctx, stage, prompt, images, schema)`. Gemini and Mock/DemoMock implement it.
- **Persistence** (`internal/store/store.go`): single `plans` table (id, created_at, genre, title,
  episodes, brief jsonb, plan jsonb). Server saves asynchronously after the plan has streamed out.
- **Auth** (`internal/auth/auth.go`): `/api/login` returns a random in-memory token; a Bearer
  middleware guards `/api/generate`, `/api/history`, `/api/history/{id}`. `/api/health` is open.
- **Two surfaces, one orchestrator**: `cmd/server` (HTTP+SSE) and `cmd/cli` both call
  `agent.New(provider, emit).Run(ctx, brief)`.

### Conventions when extending

- **Adding/changing a stage**: implement the `Stage` interface (`Name() string`,
  `Run(ctx, *PlanState) error`), register it in `AllStages()`, add a matching embedded prompt
  template, and extend `stagePayload()` (orchestrator) for SSE streaming.
- **Prompts** live in `internal/prompts/*.tmpl`, embedded via `embed.go`. Each prompt is Chinese,
  ends with the exact JSON shape of its target struct in `internal/model/plan.go`, and instructs
  Chinese-language output. Structured JSON output depends on prompt and struct staying in sync.
- **Domain types** are in `internal/model/` (`plan.go` = Brief/Plan/Episode/Image/...,
  `events.go` = SSE events). `frontend/lib/types.ts` is a hand-maintained TypeScript mirror —
  update both together.
- **Pacing rules are deterministic and pinned by tests** (`internal/tools/pacing_test.go`).
  Changing the gate thresholds means updating those tests.

### Known stale spots (code, not docs)

- `model.Brief.Market`/`Language` struct **comments** still say `fixed "US"/"English"`, but
  `ApplyDefaults()` defaults to 中国 / 中文.
- `cmd/cli` flag **defaults** are still English placeholders ("home makeover revenge", 12, 90).
- `frontend/app/page.tsx` masthead copy still reads "面向美国市场" (should be 国内市场).
