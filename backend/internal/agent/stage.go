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
