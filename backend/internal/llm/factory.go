package llm

import "os"

// FromEnv returns Gemini when GEMINI_API_KEY is set, otherwise a demo Mock so
// the chain always runs. Returns (provider, usingMock).
func FromEnv() (Provider, bool) {
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return NewGemini(key, os.Getenv("GEMINI_MODEL")), false
	}
	return DemoMock(), true
}
