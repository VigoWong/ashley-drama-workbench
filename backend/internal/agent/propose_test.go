package agent

import (
	"context"
	"testing"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

func TestProposeReturnsThreeConcepts(t *testing.T) {
	m := llm.NewMock()
	m.Register("propose", `{"concepts":[
		{"logline":"方向一","payoffEngine":"逆袭打脸"},
		{"logline":"方向二","payoffEngine":"双向奔赴"},
		{"logline":"方向三","payoffEngine":"悬疑反转"}
	]}`)

	concepts, err := Propose(context.Background(), m, model.Brief{Requirement: "家装改造", Episodes: 5})
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if len(concepts) != 3 {
		t.Fatalf("expected 3 concepts, got %d", len(concepts))
	}
	if concepts[0].Logline != "方向一" || concepts[2].PayoffEngine != "悬疑反转" {
		t.Fatalf("unexpected concept content: %+v", concepts)
	}
}

func TestProposeTrimsToMax(t *testing.T) {
	m := llm.NewMock()
	m.Register("propose", `{"concepts":[
		{"logline":"a"},{"logline":"b"},{"logline":"c"},{"logline":"d"}
	]}`)
	concepts, err := Propose(context.Background(), m, model.Brief{})
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if len(concepts) != maxProposals {
		t.Fatalf("expected %d concepts after trim, got %d", maxProposals, len(concepts))
	}
}
