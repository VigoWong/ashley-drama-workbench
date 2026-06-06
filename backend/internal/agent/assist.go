package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/prompts"
)

// Assist powers the front-end「提示词助手」: it takes the user's rough idea (raw —
// may be a few keywords, half a sentence, or empty) and a single lightweight LLM
// call expands/polishes it into one complete, ready-to-generate 中文生成需求 that
// fuses 题材/套路、爽点引擎、人物冲突 and the Ashley furniture placement focus. It
// does NOT run the pipeline; the returned text just pre-fills the requirement box,
// and the user can edit it freely before /api/propose or /api/generate.
func Assist(ctx context.Context, provider llm.Provider, raw string, episodes, episodeSecs int, images []model.Image) (string, error) {
	if episodes <= 0 {
		episodes = 5
	}
	if episodeSecs <= 0 {
		episodeSecs = 30
	}
	prompt, err := prompts.Render("assist", map[string]any{
		"Raw": raw, "Episodes": episodes, "EpisodeSecs": episodeSecs,
		"HasImages": len(images) > 0,
	})
	if err != nil {
		return "", fmt.Errorf("assist: render: %w", err)
	}

	// Feed any reference images so the expanded 需求 is grounded in what the user
	// actually uploaded (room/color/product/mood) — the same multimodal anchoring
	// propose/generate already use.
	out, err := provider.GenerateJSON(ctx, "assist", prompt, images, nil)
	if err != nil {
		return "", fmt.Errorf("assist: generate: %w", err)
	}

	var wrap struct {
		Requirement string `json:"requirement"`
	}
	if err := json.Unmarshal(out, &wrap); err != nil {
		return "", fmt.Errorf("assist: unmarshal: %w (raw=%s)", err, truncate(out))
	}
	if wrap.Requirement == "" {
		return "", fmt.Errorf("assist: model returned empty requirement")
	}
	return wrap.Requirement, nil
}
