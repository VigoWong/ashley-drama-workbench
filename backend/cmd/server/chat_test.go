package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleChatStreamsReActEvents(t *testing.T) {
	body := `{"message":"做个家装逆袭短剧,植入沙发"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handleChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	out := rec.Body.String()
	// The no-key demo script must drive a full trace: tool calls + block events +
	// a closing turn.done, all over SSE ("data: " framed).
	for _, want := range []string{
		"data: ", `"type":"tool.start"`, `"type":"block.done"`,
		`"stage":"concept"`, `"type":"turn.done"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("SSE output missing %q\n---\n%s", want, out)
		}
	}
}
