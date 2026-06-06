package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/prompts"
	"github.com/ashley/drama-workbench/internal/tools"
)

// maxProposals caps how many candidate concepts we surface to the user. The
// propose prompt asks for exactly 3, but we defensively trim in case the model
// returns more.
const maxProposals = 3

// Propose runs a single, lightweight LLM call that returns 2-3 "显著不同" 的立意
// 方向（candidate concepts）for the given brief — the front of the multi-direction
// selection flow. It deliberately does NOT run the rest of the pipeline: the user
// picks (and may tweak) one direction, after which /api/generate continues from
// the bible stage with the chosen concept. The prompt input mirrors ConceptStage
// (genre/episodes/episodeSecs/brandFocus + winning tropes) so a proposal reads as
// a real first-pass concept rather than a throwaway pitch.
func Propose(ctx context.Context, provider llm.Provider, brief model.Brief) ([]model.Concept, error) {
	brief.ApplyDefaults()

	tr := tools.GetWinningTropes(brief.Market, "home")
	trJSON, _ := json.Marshal(tr)
	data := map[string]any{
		"Requirement": brief.Requirement, "Episodes": brief.Episodes,
		"EpisodeSecs": brief.EpisodeSecs,
		"Tropes":      string(trJSON),
	}

	prompt, err := prompts.Render("propose", data)
	if err != nil {
		return nil, fmt.Errorf("propose: render: %w", err)
	}

	// Feed any reference images so the proposed 立意方向 are grounded in the user's
	// uploaded room/product photos — this is the earliest, highest-leverage place
	// for the images to shape the creative direction.
	raw, err := provider.GenerateJSON(ctx, "propose", prompt, brief.Images, nil)
	if err != nil {
		return nil, fmt.Errorf("propose: generate: %w", err)
	}

	var wrap struct {
		Concepts []model.Concept `json:"concepts"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("propose: unmarshal: %w (raw=%s)", err, truncate(raw))
	}
	if len(wrap.Concepts) == 0 {
		return nil, fmt.Errorf("propose: model returned no concepts")
	}
	if len(wrap.Concepts) > maxProposals {
		wrap.Concepts = wrap.Concepts[:maxProposals]
	}
	return wrap.Concepts, nil
}
