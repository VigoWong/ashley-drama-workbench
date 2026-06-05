package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

type Mock struct{ fixtures map[string]string }

func NewMock() *Mock { return &Mock{fixtures: map[string]string{}} }

func (m *Mock) Register(stage, jsonOut string) { m.fixtures[stage] = jsonOut }

func (m *Mock) GenerateJSON(_ context.Context, stage, _ string, _ map[string]any) ([]byte, error) {
	v, ok := m.fixtures[stage]
	if !ok {
		return nil, fmt.Errorf("mock: no fixture registered for stage %q", stage)
	}
	return []byte(v), nil
}

// DemoMock returns a *Mock pre-registered with rich, plausible fixtures for all
// 7 pipeline stages so a keyless server/CLI still produces a full, sensible
// plan. The theme is a US-market Ashley-furniture branded makeover/revenge
// drama using real SKUs from the product catalog. The Mock ignores the prompt
// and returns the same fixture per stage regardless of requested episode count,
// so the episode array is a fixed ~12-episode season.
func DemoMock() *Mock {
	m := NewMock()
	m.Register("concept", `{
		"logline": "A penniless interior designer is thrown out by her cheating husband — then returns as the secret heiress who buys the entire neighborhood, one stunning home at a time.",
		"theme": "Self-worth, reinvention, and the home you build for yourself",
		"audience": "US women 25-45 on ReelShort & DramaBox who binge revenge-glow-up dramas",
		"tone": "Glossy, emotionally charged, satisfying reversals every episode",
		"payoffEngine": "Status-climb revenge: each episode the heroine reclaims power and unveils a more beautiful, fully-furnished space than her enemies can afford.",
		"coreConflict": "Cast out with nothing, she must rebuild her life and home while hiding her inheritance to expose the people who betrayed her.",
		"tropesUsed": ["From-broke-to-dream-home", "Secret-heir-renovation", "Fresh-start-after-divorce"]
	}`)

	m.Register("bible", `{
		"title": "The House She Built",
		"genreTags": ["revenge", "makeover", "romance", "rags-to-riches"],
		"episodes": 12,
		"episodeSecs": 90,
		"totalRuntimeMin": 18,
		"platform": "ReelShort / DramaBox (vertical 9:16)",
		"integrationThesis": "Every act of the heroine's comeback is staged in a fully realized Ashley living space — the furniture is the visible scoreboard of her rising status, so brand reveals double as emotional payoffs."
	}`)

	m.Register("characters", `{"characters": [
		{
			"name": "Mia Calloway",
			"role": "protagonist",
			"bio": "A gifted interior designer who gave up her career to support her husband's firm, only to be discarded the moment she became inconvenient.",
			"arc": "From humiliated and broke to a self-made tastemaker who turns every room into a statement of independence.",
			"relationships": "Estranged wife of Derek; secret granddaughter of the late real-estate magnate Arthur Vance."
		},
		{
			"name": "Derek Sloan",
			"role": "antagonist",
			"bio": "Mia's social-climbing husband who traded her in for a wealthier partner the night before her inheritance was revealed.",
			"arc": "From smug and untouchable to publicly bankrupt as Mia buys out everything he overleveraged.",
			"relationships": "Mia's soon-to-be ex-husband; business rival of Julian."
		},
		{
			"name": "Julian Reyes",
			"role": "love-interest",
			"bio": "A principled custom-furniture craftsman who helps Mia furnish her first new home and slowly earns her trust.",
			"arc": "From guarded loner to Mia's partner in both life and her design empire.",
			"relationships": "Mia's collaborator and eventual love interest; Derek's professional rival."
		}
	]}`)

	m.Register("episodes", episodesFixture())

	m.Register("placements", `{"placements": [
		{"episode": 1, "scene": "Mia is thrown out and spends the night on a bare floor", "productSku": "ASH-SOFA-001", "category": "sofa", "emotionalBeat": "rock-bottom contrast against the empty room", "ctaTiming": "soft lower-third at 0:75"},
		{"episode": 2, "scene": "First morning in her tiny new apartment", "productSku": "ASH-BED-001", "category": "bed", "emotionalBeat": "fresh-start hope", "ctaTiming": "end-card CTA"},
		{"episode": 4, "scene": "Mia hosts a tense dinner to confront Derek's new partner", "productSku": "ASH-DINE-001", "category": "dining", "emotionalBeat": "confrontation power-play", "ctaTiming": "mid-roll product pin at 0:45"},
		{"episode": 6, "scene": "Late-night reconciliation talk with Julian", "productSku": "ASH-RECL-001", "category": "recliner", "emotionalBeat": "warmth and trust building", "ctaTiming": "soft lower-third"},
		{"episode": 8, "scene": "Mia launches her design studio from home", "productSku": "ASH-DESK-001", "category": "office", "emotionalBeat": "underdog-builds-empire momentum", "ctaTiming": "end-card CTA"},
		{"episode": 12, "scene": "Dream-home reveal party as Mia takes over the neighborhood", "productSku": "ASH-OUT-001", "category": "outdoor", "emotionalBeat": "triumphant status payoff", "ctaTiming": "hero shoppable end-card 0:80"}
	]}`)

	m.Register("hero", `{"heroScenes": [
		{"episode": 12, "title": "The Dream-Home Reveal", "shots": [
			{"number": 1, "shotType": "WS", "action": "Drone-style vertical push-in on the illuminated reveal party at Mia's new estate", "dialogue": ""},
			{"number": 2, "shotType": "MS", "action": "Mia steps onto the patio in front of the Clare View Outdoor Set as guests gasp", "dialogue": "Welcome to the home you said I'd never have."},
			{"number": 3, "shotType": "CU", "action": "Derek's face falls as he recognizes the deed in Mia's hand", "dialogue": "You... bought all of it?"},
			{"number": 4, "shotType": "POV", "action": "Mia's gaze sweeps the fully furnished living room, lingering on the Maeford Sectional", "dialogue": ""},
			{"number": 5, "shotType": "MS", "action": "Julian takes Mia's hand as the camera pulls back to the glowing house", "dialogue": "Now it's really home."}
		]},
		{"episode": 4, "title": "The Confrontation Dinner", "shots": [
			{"number": 1, "shotType": "WS", "action": "The Haddigan Dining Set is set for a tense dinner, candlelight flickering", "dialogue": ""},
			{"number": 2, "shotType": "CU", "action": "Mia sets down a glass with deliberate calm", "dialogue": "Sit. We have a lot to settle."},
			{"number": 3, "shotType": "MS", "action": "Derek's new partner shifts uncomfortably as Mia slides a folder across the table", "dialogue": "Do you know whose money you're spending?"}
		]}
	]}`)

	m.Register("production_distribution", `{
		"production": {
			"format": "9:16 vertical",
			"budgetTier": "mid (single-location heavy, 3-day shoot)",
			"shotCount": 240,
			"castSize": 6,
			"locations": ["Tiny starter apartment", "Modern design studio", "Luxury reveal estate", "Restaurant dining room"],
			"furnitureProps": ["Maeford Sectional (ASH-SOFA-001)", "Realyn Queen Bed (ASH-BED-001)", "Haddigan Dining Set (ASH-DINE-001)", "Boxberg Recliner (ASH-RECL-001)", "Camiburg Home Office Desk (ASH-DESK-001)", "Clare View Outdoor Set (ASH-OUT-001)"]
		},
		"distribution": {
			"ctaCopy": "She built her dream home from nothing. Shop the look at Ashley.",
			"linkPlacement": "Shoppable end-card + pinned comment with SKU links",
			"hashtags": ["#shortdrama", "#revengeglowup", "#dreamhome", "#AshleyFurniture", "#makeover", "#homemakeover"]
		}
	}`)

	return m
}

