package tools

import (
	"fmt"

	"github.com/ashley/drama-workbench/internal/model"
)

type Issue struct {
	Episode int    `json:"episode"` // 0 = season-level
	Message string `json:"message"`
}

type PacingReport struct {
	Pass   bool    `json:"pass"`
	Score  float64 `json:"score"` // 0..100
	Issues []Issue `json:"issues"`
}

// ValidatePacing enforces short-drama pacing rules deterministically:
//   - every episode must have a non-empty hook and cliffhanger
//   - payoff (爽点/反转) density across the season must be >= 0.6
//   - episode 1 hook must exist (golden 3 seconds)
func ValidatePacing(eps []model.Episode) PacingReport {
	r := PacingReport{Pass: true, Score: 100}
	if len(eps) == 0 {
		return PacingReport{Pass: false, Score: 0, Issues: []Issue{{0, "no episodes"}}}
	}
	payoffs := 0
	for _, e := range eps {
		if e.Hook == "" {
			r.Issues = append(r.Issues, Issue{e.Number, "missing opening hook (golden 3 seconds)"})
		}
		if e.Cliffhanger == "" {
			r.Issues = append(r.Issues, Issue{e.Number, "missing ending cliffhanger"})
		}
		if e.Payoff != "" {
			payoffs++
		}
	}
	density := float64(payoffs) / float64(len(eps))
	if density < 0.6 {
		r.Issues = append(r.Issues, Issue{0, fmt.Sprintf("payoff density %.0f%% < 60%%", density*100)})
	}
	r.Score = 100 - float64(len(r.Issues))*10
	if r.Score < 0 {
		r.Score = 0
	}
	r.Pass = len(r.Issues) == 0
	return r
}

// FormatIssues renders issues as a feedback string for the LLM refine pass.
func (r PacingReport) FormatIssues() string {
	s := ""
	for _, is := range r.Issues {
		if is.Episode == 0 {
			s += "- (season) " + is.Message + "\n"
		} else {
			s += fmt.Sprintf("- (ep %d) %s\n", is.Episode, is.Message)
		}
	}
	return s
}
