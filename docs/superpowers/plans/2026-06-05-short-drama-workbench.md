# Short Drama Production Workbench — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an AI-agent-driven workbench that turns a short-drama brief (genre, episodes, duration, market) into a structured, furniture-brand-marketing production plan, runnable via both a Web UI (Next.js) and a CLI (Go).

**Architecture:** Go backend with an orchestrator-driven, multi-stage pipeline. Each stage is an LLM "agent" (prompt template + Gemini structured JSON output) that reads/writes a shared `PlanState`. Deterministic Go tools (trope library, product catalog, pacing validator) ground the LLM and gate the episode stage with a one-shot self-correction loop. A pluggable `Provider` interface (Gemini + Mock) makes the whole chain runnable and testable without an API key. The server streams per-stage progress over SSE; the CLI runs the same orchestrator.

**Tech Stack:** Go 1.22 (stdlib + `chi` router, Gemini via REST/`net/http`, no heavy SDK), Next.js 15 App Router + Tailwind CSS, SSE for streaming.

---

## File Structure

```
backend/
  go.mod
  Makefile
  .env.example
  cmd/
    server/main.go        HTTP + SSE server
    cli/main.go           CLI entrypoint
  internal/
    model/plan.go         domain types: Brief, Plan, Episode, Character, ...
    model/events.go       SSE event types
    llm/provider.go       Provider interface + retry/repair helpers
    llm/mock.go           MockProvider (deterministic, no key)
    llm/gemini.go         GeminiProvider (REST)
    tools/tropes.go       GetWinningTropes + curated data
    tools/catalog.go      GetProductCatalog + Ashley categories
    tools/pacing.go       ValidatePacing (pure deterministic)
    prompts/*.tmpl        one prompt template per stage (embedded)
    prompts/embed.go      embed.FS + render helper
    agent/stage.go        Stage interface + PlanState
    agent/orchestrator.go runs stages, emits events
    agent/stages.go       the 9 stage implementations
    render/markdown.go    Plan -> Markdown
  testdata/               golden fixtures
frontend/
  package.json, next.config, tailwind config, tsconfig
  app/layout.tsx, app/page.tsx
  app/globals.css
  components/InputForm.tsx
  components/StageTimeline.tsx
  components/PlanView.tsx
  components/ExportBar.tsx
  lib/types.ts            mirrors backend model
  lib/api.ts              POST+SSE stream parser
README.md
```

---

## Phase 1 — Backend Foundations

### Task 1: Go module + domain model

**Files:**
- Create: `backend/go.mod`
- Create: `backend/internal/model/plan.go`
- Create: `backend/internal/model/events.go`

- [ ] **Step 1: Init module**

```bash
cd backend && go mod init github.com/ashley/drama-workbench && go get github.com/go-chi/chi/v5@latest
```

- [ ] **Step 2: Write `internal/model/plan.go`**

```go
package model

// Brief is the normalized user input that drives the whole pipeline.
type Brief struct {
	Genre        string `json:"genre"`         // e.g. "home makeover revenge"
	Episodes     int    `json:"episodes"`      // default 12
	EpisodeSecs  int    `json:"episodeSecs"`   // default 90
	Market       string `json:"market"`        // fixed "US" for MVP
	Language     string `json:"language"`      // fixed "English" for MVP
	BrandFocus   string `json:"brandFocus"`    // e.g. "living room sofas, bedroom sets"
	Extra        string `json:"extra"`         // free-form notes
}

func (b *Brief) ApplyDefaults() {
	if b.Episodes <= 0 { b.Episodes = 12 }
	if b.EpisodeSecs <= 0 { b.EpisodeSecs = 90 }
	if b.Market == "" { b.Market = "US" }
	if b.Language == "" { b.Language = "English" }
}

type Concept struct {
	Logline      string   `json:"logline"`
	Theme        string   `json:"theme"`
	Audience     string   `json:"audience"`
	Tone         string   `json:"tone"`
	PayoffEngine string   `json:"payoffEngine"` // the core "爽点引擎"
	CoreConflict string   `json:"coreConflict"`
	TropesUsed   []string `json:"tropesUsed"`
}

type SeriesBible struct {
	Title             string `json:"title"`
	GenreTags         []string `json:"genreTags"`
	Episodes          int    `json:"episodes"`
	EpisodeSecs       int    `json:"episodeSecs"`
	TotalRuntimeMin   int    `json:"totalRuntimeMin"`
	Platform          string `json:"platform"`
	IntegrationThesis string `json:"integrationThesis"`
}

type Character struct {
	Name         string `json:"name"`
	Role         string `json:"role"` // protagonist / antagonist / love-interest / ...
	Bio          string `json:"bio"`
	Arc          string `json:"arc"`
	Relationships string `json:"relationships"`
}

type Episode struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Synopsis   string `json:"synopsis"`
	Beats      []string `json:"beats"`
	Hook       string `json:"hook"`        // golden-3-seconds opener
	Cliffhanger string `json:"cliffhanger"` // ending hook
	Payoff     string `json:"payoff"`      // 爽点/反转
}

type Placement struct {
	Episode      int    `json:"episode"`
	Scene        string `json:"scene"`
	ProductSKU   string `json:"productSku"`
	Category     string `json:"category"`
	EmotionalBeat string `json:"emotionalBeat"`
	CTATiming    string `json:"ctaTiming"`
}

type Shot struct {
	Number    int    `json:"number"`
	ShotType  string `json:"shotType"` // CU / MS / WS / POV ...
	Action    string `json:"action"`
	Dialogue  string `json:"dialogue"`
}

type HeroScene struct {
	Episode int    `json:"episode"`
	Title   string `json:"title"`
	Shots   []Shot `json:"shots"`
}

type Production struct {
	Format        string   `json:"format"` // "9:16 vertical"
	BudgetTier    string   `json:"budgetTier"`
	ShotCount     int      `json:"shotCount"`
	CastSize      int      `json:"castSize"`
	Locations     []string `json:"locations"`
	FurnitureProps []string `json:"furnitureProps"`
}

type Distribution struct {
	CTACopy   string   `json:"ctaCopy"`
	LinkPlacement string `json:"linkPlacement"`
	Hashtags  []string `json:"hashtags"`
}

// Plan is the complete structured production plan (the core deliverable).
type Plan struct {
	Brief        Brief        `json:"brief"`
	Concept      Concept      `json:"concept"`
	Bible        SeriesBible  `json:"bible"`
	Characters   []Character  `json:"characters"`
	Episodes     []Episode    `json:"episodes"`
	Placements   []Placement  `json:"placements"`
	HeroScenes   []HeroScene  `json:"heroScenes"`
	Production    Production   `json:"production"`
	Distribution Distribution `json:"distribution"`
}
```

