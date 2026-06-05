package render

import (
	"strings"
	"testing"

	"github.com/ashley/drama-workbench/internal/model"
)

func TestMarkdownIncludesTitleAndEpisodes(t *testing.T) {
	p := &model.Plan{
		Bible:    model.SeriesBible{Title: "Dream Home"},
		Episodes: []model.Episode{{Number: 1, Title: "Pilot", Hook: "h"}},
	}
	md := Markdown(p)
	if !strings.Contains(md, "Dream Home") {
		t.Fatal("missing title")
	}
	if !strings.Contains(md, "Pilot") {
		t.Fatal("missing episode")
	}
}
