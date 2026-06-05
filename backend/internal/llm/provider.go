package llm

import (
	"context"

	"github.com/ashley/drama-workbench/internal/model"
)

// Provider abstracts the LLM. `stage` is a logical name used by Mock to route
// fixtures and by Gemini for logging. `images` are optional multimodal
// reference images sent alongside the prompt (Mock ignores them). `schema` is
// an optional JSON schema (map[string]any) passed to Gemini's responseSchema;
// Mock ignores it.
type Provider interface {
	GenerateJSON(ctx context.Context, stage, prompt string, images []model.Image, schema map[string]any) ([]byte, error)
}
