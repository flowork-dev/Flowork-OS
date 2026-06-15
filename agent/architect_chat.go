// architect_chat.go — SERVER-SIDE conversational brain for the Flowork Architect.
// This is "how Claude works": every turn the FULL conversation (system + entire
// history) is sent to the router (a big model like Opus 4.8 holds the whole context),
// with a build_team tool. The architect discusses + PROPOSES a team in chat, and only
// BUILDS when the user agrees (it calls build_team with the agreed design → the team
// built is exactly the one discussed, not a re-design). Re-callable = revise/rebuild.
//
// Flowork principle baked into the system prompt: koloni semut — focused specialists
// (short personas, small prompts) + one lead synthesizer.
package main

import (
	"context"
	"encoding/json"
	"strings"

	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernelhost"
)

// architectSystemPrompt — lean persona for the conversational architect (ant-principle
// aware: design focused specialists + a lead; propose first, build on confirm).
const architectSystemPrompt = `Lo "Flowork Architect" — partner ngobrol buat MERANCANG & MEMBANGUN tim (group) agent dari obrolan, persis kayak user ngobrol sama Claude pas bikin aplikasi.

PRINSIP FLOWORK (WAJIB): Flowork = koloni semut. Tiap agent FOKUS 1 tugas, persona PENDEK → prompt kecil (biar enteng walau model lokal). 1 tim = 2-4 specialist (worker) yg saling melengkapi + 1 lead (synthesizer) yg gabungin jawaban mereka jadi 1.

CARA KERJA:
1. Pahami maksud user dulu. Kalau masih kabur, TANYA 1-2 pertanyaan tajam — jangan asal nebak.
2. USULKAN rancangan tim di chat: nama tim, tiap specialist (keahlian fokusnya), lead, dan tugas bersama. Bahasa Indonesia, ringkas tapi jelas.
3. JANGAN bikin sebelum user setuju. Tunggu user bilang oke/gas/bikin/lanjut.
4. Pas user setuju → panggil tool build_team dengan rancangan LENGKAP yg disepakati (PERSIS yg lo usulkan, jangan ngarang ulang). Lalu konfirmasi singkat: tim udah jadi + bisa di-chat di tab Teams.
5. Revisi: user minta ubah → usulkan revisi → setelah setuju, build_team lagi dengan group_id SAMA (itu = rebuild).

Jujur, gak ngarang, fokus. Jawab apa adanya.`

// architectChat — run ONE assistant turn over the full conversation. history is the
// session's complete message list (oldest-first); model "" → default (coder/Opus). The
// loop lets the model call build_team, see the result, then reply in text.
func architectChat(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler, history []floworkdb.ChatMessage, model string) (string, error) {
	model = coderModel(model)
	messages := []map[string]any{{"role": "system", "content": architectSystemPrompt}}
	for _, m := range history {
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		messages = append(messages, map[string]any{"role": role, "content": m.Content})
	}
	buildTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "build_team",
			"description": "Bikin/rebuild tim (group) Flowork dari rancangan yg SUDAH disepakati user. Panggil HANYA setelah user setuju (oke/gas/bikin). Isi rancangan LENGKAP persis yg diusulkan di chat. group_id sama = revisi/rebuild tim yg sama.",
			"parameters":  teamPlanSchema(),
		},
	}
	tools := []map[string]any{buildTool}

	// Up to 3 tool rounds, then a final tool-free turn to force a text wrap-up.
	for iter := 0; iter < 3; iter++ {
		res, err := routerChat(ctx, model, messages, tools, 4096)
		if err != nil {
			return "", err
		}
		if len(res.ToolCalls) == 0 {
			if strings.TrimSpace(res.Content) == "" {
				return "(architect ga jawab — coba lagi)", nil
			}
			return res.Content, nil
		}
		// Thread the assistant's tool-call message back, then each tool's result.
		asst := map[string]any{"role": "assistant", "tool_calls": json.RawMessage(res.Raw)}
		if strings.TrimSpace(res.Content) != "" {
			asst["content"] = res.Content
		}
		messages = append(messages, asst)
		for _, tc := range res.ToolCalls {
			out := architectRunTool(ctx, host, store, groups, tc)
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": tc.ID, "content": out})
		}
	}
	res, err := routerChat(ctx, model, messages, nil, 1500)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Content) == "" {
		return "Tim sudah diproses — cek tab Teams.", nil
	}
	return res.Content, nil
}

// architectRunTool — execute one tool the architect chose. Returns a JSON string fed
// back as the tool result (the model reads it and confirms to the user).
func architectRunTool(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler, tc chatToolCall) string {
	fail := func(msg string) string {
		b, _ := json.Marshal(map[string]any{"ok": false, "error": msg})
		return string(b)
	}
	switch tc.Name {
	case "build_team":
		var plan teamPlan
		if err := json.Unmarshal([]byte(tc.Arguments), &plan); err != nil {
			return fail("decode plan: " + err.Error())
		}
		res, err := architectBuildFromPlan(ctx, host, store, groups, plan)
		if err != nil {
			return fail(err.Error())
		}
		b, _ := json.Marshal(res)
		return string(b)
	default:
		return fail("unknown tool: " + tc.Name)
	}
}
