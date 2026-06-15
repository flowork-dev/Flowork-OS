// selfevolve.go — R7 SELF-EVOLUTION fase-1 (sisi main: proposer LLM). Owner-approved
// 2026-06-15 (FASE 2 autonomi). Wire routerChat ke agentmgr.EvolveReflectHandler:
// kasih self-map semantik (R6) → LLM usulin perbaikan ADDITIVE & AMAN. FASE-1 = usulan
// doang (nol ubah kode). Prompt nge-LARANG delete / sentuh file LOCKED (pelajaran zombie:
// jangan asal, verifikasi dulu). Eksekusi auto-commit = fase-2 di-gate karma.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"flowork-gui/internal/agentmgr"
)

func evolveProposer() agentmgr.EvolveProposer {
	return func(ctx context.Context, selfMapContext, focus string) ([]agentmgr.ProposalDraft, error) {
		model := coderModel("")
		foc := strings.TrimSpace(focus)
		if foc == "" {
			foc = "perbaikan yang naikin autonomi, ketahanan (resilience), atau ngisi celah kemampuan"
		}
		sys := "You are Flowork's self-evolution architect. You receive a SEMANTIC SELF-MAP of the codebase " +
			"(lines: 'path [domain/role]: summary'). Propose 3-5 CONCRETE, SAFE, ADDITIVE improvements. " +
			`Reply ONLY a JSON array: [{"target_file":"path (or NEW:path for new file)","kind":"add-agent|add-skill|add-app|fix|refactor|doc|test","rationale":"1-2 sentences: what + why","risk":"low|medium|high"}]. ` +
			"Prefer ADDITIVE (new agent/skill/app/test/docs). NEVER propose deleting files or editing files marked LOCKED. No prose, JSON array only."
		user := "FOCUS: " + foc + "\n\nSELF-MAP (semantik):\n" + selfMapContext
		res, e := routerChat(ctx, model, []map[string]any{
			{"role": "system", "content": sys},
			{"role": "user", "content": user},
		}, nil, 1400)
		if e != nil {
			return nil, e
		}
		var arr []agentmgr.ProposalDraft
		if jerr := json.Unmarshal([]byte(jsonArraySlice(res.Content)), &arr); jerr != nil {
			return nil, fmt.Errorf("bad json from model: %s", trimStr(res.Content, 100))
		}
		for i := range arr {
			arr[i].Model = model
		}
		return arr, nil
	}
}

// jsonArraySlice — ambil [...] pertama..terakhir dari output LLM (buang fence/prosa).
func jsonArraySlice(s string) string {
	i := strings.Index(s, "[")
	j := strings.LastIndex(s, "]")
	if i >= 0 && j > i {
		return s[i : j+1]
	}
	return s
}
