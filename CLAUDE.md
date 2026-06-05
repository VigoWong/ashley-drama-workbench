# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Ashley is an AI-agent short-drama production workbench: a one-line brief becomes an
8-block, furniture-brand-marketing short-drama production plan. A Go backend runs the
orchestration; a Next.js frontend drives it. The README is the authoritative deep
reference (architecture diagram, business rationale, design trade-offs) — read it for
the "why". This file covers commands and the structural facts you need before editing.

## Commands

All backend commands run from `backend/` (Go 1.23, module `github.com/ashley/drama-workbench`):

```bash
make test                              # go test ./...  (entire suite runs with NO API key)
go test ./internal/tools/ -run TestPacingMissingHookFails   # single test
go test ./internal/agent/ -v           # one package, verbose
make server                            # HTTP + SSE on :8080 (POST /api/generate)
make cli                               # CLI; pass flags via ARGS, e.g. make cli ARGS="-episodes 3 -format json"
go run ./cmd/cli -episodes 3 -format markdown   # run CLI directly
make build                             # builds bin/server and bin/cli
```

Frontend (from `frontend/`, Next.js, npm):

```bash
npm install && npm run dev   # http://localhost:3000, talks to :8080 by default
npm run build                # production build
npm run lint                 # eslint
```

`NEXT_PUBLIC_API` overrides the backend URL the UI calls.

## Keyless operation (important for dev & tests)

With no `GEMINI_API_KEY`, `llm.NewFromEnv` (`backend/internal/llm/factory.go`) returns a
`DemoMock` provider that emits a fixed, complete sample plan. The CLI, server, and the
**entire test suite** run this way — you never need a real key to develop or test. The
demo mock ignores the requested episode count (it demonstrates the chain, not variation).
With a key set, you get `GeminiProvider` (REST, structured JSON, 3-attempt retry + JSON-repair).

## Architecture essentials

A linear, orchestrator-driven pipeline of single-purpose LLM stages, grounded and gated
by deterministic Go. The core design stance: **the LLM creates, deterministic Go verifies,
a feedback loop rewrites.**

- **Orchestrator** (`internal/agent/orchestrator.go`): runs `AllStages()` in order,
  threads a shared `*PlanState` (`internal/agent/stage.go` — holds `*model.Plan` + the
  `Provider`), and emits an `Event` per stage via an `Emitter` (server → SSE, CLI → stdout).
- **7 top-level stages** (`internal/agent/stages.go`): concept → bible → characters →
  episodes → placements → hero → production_distribution.
- **The one genuinely agentic part is the episodes gate.** `EpisodeStage.Run` generates
  episodes, then calls `tools.ValidatePacing` (pure Go: every episode needs a non-empty
  hook + cliffhanger; payoff density ≥ 60%). On failure it renders `PacingReport.FormatIssues()`
  and feeds it plus the rejected draft into the `episodes_refine` prompt for **exactly one**
  corrective rewrite. `episodes_refine` is internal to this stage, not a top-level stage.
- **3 deterministic tools** (`internal/tools/`): `GetWinningTropes` (trope grounding,
  `tropes.go`), `GetProductCatalog` (Ashley SKU grounding, `catalog.go`), `ValidatePacing`
  (the gate, `pacing.go`).
- **Provider abstraction** (`internal/llm/provider.go`): every stage calls
  `Provider.GenerateJSON(ctx, stage, prompt, schema)`. Gemini and Mock/DemoMock implement it.
- **Two surfaces, one orchestrator**: `cmd/server` and `cmd/cli` both call
  `agent.New(provider, emit).Run(ctx, brief)`.

### Conventions when extending

- **Adding/changing a stage**: implement the `Stage` interface (`Name() string`,
  `Run(ctx, *PlanState) error`), register it in `AllStages()`, and add a matching embedded
  prompt template. Each stage reads/writes fields on the shared `PlanState.Plan`.
- **Prompts** live in `internal/prompts/*.tmpl`, embedded via `embed.go`. Each prompt ends
  with the exact JSON shape of its target struct in `internal/model/plan.go`; structured
  JSON output depends on prompt and struct staying in sync.
- **Domain types** are in `internal/model/` (`plan.go` = Brief/Plan/Episode/...,
  `events.go` = SSE events). `frontend/lib/types.ts` is a hand-maintained TypeScript mirror
  of these — update both together.
- **Pacing rules are deterministic and pinned by tests** (`internal/tools/pacing_test.go`).
  Changing the gate thresholds means updating those tests.

### Out of scope (deliberate MVP omissions)

No auth, no database, no persistence (generation is stateless), no per-stage human editing,
single-market (US/English/home vertical). See README §8 before adding any of these.