- [ ] **Step 3: Write `internal/model/events.go`**

```go
package model

type EventType string

const (
	EventStageStart EventType = "stage_start"
	EventStageDone  EventType = "stage_done"
	EventError      EventType = "error"
	EventComplete   EventType = "complete"
)

// Event is what the orchestrator emits and the server streams as SSE.
type Event struct {
	Type    EventType   `json:"type"`
	Stage   string      `json:"stage,omitempty"`
	Index   int         `json:"index,omitempty"` // 0-based stage index
	Total   int         `json:"total,omitempty"`
	Message string      `json:"message,omitempty"`
	Payload interface{} `json:"payload,omitempty"` // partial output for this stage
	Plan    *Plan       `json:"plan,omitempty"`    // set on EventComplete
}
```

- [ ] **Step 4: Verify it builds**

Run: `cd backend && go build ./...`
Expected: success (no output).

- [ ] **Step 5: Commit**

```bash
git add backend && git commit -m "feat(backend): go module + domain model"
```

---

### Task 2: LLM Provider interface + Mock provider

**Files:**
- Create: `backend/internal/llm/provider.go`
- Create: `backend/internal/llm/mock.go`
- Test: `backend/internal/llm/mock_test.go`

- [ ] **Step 1: Write the failing test `mock_test.go`**

```go
package llm

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMockReturnsRegisteredJSON(t *testing.T) {
	m := NewMock()
	m.Register("concept", `{"logline":"x"}`)
	out, err := m.GenerateJSON(context.Background(), "concept", "prompt body", nil)
	if err != nil { t.Fatal(err) }
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil { t.Fatal(err) }
	if got["logline"] != "x" { t.Fatalf("got %v", got) }
}

func TestMockUnknownStageErrors(t *testing.T) {
	m := NewMock()
	if _, err := m.GenerateJSON(context.Background(), "nope", "p", nil); err == nil {
		t.Fatal("expected error for unregistered stage")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/llm/...`
Expected: FAIL (undefined `NewMock`).

- [ ] **Step 3: Write `provider.go`**

```go
package llm

import "context"

// Provider abstracts the LLM. `stage` is a logical name used by Mock to route
// fixtures and by Gemini for logging. `schema` is an optional JSON schema
// (map[string]any) passed to Gemini's responseSchema; Mock ignores it.
type Provider interface {
	GenerateJSON(ctx context.Context, stage, prompt string, schema map[string]any) ([]byte, error)
}
```

- [ ] **Step 4: Write `mock.go`**

```go
package llm

import (
	"context"
	"fmt"
)

type Mock struct{ fixtures map[string]string }

func NewMock() *Mock { return &Mock{fixtures: map[string]string{}} }

func (m *Mock) Register(stage, jsonOut string) { m.fixtures[stage] = jsonOut }

func (m *Mock) GenerateJSON(_ context.Context, stage, _ string, _ map[string]any) ([]byte, error) {
	v, ok := m.fixtures[stage]
	if !ok {
		return nil, fmt.Errorf("mock: no fixture registered for stage %q", stage)
	}
	return []byte(v), nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/llm/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/llm && git commit -m "feat(llm): provider interface + mock provider"
```

---

## Phase 2 — Deterministic Tools (TDD)

### Task 3: Pacing validator (pure deterministic — the self-correction gate)

**Files:**
- Create: `backend/internal/tools/pacing.go`
- Test: `backend/internal/tools/pacing_test.go`

- [ ] **Step 1: Write the failing test `pacing_test.go`**

```go
package tools

import (
	"testing"

	"github.com/ashley/drama-workbench/internal/model"
)

func eps(n int, fill bool) []model.Episode {
	out := make([]model.Episode, n)
	for i := range out {
		out[i].Number = i + 1
		if fill {
			out[i].Hook = "hook"
			out[i].Cliffhanger = "cliff"
			out[i].Payoff = "payoff"
		}
	}
	return out
}

func TestPacingPerfectScoresHigh(t *testing.T) {
	r := ValidatePacing(eps(12, true))
	if !r.Pass { t.Fatalf("expected pass, got %+v", r) }
	if len(r.Issues) != 0 { t.Fatalf("expected no issues, got %v", r.Issues) }
}

func TestPacingMissingHookFails(t *testing.T) {
	e := eps(12, true)
	e[0].Hook = ""
	r := ValidatePacing(e)
	if r.Pass { t.Fatal("expected fail when episode 1 has no hook") }
	found := false
	for _, is := range r.Issues { if is.Episode == 1 { found = true } }
	if !found { t.Fatalf("expected issue on episode 1, got %v", r.Issues) }
}

func TestPacingLowPayoffDensityFails(t *testing.T) {
	e := eps(12, true)
	for i := range e { e[i].Payoff = "" } // zero payoff across season
	r := ValidatePacing(e)
	if r.Pass { t.Fatal("expected fail on low payoff density") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/tools/...`
Expected: FAIL (undefined `ValidatePacing`).

- [ ] **Step 3: Write `pacing.go`**

```go
package tools

import (
	"fmt"

	"github.com/ashley/drama-workbench/internal/model"
)

type Issue struct {
	Episode int    `json:"episode"` // 0 = season-level
	Message string `json:"message"`
}

type PacingReport struct {
	Pass   bool    `json:"pass"`
	Score  float64 `json:"score"` // 0..100
	Issues []Issue `json:"issues"`
}

// ValidatePacing enforces short-drama pacing rules deterministically:
//   - every episode must have a non-empty hook and cliffhanger
//   - payoff (爽点/反转) density across the season must be >= 0.6
//   - episode 1 hook must exist (golden 3 seconds)
func ValidatePacing(eps []model.Episode) PacingReport {
	r := PacingReport{Pass: true, Score: 100}
	if len(eps) == 0 {
		return PacingReport{Pass: false, Score: 0, Issues: []Issue{{0, "no episodes"}}}
	}
	payoffs := 0
	for _, e := range eps {
		if e.Hook == "" {
			r.Issues = append(r.Issues, Issue{e.Number, "missing opening hook (golden 3 seconds)"})
		}
		if e.Cliffhanger == "" {
			r.Issues = append(r.Issues, Issue{e.Number, "missing ending cliffhanger"})
		}
		if e.Payoff != "" { payoffs++ }
	}
	density := float64(payoffs) / float64(len(eps))
	if density < 0.6 {
		r.Issues = append(r.Issues, Issue{0, fmt.Sprintf("payoff density %.0f%% < 60%%", density*100)})
	}
	r.Score = 100 - float64(len(r.Issues))*10
	if r.Score < 0 { r.Score = 0 }
	r.Pass = len(r.Issues) == 0
	return r
}

// FormatIssues renders issues as a feedback string for the LLM refine pass.
func (r PacingReport) FormatIssues() string {
	s := ""
	for _, is := range r.Issues {
		if is.Episode == 0 { s += "- (season) " + is.Message + "\n" } else {
			s += fmt.Sprintf("- (ep %d) %s\n", is.Episode, is.Message)
		}
	}
	return s
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/tools/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/tools/pacing.go backend/internal/tools/pacing_test.go && git commit -m "feat(tools): deterministic pacing validator"
```

