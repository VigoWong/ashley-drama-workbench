package llm

import "context"

// Provider abstracts the LLM. `stage` is a logical name used by Mock to route
// fixtures and by Gemini for logging. `schema` is an optional JSON schema
// (map[string]any) passed to Gemini's responseSchema; Mock ignores it.
type Provider interface {
	GenerateJSON(ctx context.Context, stage, prompt string, schema map[string]any) ([]byte, error)
}
