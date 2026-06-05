package agent

import (
	"context"
	"testing"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

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
