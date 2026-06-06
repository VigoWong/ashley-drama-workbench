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
