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

// When a stage's LLM call fails AFTER block.start, the tool must resolve the
// block so the canvas never hangs in "writing": it emits a stage-scoped error
// event (and no block.done) for that stage.
func TestStageToolErrorResolvesBlock(t *testing.T) {
	reg := DefaultRegistry()
	tool, _ := reg.Get("generateConcept") // no precondition → reaches stage.Run
	var evs []model.ChatEvent
	tc := &ToolCtx{
		Plan:     &model.Plan{},
		Provider: llm.NewMock(), // empty mock: no "concept" fixture → stage.Run errors
		Emit:     func(e model.ChatEvent) { evs = append(evs, e) },
	}
	obs := tool.Run(context.Background(), tc, nil)
	if obs.OK {
		t.Fatal("expected the stage to fail with an empty mock")
	}
	var sawStart, sawErr, sawDone bool
	for _, e := range evs {
		switch {
		case e.Type == model.ChatBlockStart && e.Stage == "concept":
			sawStart = true
		case e.Type == model.ChatErrorEvent && e.Stage == "concept":
			sawErr = true
		case e.Type == model.ChatBlockDone && e.Stage == "concept":
			sawDone = true
		}
	}
	if !sawStart {
		t.Fatal("expected block.start for concept")
	}
	if !sawErr {
		t.Fatal("expected a stage-scoped error event resolving the block")
	}
	if sawDone {
		t.Fatal("must NOT emit block.done on failure")
	}
}
