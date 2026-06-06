package agent

import (
	"context"
	"fmt"
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
	plan, err := o.Run(context.Background(), model.Brief{Genre: "makeover", Episodes: 2})
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

func TestOrchestratorRetriesTransientStageFailure(t *testing.T) {
	// First call (concept stage) fails once, then succeeds — pipeline must recover.
	p := &flakyProvider{inner: mockAll(), failFirst: 1}
	var errored bool
	o := New(p, func(e model.Event) {
		if e.Type == model.EventError {
			errored = true
		}
	})
	plan, err := o.Run(context.Background(), model.Brief{Genre: "makeover", Episodes: 2})
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
