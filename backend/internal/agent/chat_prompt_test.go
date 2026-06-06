// backend/internal/agent/chat_prompt_test.go
package agent

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptListsToolsAndGuidance(t *testing.T) {
	p := BuildSystemPrompt(DefaultRegistry())
	// must name every tool so the model knows its action space
	for _, name := range []string{"generateConcept", "generateEpisodes", "validatePacing", "refineBlock"} {
		if !strings.Contains(p, name) {
			t.Fatalf("system prompt missing tool %q", name)
		}
	}
	// must carry the guided-ReAct guidance: recommended order + self-check
	if !strings.Contains(p, "validatePacing") || !strings.Contains(p, "顺序") {
		t.Fatal("system prompt missing guided-order / self-check guidance")
	}
}
