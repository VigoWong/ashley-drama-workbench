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