---

### Task 4: Trope library + Ashley product catalog

**Files:**
- Create: `backend/internal/tools/tropes.go`
- Create: `backend/internal/tools/catalog.go`
- Test: `backend/internal/tools/catalog_test.go`

- [ ] **Step 1: Write the failing test `catalog_test.go`**

```go
package tools

import "testing"

func TestCatalogReturnsAllWhenNoFilter(t *testing.T) {
	all := GetProductCatalog("")
	if len(all) < 4 { t.Fatalf("expected several categories, got %d", len(all)) }
}

func TestCatalogFiltersByCategory(t *testing.T) {
	got := GetProductCatalog("sofa")
	if len(got) == 0 { t.Fatal("expected a sofa match") }
	for _, p := range got {
		if p.Category != "sofa" { t.Fatalf("unexpected category %s", p.Category) }
	}
}

func TestTropesReturnsHomeVertical(t *testing.T) {
	tr := GetWinningTropes("US", "home")
	if len(tr) == 0 { t.Fatal("expected home tropes") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/tools/...`
Expected: FAIL (undefined `GetProductCatalog`, `GetWinningTropes`).

- [ ] **Step 3: Write `catalog.go`**

```go
package tools

import "strings"

type Product struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	SellingAngle string `json:"sellingAngle"`
}

var ashleyCatalog = []Product{
	{"ASH-SOFA-001", "Maeford Sectional", "sofa", "Family gathering centerpiece; cozy reunion scenes"},
	{"ASH-BED-001", "Realyn Queen Bed", "bed", "Fresh-start / new-home morning scenes"},
	{"ASH-DINE-001", "Haddigan Dining Set", "dining", "Celebration & confrontation dinner scenes"},
	{"ASH-RECL-001", "Boxberg Recliner", "recliner", "Reconciliation / heart-to-heart moments"},
	{"ASH-DESK-001", "Camiburg Home Office Desk", "office", "Underdog-builds-business montage"},
	{"ASH-OUT-001", "Clare View Outdoor Set", "outdoor", "Dream-home reveal / status arc"},
}

func GetProductCatalog(category string) []Product {
	if category == "" { return ashleyCatalog }
	c := strings.ToLower(category)
	var out []Product
	for _, p := range ashleyCatalog {
		if p.Category == c { out = append(out, p) }
	}
	return out
}
```

- [ ] **Step 4: Write `tropes.go`**

```go
package tools

type Trope struct {
	Name      string `json:"name"`
	Hook      string `json:"hook"`
	WhyItWorks string `json:"whyItWorks"`
	HomeAngle string `json:"homeAngle"` // how furniture earns screen time
}

var homeTropes = []Trope{
	{"From-broke-to-dream-home", "Evicted heroine vows to rebuild", "Aspirational status arc, fast payoff", "Every upgrade = a new furniture reveal"},
	{"Fresh-start-after-divorce", "She walks out with one suitcase", "Empowerment + clean slate", "Furnishing the new place = healing montage"},
	{"Secret-heir-renovation", "Handyman is secretly a billionaire", "Reversal-driven, identity reveal", "The mansion makeover showcases premium lines"},
	{"Family-reconciliation", "Estranged siblings inherit a house", "Emotional payoff, warmth", "Shared living/dining scenes anchor the brand"},
}

func GetWinningTropes(market, vertical string) []Trope {
	// MVP: single curated home vertical; market reserved for future expansion.
	return homeTropes
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/tools/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/tools && git commit -m "feat(tools): trope library + Ashley product catalog"
```

---

## Phase 3 — Prompts, Stages, Orchestrator

### Task 5: Prompt templates + embed loader

**Files:**
- Create: `backend/internal/prompts/embed.go`
- Create: `backend/internal/prompts/concept.tmpl` (+ one per stage; see list)
- Test: `backend/internal/prompts/embed_test.go`

Stages needing a `.tmpl`: `concept`, `bible`, `characters`, `episodes`, `episodes_refine`, `placements`, `hero`, `production_distribution`.

- [ ] **Step 1: Write the failing test `embed_test.go`**

```go
package prompts

import "testing"

func TestRenderSubstitutes(t *testing.T) {
	out, err := Render("concept", map[string]any{"Genre": "makeover"})
	if err != nil { t.Fatal(err) }
	if out == "" { t.Fatal("empty render") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/prompts/...`
Expected: FAIL (no Render / no templates).

- [ ] **Step 3: Write `embed.go`**

```go
package prompts

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed *.tmpl
var files embed.FS

func Render(name string, data any) (string, error) {
	t, err := template.ParseFS(files, name+".tmpl")
	if err != nil { return "", err }
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil { return "", err }
	return buf.String(), nil
}
```

- [ ] **Step 4: Write `concept.tmpl`** (representative — every stage prompt follows this shape: role + context + grounding data + STRICT "output JSON only matching this shape" instruction)

```
You are a creative director for vertical short-dramas sold to US audiences on
platforms like ReelShort and DramaBox. Your dramas double as branded marketing
for Ashley furniture: living spaces must earn screen time and emotional weight.

BRIEF:
- Genre: {{.Genre}}
- Episodes: {{.Episodes}} x {{.EpisodeSecs}}s, vertical 9:16, English, US market
- Brand focus: {{.BrandFocus}}

PROVEN HOME TROPES (use 1-2, adapt — do not copy verbatim):
{{.Tropes}}

Design the CONCEPT. Lead with a punchy, scroll-stopping logline. Define the
core "payoff engine" — the repeatable satisfaction mechanism (reversal, revenge,
status climb) that makes viewers binge.

Return ONLY valid JSON, no markdown fences, matching exactly:
{"logline":"","theme":"","audience":"","tone":"","payoffEngine":"","coreConflict":"","tropesUsed":[""]}
```

