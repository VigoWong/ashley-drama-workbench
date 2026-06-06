package llm

import (
	"context"
	"errors"

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

// ErrImagesUnsupported is returned by ImageProvider implementations that cannot
// generate images in their current mode (e.g. AI Studio Gemini, or the Mock).
// Callers (VisualStage) treat it as a graceful "no images" signal, not a fatal
// error — the text plan is still complete.
var ErrImagesUnsupported = errors.New("image generation not supported by this provider")

// ImageProvider is an optional capability: a Provider that can also synthesize
// images from a text prompt (text-to-image). Returns raw image bytes and the
// MIME type, or ErrImagesUnsupported when unavailable. Implemented by the Vertex
// AI Gemini provider (via Imagen); other providers may not implement it at all.
type ImageProvider interface {
	GenerateImage(ctx context.Context, prompt string) (data []byte, mimeType string, err error)
}
