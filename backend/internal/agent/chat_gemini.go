package agent

import (
	"context"

	"github.com/ashley/drama-workbench/internal/llm"
)

// geminiChatLLM adapts any llm.ToolCaller (the Gemini provider) to the engine's
// ChatLLM interface, translating between the agent conversation types and the
// provider's tool-calling types.
type geminiChatLLM struct{ tc llm.ToolCaller }

// NewGeminiChatLLM wraps a provider for the chat engine. The provider must
// implement llm.ToolCaller (Gemini does); callers gate on usingMock so this is
// only constructed when real credentials are present. If the provider does not
// implement ToolCaller (e.g. DemoMock), it falls back to a scripted fallback
// message rather than panicking.
func NewGeminiChatLLM(p llm.Provider) ChatLLM {
	tc, ok := p.(llm.ToolCaller)
	if !ok {
		// Defensive: fall back to a single "I can't act" message rather than panic.
		return NewScriptedLLM([]Turn{{Message: "当前后端未启用工具调用能力。"}})
	}
	return &geminiChatLLM{tc: tc}
}

func (g *geminiChatLLM) NextTurn(ctx context.Context, system string, history []Message, defs []ToolDef) (Turn, error) {
	specs := make([]llm.ToolSpec, len(defs))
	for i, d := range defs {
		specs[i] = llm.ToolSpec{Name: d.Name, Description: d.Description, Parameters: d.Parameters}
	}

	msgs := make([]llm.ToolMessage, len(history))
	for i, m := range history {
		switch m.Role {
		case RoleTool:
			msgs[i] = llm.ToolMessage{Role: "tool", Name: m.ToolName, Result: m.ToolResult}
		case RoleAssistant:
			// Carry the function calls so GenerateWithTools can emit functionCall
			// parts in the "model" history content, as required by Gemini's
			// multi-turn function-calling protocol.
			var refs []llm.ToolCallRef
			if len(m.ToolCalls) > 0 {
				refs = make([]llm.ToolCallRef, len(m.ToolCalls))
				for j, tc := range m.ToolCalls {
					refs[j] = llm.ToolCallRef{Name: tc.Name, Args: tc.Args}
				}
			}
			msgs[i] = llm.ToolMessage{Role: "model", Text: m.Text, ToolCalls: refs}
		default: // RoleUser
			msgs[i] = llm.ToolMessage{Role: "user", Text: m.Text}
		}
	}

	reply, err := g.tc.GenerateWithTools(ctx, system, msgs, specs)
	if err != nil {
		return Turn{}, err
	}

	turn := Turn{Thought: reply.Thought, Message: reply.Text}
	for _, c := range reply.ToolCalls {
		turn.ToolCalls = append(turn.ToolCalls, ToolCall{ID: c.ID, Name: c.Name, Args: c.Args})
	}
	return turn, nil
}