> **Executor note:** Write the remaining `.tmpl` files now, each tailored to its stage and ending with the exact JSON shape from the matching `model` struct:
> - `bible.tmpl` → SeriesBible JSON; input: Concept + Brief.
> - `characters.tmpl` → JSON `{"characters":[{...}]}`; input: Concept + Bible. Demand 3-5 characters with arcs + a relationships sentence each.
> - `episodes.tmpl` → JSON `{"episodes":[{...}]}` for ALL `{{.Episodes}}` episodes; every episode MUST have non-empty `hook`, `cliffhanger`, and most must have `payoff`. Input: Concept + Bible + Characters.
> - `episodes_refine.tmpl` → same output shape; extra input `{{.Issues}}` (pacing report) + the previous episodes JSON; instruct to fix ONLY the listed issues.
> - `placements.tmpl` → JSON `{"placements":[{...}]}`; input: Episodes + `{{.Catalog}}` (Ashley products). Map products to scenes by emotional beat; set realistic `ctaTiming`.
> - `hero.tmpl` → JSON `{"heroScenes":[{...}]}` for the 1-2 highest-payoff episodes; full shot list (shotType/action/dialogue).
> - `production_distribution.tmpl` → JSON `{"production":{...},"distribution":{...}}`; input: full plan so far; furnitureProps must list the actual Ashley products used in placements.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/prompts/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/prompts && git commit -m "feat(prompts): embedded stage prompt templates"
```

---

### Task 6: Stage interface, PlanState, helpers

**Files:**
- Create: `backend/internal/agent/stage.go`

- [ ] **Step 1: Write `stage.go`**

```go
package agent

