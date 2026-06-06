package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

// maxStageAttempts bounds how many times a single stage is retried before the
// pipeline gives up. Real LLM backends (Vertex/Gemini) occasionally return a
// transient error or a malformed payload for one stage; retrying the stage as a
// whole recovers from that without failing the entire generation.
const maxStageAttempts = 3

type Orchestrator struct {
	provider llm.Provider
	emit     Emitter
	stages   []Stage
}

func New(p llm.Provider, emit Emitter) *Orchestrator {
	if emit == nil {
		emit = func(model.Event) {}
	}
	return &Orchestrator{provider: p, emit: emit, stages: AllStages()}
}

func (o *Orchestrator) Run(ctx context.Context, brief model.Brief) (*model.Plan, error) {
	brief.ApplyDefaults()
	state := &PlanState{Plan: &model.Plan{Brief: brief}, Provider: o.provider}
	return o.runStages(ctx, state, o.stages)
}

// RunFrom reruns a subset of the pipeline against an existing plan — the core of
// the human-in-the-loop refine flow. With only=true it regenerates exactly the
// named stage; otherwise it reruns that stage and everything downstream so the
// edited/regenerated section propagates forward. note is an optional transient
// instruction appended to each reran stage's prompt. The incoming plan is reused
// as-is (its Brief, Concept, … stay in place); only zero-value brief fields are
// backfilled.
func (o *Orchestrator) RunFrom(ctx context.Context, plan *model.Plan, fromStage string, only bool, note string) (*model.Plan, error) {
	if plan == nil {
		return nil, fmt.Errorf("refine: plan is nil")
	}
	plan.Brief.ApplyDefaults()

	start := -1
	for i, st := range o.stages {
		if st.Name() == fromStage {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("refine: unknown stage %q", fromStage)
	}

	var subset []Stage
	if only {
		subset = o.stages[start : start+1]
	} else {
		subset = o.stages[start:]
	}

	state := &PlanState{Plan: plan, Provider: o.provider, Note: note}
	return o.runStages(ctx, state, subset)
}

// runStages drives a sequence of stages with per-stage retry, emitting the same
// StageStart/StageDone/Error/Complete events for both full runs and refines.
func (o *Orchestrator) runStages(ctx context.Context, state *PlanState, stages []Stage) (*model.Plan, error) {
	total := len(stages)
	for i, st := range stages {
		o.emit(model.Event{Type: model.EventStageStart, Stage: st.Name(), Index: i, Total: total})
		if err := o.runStage(ctx, st, state); err != nil {
			o.emit(model.Event{Type: model.EventError, Stage: st.Name(), Index: i, Message: err.Error()})
			return nil, err
		}
		o.emit(model.Event{Type: model.EventStageDone, Stage: st.Name(), Index: i, Total: total, Payload: stagePayload(st.Name(), state.Plan)})
	}
	o.emit(model.Event{Type: model.EventComplete, Plan: state.Plan})
	return state.Plan, nil
}

// runStage runs one stage, retrying transient failures up to maxStageAttempts
// with linear backoff. It stops early if the context is cancelled (client gone).
func (o *Orchestrator) runStage(ctx context.Context, st Stage, state *PlanState) error {
	var err error
	for attempt := 1; attempt <= maxStageAttempts; attempt++ {
		if err = st.Run(ctx, state); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return err // cancelled — don't keep retrying
		}
		if attempt < maxStageAttempts {
			log.Printf("stage %s failed (attempt %d/%d): %v — retrying", st.Name(), attempt, maxStageAttempts, err)
			select {
			case <-ctx.Done():
				return err
			case <-time.After(time.Duration(attempt) * 700 * time.Millisecond):
			}
		}
	}
	return err
}

// stagePayload returns just the slice of the plan a stage produced, for streaming.
func stagePayload(name string, p *model.Plan) any {
	switch name {
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
	}
	return nil
}
