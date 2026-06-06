# 子项目 A:后端 ReAct 引擎 + 流式契约 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给短剧工作台加一个引导式 ReAct 引擎:用户用自然语言发一条消息,agent 自己「思考 → 调工具 → 看结果 → 再决定」,通过新的细粒度 SSE 事件流(`thought.* / tool.* / message.* / block.* / turn.done`)把推理与工具调用过程暴露出来,并在共享 `*model.Plan` 上增量产出 8 块方案。

**Architecture:** 引擎放在**现有 `agent` 包内**(无 import cycle),把现有 8 个 `Stage` 直接包装成「生成类工具」,3 个 `internal/tools` 函数包装成「确定性工具」,`RunFrom` 包装成 `refineBlock`。引擎依赖一个 `ChatLLM` 接口:`ScriptedLLM`(无 key / 测试用,回放固定 ReAct 轨迹)与 `geminiChatLLM`(真实 function-calling)两种实现。新增 `POST /api/chat` SSE 端点;整个测试套件继续无 key 全绿。

**Tech Stack:** Go 1.23 (`github.com/ashley/drama-workbench`)、标准库 `net/http` + SSE、`encoding/json`、`text/template`(已有 prompts)、Gemini function-calling(Vertex / AI Studio)。

参考主设计:`docs/superpowers/specs/2026-06-07-conversational-react-agent-design.md`。

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `backend/internal/model/chat.go`(新建) | 流式契约类型:`ChatEvent` + 事件常量 |
| `backend/internal/agent/chat_llm.go`(新建) | `ChatLLM` 接口 + `Message`/`Turn`/`ToolCall` 会话类型 |
| `backend/internal/agent/chat_tool.go`(新建) | `Tool`/`ToolDef`/`Observation`/`ToolCtx`/`Registry`、`stageTool` 包装器、`refineTool`、确定性工具、`DefaultRegistry()`、`chatBlockPayload` |
| `backend/internal/agent/chat_prompt.go`(新建) | `BuildSystemPrompt(registry)`:引导式系统提示词 |
| `backend/internal/agent/chat_engine.go`(新建) | `RunChat(...)`:ReAct 主循环,发 `ChatEvent` |
| `backend/internal/agent/chat_mock.go`(新建) | `ScriptedLLM` + `DemoChatScript()`:固定 ReAct 轨迹 |
| `backend/internal/agent/chat_engine_test.go`(新建) | 引擎循环单测(用 ScriptedLLM + 捕获事件) |
| `backend/internal/agent/chat_tool_test.go`(新建) | 工具前置校验 / 确定性工具单测 |
| `backend/internal/llm/gemini_tools.go`(新建) | Gemini function-calling 实现(供 `geminiChatLLM` 适配) |
| `backend/internal/agent/chat_gemini.go`(新建) | `geminiChatLLM`:把 llm 的 function-calling 适配成 `ChatLLM` |
| `backend/cmd/server/main.go`(修改) | 注册 `POST /api/chat` + `handleChat` |
| `backend/cmd/server/chat_test.go`(新建) | `/api/chat` 端到端 SSE 测试(mock) |
| `frontend/lib/types.ts`(修改) | 前端契约镜像:`ChatEvent` 联合类型 + `ChatMessage` |

---

## Task 1: 流式契约类型 `ChatEvent`

**Files:**
- Create: `backend/internal/model/chat.go`
- Test: `backend/internal/model/chat_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/model/chat_test.go
package model

import (
	"encoding/json"
	"testing"
)

func TestChatEventMarshalsToolResult(t *testing.T) {
	e := ChatEvent{
		Type:         ChatToolResult,
		ToolID:       "t1",
		ToolName:     "validatePacing",
		Status:       "ok",
		Output:       map[string]any{"pass": true},
		AffectsStage: "episodes",
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if !contains(got, `"type":"tool.result"`) || !contains(got, `"affectsStage":"episodes"`) {
		t.Fatalf("unexpected json: %s", got)
	}
	// zero-value fields must be omitted so the FE union stays clean
	if contains(got, `"text"`) || contains(got, `"plan"`) {
		t.Fatalf("zero fields not omitted: %s", got)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/model/ -run TestChatEventMarshalsToolResult`
Expected: FAIL — `undefined: ChatEvent` / `ChatToolResult`.

- [ ] **Step 3: Write the implementation**

```go
// backend/internal/model/chat.go
package model

// ChatEventType is the discriminator for the conversational ReAct stream. These
// are the fine-grained successors to the stage-oriented EventType: the front end
// drives the chat column (thought/tool/message) and the live canvas (block.*)
// off these.
type ChatEventType string

const (
	ChatThoughtDelta ChatEventType = "thought.delta" // streaming reasoning token(s)
	ChatThoughtDone  ChatEventType = "thought.done"  // reasoning finished; Text = one-line summary
	ChatToolStart    ChatEventType = "tool.start"    // a tool call began (running)
	ChatToolResult   ChatEventType = "tool.result"   // a tool call finished (ok/fail) + I/O
	ChatMessageDelta ChatEventType = "message.delta" // streaming assistant reply token(s)
	ChatMessageDone  ChatEventType = "message.done"  // assistant reply finished
	ChatBlockStart   ChatEventType = "block.start"   // a plan block entered "writing" state
	ChatBlockDone    ChatEventType = "block.done"    // a plan block finished; Payload = its content
	ChatTurnDone     ChatEventType = "turn.done"     // the whole agent turn is done; input re-enabled
	ChatErrorEvent   ChatEventType = "error"         // a fatal error for this turn
)

// ChatEvent is what the ReAct engine emits and the server streams as SSE on
// /api/chat. All non-essential fields are omitempty so each event serialises to
// just the keys its Type needs (a discriminated union on the wire).
type ChatEvent struct {
	Type ChatEventType `json:"type"`

	// thought.* / message.* / error
	Text    string `json:"text,omitempty"`
	Message string `json:"message,omitempty"`

	// tool.*
	ToolID       string      `json:"toolId,omitempty"`
	ToolName     string      `json:"toolName,omitempty"`
	FriendlyName string      `json:"friendlyName,omitempty"`
	Input        interface{} `json:"input,omitempty"`
	Output       interface{} `json:"output,omitempty"`
	Status       string      `json:"status,omitempty"`       // "ok" | "fail" (tool.result)
	AffectsStage string      `json:"affectsStage,omitempty"` // canvas linkage, if the tool touched a block

	// block.*
	Stage   string      `json:"stage,omitempty"`
	Payload interface{} `json:"payload,omitempty"`

	// turn.done (optional snapshot)
	Plan *Plan `json:"plan,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/model/ -run TestChatEventMarshalsToolResult`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/model/chat.go internal/model/chat_test.go
