// backend/internal/model/chat_test.go
package model

import (
	"encoding/json"
	"testing"
)

func TestChatEventMarshalsToolResult(t *testing.T) {
	e := ChatEvent{
		Type:         ChatToolResult,
		ToolID:       "t1",
		ToolName:     "validatePacing",
		Status:       "ok",
		Output:       map[string]any{"pass": true},
		AffectsStage: "episodes",
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if !contains(got, `"type":"tool.result"`) || !contains(got, `"affectsStage":"episodes"`) {
		t.Fatalf("unexpected json: %s", got)
	}
	// zero-value fields must be omitted so the FE union stays clean
	if contains(got, `"text"`) || contains(got, `"plan"`) {
		t.Fatalf("zero fields not omitted: %s", got)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
