// backend/internal/agent/chat_llm.go
package agent

import "context"

// Role identifies who produced a conversation Message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool" // a tool's observation fed back to the model
)

// Message is one entry in the conversation history handed to the ChatLLM. For
// RoleTool entries, ToolCallID/ToolName identify which call this observes and
// ToolResult is the JSON-encoded Observation. For RoleAssistant entries that
// triggered tool calls, ToolCalls carries the calls so the Gemini adapter can
// reconstruct the required functionCall parts in multi-turn history.
type Message struct {
	Role       Role       `json:"role"`
	Text       string     `json:"text,omitempty"`
	ToolCallID string     `json:"toolCallId,omitempty"`
	ToolName   string     `json:"toolName,omitempty"`
	ToolResult string     `json:"toolResult,omitempty"`
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`
}

// ToolCall is a single tool invocation the model wants the engine to run.
type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// Turn is one model step. Exactly one of {ToolCalls non-empty} or {Message set}
// is the "payload": if the model wants to act it returns ToolCalls; if it's done
// it returns a final Message. Thought is optional reasoning shown before either.
type Turn struct {
	Thought   string     `json:"thought,omitempty"`
	Message   string     `json:"message,omitempty"`
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// ChatLLM is the model abstraction the ReAct engine drives. Implementations:
// ScriptedLLM (no-key/tests) and geminiChatLLM (real function-calling).
type ChatLLM interface {
	NextTurn(ctx context.Context, system string, history []Message, tools []ToolDef) (Turn, error)
}
