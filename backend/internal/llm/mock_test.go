package llm

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMockReturnsRegisteredJSON(t *testing.T) {
	m := NewMock()
	m.Register("concept", `{"logline":"x"}`)
	out, err := m.GenerateJSON(context.Background(), "concept", "prompt body", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got["logline"] != "x" {
		t.Fatalf("got %v", got)
	}
}

func TestMockUnknownStageErrors(t *testing.T) {
	m := NewMock()
	if _, err := m.GenerateJSON(context.Background(), "nope", "p", nil, nil); err == nil {
		t.Fatal("expected error for unregistered stage")
	}
}
