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
)

type Gemini struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGemini(apiKey, model string) *Gemini {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &Gemini{apiKey: apiKey, model: model, client: &http.Client{Timeout: 90 * time.Second}}
}

type geminiReq struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig genConfig       `json:"generationConfig"`
}
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}
type geminiPart struct {
	Text string `json:"text"`
}
type genConfig struct {
	ResponseMimeType string  `json:"responseMimeType"`
	Temperature      float64 `json:"temperature"`
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
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	body, _ := json.Marshal(geminiReq{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: genConfig{ResponseMimeType: "application/json", Temperature: 0.9},
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
