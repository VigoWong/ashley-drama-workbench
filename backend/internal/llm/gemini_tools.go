package llm

import (
	"context"
	"encoding/json"
	"fmt"
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

// ToolMessage is one conversation entry for the tool-calling API. Role is
// "user" | "model" | "tool". For tool messages, Name + Result (JSON string)
// carry the observation.
type ToolMessage struct {
	Role   string
	Text   string
	Name   string
	Result string
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
			contents = append(contents, rawContent{Role: "model", Parts: []any{textPart{Text: m.Text}}})
		default: // "user"
			contents = append(contents, rawContent{Role: "user", Parts: []any{textPart{Text: m.Text}}})
		}
	}

	reqBody := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]any{{"text": system}},
		},
		"contents": contents,
		"tools":    []map[string]any{{"functionDeclarations": decls}},
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
