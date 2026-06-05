package model

type EventType string

const (
	EventStageStart EventType = "stage_start"
	EventStageDone  EventType = "stage_done"
	EventError      EventType = "error"
	EventComplete   EventType = "complete"
)

// Event is what the orchestrator emits and the server streams as SSE.
type Event struct {
	Type    EventType   `json:"type"`
	Stage   string      `json:"stage,omitempty"`
	Index   int         `json:"index,omitempty"` // 0-based stage index
	Total   int         `json:"total,omitempty"`
	Message string      `json:"message,omitempty"`
	Payload interface{} `json:"payload,omitempty"` // partial output for this stage
	Plan    *Plan       `json:"plan,omitempty"`    // set on EventComplete
}
