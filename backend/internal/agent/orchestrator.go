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
	if emit == nil {
		emit = func(model.Event) {}
	}
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
	}
	return nil
}