// episodesFixture builds a realistic 12-episode season as a JSON string, each
// episode with a non-empty hook, cliffhanger and payoff so the pacing gate
// passes in keyless demo mode.
func episodesFixture() string {
	type ep struct {
		Number      int      `json:"number"`
		Title       string   `json:"title"`
		Synopsis    string   `json:"synopsis"`
		Beats       []string `json:"beats"`
		Hook        string   `json:"hook"`
		Cliffhanger string   `json:"cliffhanger"`
		Payoff      string   `json:"payoff"`
	}
	titles := []string{
		"Thrown Out", "One Suitcase", "The Inheritance Letter", "The Confrontation Dinner",
		"Buying Back the Block", "The Craftsman", "Studio Launch", "Empire Rising",
		"Derek's Margin Call", "The Reveal Invitations", "Exposed", "The House She Built",
	}
	hooks := []string{
		"A wedding ring hits the marble floor as the door slams in her face.",
		"She counts her last $40 on a bare apartment floor at dawn.",
		"A lawyer calls: 'Your grandfather left you everything.'",
		"She texts Derek one line: 'Dinner. Tonight. Bring her.'",
		"A 'SOLD' sign goes up on the house Derek wanted most.",
		"Sparks fly — literally — in Julian's furniture workshop.",
		"Her first client walks in and it's Derek's new partner.",
		"A magazine names her 'Designer of the Year' overnight.",
		"Derek's phone explodes: every loan called in at once.",
		"Gold-embossed invitations land on every enemy's doorstep.",
		"A hidden folder reveals exactly who betrayed her.",
		"Floodlights snap on over an estate no one knew she owned.",
	}
	cliffs := []string{
		"As she leaves, a black car pulls up — for her.",
		"The lawyer's business card bears her grandfather's name.",
		"She signs the papers — then learns Derek owes her firm millions.",
		"Derek's partner faints when she sees the deed Mia holds.",
		"Julian recognizes Mia from a photo he was never supposed to see.",
		"Mia finds her sketches stolen and sold under Derek's brand.",
		"Her studio's biggest backer turns out to be Derek's bank.",
		"A rival offers to buy her out — or bury her.",
		"Derek shows up at her door, desperate and dangerous.",
		"One name on the guest list shouldn't be alive.",
		"The traitor is someone she still loved.",
		"She hands Julian a second deed — to a home with both their names.",
	}
	payoffs := []string{
		"The 'nobody' is revealed to be the city's newest billionaire.",
		"She turns a bare room into a stunning Ashley-furnished sanctuary overnight.",
		"She secures the inheritance and the leverage to ruin Derek.",
		"She publicly humiliates Derek at his own table.",
		"She out-bids Derek for the trophy property he bragged about.",
		"She and Julian design a viral, sold-out collection.",
		"She lands the contract Derek spent years chasing.",
		"Her brand eclipses Derek's firm in a single quarter.",
		"Derek's empire collapses as Mia buys the debt.",
		"Every person who wronged her watches her ascend.",
		"She exposes the betrayal on a live broadcast.",
		"She walks into a fully furnished dream home that is finally, truly hers.",
	}
	eps := make([]ep, 12)
	for i := 0; i < 12; i++ {
		eps[i] = ep{
			Number:      i + 1,
			Title:       titles[i],
			Synopsis:    "Episode " + titles[i] + ": Mia advances her comeback while a new Ashley-furnished space anchors the emotional turn.",
			Beats:       []string{"setup", "escalation", "reversal"},
			Hook:        hooks[i],
			Cliffhanger: cliffs[i],
			Payoff:      payoffs[i],
		}
	}
	wrap := map[string]any{"episodes": eps}
	b, _ := json.Marshal(wrap)
	return string(b)
}
