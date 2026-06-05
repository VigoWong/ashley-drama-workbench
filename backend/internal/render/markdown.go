package render

import (
	"fmt"
	"strings"

	"github.com/ashley/drama-workbench/internal/model"
)

func Markdown(p *model.Plan) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# %s\n\n", nz(p.Bible.Title, "未命名剧集"))
	w("## 立意\n- **一句话梗概：** %s\n- **主题：** %s\n- **目标观众：** %s\n- **基调：** %s\n- **爽点引擎：** %s\n- **核心冲突：** %s\n\n",
		p.Concept.Logline, p.Concept.Theme, p.Concept.Audience, p.Concept.Tone, p.Concept.PayoffEngine, p.Concept.CoreConflict)
	w("## 剧集圣经\n- 集数：%d 集 × %d 秒 | 平台：%s\n- 植入主线：%s\n\n",
		p.Bible.Episodes, p.Bible.EpisodeSecs, p.Bible.Platform, p.Bible.IntegrationThesis)
	w("## 人物\n")
	for _, c := range p.Characters {
		w("- **%s**（%s）：%s _弧线：_ %s\n", c.Name, c.Role, c.Bio, c.Arc)
	}
	w("\n## 分集\n")
	for _, e := range p.Episodes {
		w("### 第 %d 集 — %s\n%s\n- **钩子：** %s\n- **悬念：** %s\n- **爽点：** %s\n\n",
			e.Number, e.Title, e.Synopsis, e.Hook, e.Cliffhanger, e.Payoff)
	}
	w("## 品牌植入\n")
	for _, pl := range p.Placements {
		w("- 第 %d 集 — %s（%s）：%s | CTA：%s\n", pl.Episode, pl.Category, pl.ProductSKU, pl.EmotionalBeat, pl.CTATiming)
	}
	w("\n## 英雄场景\n")
	for _, h := range p.HeroScenes {
		w("### 第 %d 集 — %s\n", h.Episode, h.Title)
		for _, s := range h.Shots {
			w("%d. [%s] %s — “%s”\n", s.Number, s.ShotType, s.Action, s.Dialogue)
		}
	}
	w("\n## 制作\n- 画幅：%s | 预算档位：%s | 镜头数：%d | 演员数：%d\n- 家具道具：%s\n\n",
		p.Production.Format, p.Production.BudgetTier, p.Production.ShotCount, p.Production.CastSize, strings.Join(p.Production.FurnitureProps, "、"))
	w("## 分发\n- CTA：%s\n- 话题标签：%s\n", p.Distribution.CTACopy, strings.Join(p.Distribution.Hashtags, " "))
	return b.String()
}

func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
