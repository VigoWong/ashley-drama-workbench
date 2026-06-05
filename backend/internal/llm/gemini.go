package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// defaultGeminiBase is Google's native Generative Language API host. It can be
// overridden (e.g. to an OpenAI-compatible proxy that also speaks the native
// generateContent shape, like yunwu.ai) via GEMINI_BASE_URL.
const defaultGeminiBase = "https://generativelanguage.googleapis.com"

// Gemini calls Google's native generateContent API. It runs in one of two
// modes depending on how it was constructed:
//   - AI Studio (NewGemini): API key in the query string, against baseURL.
//   - Vertex AI (NewVertex): OAuth2 bearer token from a service account,
//     against the regional aiplatform.googleapis.com endpoint.
//
// The request/response wire format is identical in both modes; only the URL
// and auth differ.
type Gemini struct {
	model  string
	client *http.Client

	// AI Studio mode
	apiKey  string
	baseURL string

	// Vertex AI mode (active when tokenSource != nil)
	tokenSource oauth2.TokenSource
	project     string
	location    string
}

func NewGemini(apiKey, model, baseURL string) *Gemini {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	if baseURL == "" {
		baseURL = defaultGeminiBase
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Gemini{apiKey: apiKey, model: model, baseURL: baseURL, client: &http.Client{Timeout: 90 * time.Second}}
}

// NewVertex builds a provider that authenticates to Vertex AI with a Google
// service-account JSON and talks to the regional endpoint for project/location.
// The token source refreshes access tokens automatically.
func NewVertex(saJSON []byte, project, location, model string) (*Gemini, error) {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if location == "" {
		location = "us-central1"
	}
	cfg, err := google.JWTConfigFromJSON(saJSON, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("vertex credentials: %w", err)
	}
	if project == "" {
		// Fall back to the project the service account itself belongs to.
		var sa struct {
			ProjectID string `json:"project_id"`
		}
		_ = json.Unmarshal(saJSON, &sa)
		project = sa.ProjectID
	}
	if project == "" {
		return nil, fmt.Errorf("vertex: project id missing (set VERTEX_PROJECT or use a key with project_id)")
	}
	return &Gemini{
		model:       model,
		project:     project,
		location:    location,
		tokenSource: cfg.TokenSource(context.Background()),
		client:      &http.Client{Timeout: 90 * time.Second},
	}, nil
}

type geminiReq struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig genConfig       `json:"generationConfig"`
}
type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}
type geminiPart struct {
	Text string `json:"text"`
}
type genConfig struct {
	ResponseMimeType string          `json:"responseMimeType"`
	Temperature      float64         `json:"temperature"`
	ThinkingConfig   *thinkingConfig `json:"thinkingConfig,omitempty"`
}
type thinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}
type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (g *Gemini) GenerateJSON(ctx context.Context, stage, prompt string, _ map[string]any) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		raw, err := g.once(ctx, prompt)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			continue
		}
		clean := stripFences(raw)
		if json.Valid([]byte(clean)) {
			return []byte(clean), nil
		}
		// repair pass: ask the model to fix to strict JSON
		prompt = "Your previous output was not valid JSON. Return ONLY valid JSON, no prose, no fences.\nPREVIOUS:\n" + raw
		lastErr = fmt.Errorf("%s: invalid JSON after attempt %d", stage, attempt)
	}
	return nil, lastErr
}

func (g *Gemini) once(ctx context.Context, prompt string) (string, error) {
	cfg := genConfig{ResponseMimeType: "application/json", Temperature: 0.9}
	// Gemini 2.5 models think by default and leak the reasoning into the text
	// part, which breaks strict-JSON parsing. Disable it so we get clean JSON.
	if strings.Contains(g.model, "2.5") {
		cfg.ThinkingConfig = &thinkingConfig{ThinkingBudget: 0}
	}
	body, _ := json.Marshal(geminiReq{
		Contents:         []geminiContent{{Role: "user", Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: cfg,
	})

	var url string
	if g.tokenSource != nil {
		url = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
			g.location, g.project, g.location, g.model)
	} else {
		url = fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if g.tokenSource != nil {
		tok, err := g.tokenSource.Token()
		if err != nil {
			return "", fmt.Errorf("vertex token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gemini %d: %s", resp.StatusCode, truncStr(string(data)))
	}
	var gr geminiResp
	if err := json.Unmarshal(data, &gr); err != nil {
		return "", err
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	return gr.Candidates[0].Content.Parts[0].Text, nil
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncStr(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
