package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolSpec is a provider-agnostic tool declaration passed to GenerateWithTools.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON schema
}

// ToolCallOut is a tool invocation the model asked for.
type ToolCallOut struct {
	ID   string
	Name string
	Args map[string]any
}

// ToolCallRef carries the name and args of a single function call that the
// model previously requested. It is stored on model-role ToolMessages so that
// GenerateWithTools can reconstruct the functionCall parts Gemini requires in
// the "model" content that precedes each functionResponse turn.
type ToolCallRef struct {
	Name string
	Args map[string]any
}

// ToolMessage is one conversation entry for the tool-calling API. Role is
// "user" | "model" | "tool". For tool messages, Name + Result (JSON string)
// carry the observation.
// ToolCalls is populated only for model-role messages that previously triggered
// function calls; it drives reconstruction of functionCall parts in the history.
type ToolMessage struct {
	Role      string
	Text      string
	Name      string
	Result    string
	ToolCalls []ToolCallRef // non-nil only for role=="model" that had function calls
}

// ToolReply is the model's response: either Text (final) or ToolCalls (act).
// Thought accumulates any leading text parts that precede function calls.
type ToolReply struct {
	Thought   string
	Text      string
	ToolCalls []ToolCallOut
}

// ToolCaller is implemented by providers that support native function-calling.
// Gemini (Vertex + AI Studio) implements it; Mock does not.
type ToolCaller interface {
	GenerateWithTools(ctx context.Context, system string, history []ToolMessage, tools []ToolSpec) (ToolReply, error)
}

// GenerateWithTools maps the conversation + tool specs to Gemini's
// functionDeclarations / functionCall / functionResponse protocol, issues one
// :generateContent call, and parses the first candidate into a ToolReply.
//
// History roles are mapped as:
//   - "user"  → "user" content with a text part
//   - "model" → "model" content with a text part
//   - "tool"  → "user" content with a functionResponse part (Gemini groups
//     function responses under the "user" role in multi-turn conversations)
//
// System instructions go into the top-level "systemInstruction" field.
// Tools are declared under "tools":[{"functionDeclarations":[...]}].
func (g *Gemini) GenerateWithTools(ctx context.Context, system string, history []ToolMessage, tools []ToolSpec) (ToolReply, error) {
	type fnDecl struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters,omitempty"`
	}
	decls := make([]fnDecl, len(tools))
	for i, t := range tools {
		decls[i] = fnDecl{Name: t.Name, Description: t.Description, Parameters: t.Parameters}
	}

	// Wire types for the :generateContent request.
	type textPart struct {
		Text string `json:"text,omitempty"`
	}
	type fnRespInner struct {
		Name     string `json:"name"`
		Response any    `json:"response"`
	}
	type fnRespPart struct {
		FunctionResponse fnRespInner `json:"functionResponse"`
	}
	type rawContent struct {
		Role  string `json:"role"`
		Parts []any  `json:"parts"`
	}

	// fnCallPart is the wire shape for a functionCall part in a "model" content.
	type fnCallInner struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	}
	type fnCallPart struct {
		FunctionCall fnCallInner `json:"functionCall"`
	}

	contents := make([]rawContent, 0, len(history))
	for _, m := range history {
		switch m.Role {
		case "tool":
			// Gemini expects functionResponse parts inside a "user" role turn.
			var resp any
			if err := json.Unmarshal([]byte(m.Result), &resp); err != nil {
				resp = m.Result // fall back to raw string if not valid JSON
			}
			part := fnRespPart{FunctionResponse: fnRespInner{
				Name:     m.Name,
				Response: map[string]any{"result": resp},
			}}
			contents = append(contents, rawContent{Role: "user", Parts: []any{part}})
		case "model":
			// Reconstruct the full "model" content. If this turn had function calls
			// (stored in ToolCalls), we must include the functionCall parts so Gemini
			// can correlate them with the subsequent functionResponse parts. Omitting
			// them causes a 400 on the second+ tool iteration.
			parts := make([]any, 0, len(m.ToolCalls)+1)
			if m.Text != "" {
				parts = append(parts, textPart{Text: m.Text})
			}
			for _, tc := range m.ToolCalls {
				parts = append(parts, fnCallPart{FunctionCall: fnCallInner{Name: tc.Name, Args: tc.Args}})
			}
			if len(parts) == 0 {
				// Safety fallback: Gemini requires at least one part.
				parts = append(parts, textPart{Text: ""})
			}
			contents = append(contents, rawContent{Role: "model", Parts: parts})
		default: // "user"
			contents = append(contents, rawContent{Role: "user", Parts: []any{textPart{Text: m.Text}}})
		}
	}

	// Fix 1: disable thinking for 2.5 models during tool-calling turns, matching
	// the same guard used in once() for plain GenerateJSON calls.
	var generationConfig *struct {
		ThinkingConfig *thinkingConfig `json:"thinkingConfig,omitempty"`
	}
	if strings.Contains(g.model, "2.5") {
		generationConfig = &struct {
			ThinkingConfig *thinkingConfig `json:"thinkingConfig,omitempty"`
		}{
			ThinkingConfig: &thinkingConfig{ThinkingBudget: 0},
		}
	}

	reqBody := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]any{{"text": system}},
		},
		"contents": contents,
		"tools":    []map[string]any{{"functionDeclarations": decls}},
	}
	if generationConfig != nil {
		reqBody["generationConfig"] = generationConfig
	}

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return ToolReply{}, fmt.Errorf("gemini tools: marshal request: %w", err)
	}

	raw, err := g.doGenerate(ctx, encoded)
	if err != nil {
		return ToolReply{}, err
	}

	// Parse the response. A part can be either a text part or a functionCall part.
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ToolReply{}, fmt.Errorf("gemini tools: parse response: %w", err)
	}

	var reply ToolReply
	if len(parsed.Candidates) == 0 {
		return reply, nil
	}
	parts := parsed.Candidates[0].Content.Parts
	hasFnCalls := false
	for _, p := range parts {
		if p.FunctionCall != nil {
			hasFnCalls = true
			break
		}
	}
	for i, p := range parts {
		if p.FunctionCall != nil {
			reply.ToolCalls = append(reply.ToolCalls, ToolCallOut{
				ID:   fmt.Sprintf("%s_%d", p.FunctionCall.Name, i),
				Name: p.FunctionCall.Name,
				Args: p.FunctionCall.Args,
			})
		} else if p.Text != "" {
			if hasFnCalls {
				// Text that accompanies tool calls is reasoning / thought.
				reply.Thought += p.Text
			} else {
				reply.Text += p.Text
			}
		}
	}
	return reply, nil
}
