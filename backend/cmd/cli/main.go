package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ashley/drama-workbench/internal/agent"
	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/render"
)

func main() {
	genre := flag.String("genre", "home makeover revenge", "drama genre/theme")
	episodes := flag.Int("episodes", 12, "number of episodes")
	secs := flag.Int("secs", 90, "seconds per episode")
	brand := flag.String("brand", "living room & bedroom", "brand focus")
	format := flag.String("format", "markdown", "output: markdown|json")
	out := flag.String("out", "", "write to file instead of stdout")
	flag.Parse()

	provider, mock := llm.FromEnv()
	if mock {
		fmt.Fprintln(os.Stderr, "[demo mode: no GEMINI_API_KEY, using mock provider]")
	}

	emit := func(e model.Event) {
		if e.Type == model.EventStageStart {
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s...\n", e.Index+1, e.Total, e.Stage)
		}
	}
	o := agent.New(provider, emit)
	plan, err := o.Run(context.Background(), model.Brief{Genre: *genre, Episodes: *episodes, EpisodeSecs: *secs, BrandFocus: *brand})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	var result string
	if *format == "json" {
		b, _ := json.MarshalIndent(plan, "", "  ")
		result = string(b)
	} else {
		result = render.Markdown(plan)
	}
	if *out != "" {
		if err := os.WriteFile(*out, []byte(result), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
	} else {
		fmt.Println(result)
	}
}
