package tools

import (
	"testing"

	"github.com/ashley/drama-workbench/internal/model"
)

func eps(n int, fill bool) []model.Episode {
	out := make([]model.Episode, n)
	for i := range out {
		out[i].Number = i + 1
		if fill {
			out[i].Hook = "hook"
			out[i].Cliffhanger = "cliff"
			out[i].Payoff = "payoff"
		}
	}
	return out
}

func TestPacingPerfectScoresHigh(t *testing.T) {
	r := ValidatePacing(eps(12, true))
	if !r.Pass {
		t.Fatalf("expected pass, got %+v", r)
	}
	if len(r.Issues) != 0 {
		t.Fatalf("expected no issues, got %v", r.Issues)
	}
}

func TestPacingMissingHookFails(t *testing.T) {
	e := eps(12, true)
	e[0].Hook = ""
	r := ValidatePacing(e)
	if r.Pass {
		t.Fatal("expected fail when episode 1 has no hook")
	}
	found := false
	for _, is := range r.Issues {
		if is.Episode == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected issue on episode 1, got %v", r.Issues)
	}
}

func TestPacingLowPayoffDensityFails(t *testing.T) {
	e := eps(12, true)
	for i := range e {
		e[i].Payoff = ""
	} // zero payoff across season
	r := ValidatePacing(e)
	if r.Pass {
		t.Fatal("expected fail on low payoff density")
	}
}
