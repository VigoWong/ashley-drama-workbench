package llm

import (
	"log"
	"os"
)

// FromEnv picks an LLM provider from the environment, in priority order:
//
//  1. Vertex AI — when VERTEX_CREDENTIALS_FILE points at a service-account JSON.
//     Project defaults to the SA's own project_id (override: VERTEX_PROJECT),
//     location defaults to us-central1 (override: VERTEX_LOCATION), model from
//     GEMINI_MODEL (default gemini-2.5-flash).
//  2. AI Studio — when GEMINI_API_KEY is set (optional GEMINI_BASE_URL proxy).
//  3. Demo Mock — so the chain always runs without any credentials.
//
// Returns (provider, usingMock).
func FromEnv() (Provider, bool) {
	if credFile := os.Getenv("VERTEX_CREDENTIALS_FILE"); credFile != "" {
		saJSON, err := os.ReadFile(credFile)
		if err != nil {
			log.Printf("vertex: cannot read %s: %v — falling back", credFile, err)
		} else if p, err := NewVertex(saJSON, os.Getenv("VERTEX_PROJECT"), os.Getenv("VERTEX_LOCATION"), os.Getenv("GEMINI_MODEL")); err != nil {
			log.Printf("vertex: init failed: %v — falling back", err)
		} else {
			return p, false
		}
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return NewGemini(key, os.Getenv("GEMINI_MODEL"), os.Getenv("GEMINI_BASE_URL")), false
	}
	return DemoMock(), true
}
