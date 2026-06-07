// backend/internal/agent/chat_engine.go
package agent

import (
	"context"

	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
)

// maxChatSteps bounds the ReAct loop so a misbehaving model can't spin forever.
const maxChatSteps = 24

// maxObsChars caps the size of a tool observation fed back into the LLM history,
// a defense-in-depth guard against context-token blowups (e.g. a tool ever
// returning large/base64 data). Tools already return compact model-facing data;
// this bounds anything that slips through.
const maxObsChars = 8000

// capJSON truncates an observation JSON string to at most n chars, appending a
// marker so the model knows it was clipped (the string need not stay valid JSON).
func capJSON(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + `…"(truncated)"`
}

// affectsStageFor maps a tool name to the plan block it touches, for canvas
// linkage on tool.result. Deterministic tools that don't mutate a block return "".
func affectsStageFor(name string) string {
	switch name {
	case "generateConcept":
		return "concept"
	case "writeBible":
		return "bible"
	case "writeCharacters":
		return "characters"
	case "generateEpisodes":
		return "episodes"
	case "planPlacements":
		return "placements"
	case "designHeroScenes":
		return "hero"
	case "planProductionDistribution":
		return "production_distribution"
	case "renderVisuals":
		return "visuals"
	default:
		return "" // tropes/catalog/pacing/refineBlock(dynamic) → no fixed block
	}
}

// RunChat drives one user turn through the guided-ReAct loop: it appends the user
// message to history, then repeatedly asks the ChatLLM for the next Turn,
// executing any tool calls against the shared plan and feeding observations back,
// until the model returns a final message (or maxChatSteps is hit). Every step
// is surfaced via emit as ChatEvents. Returns the updated history.
func RunChat(
	ctx context.Context,
	chatLLM ChatLLM,
	plan *model.Plan,
	provider llm.Provider,
	reg *Registry,
	history []Message,
	userMsg string,
	emit func(model.ChatEvent),
) ([]Message, error) {
	history = append(history, Message{Role: RoleUser, Text: userMsg})
	system := BuildSystemPrompt(reg)
	defs := reg.Defs()

	for step := 0; step < maxChatSteps; step++ {
		turn, err := chatLLM.NextTurn(ctx, system, history, defs)
		if err != nil {
			emit(model.ChatEvent{Type: model.ChatErrorEvent, Message: err.Error()})
			return history, err
		}

		if turn.Thought != "" {
			emit(model.ChatEvent{Type: model.ChatThoughtDelta, Text: turn.Thought})
			emit(model.ChatEvent{Type: model.ChatThoughtDone, Text: turn.Thought})
		}

		// No tool calls → this is the final assistant message; end the turn.
		if len(turn.ToolCalls) == 0 {
			emit(model.ChatEvent{Type: model.ChatMessageDelta, Text: turn.Message})
			emit(model.ChatEvent{Type: model.ChatMessageDone, Text: turn.Message})
			history = append(history, Message{Role: RoleAssistant, Text: turn.Message})
			emit(model.ChatEvent{Type: model.ChatTurnDone, Plan: plan})
			return history, nil
		}

		// Record the assistant's intent, then execute each tool call. Store the
		// ToolCalls so the Gemini adapter can reconstruct functionCall parts in the
		// multi-turn history (Gemini requires the model content preceding each
		// functionResponse to contain the matching functionCall parts).
		history = append(history, Message{Role: RoleAssistant, Text: turn.Message, ToolCalls: turn.ToolCalls})
		for _, call := range turn.ToolCalls {
			tool, ok := reg.Get(call.Name)
			emit(model.ChatEvent{
				Type: model.ChatToolStart, ToolID: call.ID, ToolName: call.Name,
				FriendlyName: friendlyName(reg, call.Name), Input: call.Args,
			})
			var obs Observation
			if !ok {
				obs = Observation{OK: false, Error: "未知工具:" + call.Name}
			} else {
				obs = tool.Run(ctx, &ToolCtx{Plan: plan, Provider: provider, Emit: emit}, call.Args)
			}
			status := "ok"
			if !obs.OK {
				status = "fail"
			}
			emit(model.ChatEvent{
				Type: model.ChatToolResult, ToolID: call.ID, ToolName: call.Name,
				Status: status, Output: obs, AffectsStage: affectsStageFor(call.Name),
			})
			history = append(history, Message{
				Role: RoleTool, ToolCallID: call.ID, ToolName: call.Name, ToolResult: capJSON(mustJSON(obs), maxObsChars),
			})
		}
	}

	// Hit the step ceiling: close the turn gracefully. Emit delta before done so
	// the front end always sees the message.delta→message.done pair, same as the
	// normal final-message path above.
	ceiling := "本轮步骤已达上限,请补充指令后我再继续。"
	emit(model.ChatEvent{Type: model.ChatMessageDelta, Text: ceiling})
	emit(model.ChatEvent{Type: model.ChatMessageDone, Text: ceiling})
	emit(model.ChatEvent{Type: model.ChatTurnDone, Plan: plan})
	return history, nil
}

func friendlyName(reg *Registry, name string) string {
	if t, ok := reg.Get(name); ok {
		return t.Def().FriendlyName
	}
	return name
}
