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
