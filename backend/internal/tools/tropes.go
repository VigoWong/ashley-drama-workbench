package tools

type Trope struct {
	Name       string `json:"name"`
	Hook       string `json:"hook"`
	WhyItWorks string `json:"whyItWorks"`
	HomeAngle  string `json:"homeAngle"` // how furniture earns screen time
}

var homeTropes = []Trope{
	{"From-broke-to-dream-home", "Evicted heroine vows to rebuild", "Aspirational status arc, fast payoff", "Every upgrade = a new furniture reveal"},
	{"Fresh-start-after-divorce", "She walks out with one suitcase", "Empowerment + clean slate", "Furnishing the new place = healing montage"},
	{"Secret-heir-renovation", "Handyman is secretly a billionaire", "Reversal-driven, identity reveal", "The mansion makeover showcases premium lines"},
	{"Family-reconciliation", "Estranged siblings inherit a house", "Emotional payoff, warmth", "Shared living/dining scenes anchor the brand"},
}

func GetWinningTropes(market, vertical string) []Trope {
	// MVP: single curated home vertical; market reserved for future expansion.
	return homeTropes
}
