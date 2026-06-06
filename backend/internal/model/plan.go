package model

// Image is a multimodal reference image (preset material or user upload) fed to
// Gemini alongside the text prompt. Data is raw base64 (no `data:` URI prefix).
type Image struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
	Label    string `json:"label,omitempty"`
}

// Brief is the normalized user input that drives the whole pipeline.
//
// Requirement is the single free-text "生成需求": it fuses what used to be two
// separate fields (题材/套路 + Ashley 品牌植入重点) into one long prompt the user
// writes (often via the front-end prompt assistant). It carries the genre, the
// 爽点 direction, character/conflict hints and the Ashley furniture placement
// focus all at once.
type Brief struct {
	Requirement string  `json:"requirement"` // 一段式生成需求(题材/爽点/Ashley 植入等)
	Episodes    int     `json:"episodes"`    // default 5
	EpisodeSecs int     `json:"episodeSecs"` // default 30
	Market      string  `json:"market"`      // 默认 "中国"
	Language    string  `json:"language"`    // 默认 "中文"
	Extra       string  `json:"extra"`       // free-form notes
	Images      []Image `json:"images,omitempty"` // optional multimodal reference images
}

func (b *Brief) ApplyDefaults() {
	if b.Episodes <= 0 {
		b.Episodes = 5
	}
	if b.EpisodeSecs <= 0 {
		b.EpisodeSecs = 30
	}
	if b.Market == "" {
		b.Market = "中国"
	}
	if b.Language == "" {
		b.Language = "中文"
	}
}

type Concept struct {
	Logline      string   `json:"logline"`
	Theme        string   `json:"theme"`
	Audience     string   `json:"audience"`
	Tone         string   `json:"tone"`
	PayoffEngine string   `json:"payoffEngine"` // the core "爽点引擎"
	CoreConflict string   `json:"coreConflict"`
	TropesUsed   []string `json:"tropesUsed"`
}

type SeriesBible struct {
	Title             string   `json:"title"`
	GenreTags         []string `json:"genreTags"`
	Episodes          int      `json:"episodes"`
	EpisodeSecs       int      `json:"episodeSecs"`
	TotalRuntimeMin   int      `json:"totalRuntimeMin"`
	Platform          string   `json:"platform"`
	IntegrationThesis string   `json:"integrationThesis"`
}

type Character struct {
	Name          string `json:"name"`
	Role          string `json:"role"` // protagonist / antagonist / love-interest / ...
	Bio           string `json:"bio"`
	Arc           string `json:"arc"`
	Relationships string `json:"relationships"`
}

type Episode struct {
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Synopsis    string   `json:"synopsis"`
	Beats       []string `json:"beats"`
	Hook        string   `json:"hook"`        // golden-3-seconds opener
	Cliffhanger string   `json:"cliffhanger"` // ending hook
	Payoff      string   `json:"payoff"`      // 爽点/反转
}

type Placement struct {
	Episode       int    `json:"episode"`
	Scene         string `json:"scene"`
	ProductSKU    string `json:"productSku"`
	Category      string `json:"category"`
	EmotionalBeat string `json:"emotionalBeat"`
	CTATiming     string `json:"ctaTiming"`
}

type Shot struct {
	Number   int    `json:"number"`
	ShotType string `json:"shotType"` // CU / MS / WS / POV ...
	Action   string `json:"action"`
	Dialogue string `json:"dialogue"`
}

type HeroScene struct {
	Episode int    `json:"episode"`
	Title   string `json:"title"`
	Shots   []Shot `json:"shots"`
}

type Production struct {
	Format         string   `json:"format"` // "9:16 vertical"
	BudgetTier     string   `json:"budgetTier"`
	ShotCount      int      `json:"shotCount"`
	CastSize       int      `json:"castSize"`
	Locations      []string `json:"locations"`
	FurnitureProps []string `json:"furnitureProps"`
}

type Distribution struct {
	CTACopy       string   `json:"ctaCopy"`
	LinkPlacement string   `json:"linkPlacement"`
	Hashtags      []string `json:"hashtags"`
}

// Visual is an AI-generated concept image (poster / storyboard key frame). Data
// is raw base64-encoded PNG (no `data:` URI prefix). Visuals are produced by the
// optional VisualStage via an ImageProvider (Vertex Imagen); when image
// generation is unavailable the slice stays empty and the text plan is complete.
type Visual struct {
	Label    string `json:"label"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // raw base64 PNG
}

// Plan is the complete structured production plan (the core deliverable).
type Plan struct {
	Brief        Brief        `json:"brief"`
	Concept      Concept      `json:"concept"`
	Bible        SeriesBible  `json:"bible"`
	Characters   []Character  `json:"characters"`
	Episodes     []Episode    `json:"episodes"`
	Placements   []Placement  `json:"placements"`
	HeroScenes   []HeroScene  `json:"heroScenes"`
	Production    Production   `json:"production"`
	Distribution Distribution `json:"distribution"`
	Visuals      []Visual     `json:"visuals,omitempty"`
}