git commit -m "feat(chat): ChatEvent streaming contract types"
```

---

## Task 2: 会话类型 + `ChatLLM` 接口

**Files:**
- Create: `backend/internal/agent/chat_llm.go`

This task has no behaviour of its own (pure type/interface declarations consumed by later tasks), so it is verified by compilation rather than a unit test.

- [ ] **Step 1: Write the implementation**

```go
// backend/internal/agent/chat_llm.go
package agent

import "context"

// Role identifies who produced a conversation Message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool" // a tool's observation fed back to the model
)

// Message is one entry in the conversation history handed to the ChatLLM. For
// RoleTool entries, ToolCallID/ToolName identify which call this observes and
// ToolResult is the JSON-encoded Observation.
type Message struct {
	Role       Role   `json:"role"`
	Text       string `json:"text,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	ToolResult string `json:"toolResult,omitempty"`
}

// ToolCall is a single tool invocation the model wants the engine to run.
type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// Turn is one model step. Exactly one of {ToolCalls non-empty} or {Message set}
// is the "payload": if the model wants to act it returns ToolCalls; if it's done
// it returns a final Message. Thought is optional reasoning shown before either.
type Turn struct {
	Thought   string     `json:"thought,omitempty"`
	Message   string     `json:"message,omitempty"`
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// ChatLLM is the model abstraction the ReAct engine drives. Implementations:
// ScriptedLLM (no-key/tests) and geminiChatLLM (real function-calling).
type ChatLLM interface {
	NextTurn(ctx context.Context, system string, history []Message, tools []ToolDef) (Turn, error)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd backend && go build ./internal/agent/`
Expected: build fails with `undefined: ToolDef` (defined in Task 3) — that is expected; do NOT add ToolDef here. Proceed to Task 3, which makes the package compile.

- [ ] **Step 3: Commit**

```bash
cd backend && git add internal/agent/chat_llm.go
git commit -m "feat(chat): conversation types + ChatLLM interface"
```

---

## Task 3: 工具抽象 + 确定性工具 + Registry

**Files:**
- Create: `backend/internal/agent/chat_tool.go`
- Test: `backend/internal/agent/chat_tool_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/agent/chat_tool_test.go
package agent

import (
	"context"
	"testing"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

func TestDeterministicToolReturnsData(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.Get("getProductCatalog")
	if !ok {
		t.Fatal("getProductCatalog not registered")
	}
	tc := &ToolCtx{Plan: &model.Plan{}, Provider: llm.NewMock(), Emit: func(model.ChatEvent) {}}
	obs := tool.Run(context.Background(), tc, map[string]any{"category": ""})
	if !obs.OK {
		t.Fatalf("expected ok, got error %q", obs.Error)
	}
	if obs.Data == nil {
		t.Fatal("expected catalog data")
	}
}

func TestStageToolPreconditionFails(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.Get("generateEpisodes")
	if !ok {
		t.Fatal("generateEpisodes not registered")
	}
	// Empty plan: no concept/bible/characters yet → precondition must reject with
	// a structured observation the agent can self-correct from (NOT a hard error).
	tc := &ToolCtx{Plan: &model.Plan{}, Provider: llm.NewMock(), Emit: func(model.ChatEvent) {}}
	obs := tool.Run(context.Background(), tc, nil)
	if obs.OK {
		t.Fatal("expected precondition failure")
	}
	if obs.Error == "" {
		t.Fatal("expected a non-empty error message describing the missing dependency")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/agent/ -run 'TestDeterministicToolReturnsData|TestStageToolPreconditionFails'`
Expected: FAIL — `undefined: DefaultRegistry` / `ToolCtx`.

- [ ] **Step 3: Write the implementation**

```go
// backend/internal/agent/chat_tool.go
package agent

import (
	"context"
	"encoding/json"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/tools"
)

// ToolDef is the model-facing declaration of a tool (name + JSON-schema params).
type ToolDef struct {
	Name         string         `json:"name"`
	FriendlyName string         `json:"friendlyName"`
	Description  string         `json:"description"`
	Parameters   map[string]any `json:"parameters"`
}

// Observation is the structured result of running a tool. It is always non-nil;
// a failed precondition or error is reported via OK=false + Error (NOT a Go
// error) so the agent can read it as an observation and self-correct.
type Observation struct {
	OK    bool        `json:"ok"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// ToolCtx is the shared state a tool runs against.
type ToolCtx struct {
	Plan     *model.Plan
	Provider llm.Provider
	Emit     func(model.ChatEvent)
}

// Tool is one capability the ReAct agent can call.
type Tool interface {
	Def() ToolDef
	Run(ctx context.Context, tc *ToolCtx, args map[string]any) Observation
}

// Registry is an ordered set of tools.
type Registry struct {
	order []string
	byName map[string]Tool
}

func NewRegistry() *Registry { return &Registry{byName: map[string]Tool{}} }

func (r *Registry) Add(t Tool) {
	name := t.Def().Name
	if _, dup := r.byName[name]; !dup {
		r.order = append(r.order, name)
	}
	r.byName[name] = t
}

func (r *Registry) Get(name string) (Tool, bool) { t, ok := r.byName[name]; return t, ok }

func (r *Registry) Defs() []ToolDef {
	out := make([]ToolDef, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.byName[n].Def())
	}
	return out
}

// ---- helpers ----------------------------------------------------------------

func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

// chatBlockPayload returns just the slice of the plan a given stage produced —
// the content the canvas renders for that block. Mirrors orchestrator's
// stagePayload but is explicit here to keep the chat layer self-contained.
func chatBlockPayload(p *model.Plan, stage string) any {
	switch stage {
	case "concept":
		return p.Concept
	case "bible":
		return p.Bible
	case "characters":
		return p.Characters
	case "episodes":
		return p.Episodes
	case "placements":
		return p.Placements
	case "hero":
		return p.HeroScenes
	case "production_distribution":
		return map[string]any{"production": p.Production, "distribution": p.Distribution}
	case "visuals":
		return p.Visuals
	default:
		return nil
	}
}

// ---- deterministic tools ----------------------------------------------------

type tropesTool struct{}

func (tropesTool) Def() ToolDef {
	return ToolDef{
		Name: "getWinningTropes", FriendlyName: "查爆款套路库",
		Description: "返回某市场某垂类的爆款短剧套路候选(用于给立意打底)。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"market":   map[string]any{"type": "string", "description": "市场,如「中国」"},
				"vertical": map[string]any{"type": "string", "description": "垂类,如「home」"},
			},
		},
	}
}

func (tropesTool) Run(_ context.Context, tc *ToolCtx, args map[string]any) Observation {
	market := argString(args, "market")
	if market == "" {
		market = tc.Plan.Brief.Market
	}
	vertical := argString(args, "vertical")
	if vertical == "" {
		vertical = "home"
	}
	return Observation{OK: true, Data: tools.GetWinningTropes(market, vertical)}
}

