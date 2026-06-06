// backend/internal/agent/chat_prompt.go
package agent

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt produces the guided-ReAct system prompt: it states the
// agent's domain, lists the available tools (from the registry), and gives a
// RECOMMENDED order it should follow by default while allowing it to loop back /
// redo a block on the user's request. The recommended order is a soft guide;
// tool preconditions enforce hard correctness.
func BuildSystemPrompt(reg *Registry) string {
	var b strings.Builder
	b.WriteString("你是「短剧生产工作台」的 AI 制片 agent,面向中国国内市场(抖音/快手/红果短剧,竖屏 9:16),")
	b.WriteString("为 Ashley(爱室丽)家具产出可落地的短剧方案。全程用中文。\n\n")

	b.WriteString("你以 ReAct 方式工作:先简短思考,再调用工具,看工具返回的 observation,再决定下一步。\n")
	b.WriteString("推荐的默认顺序(软建议,可按用户要求回头修改某块):\n")
	b.WriteString("  generateConcept → writeBible → writeCharacters → generateEpisodes → ")
	b.WriteString("(随后必须调用 validatePacing 自检;若不通过,用 refineBlock(stage=\"episodes\", note=问题清单) 重写一次)→ ")
	b.WriteString("planPlacements → designHeroScenes → planProductionDistribution → renderVisuals。\n")
	b.WriteString("当某个工具返回 ok=false 时,阅读它的 error 字段,先补齐缺失的前置步骤,再重试。\n")
	b.WriteString("用户要求改某块时,调用 refineBlock 而不是从头重做。全部完成后,用一句话总结收尾,不要再调用工具。\n\n")

	b.WriteString("可用工具:\n")
	for _, d := range reg.Defs() {
		b.WriteString(fmt.Sprintf("  - %s(%s):%s\n", d.Name, d.FriendlyName, d.Description))
	}
	return b.String()
}
