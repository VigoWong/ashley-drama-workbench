package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/ashley/drama-workbench/internal/llm"
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

// VisualStage generates AI concept art (a series key-art poster plus a few hero
// scene stills) via the provider's optional ImageProvider capability (Vertex
// Imagen). It is best-effort and resilient by design:
//   - If the provider does not support image generation, it is a no-op (no error,
//     no images) — the text plan is already complete.
//   - Individual image failures are logged and skipped; one bad image never
//     drops the others.
//   - The stage always returns nil so the orchestrator never retries or aborts
//     the whole pipeline over visuals.
type VisualStage struct{}

func (VisualStage) Name() string { return "visuals" }

// maxVisuals caps how many concept images we request (1 poster + up to 2 hero
// scenes). Each Imagen call is slow and costly, so we keep the gallery tight.
const maxVisuals = 3

func (VisualStage) Run(ctx context.Context, s *PlanState) error {
	ip, ok := s.Provider.(llm.ImageProvider)
	if !ok {
		return nil // provider cannot make images; text plan stands on its own
	}

	type job struct {
		label  string
		prompt string
	}
	jobs := make([]job, 0, maxVisuals)

	// 1) Series key-art poster.
	title := s.Plan.Bible.Title
	if title == "" {
		title = s.Plan.Concept.Logline
	}
	posterPrompt := fmt.Sprintf(
		"vertical 9:16 cinematic key art poster for a Chinese vertical short drama titled %q. "+
			"Genre and tone: %s. Premise: %s. "+
			"Modern Chinese home interior with stylish furniture, dramatic moody lighting, "+
			"high-end streaming poster look, photorealistic, no text, no watermark, no logo.",
		title, s.Plan.Concept.Tone, s.Plan.Concept.Logline,
	)
	jobs = append(jobs, job{label: "系列海报", prompt: posterPrompt})

	// 2) One still per hero scene, until we hit the cap.
	for _, h := range s.Plan.HeroScenes {
		if len(jobs) >= maxVisuals {
			break
		}
		var actions []string
		for _, sh := range h.Shots {
			if sh.Action != "" {
				actions = append(actions, sh.Action)
			}
		}
		desc := strings.Join(actions, "; ")
		scenePrompt := fmt.Sprintf(
			"vertical 9:16 cinematic film still for a Chinese vertical short drama scene titled %q. "+
				"Scene action: %s. "+
				"Modern Chinese home interior with realistic furniture, dramatic cinematic lighting, "+
				"photorealistic, shallow depth of field, no text, no watermark, no logo.",
			h.Title, desc,
		)
		jobs = append(jobs, job{label: "分镜·" + h.Title, prompt: scenePrompt})
	}

	for _, j := range jobs {
		data, mime, err := ip.GenerateImage(ctx, j.prompt)
		if err != nil {
			if errors.Is(err, llm.ErrImagesUnsupported) {
				return nil // capability vanished; stop quietly
			}
			log.Printf("visuals: skip %q: %v", j.label, err)
			continue
		}
		s.Plan.Visuals = append(s.Plan.Visuals, model.Visual{
			Label:    j.label,
			MimeType: mime,
			Data:     base64.StdEncoding.EncodeToString(data),
		})
	}
	return nil
}

// AllStages returns the ordered pipeline.
func AllStages() []Stage {
	return []Stage{
		ConceptStage{}, BibleStage{}, CharacterStage{}, EpisodeStage{},
		PlacementStage{}, HeroStage{}, ProducerStage{}, VisualStage{},
	}
}