type catalogTool struct{}

func (catalogTool) Def() ToolDef {
	return ToolDef{
		Name: "getProductCatalog", FriendlyName: "查 Ashley SKU 目录",
		Description: "返回 Ashley 家具 SKU 目录,可按 category 过滤。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"category": map[string]any{"type": "string", "description": "品类过滤,空字符串返回全部"},
			},
		},
	}
}

func (catalogTool) Run(_ context.Context, _ *ToolCtx, args map[string]any) Observation {
	return Observation{OK: true, Data: tools.GetProductCatalog(argString(args, "category"))}
}

type pacingTool struct{}

func (pacingTool) Def() ToolDef {
	return ToolDef{
		Name: "validatePacing", FriendlyName: "节奏校验",
		Description: "对当前分集做确定性节奏校验,返回通过/不通过 + 问题清单 + 评分。",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
	}
}

func (pacingTool) Run(_ context.Context, tc *ToolCtx, _ map[string]any) Observation {
	rep := tools.ValidatePacing(tc.Plan.Episodes)
	return Observation{OK: rep.Pass, Error: pacingErr(rep), Data: rep}
}

func pacingErr(rep tools.PacingReport) string {
	if rep.Pass {
		return ""
	}
	return rep.FormatIssues()
}

// ---- generation tools: wrap the existing 8 Stages ---------------------------

// stageTool adapts an existing Stage into a ReAct Tool: it checks preconditions
// (returning a self-correctable observation when unmet), emits block.start/done
// around the stage Run, and reports the produced block as the observation Data.
type stageTool struct {
	name     string
	friendly string
	desc     string
	stage    Stage
	requires func(*model.Plan) string // "" if satisfied, else a missing-dependency message
}

func (t stageTool) Def() ToolDef {
	return ToolDef{
		Name: t.name, FriendlyName: t.friendly, Description: t.desc,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"note": map[string]any{"type": "string", "description": "可选:对本块的额外创作指令"},
			},
		},
	}
}

func (t stageTool) Run(ctx context.Context, tc *ToolCtx, args map[string]any) Observation {
	if t.requires != nil {
		if miss := t.requires(tc.Plan); miss != "" {
			return Observation{OK: false, Error: miss}
		}
	}
	stage := t.stage.Name()
	tc.Emit(model.ChatEvent{Type: model.ChatBlockStart, Stage: stage})
	st := &PlanState{Plan: tc.Plan, Provider: tc.Provider, Note: argString(args, "note")}
	if err := t.stage.Run(ctx, st); err != nil {
		return Observation{OK: false, Error: err.Error()}
	}
	payload := chatBlockPayload(tc.Plan, stage)
	tc.Emit(model.ChatEvent{Type: model.ChatBlockDone, Stage: stage, Payload: payload})
	return Observation{OK: true, Data: payload}
}

// ---- refine tool ------------------------------------------------------------

type refineTool struct{}

func (refineTool) Def() ToolDef {
	return ToolDef{
		Name: "refineBlock", FriendlyName: "重生成某块",
		Description: "按用户指令重生成某一块(stage),仅重跑该块。stage 取值见各生成工具名对应的 stage。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"stage": map[string]any{"type": "string", "description": "要重生成的 stage 名,如 concept/bible/episodes…"},
				"note":  map[string]any{"type": "string", "description": "重生成的具体指令,如「立意换狗血一点」"},
			},
			"required": []string{"stage"},
		},
	}
}

func (refineTool) Run(ctx context.Context, tc *ToolCtx, args map[string]any) Observation {
	stage := argString(args, "stage")
	if !IsStage(stage) {
		return Observation{OK: false, Error: "未知的 stage:" + stage}
	}
	tc.Emit(model.ChatEvent{Type: model.ChatBlockStart, Stage: stage})
	o := New(tc.Provider, func(model.Event) {}) // reuse RunFrom; swallow legacy events
	if _, err := o.RunFrom(ctx, tc.Plan, stage, true, argString(args, "note")); err != nil {
		return Observation{OK: false, Error: err.Error()}
	}
	payload := chatBlockPayload(tc.Plan, stage)
	tc.Emit(model.ChatEvent{Type: model.ChatBlockDone, Stage: stage, Payload: payload})
	return Observation{OK: true, Data: payload}
}

// ---- default registry -------------------------------------------------------

// hasConcept etc. are the precondition predicates. They return a Chinese message
// naming the missing dependency so the agent's observation is self-explanatory.
func need(cond bool, msg string) string {
	if cond {
		return ""
	}
	return msg
}

// DefaultRegistry wires every tool the guided ReAct agent can call: the 8 stage
// wrappers (with dependency preconditions), refineBlock, and the 3 deterministic
// tools. Stage order/deps mirror AllStages().
func DefaultRegistry() *Registry {
	r := NewRegistry()

	stages := map[string]Stage{}
	for _, s := range AllStages() {
		stages[s.Name()] = s
	}

	conceptSet := func(p *model.Plan) bool { return p.Concept.Logline != "" }
	bibleSet := func(p *model.Plan) bool { return p.Bible.Title != "" }
	charsSet := func(p *model.Plan) bool { return len(p.Characters) > 0 }
	epsSet := func(p *model.Plan) bool { return len(p.Episodes) > 0 }
	placeSet := func(p *model.Plan) bool { return len(p.Placements) > 0 }

	r.Add(stageTool{
		name: "generateConcept", friendly: "生成立意", desc: "根据生成需求产出一个立意方向(logline/爽点引擎/冲突)。",
		stage: stages["concept"], requires: nil,
	})
	r.Add(stageTool{
		name: "writeBible", friendly: "写剧集圣经", desc: "产出标题/平台/集数/植入策略。需先有立意。",
		stage: stages["bible"], requires: func(p *model.Plan) string { return need(conceptSet(p), "需先调用 generateConcept 生成立意") },
	})
	r.Add(stageTool{
		name: "writeCharacters", friendly: "写人物", desc: "产出主要人物卡司/弧光/关系。需先有圣经。",
		stage: stages["characters"], requires: func(p *model.Plan) string { return need(bibleSet(p), "需先调用 writeBible 写剧集圣经") },
	})
	r.Add(stageTool{
		name: "generateEpisodes", friendly: "生成分集", desc: "产出逐集大纲(钩子/悬念/爽点)。需先有人物。建议生成后调用 validatePacing 校验。",
		stage: stages["episodes"], requires: func(p *model.Plan) string { return need(charsSet(p), "需先调用 writeCharacters 写人物") },
	})
	r.Add(stageTool{
		name: "planPlacements", friendly: "排布品牌植入", desc: "把 Ashley SKU 映射到分集场景与 CTA。需先有分集。",
		stage: stages["placements"], requires: func(p *model.Plan) string { return need(epsSet(p), "需先调用 generateEpisodes 生成分集") },
	})
	r.Add(stageTool{
		name: "designHeroScenes", friendly: "设计英雄场景", desc: "为关键集产出分镜表。需先有分集。",
		stage: stages["hero"], requires: func(p *model.Plan) string { return need(epsSet(p), "需先调用 generateEpisodes 生成分集") },
	})
	r.Add(stageTool{
		name: "planProductionDistribution", friendly: "制作与分发", desc: "产出预算/格式/CTA 文案/标签。需先有分集与植入。",
		stage: stages["production_distribution"], requires: func(p *model.Plan) string {
			return need(epsSet(p) && placeSet(p), "需先有分集与品牌植入")
		},
	})
	r.Add(stageTool{
		name: "renderVisuals", friendly: "生成概念图", desc: "尽力生成关键概念图(无图能力则跳过)。需先有英雄场景。",
		stage: stages["visuals"], requires: func(p *model.Plan) string { return need(epsSet(p), "需先生成分集与英雄场景") },
	})

	r.Add(refineTool{})
	r.Add(tropesTool{})
	r.Add(catalogTool{})
	r.Add(pacingTool{})
	return r
}

