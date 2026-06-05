package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/prompts"
	"github.com/ashley/drama-workbench/internal/tools"
)

// call renders a prompt and unmarshals the provider's JSON into out. When
// withImages is true the Brief's reference images are attached as multimodal
// input — reserved for visual stages (concept/placements/hero) to control token
// cost; other stages pass false.
func call(ctx context.Context, s *PlanState, stage string, data any, out any, withImages bool) error {
	prompt, err := prompts.Render(stage, data)
	if err != nil {
		return fmt.Errorf("%s: render: %w", stage, err)
	}
	var images []model.Image
	if withImages {
		images = s.Plan.Brief.Images
	}
	raw, err := s.Provider.GenerateJSON(ctx, stage, prompt, images, nil)
	if err != nil {
		return fmt.Errorf("%s: generate: %w", stage, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: unmarshal: %w (raw=%s)", stage, err, truncate(raw))
	}
	return nil
}

func truncate(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "..."
	}
	return string(b)
}

type ConceptStage struct{}

func (ConceptStage) Name() string { return "concept" }
func (ConceptStage) Run(ctx context.Context, s *PlanState) error {
	tr := tools.GetWinningTropes(s.Plan.Brief.Market, "home")
	trJSON, _ := json.Marshal(tr)
	data := map[string]any{
		"Genre": s.Plan.Brief.Genre, "Episodes": s.Plan.Brief.Episodes,
		"EpisodeSecs": s.Plan.Brief.EpisodeSecs, "BrandFocus": s.Plan.Brief.BrandFocus,
		"Tropes": string(trJSON),
	}
	return call(ctx, s, "concept", data, &s.Plan.Concept, true)
}

type BibleStage struct{}

func (BibleStage) Name() string { return "bible" }
func (BibleStage) Run(ctx context.Context, s *PlanState) error {
	data := map[string]any{"Brief": s.Plan.Brief, "Concept": s.Plan.Concept}
	if err := call(ctx, s, "bible", data, &s.Plan.Bible, false); err != nil {
		return err
	}
	if s.Plan.Bible.Episodes == 0 {
		s.Plan.Bible.Episodes = s.Plan.Brief.Episodes
	}
	if s.Plan.Bible.EpisodeSecs == 0 {
		s.Plan.Bible.EpisodeSecs = s.Plan.Brief.EpisodeSecs
	}
	return nil
}

type CharacterStage struct{}

func (CharacterStage) Name() string { return "characters" }
func (CharacterStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct {
		Characters []model.Character `json:"characters"`
	}
	data := map[string]any{"Concept": s.Plan.Concept, "Bible": s.Plan.Bible}
	if err := call(ctx, s, "characters", data, &wrap, false); err != nil {
		return err
	}
	s.Plan.Characters = wrap.Characters
	return nil
}

type EpisodeStage struct{}

func (EpisodeStage) Name() string { return "episodes" }
func (EpisodeStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct {
		Episodes []model.Episode `json:"episodes"`
	}
	data := map[string]any{
		"Concept": s.Plan.Concept, "Bible": s.Plan.Bible,
		"Characters": s.Plan.Characters, "Episodes": s.Plan.Brief.Episodes,
	}
	if err := call(ctx, s, "episodes", data, &wrap, false); err != nil {
		return err
	}
	// Deterministic pacing gate + one-shot self-correction.
	if rep := tools.ValidatePacing(wrap.Episodes); !rep.Pass {
		prev, _ := json.Marshal(wrap.Episodes)
		refine := map[string]any{
			"Concept": s.Plan.Concept, "Bible": s.Plan.Bible,
			"Characters": s.Plan.Characters, "Episodes": s.Plan.Brief.Episodes,
			"Issues": rep.FormatIssues(), "Previous": string(prev),
		}
		var rewrap struct {
			Episodes []model.Episode `json:"episodes"`
		}
		if err := call(ctx, s, "episodes_refine", refine, &rewrap, false); err == nil && len(rewrap.Episodes) > 0 {
			wrap.Episodes = rewrap.Episodes
		}
	}
	s.Plan.Episodes = wrap.Episodes
	return nil
}

type PlacementStage struct{}

func (PlacementStage) Name() string { return "placements" }
func (PlacementStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct {
		Placements []model.Placement `json:"placements"`
	}
	cat, _ := json.Marshal(tools.GetProductCatalog(""))
	data := map[string]any{"Episodes": s.Plan.Episodes, "Catalog": string(cat)}
	if err := call(ctx, s, "placements", data, &wrap, true); err != nil {
		return err
	}
	s.Plan.Placements = wrap.Placements
	return nil
}

type HeroStage struct{}

func (HeroStage) Name() string { return "hero" }
func (HeroStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct {
		HeroScenes []model.HeroScene `json:"heroScenes"`
	}
	data := map[string]any{"Episodes": s.Plan.Episodes, "Concept": s.Plan.Concept}
	if err := call(ctx, s, "hero", data, &wrap, true); err != nil {
		return err
	}
	s.Plan.HeroScenes = wrap.HeroScenes
	return nil
}

type ProducerStage struct{}

func (ProducerStage) Name() string { return "production_distribution" }
func (ProducerStage) Run(ctx context.Context, s *PlanState) error {
	var wrap struct {
		Production   model.Production   `json:"production"`
		Distribution model.Distribution `json:"distribution"`
	}
	data := map[string]any{"Plan": s.Plan}
	if err := call(ctx, s, "production_distribution", data, &wrap, false); err != nil {
		return err
	}
	s.Plan.Production = wrap.Production
	s.Plan.Distribution = wrap.Distribution
	return nil
}

// AllStages returns the ordered pipeline.
func AllStages() []Stage {
	return []Stage{
		ConceptStage{}, BibleStage{}, CharacterStage{}, EpisodeStage{},
		PlacementStage{}, HeroStage{}, ProducerStage{},
	}
}
