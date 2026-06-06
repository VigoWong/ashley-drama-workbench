# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

短剧生产工作台 (Short-Drama Production Workbench): a one-line brief becomes an 8-block,
Chinese-language short-drama production plan that markets Ashley furniture for the **China
domestic market** (抖音 / 快手 / 红果短剧, vertical 9:16). A Go backend runs the orchestration;
a Next.js + Tailwind frontend drives a four-step wizard (填需求 → 选立意 → 生成 → 方案).
The README is the authoritative deep reference (architecture, business rationale, run guide,
design trade-offs) — read it for the "why". This file covers commands and the structural
facts you need before editing. Design docs / plans live in `docs/superpowers/` (historical
process artifacts, English).

## Commands

Backend commands run from `backend/` (Go 1.23, module `github.com/ashley/drama-workbench`):

```bash
make test                                  # go test ./...  (entire suite runs with NO API key / NO DB)
go test ./internal/tools/ -run TestPacingMissingHookFails   # single test
go test ./internal/agent/ -v               # one package, verbose
make server                                # HTTP + SSE on :8080 (PORT overrides)
make cli ARGS="-episodes 5 -format json"   # CLI; pass flags via ARGS
go run ./cmd/cli -req "家装改造逆袭，主打逆袭打脸，植入 Ashley 客厅沙发" -episodes 5 -secs 30   # run CLI directly (-req defaults to a full Chinese requirement)
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
   `GEMINI_MODEL` (default gemini-2.5-flash), `IMAGEN_MODEL` (default imagen-3.0-generate-002,
   used for the visuals stage). Mints OAuth tokens, hits the regional endpoint. **Only Vertex
   supports image generation** (Imagen `:predict`).
2. **AI Studio** — when `GEMINI_API_KEY` is set. `GEMINI_MODEL` (default gemini-2.0-flash),
   optional `GEMINI_BASE_URL` proxy. Text only — no image generation.
3. **DemoMock** — no credentials: emits a fixed, complete Chinese sample plan (incl. a propose
   fixture for the 立意选型 step). The CLI, server, and the **entire test suite** run this way;
   you never need a real key to develop or test. The demo mock ignores the requested episode
   count (fixed ~12-ep season) and produces no images.

Gemini provider (both modes, `internal/llm/gemini.go`): structured JSON output, 3-attempt
retry + JSON-repair; Vertex content carries `role`; `thinkingBudget=0` for 2.5 models. In
Vertex mode it also implements `ImageProvider.GenerateImage` (Imagen `:predict`, 9:16);
AI Studio / Mock return `ErrImagesUnsupported`. `vertex-sa.json` / `run-dev.sh` / `.env.local`
are gitignored — **never commit secrets**.

Auth env (`internal/auth/auth.go`): `AUTH_USERNAME` / `AUTH_PASSWORD` (default admin/admin).
`DATABASE_URL` enables history persistence; unset/unreachable → graceful degradation.
`PORT` (default 8080).

## Architecture essentials

A linear, orchestrator-driven pipeline of single-purpose LLM stages, grounded and gated
by deterministic Go. Core design stance: **the LLM creates, deterministic Go verifies,
a feedback loop rewrites.**

- **Orchestrator** (`internal/agent/orchestrator.go`): `Run(ctx, brief)` runs `AllStages()` in
  order from the concept stage; `RunFrom(ctx, plan, fromStage, only, note)` reruns a subset (only
  that stage, or that stage onward) against an existing plan — the core of the refine / 选立意 flows.
  Threads a shared `*PlanState` (`internal/agent/stage.go` — holds `*model.Plan`, the `Provider`,
  and a transient refine `Note`), emits an `Event` per stage via an `Emitter` (server → SSE, CLI →
  stdout). **Retries a single stage's transient failures up to `maxStageAttempts` (3)** with linear
  backoff (attempt × 700ms), honoring ctx cancel.
- **Propose** (`internal/agent/propose.go`): `Propose(ctx, provider, brief)` is a single
  lightweight LLM call returning 2-3 differentiated 立意方向 (candidate `Concept`s), trimmed to
  `maxProposals` (3). It does NOT run the pipeline — the user picks/tweaks one, then `/api/generate`
  runs `RunFrom("bible", …)` with the chosen concept.
- **8 top-level stages** (`internal/agent/stages.go`, `AllStages()`): concept → bible → characters →
  episodes → placements → hero → production_distribution → **visuals**. `IsStage(name)` validates a
  refine's `fromStage`.
- **The genuinely agentic part is the episodes gate.** `EpisodeStage.Run` generates episodes,
  then calls `tools.ValidatePacing` (pure Go: every episode needs a non-empty hook + cliffhanger;
  payoff density ≥ 60%). On failure it renders `PacingReport.FormatIssues()` and feeds it plus the
  rejected draft into the `episodes_refine` prompt for **exactly one** corrective rewrite.
  `episodes_refine` is internal to this stage, not a top-level stage.
- **VisualStage** (`internal/agent/stages.go`): best-effort AI concept art (1 key-art poster + ≤2
  hero-scene stills, `maxVisuals` 3) via the provider's optional `ImageProvider`. If the provider
  can't make images it's a no-op; individual failures are logged and skipped; it always returns nil
  so visuals never abort the pipeline. Results land in `Plan.Visuals` (base64).
- **3 deterministic tools** (`internal/tools/`): `GetWinningTropes` (Chinese home tropes,
  `tropes.go`), `GetProductCatalog` (Ashley SKUs, `catalog.go`), `ValidatePacing` (the gate,
  `pacing.go`).
- **Multimodal**: `Brief.Images` (`model.Image{mimeType, data=raw base64, label}`) are attached
  only to the **concept / placements / hero** stages (the `withImages` flag in `call`) to limit
  token cost.
- **Provider abstraction** (`internal/llm/provider.go`):
  `Provider.GenerateJSON(ctx, stage, prompt, images, schema)`; optional `ImageProvider.GenerateImage(ctx, prompt)`
  + `ErrImagesUnsupported`. Gemini and Mock/DemoMock implement both (Mock's GenerateImage always
  reports unsupported).
- **Persistence** (`internal/store/store.go`): single `plans` table (id, created_at, genre, title,
  episodes, brief jsonb, plan jsonb — the `genre` column name is kept to avoid a migration and now
  holds a truncated `Brief.Requirement` snippet used only as the history-list label). Server saves
  asynchronously after `/api/generate` streams out. Refines are NOT persisted (interactive drafts).
- **Auth** (`internal/auth/auth.go`): `/api/login` returns a random in-memory token; a Bearer
  middleware guards `/api/assist`, `/api/propose`, `/api/generate`, `/api/refine`, `/api/history`,
  `/api/history/{id}` (GET + DELETE). `/api/health` is open. Default creds admin/admin
  (`AUTH_USERNAME`/`AUTH_PASSWORD`).
- **Two surfaces, one orchestrator**: `cmd/server` (HTTP+SSE: assist/propose/generate/refine/history)
  and `cmd/cli` (full `Run` only) both call `agent.New(provider, emit)`.

### HTTP endpoints (`cmd/server/main.go`)

- `POST /api/login` `{username,password}` → `{token}` (open)
- `POST /api/assist` `{requirement,episodes,episodeSecs,images}` → `{requirement}` (Bearer; plain
  JSON, not SSE; expands a rough idea into one full 中文需求; image-aware; not persisted)
- `POST /api/propose` `Brief` → `{concepts:[…2-3…]}` (Bearer; plain JSON, not SSE; not persisted)
- `POST /api/generate` `Brief` (+ optional `concept`) → SSE (Bearer; with `concept` skips the concept
  stage and runs from bible; persisted)
- `POST /api/refine` `{plan,fromStage,only,note}` → SSE (Bearer; rerun via `RunFrom`; NOT persisted)
- `GET /api/history` / `GET /api/history/{id}` / `DELETE /api/history/{id}` (Bearer; empty/404 without DB)
- `GET /api/health` (open)

### Conventions when extending

- **Adding/changing a stage**: implement the `Stage` interface (`Name() string`,
  `Run(ctx, *PlanState) error`), register it in `AllStages()`, add a matching embedded prompt
  template, and extend `stagePayload()` (orchestrator) for SSE streaming.
- **Prompts** live in `internal/prompts/*.tmpl`, embedded via `embed.go`. Each prompt is Chinese,
  ends with the exact JSON shape of its target struct in `internal/model/plan.go`, and instructs
  Chinese-language output. Structured JSON output depends on prompt and struct staying in sync.
- **Domain types** are in `internal/model/` (`plan.go` = Brief/Plan/Concept/Episode/Image/Visual/...,
  `events.go` = SSE events). `frontend/lib/types.ts` is a hand-maintained TypeScript mirror (incl.
  `Concept` / `Visual` / `RefineReq` / `ProposeResp`) — update both together.
- **Pacing rules are deterministic and pinned by tests** (`internal/tools/pacing_test.go`).
  Changing the gate thresholds means updating those tests.

### Notes

- **`Brief` uses a single `Requirement` string** — the former `Genre` + `BrandFocus` were merged
  into one free-text 生成需求 paragraph (`internal/model/plan.go`). The front-end 提示词助手 lets
  users scaffold it from template/chips or expand it via `POST /api/assist` (`internal/agent/assist.go`,
  `internal/prompts/assist.tmpl`), which is image-aware (reuses the same multimodal anchoring as
  propose/generate). Assist is server-only (no CLI path).
- `cmd/cli` runs the full `Run` only (no propose / refine / assist). Flag defaults are Chinese
  (`-req "<一段完整需求>" -episodes 5 -secs 30`); `-format markdown|json`, `-out <file>`.
- `frontend/lib/types.ts` `Brief` intentionally omits `language` (server defaults it); the mirror
  is deliberately partial there.
- Imagen renders in-image Chinese text poorly, so visuals prompts request "no text/watermark/logo".