// mustJSON encodes an Observation for feeding back as a tool Message; never fails
// in practice (Observation is plain data).
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"ok":false,"error":"encode failed"}`
	}
	return string(b)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/agent/ -run 'TestDeterministicToolReturnsData|TestStageToolPreconditionFails'`
Expected: PASS. (The package now compiles — Task 2's `ToolDef` reference is satisfied.)

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/agent/chat_tool.go internal/agent/chat_tool_test.go
git commit -m "feat(chat): tool registry, stage wrappers, deterministic tools"
```

---

## Task 4: 引导式系统提示词

**Files:**
- Create: `backend/internal/agent/chat_prompt.go`
- Test: `backend/internal/agent/chat_prompt_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/agent/chat_prompt_test.go
package agent

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptListsToolsAndGuidance(t *testing.T) {
	p := BuildSystemPrompt(DefaultRegistry())
	// must name every tool so the model knows its action space
	for _, name := range []string{"generateConcept", "generateEpisodes", "validatePacing", "refineBlock"} {
		if !strings.Contains(p, name) {
			t.Fatalf("system prompt missing tool %q", name)
		}
	}
	// must carry the guided-ReAct guidance: recommended order + self-check
	if !strings.Contains(p, "validatePacing") || !strings.Contains(p, "顺序") {
		t.Fatal("system prompt missing guided-order / self-check guidance")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/agent/ -run TestBuildSystemPromptListsToolsAndGuidance`
Expected: FAIL — `undefined: BuildSystemPrompt`.

- [ ] **Step 3: Write the implementation**

```go
// backend/internal/agent/chat_prompt.go
package agent

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt produces the guided-ReAct system prompt: it states the
// agent's domain, lists the available tools (from the registry), and gives a
// RECOMMENDED order it should follow by default while allowing it to loop back /
// redo a block on the user's request. The recommended order is a soft guide;
// tool preconditions enforce hard correctness.
func BuildSystemPrompt(reg *Registry) string {
	var b strings.Builder
	b.WriteString("你是「短剧生产工作台」的 AI 制片 agent,面向中国国内市场(抖音/快手/红果短剧,竖屏 9:16),")
	b.WriteString("为 Ashley(爱室丽)家具产出可落地的短剧方案。全程用中文。\n\n")

	b.WriteString("你以 ReAct 方式工作:先简短思考,再调用工具,看工具返回的 observation,再决定下一步。\n")
	b.WriteString("推荐的默认顺序(软建议,可按用户要求回头修改某块):\n")
	b.WriteString("  generateConcept → writeBible → writeCharacters → generateEpisodes → ")
	b.WriteString("(随后必须调用 validatePacing 自检;若不通过,用 refineBlock(stage=\"episodes\", note=问题清单) 重写一次)→ ")
	b.WriteString("planPlacements → designHeroScenes → planProductionDistribution → renderVisuals。\n")
	b.WriteString("当某个工具返回 ok=false 时,阅读它的 error 字段,先补齐缺失的前置步骤,再重试。\n")
	b.WriteString("用户要求改某块时,调用 refineBlock 而不是从头重做。全部完成后,用一句话总结收尾,不要再调用工具。\n\n")

	b.WriteString("可用工具:\n")
	for _, d := range reg.Defs() {
		b.WriteString(fmt.Sprintf("  - %s(%s):%s\n", d.Name, d.FriendlyName, d.Description))
	}
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/agent/ -run TestBuildSystemPromptListsToolsAndGuidance`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/agent/chat_prompt.go internal/agent/chat_prompt_test.go
git commit -m "feat(chat): guided-ReAct system prompt builder"
```

---

## Task 5: ScriptedLLM(无 key / 测试用的固定 ReAct 轨迹)

**Files:**
- Create: `backend/internal/agent/chat_mock.go`
- Test: `backend/internal/agent/chat_mock_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/agent/chat_mock_test.go
package agent

import (
	"context"
	"testing"
)

func TestScriptedLLMReplaysTurnsInOrder(t *testing.T) {
	llm := NewScriptedLLM([]Turn{
		{Thought: "先查套路库", ToolCalls: []ToolCall{{ID: "1", Name: "getWinningTropes", Args: map[string]any{}}}},
		{Message: "完成"},
	})
	t1, _ := llm.NextTurn(context.Background(), "", nil, nil)
	if len(t1.ToolCalls) != 1 || t1.ToolCalls[0].Name != "getWinningTropes" {
		t.Fatalf("turn 1 wrong: %+v", t1)
	}
	t2, _ := llm.NextTurn(context.Background(), "", nil, nil)
	if t2.Message != "完成" {
		t.Fatalf("turn 2 wrong: %+v", t2)
	}
	// running past the script returns a final empty message, never panics
	t3, err := llm.NextTurn(context.Background(), "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(t3.ToolCalls) != 0 {
		t.Fatal("past-script turn should not request tools")
	}
}

func TestDemoChatScriptDrivesFullPlan(t *testing.T) {
	// The demo script must take an empty plan all the way through the pipeline
	// using DemoMock fixtures. Sanity: it ends with a final message turn.
	script := DemoChatScript()
	if len(script) == 0 {
		t.Fatal("empty demo script")
	}
	last := script[len(script)-1]
	if last.Message == "" || len(last.ToolCalls) != 0 {
		t.Fatal("demo script must end with a final assistant message (no tool calls)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/agent/ -run 'TestScriptedLLMReplaysTurnsInOrder|TestDemoChatScriptDrivesFullPlan'`
Expected: FAIL — `undefined: NewScriptedLLM` / `DemoChatScript`.

- [ ] **Step 3: Write the implementation**

```go
// backend/internal/agent/chat_mock.go
package agent

import "context"

// ScriptedLLM is a ChatLLM that replays a fixed list of Turns, one per NextTurn
// call. It powers the no-key demo and the engine tests so the front end and the
// test suite see a complete, deterministic ReAct trace without a real model.
// Calls past the end of the script return an empty final message (graceful stop).
type ScriptedLLM struct {
	turns []Turn
	i     int
}

func NewScriptedLLM(turns []Turn) *ScriptedLLM { return &ScriptedLLM{turns: turns} }

func (s *ScriptedLLM) NextTurn(_ context.Context, _ string, _ []Message, _ []ToolDef) (Turn, error) {
	if s.i >= len(s.turns) {
		return Turn{Message: "(已完成)"}, nil
	}
	t := s.turns[s.i]
	s.i++
	return t, nil
}

// DemoChatScript is the canonical no-key ReAct trace: it consults the trope
// library, then walks the full pipeline via the stage tools (each backed by the
// DemoMock fixtures), self-checks pacing, plans placements/hero/distribution,
// renders visuals (no-op without an image provider), and signs off. Keep this in
// sync with the tool names in DefaultRegistry().
func DemoChatScript() []Turn {
	tc := func(name string) ToolCall { return ToolCall{ID: name, Name: name, Args: map[string]any{}} }
	return []Turn{
		{Thought: "用户想要一部家装逆袭短剧。先查爆款套路库给立意打底。", ToolCalls: []ToolCall{tc("getWinningTropes")}},
		{Thought: "套路里「家装改造逆袭」最贴题,生成立意。", ToolCalls: []ToolCall{tc("generateConcept")}},
		{Thought: "立意已定,写剧集圣经。", ToolCalls: []ToolCall{tc("writeBible")}},
		{Thought: "写主要人物。", ToolCalls: []ToolCall{tc("writeCharacters")}},
		{Thought: "生成逐集大纲。", ToolCalls: []ToolCall{tc("generateEpisodes")}},
		{Thought: "按规矩跑一遍节奏自检。", ToolCalls: []ToolCall{tc("validatePacing")}},
		{Thought: "节奏过关。查 SKU 目录并排布植入。", ToolCalls: []ToolCall{tc("getProductCatalog"), tc("planPlacements")}},
		{Thought: "设计关键集的英雄场景。", ToolCalls: []ToolCall{tc("designHeroScenes")}},
		{Thought: "产出制作与分发方案。", ToolCalls: []ToolCall{tc("planProductionDistribution")}},
		{Thought: "尝试生成概念图。", ToolCalls: []ToolCall{tc("renderVisuals")}},
		{Message: "方案已生成完毕,右侧画布可查看与导出。需要我调整某一块吗?"},
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/agent/ -run 'TestScriptedLLMReplaysTurnsInOrder|TestDemoChatScriptDrivesFullPlan'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/agent/chat_mock.go internal/agent/chat_mock_test.go
git commit -m "feat(chat): ScriptedLLM + demo ReAct trace"
```

---

## Task 6: ReAct 引擎主循环 `RunChat`

**Files:**
- Create: `backend/internal/agent/chat_engine.go`
- Test: `backend/internal/agent/chat_engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/agent/chat_engine_test.go
package agent

import (
	"context"
	"testing"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

func collectChat(turns []Turn, plan *model.Plan) []model.ChatEvent {
	var got []model.ChatEvent
	emit := func(e model.ChatEvent) { got = append(got, e) }
	RunChat(context.Background(), NewScriptedLLM(turns), plan, llm.DemoMock(),
		DefaultRegistry(), nil, "做个家装逆袭短剧", emit)
	return got
}

func types(evs []model.ChatEvent) []model.ChatEventType {
	out := make([]model.ChatEventType, len(evs))
	for i, e := range evs {
		out[i] = e.Type
	}
	return out
}

func TestRunChatEmitsThoughtToolBlockThenTurnDone(t *testing.T) {
	plan := &model.Plan{}
	evs := collectChat([]Turn{
		{Thought: "查套路库", ToolCalls: []ToolCall{{ID: "1", Name: "getWinningTropes", Args: map[string]any{}}}},
		{Message: "好了"},
	}, plan)

	seq := types(evs)
	// expected: thought.delta, thought.done, tool.start, tool.result, message.delta, message.done, turn.done
	want := []model.ChatEventType{
		model.ChatThoughtDelta, model.ChatThoughtDone,
		model.ChatToolStart, model.ChatToolResult,
		model.ChatMessageDelta, model.ChatMessageDone, model.ChatTurnDone,
	}
	if len(seq) != len(want) {
		t.Fatalf("event count: got %v", seq)
	}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("event %d: want %s got %s (all=%v)", i, want[i], seq[i], seq)
		}
	}
}

func TestRunChatBlockToolEmitsBlockEvents(t *testing.T) {
	plan := &model.Plan{}
	plan.Brief.ApplyDefaults()
	evs := collectChat([]Turn{
		{ToolCalls: []ToolCall{{ID: "1", Name: "generateConcept", Args: map[string]any{}}}},
		{Message: "立意已生成"},
	}, plan)

	var sawStart, sawDone bool
	for _, e := range evs {
		if e.Type == model.ChatBlockStart && e.Stage == "concept" {
			sawStart = true
		}
		if e.Type == model.ChatBlockDone && e.Stage == "concept" {
			sawDone = true
		}
	}
	if !sawStart || !sawDone {
		t.Fatalf("expected concept block.start+block.done, got %v", types(evs))
	}
	if plan.Concept.Logline == "" {
		t.Fatal("generateConcept should have populated plan.Concept (DemoMock fixture)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/agent/ -run 'TestRunChat'`
Expected: FAIL — `undefined: RunChat`.

- [ ] **Step 3: Write the implementation**

```go
// backend/internal/agent/chat_engine.go
package agent

import (
	"context"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

// maxChatSteps bounds the ReAct loop so a misbehaving model can't spin forever.
const maxChatSteps = 24

// affectsStageFor maps a tool name to the plan block it touches, for canvas
// linkage on tool.result. Deterministic tools that don't mutate a block return "".
func affectsStageFor(name string) string {
	switch name {
	case "generateConcept":
		return "concept"
	case "writeBible":
		return "bible"
	case "writeCharacters":
		return "characters"
	case "generateEpisodes":
		return "episodes"
	case "planPlacements":
		return "placements"
	case "designHeroScenes":
		return "hero"
	case "planProductionDistribution":
		return "production_distribution"
	case "renderVisuals":
		return "visuals"
	default:
		return "" // tropes/catalog/pacing/refineBlock(dynamic) → no fixed block
	}
}

// RunChat drives one user turn through the guided-ReAct loop: it appends the user
// message to history, then repeatedly asks the ChatLLM for the next Turn,
// executing any tool calls against the shared plan and feeding observations back,
// until the model returns a final message (or maxChatSteps is hit). Every step
// is surfaced via emit as ChatEvents. Returns the updated history.
func RunChat(
	ctx context.Context,
	model_llm ChatLLM,
	plan *model.Plan,
	provider llm.Provider,
	reg *Registry,
	history []Message,
	userMsg string,
	emit func(model.ChatEvent),
) ([]Message, error) {
	history = append(history, Message{Role: RoleUser, Text: userMsg})
	system := BuildSystemPrompt(reg)
	defs := reg.Defs()

	for step := 0; step < maxChatSteps; step++ {
		turn, err := model_llm.NextTurn(ctx, system, history, defs)
		if err != nil {
			emit(model.ChatEvent{Type: model.ChatErrorEvent, Message: err.Error()})
			return history, err
		}

		if turn.Thought != "" {
			emit(model.ChatEvent{Type: model.ChatThoughtDelta, Text: turn.Thought})
			emit(model.ChatEvent{Type: model.ChatThoughtDone, Text: turn.Thought})
		}

		// No tool calls → this is the final assistant message; end the turn.
		if len(turn.ToolCalls) == 0 {
			emit(model.ChatEvent{Type: model.ChatMessageDelta, Text: turn.Message})
			emit(model.ChatEvent{Type: model.ChatMessageDone, Text: turn.Message})
			history = append(history, Message{Role: RoleAssistant, Text: turn.Message})
			emit(model.ChatEvent{Type: model.ChatTurnDone, Plan: plan})
			return history, nil
		}

		// Record the assistant's intent, then execute each tool call.
		history = append(history, Message{Role: RoleAssistant, Text: turn.Message})
		for _, call := range turn.ToolCalls {
			tool, ok := reg.Get(call.Name)
			emit(model.ChatEvent{
				Type: model.ChatToolStart, ToolID: call.ID, ToolName: call.Name,
				FriendlyName: friendlyName(reg, call.Name), Input: call.Args,
			})
			var obs Observation
			if !ok {
				obs = Observation{OK: false, Error: "未知工具:" + call.Name}
			} else {
				obs = tool.Run(ctx, &ToolCtx{Plan: plan, Provider: provider, Emit: emit}, call.Args)
			}
			status := "ok"
			if !obs.OK {
				status = "fail"
			}
			emit(model.ChatEvent{
				Type: model.ChatToolResult, ToolID: call.ID, ToolName: call.Name,
				Status: status, Output: obs, AffectsStage: affectsStageFor(call.Name),
			})
			history = append(history, Message{
				Role: RoleTool, ToolCallID: call.ID, ToolName: call.Name, ToolResult: mustJSON(obs),
			})
		}
	}

	// Hit the step ceiling: close the turn gracefully.
	emit(model.ChatEvent{Type: model.ChatMessageDone, Text: "本轮步骤已达上限,请补充指令后我再继续。"})
	emit(model.ChatEvent{Type: model.ChatTurnDone, Plan: plan})
	return history, nil
}

func friendlyName(reg *Registry, name string) string {
	if t, ok := reg.Get(name); ok {
		return t.Def().FriendlyName
	}
	return name
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/agent/ -run 'TestRunChat'`
Expected: PASS (both engine tests).

- [ ] **Step 5: Run the whole package + suite to confirm no regressions**

Run: `cd backend && go test ./...`
Expected: PASS (entire suite, no key/DB).

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/agent/chat_engine.go internal/agent/chat_engine_test.go
git commit -m "feat(chat): guided-ReAct engine loop with fine-grained events"
```

---

## Task 7: `POST /api/chat` SSE 端点

**Files:**
- Modify: `backend/cmd/server/main.go` (add route at the `r.With(a.Middleware).Post(...)` block near line 48-51; add `handleChat`)
- Test: `backend/cmd/server/chat_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/cmd/server/chat_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleChatStreamsReActEvents(t *testing.T) {
	body := `{"message":"做个家装逆袭短剧,植入沙发"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handleChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	out := rec.Body.String()
	// The no-key demo script must drive a full trace: tool calls + block events +
	// a closing turn.done, all over SSE ("data: " framed).
	for _, want := range []string{
		"data: ", `"type":"tool.start"`, `"type":"block.done"`,
		`"stage":"concept"`, `"type":"turn.done"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("SSE output missing %q\n---\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./cmd/server/ -run TestHandleChatStreamsReActEvents`
Expected: FAIL — `undefined: handleChat`.

- [ ] **Step 3: Register the route**

In `backend/cmd/server/main.go`, in the middleware-guarded block (currently lines 48-51), add the chat route alongside the others:

```go
	r.With(a.Middleware).Post("/api/chat", handleChat)
```

- [ ] **Step 4: Write the handler**

Append to `backend/cmd/server/main.go` (near `handleGenerate`, reusing the same SSE plumbing pattern):

```go
// chatRequest is the body for /api/chat: one user message plus optional prior
// text history (user/assistant bubbles only — tool internals are ephemeral) and
// an optional in-progress plan the client is iterating on. When plan is nil a
// fresh plan with default brief is started.
type chatRequest struct {
	Message string         `json:"message"`
	History []agent.Message `json:"history,omitempty"`
	Plan    *model.Plan    `json:"plan,omitempty"`
}

// handleChat runs one guided-ReAct turn and streams ChatEvents as SSE. It picks
// the ScriptedLLM (no-key demo) or the Gemini function-calling LLM based on
// whether real credentials are present, so the endpoint works with zero config.
func handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	plan := req.Plan
	if plan == nil {
		plan = &model.Plan{}
	}
	plan.Brief.ApplyDefaults()

	provider, usingMock := llm.FromEnv()
	var chatLLM agent.ChatLLM
	if usingMock {
		chatLLM = agent.NewScriptedLLM(agent.DemoChatScript())
	} else {
		chatLLM = agent.NewGeminiChatLLM(provider)
	}

	emit := func(e model.ChatEvent) {
		data, _ := json.Marshal(e)
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}

	if _, err := agent.RunChat(r.Context(), chatLLM, plan, provider,
		agent.DefaultRegistry(), req.History, req.Message, emit); err != nil {
		log.Printf("chat error: %v", err)
	}
}
```

> NOTE: `agent.NewGeminiChatLLM` is implemented in Task 8. To keep this task's test green before Task 8 exists, temporarily fall back to the scripted LLM in the non-mock branch (`chatLLM = agent.NewScriptedLLM(agent.DemoChatScript())`) and replace it in Task 8 — OR implement Task 8 first. The test below runs the mock branch only.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./cmd/server/ -run TestHandleChatStreamsReActEvents`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd backend && git add cmd/server/main.go cmd/server/chat_test.go
git commit -m "feat(chat): POST /api/chat SSE endpoint"
```

---

## Task 8: Gemini function-calling 适配(`geminiChatLLM`)

> Real-model path. The whole test suite stays on the ScriptedLLM, so this task's
> automated check is **build + the existing suite staying green**; full behaviour
> needs a live key (manual smoke at the end). Do not block on a live key.

**Files:**
- Create: `backend/internal/llm/gemini_tools.go`
- Create: `backend/internal/agent/chat_gemini.go`

- [ ] **Step 1: Add a function-calling entrypoint on the Gemini provider**

Inspect `backend/internal/llm/gemini.go` for the existing request/response structs and HTTP helper (the code that builds the `:generateContent` request and parses candidates). Then add a tool-aware call. Create `backend/internal/llm/gemini_tools.go`:

```go
// backend/internal/llm/gemini_tools.go
package llm

import (
	"context"
	"encoding/json"
)

// ToolSpec is a provider-agnostic tool declaration passed to GenerateWithTools.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON schema
}

// ToolCallOut is a tool invocation the model asked for.
type ToolCallOut struct {
	ID   string
	Name string
	Args map[string]any
}

// ToolMessage is one conversation entry for the tool-calling API. Role is
// "user" | "model" | "tool". For tool messages, Name + Result (JSON) carry the
// observation.
type ToolMessage struct {
	Role   string
	Text   string
	Name   string
	Result string
}

// ToolReply is the model's response: either Text (final) or ToolCalls (act).
type ToolReply struct {
	Thought   string
	Text      string
	ToolCalls []ToolCallOut
}

// ToolCaller is implemented by providers that support native function-calling.
// Gemini (Vertex + AI Studio) implements it; Mock does not.
type ToolCaller interface {
	GenerateWithTools(ctx context.Context, system string, history []ToolMessage, tools []ToolSpec) (ToolReply, error)
}

// GenerateWithTools maps the conversation + tool specs to Gemini's
// functionDeclarations / functionCall / functionResponse protocol, issues one
// :generateContent call, and parses the first candidate into a ToolReply.
//
// Implementation notes for the engineer:
//   - Reuse the existing request builder / auth / endpoint code in gemini.go.
//   - Put tools under request "tools":[{"functionDeclarations":[{name,description,parameters}]}].
//   - Map history roles: user→"user", model→"model"; a tool message becomes a
//     "function" part: {"functionResponse":{"name":Name,"response":<parsed Result>}}.
//   - system → request "systemInstruction":{"parts":[{"text":system}]}.
//   - In the response, a part with "functionCall":{name,args} → append a
//     ToolCallOut (synthesise ID = name+index since Gemini omits ids); a part with
//     "text" → Text. Concatenate any leading text parts into Thought when
//     functionCalls are also present.
func (g *Gemini) GenerateWithTools(ctx context.Context, system string, history []ToolMessage, tools []ToolSpec) (ToolReply, error) {
	type fnDecl struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters,omitempty"`
	}
	decls := make([]fnDecl, len(tools))
	for i, t := range tools {
		decls[i] = fnDecl{Name: t.Name, Description: t.Description, Parameters: t.Parameters}
	}

	type part struct {
		Text             string          `json:"text,omitempty"`
		FunctionCall     json.RawMessage `json:"functionCall,omitempty"`
		FunctionResponse json.RawMessage `json:"functionResponse,omitempty"`
	}
	type content struct {
		Role  string `json:"role"`
		Parts []part `json:"parts"`
	}

	contents := make([]content, 0, len(history))
	for _, m := range history {
		switch m.Role {
		case "tool":
			var resp any
			_ = json.Unmarshal([]byte(m.Result), &resp)
			fr, _ := json.Marshal(map[string]any{"name": m.Name, "response": map[string]any{"result": resp}})
			contents = append(contents, content{Role: "function", Parts: []part{{FunctionResponse: fr}}})
		case "model":
			contents = append(contents, content{Role: "model", Parts: []part{{Text: m.Text}}})
		default:
			contents = append(contents, content{Role: "user", Parts: []part{{Text: m.Text}}})
		}
	}

	reqBody := map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]any{{"text": system}}},
		"contents":          contents,
		"tools":             []map[string]any{{"functionDeclarations": decls}},
	}

	// doRequest is the existing low-level call in gemini.go that POSTs reqBody to
	// the model's :generateContent endpoint (with auth/region handled) and returns
	// the raw response bytes. If its name differs, adapt this line accordingly.
	raw, err := g.doGenerate(ctx, reqBody)
	if err != nil {
		return ToolReply{}, err
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ToolReply{}, err
	}
	var reply ToolReply
	if len(parsed.Candidates) > 0 {
		for i, p := range parsed.Candidates[0].Content.Parts {
			if p.FunctionCall != nil {
				reply.ToolCalls = append(reply.ToolCalls, ToolCallOut{
					ID:   p.FunctionCall.Name + "_" + itoa(i),
					Name: p.FunctionCall.Name,
					Args: p.FunctionCall.Args,
				})
			} else if p.Text != "" {
				if len(parsed.Candidates[0].Content.Parts) > 1 {
					reply.Thought += p.Text
				} else {
					reply.Text += p.Text
				}
			}
		}
	}
	return reply, nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	b := []byte{}
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
```

> If `gemini.go` does not already expose a low-level `doGenerate(ctx, body any) ([]byte, error)`, extract one from the existing `GenerateJSON` implementation (the part that signs/sends the HTTP request and returns the body) and call it from both. This is a small refactor, not new behaviour — keep `GenerateJSON`'s tests green.

- [ ] **Step 2: Adapt to the engine's `ChatLLM`**

```go
// backend/internal/agent/chat_gemini.go
package agent

import (
	"context"

	"github.com/ashley/drama-workbench/internal/llm"
)

// geminiChatLLM adapts any llm.ToolCaller (the Gemini provider) to the engine's
// ChatLLM interface, translating between the agent conversation types and the
// provider's tool-calling types.
type geminiChatLLM struct{ tc llm.ToolCaller }

// NewGeminiChatLLM wraps a provider for the chat engine. The provider must
// implement llm.ToolCaller (Gemini does); callers gate on usingMock so this is
// only constructed when real credentials are present.
func NewGeminiChatLLM(p llm.Provider) ChatLLM {
	tc, ok := p.(llm.ToolCaller)
	if !ok {
		// Defensive: fall back to a single "I can't act" message rather than panic.
		return NewScriptedLLM([]Turn{{Message: "当前后端未启用工具调用能力。"}})
	}
	return &geminiChatLLM{tc: tc}
}

func (g *geminiChatLLM) NextTurn(ctx context.Context, system string, history []Message, defs []ToolDef) (Turn, error) {
	specs := make([]llm.ToolSpec, len(defs))
	for i, d := range defs {
		specs[i] = llm.ToolSpec{Name: d.Name, Description: d.Description, Parameters: d.Parameters}
	}
	msgs := make([]llm.ToolMessage, len(history))
	for i, m := range history {
		switch m.Role {
		case RoleTool:
			msgs[i] = llm.ToolMessage{Role: "tool", Name: m.ToolName, Result: m.ToolResult}
		case RoleAssistant:
			msgs[i] = llm.ToolMessage{Role: "model", Text: m.Text}
		default:
			msgs[i] = llm.ToolMessage{Role: "user", Text: m.Text}
		}
	}
	reply, err := g.tc.GenerateWithTools(ctx, system, msgs, specs)
	if err != nil {
		return Turn{}, err
	}
	turn := Turn{Thought: reply.Thought, Message: reply.Text}
	for _, c := range reply.ToolCalls {
		turn.ToolCalls = append(turn.ToolCalls, ToolCall{ID: c.ID, Name: c.Name, Args: c.Args})
	}
	return turn, nil
}
```

- [ ] **Step 3: Wire the real branch in the handler**

If Task 7 used the temporary fallback, replace the non-mock branch in `handleChat` with the real adapter:

```go
		chatLLM = agent.NewGeminiChatLLM(provider)
```

- [ ] **Step 4: Build + run the whole suite**

Run: `cd backend && go build ./... && go test ./...`
Expected: build OK; entire suite PASS (still on mock — no key needed).

- [ ] **Step 5: (Optional, needs a real key) live smoke**

Run: `cd backend && GEMINI_API_KEY=... go run ./cmd/server` then `POST /api/chat {"message":"做个家装逆袭短剧,植入 Ashley 沙发"}` and confirm a real ReAct trace streams. Expected: thought/tool/block events backed by a real model.

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/llm/gemini_tools.go internal/agent/chat_gemini.go cmd/server/main.go
git commit -m "feat(chat): Gemini function-calling ChatLLM adapter"
```

---

## Task 9: 前端契约镜像(types.ts)

> The handshake type for sub-project B. No Go test; verified by `tsc`/`eslint`.

**Files:**
- Modify: `frontend/lib/types.ts`

- [ ] **Step 1: Add the mirror types**

Append to `frontend/lib/types.ts` (keep in sync with `internal/model/chat.go` and `agent.Message`):

```ts
// ---- conversational ReAct agent (mirrors internal/model/chat.go) ----

export type ChatEventType =
  | "thought.delta"
  | "thought.done"
  | "tool.start"
  | "tool.result"
  | "message.delta"
  | "message.done"
  | "block.start"
  | "block.done"
  | "turn.done"
  | "error"

export interface ChatEvent {
  type: ChatEventType
  // thought.* / message.* / error
  text?: string
  message?: string
  // tool.*
  toolId?: string
  toolName?: string
  friendlyName?: string
  input?: unknown
  output?: unknown
  status?: "ok" | "fail"
  affectsStage?: string
  // block.*
  stage?: string
  payload?: unknown
  // turn.done
  plan?: Plan
}

// Prior text history sent back on /api/chat (user/assistant bubbles only).
export interface ChatMessage {
  role: "user" | "assistant" | "tool"
  text?: string
}

export interface ChatReq {
  message: string
  history?: ChatMessage[]
  plan?: Plan
}
```

- [ ] **Step 2: Verify the frontend still type-checks / lints**

Run: `cd frontend && npm run lint`
Expected: no new errors from `lib/types.ts`. (If `Plan` isn't already exported in this file, confirm the existing export and reference it; do not redefine it.)

- [ ] **Step 3: Commit**

```bash
cd frontend && git add lib/types.ts
git commit -m "feat(chat): frontend ChatEvent contract mirror"
```

---

## 收尾验证

- [ ] **后端全绿(无 key / 无 DB):** `cd backend && go test ./...` → PASS
- [ ] **后端构建:** `cd backend && make build` → builds `bin/server` + `bin/cli`
- [ ] **端点冒烟(mock):** 启动 `make server`,`curl -N -X POST localhost:8080/api/chat -H 'Authorization: Bearer <token>' -d '{"message":"做个家装逆袭短剧"}'` 看到 SSE 流出 `thought.* / tool.* / block.* / turn.done`
- [ ] **前端 lint:** `cd frontend && npm run lint` → 无新增错误

---

## Self-Review(作者已核对)

**1. Spec coverage**
- 引导式 ReAct 引擎 → Task 6 + 系统提示词 Task 4 ✓
- 工具目录(8 生成类 + refine + 3 确定性)+ 前置校验兜底 → Task 3 ✓
- Provider function-calling 扩展 → Task 8 ✓
- 新 SSE 事件词汇(thought/tool/message/block/turn) → Task 1 + 引擎 Task 6 发射 ✓
- 新对话端点 `/api/chat` → Task 7 ✓
- Mock 剧本化轨迹(无 key 全绿) → Task 5 ✓
- 会话状态(单会话上下文,history+plan 随请求) → Task 7 请求体 + 引擎 history ✓
- 前端契约镜像 → Task 9 ✓
- 既有节奏校验测试不变:`validatePacing` 工具复用 `tools.ValidatePacing`,未改 `pacing.go` ✓
- 注:主设计 §4 列了 `block.delta`,本计划按 YAGNI 仅实现 `block.start/done`(结构化 JSON 无法逐 token 流式);如需块内增量,后续单开。已在此说明,非遗漏。

**2. Placeholder scan**:无 TBD/TODO;每个代码步骤含完整代码。Task 8 的 `doGenerate` 是对 `gemini.go` 既有 HTTP helper 的引用,已注明若名字不同需就地适配——这是真实集成点而非占位。

**3. Type consistency**:`ToolDef/Observation/ToolCtx/Tool/Registry`(Task 3)、`Message/Turn/ToolCall/ChatLLM`(Task 2)、`ChatEvent`+常量(Task 1)在 Task 5/6/7/8 中署名一致;`DefaultRegistry/NewScriptedLLM/DemoChatScript/RunChat/BuildSystemPrompt/NewGeminiChatLLM` 调用签名前后一致;`stagePayload` 未被复用(改用本地 `chatBlockPayload` 避免签名耦合)。
