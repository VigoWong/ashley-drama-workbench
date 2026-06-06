package prompts

import "testing"

func TestRenderSubstitutes(t *testing.T) {
	out, err := Render("concept", map[string]any{"Requirement": "makeover"})
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("empty render")
	}
}
