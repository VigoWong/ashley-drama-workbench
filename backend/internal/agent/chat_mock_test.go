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
