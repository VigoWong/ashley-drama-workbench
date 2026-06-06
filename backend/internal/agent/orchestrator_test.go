package agent

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

// flakyProvider fails the first failFirst calls, then delegates to inner.
// It models a transient LLM hiccup so we can assert stage-level retry recovers.
type flakyProvider struct {
	inner     llm.Provider
	failFirst int
	calls     int
}

func (f *flakyProvider) GenerateJSON(ctx context.Context, stage, prompt string, images []model.Image, schema map[string]any) ([]byte, error) {
	f.calls++
	if f.calls <= f.failFirst {
		return nil, fmt.Errorf("transient boom")
	}
	return f.inner.GenerateJSON(ctx, stage, prompt, images, schema)
}

func mockAll() *llm.Mock {
	m := llm.NewMock()
	m.Register("concept", `{"logline":"L","payoffEngine":"revenge"}`)
	m.Register("bible", `{"title":"Dream Home","episodes":2}`)
	m.Register("characters", `{"characters":[{"name":"Mia","role":"protagonist"}]}`)
	m.Register("episodes", `{"episodes":[{"number":1,"hook":"h","cliffhanger":"c","payoff":"p"},{"number":2,"hook":"h","cliffhanger":"c","payoff":"p"}]}`)
	m.Register("placements", `{"placements":[{"episode":1,"category":"sofa"}]}`)
	m.Register("hero", `{"heroScenes":[{"episode":1,"shots":[{"number":1,"shotType":"CU","dialogue":"hi"}]}]}`)
	m.Register("production_distribution", `{"production":{"format":"9:16"},"distribution":{"ctaCopy":"Shop now"}}`)
	return m
}

func TestOrchestratorRunsAllStages(t *testing.T) {
	var events []model.Event
	o := New(mockAll(), func(e model.Event) { events = append(events, e) })
	plan, err := o.Run(context.Background(), model.Brief{Requirement: "makeover", Episodes: 2})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Concept.Logline != "L" {
		t.Fatal("concept missing")
	}
	if plan.Production.Format != "9:16" {
		t.Fatal("production missing")
	}
	var complete bool
	for _, e := range events {
		if e.Type == model.EventComplete {
			complete = true
		}
	}
	if !complete {
		t.Fatal("expected complete event")
	}
}

// startedStages returns the stage names that emitted a stage_start event, in
// order — i.e. which stages actually ran.
func startedStages(events []model.Event) []string {
	var out []string
	for _, e := range events {
		if e.Type == model.EventStageStart {
			out = append(out, e.Stage)
		}
	}
	return out
}

func hasComplete(events []model.Event) bool {
	for _, e := range events {
		if e.Type == model.EventComplete {
			return true
		}
	}
	return false
}

func TestOrchestratorRunFrom(t *testing.T) {
	// First produce a full plan so RunFrom has a populated plan to rerun against.
	plan, err := New(mockAll(), nil).Run(context.Background(), model.Brief{Requirement: "makeover", Episodes: 2})
	if err != nil {
		t.Fatalf("seed run: %v", err)
	}

	// from-stage rerun (only=false): placements onward must all run, in order.
	var fromEvents []model.Event
	oFrom := New(mockAll(), func(e model.Event) { fromEvents = append(fromEvents, e) })
	if _, err := oFrom.RunFrom(context.Background(), plan, "placements", false, "测试备注"); err != nil {
		t.Fatalf("RunFrom from-stage: %v", err)
	}
	gotFrom := startedStages(fromEvents)
	wantFrom := []string{"placements", "hero", "production_distribution", "visuals"}
	if !reflect.DeepEqual(gotFrom, wantFrom) {
		t.Fatalf("from-stage subset = %v, want %v", gotFrom, wantFrom)
	}
	if !hasComplete(fromEvents) {
		t.Fatal("from-stage rerun: expected complete event")
	}

	// single-stage rerun (only=true): exactly one stage runs.
	var onlyEvents []model.Event
	oOnly := New(mockAll(), func(e model.Event) { onlyEvents = append(onlyEvents, e) })
	if _, err := oOnly.RunFrom(context.Background(), plan, "episodes", true, ""); err != nil {
		t.Fatalf("RunFrom single: %v", err)
	}
	gotOnly := startedStages(onlyEvents)
	if !reflect.DeepEqual(gotOnly, []string{"episodes"}) {
		t.Fatalf("single-stage subset = %v, want [episodes]", gotOnly)
	}
	if !hasComplete(onlyEvents) {
		t.Fatal("single-stage rerun: expected complete event")
	}

	// unknown stage must error and emit nothing.
	if _, err := New(mockAll(), nil).RunFrom(context.Background(), plan, "nope", true, ""); err == nil {
		t.Fatal("expected error for unknown stage")
	}
}

func TestOrchestratorRetriesTransientStageFailure(t *testing.T) {
	// First call (concept stage) fails once, then succeeds — pipeline must recover.
	p := &flakyProvider{inner: mockAll(), failFirst: 1}
	var errored bool
	o := New(p, func(e model.Event) {
		if e.Type == model.EventError {
			errored = true
		}
	})
	plan, err := o.Run(context.Background(), model.Brief{Requirement: "makeover", Episodes: 2})
	if err != nil {
		t.Fatalf("expected recovery, got %v", err)
	}
	if errored {
		t.Fatal("did not expect an error event after successful retry")
	}
	if plan.Concept.Logline != "L" {
		t.Fatal("concept missing after retry")
	}
}