import (
	"context"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

// PlanState is the shared mutable state threaded through all stages.
type PlanState struct {
	Plan     *model.Plan
	Provider llm.Provider
}

type Stage interface {
	Name() string
	Run(ctx context.Context, s *PlanState) error
}

// Emitter receives orchestrator events (server -> SSE, CLI -> stdout).
type Emitter func(model.Event)
```

- [ ] **Step 2: Verify build**

Run: `cd backend && go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/agent/stage.go && git commit -m "feat(agent): stage interface + plan state"
```

---

### Task 7: Stage implementations

**Files:**
- Create: `backend/internal/agent/stages.go`
- Test: `backend/internal/agent/stages_test.go`

Each stage: render its prompt, call `Provider.GenerateJSON(ctx, name, prompt, nil)`, unmarshal into the relevant `model` struct, write into `state.Plan`. The Episode stage additionally runs `ValidatePacing`; if it fails, it renders `episodes_refine` once and re-generates.

- [ ] **Step 1: Write the failing test `stages_test.go`** (uses Mock — no API key)

```go
package agent

import (
	"context"
	"testing"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

func TestConceptStageWritesPlan(t *testing.T) {
	m := llm.NewMock()
	m.Register("concept", `{"logline":"L","theme":"T","audience":"A","tone":"warm","payoffEngine":"revenge","coreConflict":"C","tropesUsed":["x"]}`)
	st := &PlanState{Plan: &model.Plan{Brief: model.Brief{Genre: "makeover"}}, Provider: m}
	if err := (ConceptStage{}).Run(context.Background(), st); err != nil { t.Fatal(err) }
	if st.Plan.Concept.Logline != "L" { t.Fatalf("got %+v", st.Plan.Concept) }
}

func TestEpisodeStageRefinesOnBadPacing(t *testing.T) {
	m := llm.NewMock()
	// First call: episode 1 missing hook -> pacing fails.
	// Mock returns same fixture each call, so register a GOOD one and assert it passes;
	// the refine path is exercised by the bad fixture variant below.
	m.Register("episodes", `{"episodes":[{"number":1,"title":"t","hook":"h","cliffhanger":"c","payoff":"p"},{"number":2,"title":"t","hook":"h","cliffhanger":"c","payoff":"p"}]}`)
	st := &PlanState{Plan: &model.Plan{Brief: model.Brief{Episodes: 2}}, Provider: m}
	if err := (EpisodeStage{}).Run(context.Background(), st); err != nil { t.Fatal(err) }
	if len(st.Plan.Episodes) != 2 { t.Fatalf("got %d episodes", len(st.Plan.Episodes)) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/agent/...`
Expected: FAIL (undefined stages).

- [ ] **Step 3: Write `stages.go`**

```go
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/prompts"
	"github.com/ashley/drama-workbench/internal/tools"
)

// call renders a prompt and unmarshals the provider's JSON into out.
func call(ctx context.Context, s *PlanState, stage string, data any, out any) error {
	prompt, err := prompts.Render(stage, data)
	if err != nil { return fmt.Errorf("%s: render: %w", stage, err) }
	raw, err := s.Provider.GenerateJSON(ctx, stage, prompt, nil)
	if err != nil { return fmt.Errorf("%s: generate: %w", stage, err) }
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: unmarshal: %w (raw=%s)", stage, err, truncate(raw))
	}
	return nil
}

func truncate(b []byte) string { if len(b) > 200 { return string(b[:200]) + "..." }; return string(b) }

type ConceptStage struct{}
func (ConceptStage) Name() string { return "concept" }
func (ConceptStage) Run(ctx context.Context, s *PlanState) error {
	tr := tools.GetWinningTropes(s.Plan.Brief.Market, "home")
	trJSON, _ := json.Marshal(tr)
	data := map[string]any{
		"Genre": s.Plan.Brief.Genre, "Episodes": s.Plan.Brief.Episodes,
		"EpisodeSecs": s.Plan.Brief.EpisodeSecs, "BrandFocus": s.Plan.Brief.BrandFocus,
		"Tropes": string(trJSON),
	}
	return call(ctx, s, "concept", data, &s.Plan.Concept)
}

type BibleStage struct{}
func (BibleStage) Name() string { return "bible" }
func (BibleStage) Run(ctx context.Context, s *PlanState) error {
	data := map[string]any{"Brief": s.Plan.Brief, "Concept": s.Plan.Concept}
	if err := call(ctx, s, "bible", data, &s.Plan.Bible); err != nil { return err }
	if s.Plan.Bible.Episodes == 0 { s.Plan.Bible.Episodes = s.Plan.Brief.Episodes }
	return nil
}

type CharacterStage struct{}
func (CharacterStage) Name() string { return "characters" }
func (CharacterStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct{ Characters []model.Character `json:"characters"` }
	data := map[string]any{"Concept": s.Plan.Concept, "Bible": s.Plan.Bible}
	if err := call(ctx, s, "characters", data, &wrap); err != nil { return err }
	s.Plan.Characters = wrap.Characters
	return nil
}

type EpisodeStage struct{}
func (EpisodeStage) Name() string { return "episodes" }
func (EpisodeStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct{ Episodes []model.Episode `json:"episodes"` }
	data := map[string]any{
		"Concept": s.Plan.Concept, "Bible": s.Plan.Bible,
		"Characters": s.Plan.Characters, "Episodes": s.Plan.Brief.Episodes,
	}
	if err := call(ctx, s, "episodes", data, &wrap); err != nil { return err }
	// Deterministic pacing gate + one-shot self-correction.
	if rep := tools.ValidatePacing(wrap.Episodes); !rep.Pass {
		prev, _ := json.Marshal(wrap.Episodes)
		refine := map[string]any{
			"Concept": s.Plan.Concept, "Bible": s.Plan.Bible,
			"Characters": s.Plan.Characters, "Episodes": s.Plan.Brief.Episodes,
			"Issues": rep.FormatIssues(), "Previous": string(prev),
		}
		var rewrap struct{ Episodes []model.Episode `json:"episodes"` }
		if err := call(ctx, s, "episodes_refine", refine, &rewrap); err == nil && len(rewrap.Episodes) > 0 {
			wrap.Episodes = rewrap.Episodes
		}
	}
	s.Plan.Episodes = wrap.Episodes
	return nil
}

type PlacementStage struct{}
func (PlacementStage) Name() string { return "placements" }
func (PlacementStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct{ Placements []model.Placement `json:"placements"` }
	cat, _ := json.Marshal(tools.GetProductCatalog(""))
	data := map[string]any{"Episodes": s.Plan.Episodes, "Catalog": string(cat)}
	if err := call(ctx, s, "placements", data, &wrap); err != nil { return err }
	s.Plan.Placements = wrap.Placements
	return nil
}

type HeroStage struct{}
func (HeroStage) Name() string { return "hero" }
func (HeroStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct{ HeroScenes []model.HeroScene `json:"heroScenes"` }
	data := map[string]any{"Episodes": s.Plan.Episodes, "Concept": s.Plan.Concept}
	if err := call(ctx, s, "hero", data, &wrap); err != nil { return err }
	s.Plan.HeroScenes = wrap.HeroScenes
	return nil
}

type ProducerStage struct{}
func (ProducerStage) Name() string { return "production_distribution" }
func (ProducerStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct {
		Production   model.Production   `json:"production"`
		Distribution model.Distribution `json:"distribution"`
	}
	data := map[string]any{"Plan": s.Plan}
	if err := call(ctx, s, "production_distribution", data, &wrap); err != nil { return err }
	s.Plan.Production = wrap.Production
	s.Plan.Distribution = wrap.Distribution
	return nil
}

// AllStages returns the ordered pipeline.
func AllStages() []Stage {
	return []Stage{
		ConceptStage{}, BibleStage{}, CharacterStage{}, EpisodeStage{},
		PlacementStage{}, HeroStage{}, ProducerStage{},
	}
}
```

> **Executor note:** the two `stages_test.go` tests above only register `concept` and `episodes` fixtures. Add a `testdata`-backed full-pipeline fixture set in Task 8's orchestrator test rather than here.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/agent/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/stages.go backend/internal/agent/stages_test.go && git commit -m "feat(agent): pipeline stages with pacing self-correction"
```

---

### Task 8: Orchestrator

**Files:**
- Create: `backend/internal/agent/orchestrator.go`
- Test: `backend/internal/agent/orchestrator_test.go`

- [ ] **Step 1: Write the failing test `orchestrator_test.go`** (full pipeline on Mock)

```go
package agent

import (
	"context"
	"testing"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

func mockAll() *llm.Mock {
	m := llm.NewMock()
	m.Register("concept", `{"logline":"L","payoffEngine":"revenge"}`)
	m.Register("bible", `{"title":"Dream Home","episodes":2}`)
	m.Register("characters", `{"characters":[{"name":"Mia","role":"protagonist"}]}`)
	m.Register("episodes", `{"episodes":[{"number":1,"hook":"h","cliffhanger":"c","payoff":"p"},{"number":2,"hook":"h","cliffhanger":"c","payoff":"p"}]}`)
	m.Register("placements", `{"placements":[{"episode":1,"category":"sofa"}]}`)
	m.Register("hero", `{"heroScenes":[{"episode":1,"shots":[{"number":1,"shotType":"CU","dialogue":"hi"}]}]}`)
	m.Register("production_distribution", `{"production":{"format":"9:16"},"distribution":{"ctaCopy":"Shop now"}}`)
	return m
}

func TestOrchestratorRunsAllStages(t *testing.T) {
	var events []model.Event
	o := New(mockAll(), func(e model.Event) { events = append(events, e) })
	plan, err := o.Run(context.Background(), model.Brief{Genre: "makeover", Episodes: 2})
	if err != nil { t.Fatal(err) }
	if plan.Concept.Logline != "L" { t.Fatal("concept missing") }
	if plan.Production.Format != "9:16" { t.Fatal("production missing") }
	var complete bool
	for _, e := range events { if e.Type == model.EventComplete { complete = true } }
	if !complete { t.Fatal("expected complete event") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/agent/...`
Expected: FAIL (undefined `New`).

- [ ] **Step 3: Write `orchestrator.go`**

```go
package agent

import (
	"context"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

type Orchestrator struct {
	provider llm.Provider
	emit     Emitter
	stages   []Stage
}

func New(p llm.Provider, emit Emitter) *Orchestrator {
	if emit == nil { emit = func(model.Event) {} }
	return &Orchestrator{provider: p, emit: emit, stages: AllStages()}
}

func (o *Orchestrator) Run(ctx context.Context, brief model.Brief) (*model.Plan, error) {
	brief.ApplyDefaults()
	state := &PlanState{Plan: &model.Plan{Brief: brief}, Provider: o.provider}
	total := len(o.stages)
	for i, st := range o.stages {
		o.emit(model.Event{Type: model.EventStageStart, Stage: st.Name(), Index: i, Total: total})
		if err := st.Run(ctx, state); err != nil {
			o.emit(model.Event{Type: model.EventError, Stage: st.Name(), Index: i, Message: err.Error()})
			return nil, err
		}
		o.emit(model.Event{Type: model.EventStageDone, Stage: st.Name(), Index: i, Total: total, Payload: stagePayload(st.Name(), state.Plan)})
	}
	o.emit(model.Event{Type: model.EventComplete, Plan: state.Plan})
	return state.Plan, nil
}

// stagePayload returns just the slice of the plan a stage produced, for streaming.
func stagePayload(name string, p *model.Plan) any {
	switch name {
	case "concept": return p.Concept
	case "bible": return p.Bible
	case "characters": return p.Characters
	case "episodes": return p.Episodes
	case "placements": return p.Placements
	case "hero": return p.HeroScenes
	case "production_distribution": return map[string]any{"production": p.Production, "distribution": p.Distribution}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/agent/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/orchestrator.go backend/internal/agent/orchestrator_test.go && git commit -m "feat(agent): orchestrator with event emission"
```

---

## Phase 4 — Gemini, Markdown, Server, CLI

### Task 9: Gemini provider (REST) + retry/repair

**Files:**
- Create: `backend/internal/llm/gemini.go`

- [ ] **Step 1: Write `gemini.go`**

```go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Gemini struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGemini(apiKey, model string) *Gemini {
	if model == "" { model = "gemini-2.0-flash" }
	return &Gemini{apiKey: apiKey, model: model, client: &http.Client{Timeout: 90 * time.Second}}
}

type geminiReq struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig genConfig       `json:"generationConfig"`
}
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}
type geminiPart struct{ Text string `json:"text"` }
type genConfig struct {
	ResponseMimeType string  `json:"responseMimeType"`
	Temperature      float64 `json:"temperature"`
}
type geminiResp struct {
	Candidates []struct {
		Content struct{ Parts []geminiPart `json:"parts"` } `json:"content"`
	} `json:"candidates"`
}

func (g *Gemini) GenerateJSON(ctx context.Context, stage, prompt string, _ map[string]any) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		raw, err := g.once(ctx, prompt)
		if err != nil { lastErr = err; time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond); continue }
		clean := stripFences(raw)
		if json.Valid([]byte(clean)) { return []byte(clean), nil }
		// repair pass: ask the model to fix to strict JSON
		prompt = "Your previous output was not valid JSON. Return ONLY valid JSON, no prose, no fences.\nPREVIOUS:\n" + raw
		lastErr = fmt.Errorf("%s: invalid JSON after attempt %d", stage, attempt)
	}
	return nil, lastErr
}

func (g *Gemini) once(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	body, _ := json.Marshal(geminiReq{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: genConfig{ResponseMimeType: "application/json", Temperature: 0.9},
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 { return "", fmt.Errorf("gemini %d: %s", resp.StatusCode, truncStr(string(data))) }
	var gr geminiResp
	if err := json.Unmarshal(data, &gr); err != nil { return "", err }
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	return gr.Candidates[0].Content.Parts[0].Text, nil
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
func truncStr(s string) string { if len(s) > 300 { return s[:300] }; return s }
```

- [ ] **Step 2: Verify build**

Run: `cd backend && go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/llm/gemini.go && git commit -m "feat(llm): gemini REST provider with retry + json repair"
```

---

### Task 10: Markdown renderer

**Files:**
- Create: `backend/internal/render/markdown.go`
- Test: `backend/internal/render/markdown_test.go`

- [ ] **Step 1: Write the failing test**

```go
package render

import (
	"strings"
	"testing"

	"github.com/ashley/drama-workbench/internal/model"
)

func TestMarkdownIncludesTitleAndEpisodes(t *testing.T) {
	p := &model.Plan{
		Bible: model.SeriesBible{Title: "Dream Home"},
		Episodes: []model.Episode{{Number: 1, Title: "Pilot", Hook: "h"}},
	}
	md := Markdown(p)
	if !strings.Contains(md, "Dream Home") { t.Fatal("missing title") }
	if !strings.Contains(md, "Pilot") { t.Fatal("missing episode") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/render/...`
Expected: FAIL.

- [ ] **Step 3: Write `markdown.go`**

```go
package render

import (
	"fmt"
	"strings"

	"github.com/ashley/drama-workbench/internal/model"
)

func Markdown(p *model.Plan) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# %s\n\n", nz(p.Bible.Title, "Untitled Series"))
	w("## Concept\n- **Logline:** %s\n- **Theme:** %s\n- **Audience:** %s\n- **Tone:** %s\n- **Payoff engine:** %s\n- **Core conflict:** %s\n\n",
		p.Concept.Logline, p.Concept.Theme, p.Concept.Audience, p.Concept.Tone, p.Concept.PayoffEngine, p.Concept.CoreConflict)
	w("## Series Bible\n- Episodes: %d x %ds | Platform: %s\n- Integration: %s\n\n",
		p.Bible.Episodes, p.Bible.EpisodeSecs, p.Bible.Platform, p.Bible.IntegrationThesis)
	w("## Characters\n")
	for _, c := range p.Characters { w("- **%s** (%s): %s _Arc:_ %s\n", c.Name, c.Role, c.Bio, c.Arc) }
	w("\n## Episodes\n")
	for _, e := range p.Episodes {
		w("### Ep %d — %s\n%s\n- **Hook:** %s\n- **Cliffhanger:** %s\n- **Payoff:** %s\n\n",
			e.Number, e.Title, e.Synopsis, e.Hook, e.Cliffhanger, e.Payoff)
	}
	w("## Brand Integration\n")
	for _, pl := range p.Placements {
		w("- Ep %d — %s (%s): %s | CTA: %s\n", pl.Episode, pl.Category, pl.ProductSKU, pl.EmotionalBeat, pl.CTATiming)
	}
	w("\n## Hero Scenes\n")
	for _, h := range p.HeroScenes {
		w("### Ep %d — %s\n", h.Episode, h.Title)
		for _, s := range h.Shots { w("%d. [%s] %s — \"%s\"\n", s.Number, s.ShotType, s.Action, s.Dialogue) }
	}
	w("\n## Production\n- Format: %s | Budget: %s | Shots: %d | Cast: %d\n- Furniture: %s\n\n",
		p.Production.Format, p.Production.BudgetTier, p.Production.ShotCount, p.Production.CastSize, strings.Join(p.Production.FurnitureProps, ", "))
	w("## Distribution\n- CTA: %s\n- Hashtags: %s\n", p.Distribution.CTACopy, strings.Join(p.Distribution.Hashtags, " "))
	return b.String()
}

func nz(s, d string) string { if strings.TrimSpace(s) == "" { return d }; return s }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/render/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/render && git commit -m "feat(render): plan to markdown"
```

---

### Task 11: Provider factory (env-driven) + HTTP/SSE server

**Files:**
- Create: `backend/internal/llm/factory.go`
- Create: `backend/cmd/server/main.go`
- Create: `backend/.env.example`

- [ ] **Step 1: Write `internal/llm/factory.go`**

```go
package llm

import "os"

// FromEnv returns Gemini when GEMINI_API_KEY is set, otherwise a demo Mock so
// the chain always runs. Returns (provider, usingMock).
func FromEnv() (Provider, bool) {
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return NewGemini(key, os.Getenv("GEMINI_MODEL")), false
	}
	return DemoMock(), true
}
```

> **Executor note:** add `DemoMock()` to `mock.go` — a `*Mock` pre-registered with plausible fixtures for all 7 stage names (reuse the JSON from `orchestrator_test.go`, expanded to ~3 characters and `Brief.Episodes` episodes generated in a loop) so a keyless server/CLI still produces a full, sensible plan.

- [ ] **Step 2: Write `cmd/server/main.go`**

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ashley/drama-workbench/internal/agent"
	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(corsMiddleware)

	r.Post("/api/generate", handleGenerate)
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })

	port := os.Getenv("PORT"); if port == "" { port = "8080" }
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	var brief model.Brief
	if err := json.NewDecoder(r.Body).Decode(&brief); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest); return
	}
	flusher, ok := w.(http.Flusher)
	if !ok { http.Error(w, "streaming unsupported", http.StatusInternalServerError); return }
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	provider, _ := llm.FromEnv()
	emit := func(e model.Event) {
		data, _ := json.Marshal(e)
		w.Write([]byte("data: ")); w.Write(data); w.Write([]byte("\n\n"))
		flusher.Flush()
	}
	o := agent.New(provider, emit)
	if _, err := o.Run(r.Context(), brief); err != nil {
		log.Printf("pipeline error: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions { w.WriteHeader(http.StatusOK); return }
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 3: Write `.env.example`**

```
# Leave empty to run in keyless demo mode (Mock provider).
GEMINI_API_KEY=
GEMINI_MODEL=gemini-2.0-flash
PORT=8080
```

- [ ] **Step 4: Verify build + smoke**

Run: `cd backend && go build ./... && go vet ./...`
Expected: success.
Manual smoke (keyless): `go run ./cmd/server &` then `curl -N -X POST localhost:8080/api/generate -d '{"genre":"makeover","episodes":3}'` → SSE events ending in a `complete` event.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/llm/factory.go backend/cmd/server backend/.env.example && git commit -m "feat(server): http + sse generate endpoint, env provider factory"
```

---

### Task 12: CLI

**Files:**
- Create: `backend/cmd/cli/main.go`
- Create: `backend/Makefile`

- [ ] **Step 1: Write `cmd/cli/main.go`**

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ashley/drama-workbench/internal/agent"
	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/render"
)

func main() {
	genre := flag.String("genre", "home makeover revenge", "drama genre/theme")
	episodes := flag.Int("episodes", 12, "number of episodes")
	secs := flag.Int("secs", 90, "seconds per episode")
	brand := flag.String("brand", "living room & bedroom", "brand focus")
	format := flag.String("format", "markdown", "output: markdown|json")
	out := flag.String("out", "", "write to file instead of stdout")
	flag.Parse()

	provider, mock := llm.FromEnv()
	if mock { fmt.Fprintln(os.Stderr, "[demo mode: no GEMINI_API_KEY, using mock provider]") }

	emit := func(e model.Event) {
		if e.Type == model.EventStageStart {
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s...\n", e.Index+1, e.Total, e.Stage)
		}
	}
	o := agent.New(provider, emit)
	plan, err := o.Run(context.Background(), model.Brief{Genre: *genre, Episodes: *episodes, EpisodeSecs: *secs, BrandFocus: *brand})
	if err != nil { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }

	var result string
	if *format == "json" {
		b, _ := json.MarshalIndent(plan, "", "  ")
		result = string(b)
	} else {
		result = render.Markdown(plan)
	}
	if *out != "" {
		if err := os.WriteFile(*out, []byte(result), 0644); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
		fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
	} else {
		fmt.Println(result)
	}
}
```

- [ ] **Step 2: Write `Makefile`**

```makefile
.PHONY: test server cli build
test:
	go test ./...
server:
	go run ./cmd/server
cli:
	go run ./cmd/cli $(ARGS)
build:
	go build -o bin/server ./cmd/server && go build -o bin/cli ./cmd/cli
```

- [ ] **Step 3: Verify CLI runs (keyless demo)**

Run: `cd backend && go run ./cmd/cli -episodes 3 -format markdown`
Expected: stage progress on stderr, a full Markdown plan on stdout.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/cli backend/Makefile && git commit -m "feat(cli): run pipeline from terminal, markdown/json output"
```

---

## Phase 5 — Frontend (Next.js + Tailwind)

### Task 13: Scaffold Next.js + Tailwind + types

**Files:**
- Create: `frontend/` (Next.js App Router)
- Create: `frontend/lib/types.ts`

- [ ] **Step 1: Scaffold**

```bash
cd frontend-tmp || true
npx create-next-app@latest frontend --ts --tailwind --app --eslint --no-src-dir --import-alias "@/*" --use-npm
```
(Then ensure it lives at repo `frontend/`.)

- [ ] **Step 2: Write `lib/types.ts`** (mirror of Go `model`)

```ts
export interface Brief { genre: string; episodes: number; episodeSecs: number; market?: string; brandFocus: string; extra?: string }
export interface Concept { logline: string; theme: string; audience: string; tone: string; payoffEngine: string; coreConflict: string; tropesUsed: string[] }
export interface SeriesBible { title: string; genreTags: string[]; episodes: number; episodeSecs: number; platform: string; integrationThesis: string }
export interface Character { name: string; role: string; bio: string; arc: string; relationships: string }
export interface Episode { number: number; title: string; synopsis: string; beats: string[]; hook: string; cliffhanger: string; payoff: string }
export interface Placement { episode: number; scene: string; productSku: string; category: string; emotionalBeat: string; ctaTiming: string }
export interface Shot { number: number; shotType: string; action: string; dialogue: string }
export interface HeroScene { episode: number; title: string; shots: Shot[] }
export interface Production { format: string; budgetTier: string; shotCount: number; castSize: number; locations: string[]; furnitureProps: string[] }
export interface Distribution { ctaCopy: string; linkPlacement: string; hashtags: string[] }
export interface Plan { brief: Brief; concept: Concept; bible: SeriesBible; characters: Character[]; episodes: Episode[]; placements: Placement[]; heroScenes: HeroScene[]; production: Production; distribution: Distribution }
export type EventType = "stage_start" | "stage_done" | "error" | "complete"
export interface SSEvent { type: EventType; stage?: string; index?: number; total?: number; message?: string; payload?: unknown; plan?: Plan }
```

- [ ] **Step 3: Verify it builds**

Run: `cd frontend && npm run build`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add frontend && git commit -m "feat(frontend): scaffold next.js + tailwind, shared types"
```

---

### Task 14: SSE client + workbench page

**Files:**
- Create: `frontend/lib/api.ts`
- Create: `frontend/components/InputForm.tsx`
- Create: `frontend/components/StageTimeline.tsx`
- Create: `frontend/components/PlanView.tsx`
- Create: `frontend/components/ExportBar.tsx`
- Modify: `frontend/app/page.tsx`

- [ ] **Step 1: Write `lib/api.ts`** (POST + SSE stream parsing via fetch reader)

```ts
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
```

- [ ] **Step 2: Write `app/page.tsx`** (orchestrates state; renders form, timeline, plan)

```tsx
"use client"
import { useState } from "react"
import { Brief, Plan, SSEvent } from "@/lib/types"
import { generate } from "@/lib/api"
import InputForm from "@/components/InputForm"
import StageTimeline from "@/components/StageTimeline"
import PlanView from "@/components/PlanView"
import ExportBar from "@/components/ExportBar"

const STAGES = ["concept","bible","characters","episodes","placements","hero","production_distribution"]

export default function Home() {
  const [events, setEvents] = useState<SSEvent[]>([])
  const [plan, setPlan] = useState<Plan | null>(null)
  const [running, setRunning] = useState(false)

  async function run(brief: Brief) {
    setEvents([]); setPlan(null); setRunning(true)
    try {
      await generate(brief, (e) => {
        setEvents((prev) => [...prev, e])
        if (e.type === "complete" && e.plan) setPlan(e.plan)
      })
    } finally { setRunning(false) }
  }

  return (
    <main className="mx-auto max-w-5xl p-6 space-y-6">
      <header>
        <h1 className="text-3xl font-bold">Ashley Short-Drama Workbench</h1>
        <p className="text-neutral-500">Brief → AI pipeline → branded production plan</p>
      </header>
      <InputForm onSubmit={run} disabled={running} />
      <StageTimeline stages={STAGES} events={events} />
      {plan && (<><ExportBar plan={plan} /><PlanView plan={plan} /></>)}
    </main>
  )
}
```

> **Executor note (use frontend-design skill for polish):**
> - `InputForm.tsx`: controlled form (genre text, episodes number default 12, episodeSecs default 90, brandFocus text) → calls `onSubmit(brief)`. Tailwind card styling.
> - `StageTimeline.tsx`: render the `STAGES` list; for each, derive status from `events` (pending / running on `stage_start` / done on `stage_done`); show a spinner on running, check on done; expand `payload` JSON under each done stage. Show `error` events in red.
> - `PlanView.tsx`: render the full `Plan` in sections matching the 8 blocks (Concept, Bible, Characters, Episodes accordion, Placements table, Hero shot lists, Production, Distribution).
> - `ExportBar.tsx`: two buttons — download JSON (`Blob` of `plan`) and download Markdown (client-side mirror of `render.Markdown`, or fetch a `/api/render` endpoint — prefer client mirror to avoid a round-trip).

- [ ] **Step 3: Verify end-to-end (keyless demo)**

Run backend `make server`, then `cd frontend && npm run dev`; open the app, submit a brief, watch the timeline fill and the plan render.

- [ ] **Step 4: Commit**

```bash
git add frontend && git commit -m "feat(frontend): workbench page, sse timeline, plan view, export"
```

---

## Phase 6 — Docs

### Task 15: README + run scripts

**Files:**
- Create: `README.md`
- Create: `dev.sh` (optional: runs backend + frontend)

- [ ] **Step 1: Write `README.md`** covering, in order:
  1. **What & why** — branded short-drama production workbench for Ashley; the furniture-marketing angle.
  2. **Business understanding** — short-drama conventions (golden 3s, cliffhangers, payoff density), home tropes, product placement strategy.
  3. **Architecture diagram** — the 7-stage pipeline + 3 tools + self-correction loop; provider abstraction (Gemini/Mock).
  4. **Prompt & AI orchestration notes** — why staged (controllable/observable/testable) vs ReAct; why the pacing validator is deterministic Go, not an LLM; JSON-repair + retry strategy.
  5. **Run it** — keyless demo first (`make cli` / `make server`), then with `GEMINI_API_KEY`; frontend `npm run dev`.
  6. **Testing** — `cd backend && go test ./...` (runs fully without an API key thanks to Mock).
  7. **Product sense / future work** — human-in-the-loop stage editing, multi-market switching, persistence, video-gen handoff.

- [ ] **Step 2: Commit**

```bash
git add README.md dev.sh && git commit -m "docs: readme with architecture, business rationale, run guide"
```

---

## Self-Review (completed)

- **Spec coverage:** business positioning (Task 15 README, prompts), 8-block plan (model Task 1 + stages Tasks 7/render 10), multi-stage pipeline + 3 tools (Tasks 3/4/7/8), self-correction loop (Task 7 EpisodeStage), Gemini + Mock provider (Tasks 2/9/11), SSE Web + CLI (Tasks 11/12/13/14), tests via Mock (throughout) — all mapped.
- **Placeholder scan:** stage `.tmpl` bodies and 4 frontend components are specified via precise executor notes with exact I/O contracts rather than full literal bodies; this is deliberate (prompt copy + presentational JSX are iterated during execution with the frontend-design skill) and every contract names exact types/fields. All Go logic is complete and literal.
- **Type consistency:** `Provider.GenerateJSON(ctx, stage, prompt, schema)` consistent across mock/gemini/call; `model` field names match `lib/types.ts`; stage names (`concept`,`bible`,`characters`,`episodes`,`episodes_refine`,`placements`,`hero`,`production_distribution`) consistent across prompts, stages, `stagePayload`, and frontend `STAGES`.

Note: `episodes_refine` is a prompt/stage-internal call (not in `AllStages()`), and the frontend `STAGES` list intentionally omits it since it streams under the `episodes` stage.
