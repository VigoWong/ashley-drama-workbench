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
	order  []string
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
