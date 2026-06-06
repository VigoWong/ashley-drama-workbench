// backend/internal/model/chat.go
package model

// ChatEventType is the discriminator for the conversational ReAct stream. These
// are the fine-grained successors to the stage-oriented EventType: the front end
// drives the chat column (thought/tool/message) and the live canvas (block.*)
// off these.
type ChatEventType string

const (
	ChatThoughtDelta ChatEventType = "thought.delta" // streaming reasoning token(s)
	ChatThoughtDone  ChatEventType = "thought.done"  // reasoning finished; Text = one-line summary
	ChatToolStart    ChatEventType = "tool.start"    // a tool call began (running)
	ChatToolResult   ChatEventType = "tool.result"   // a tool call finished (ok/fail) + I/O
	ChatMessageDelta ChatEventType = "message.delta" // streaming assistant reply token(s)
	ChatMessageDone  ChatEventType = "message.done"  // assistant reply finished
	ChatBlockStart   ChatEventType = "block.start"   // a plan block entered "writing" state
	ChatBlockDone    ChatEventType = "block.done"    // a plan block finished; Payload = its content
	ChatTurnDone     ChatEventType = "turn.done"     // the whole agent turn is done; input re-enabled
	ChatErrorEvent   ChatEventType = "error"         // a fatal error for this turn
)

// ChatEvent is what the ReAct engine emits and the server streams as SSE on
// /api/chat. All non-essential fields are omitempty so each event serialises to
// just the keys its Type needs (a discriminated union on the wire).
type ChatEvent struct {
	Type ChatEventType `json:"type"`

	// thought.* / message.* / error
	Text    string `json:"text,omitempty"`
	Message string `json:"message,omitempty"`

	// tool.*
	ToolID       string      `json:"toolId,omitempty"`
	ToolName     string      `json:"toolName,omitempty"`
	FriendlyName string      `json:"friendlyName,omitempty"`
	Input        interface{} `json:"input,omitempty"`
	Output       interface{} `json:"output,omitempty"`
	Status       string      `json:"status,omitempty"`       // "ok" | "fail" (tool.result)
	AffectsStage string      `json:"affectsStage,omitempty"` // canvas linkage, if the tool touched a block

	// block.*
	Stage   string      `json:"stage,omitempty"`
	Payload interface{} `json:"payload,omitempty"`

	// turn.done (optional snapshot)
	Plan *Plan `json:"plan,omitempty"`
}
