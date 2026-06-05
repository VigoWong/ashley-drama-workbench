package render

import (
	"fmt"
	"strings"

	"github.com/ashley/drama-workbench/internal/model"
)

func Markdown(p *model.Plan) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# %s\n\n", nz(p.Bible.Title, "Untitled Series"))
	w("## Concept\n- **Logline:** %s\n- **Theme:** %s\n- **Audience:** %s\n- **Tone:** %s\n- **Payoff engine:** %s\n- **Core conflict:** %s\n\n",
		p.Concept.Logline, p.Concept.Theme, p.Concept.Audience, p.Concept.Tone, p.Concept.PayoffEngine, p.Concept.CoreConflict)
	w("## Series Bible\n- Episodes: %d x %ds | Platform: %s\n- Integration: %s\n\n",
		p.Bible.Episodes, p.Bible.EpisodeSecs, p.Bible.Platform, p.Bible.IntegrationThesis)
	w("## Characters\n")
	for _, c := range p.Characters {
		w("- **%s** (%s): %s _Arc:_ %s\n", c.Name, c.Role, c.Bio, c.Arc)
	}
	w("\n## Episodes\n")
	for _, e := range p.Episodes {
		w("### Ep %d — %s\n%s\n- **Hook:** %s\n- **Cliffhanger:** %s\n- **Payoff:** %s\n\n",
			e.Number, e.Title, e.Synopsis, e.Hook, e.Cliffhanger, e.Payoff)
	}
	w("## Brand Integration\n")
	for _, pl := range p.Placements {
		w("- Ep %d — %s (%s): %s | CTA: %s\n", pl.Episode, pl.Category, pl.ProductSKU, pl.EmotionalBeat, pl.CTATiming)
	}
	w("\n## Hero Scenes\n")
	for _, h := range p.HeroScenes {
		w("### Ep %d — %s\n", h.Episode, h.Title)
		for _, s := range h.Shots {
			w("%d. [%s] %s — \"%s\"\n", s.Number, s.ShotType, s.Action, s.Dialogue)
		}
	}
	w("\n## Production\n- Format: %s | Budget: %s | Shots: %d | Cast: %d\n- Furniture: %s\n\n",
		p.Production.Format, p.Production.BudgetTier, p.Production.ShotCount, p.Production.CastSize, strings.Join(p.Production.FurnitureProps, ", "))
	w("## Distribution\n- CTA: %s\n- Hashtags: %s\n", p.Distribution.CTACopy, strings.Join(p.Distribution.Hashtags, " "))
	return b.String()
}

func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
