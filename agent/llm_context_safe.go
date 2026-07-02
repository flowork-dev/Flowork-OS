// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-21 (LOCKED ≠ FREEZE).
// Verified: unit 4/4 PASS (deteksi error PERSIS screenshot + compact 100k→<budget) + live chat OK.
package main

// llm_context_safe.go — LAYER context-safety di ATAS routerChat (llm.go = LOCKED, TIDAK disentuh).
// Owner 2026-06-21: router 502 "Context size has been exceeded" pas agent (mr-flow/AI Studio) /
// EVOLUSI muat konteks kegedean → hard-fail. BAHAYA kalau kejadian di TENGAH loop evolusi (edit
// setengah jadi / loop mati). Akar: ga ada pre-flight budget, ga ada recovery context-exceeded.
//
// Plug-and-play (doktrin: papan kosong, isolasi): bungkus call LLM —
//  1. PRE-FLIGHT budget: estimasi token; kalau lewat jendela model → compact DULU.
//  2. DETEKSI context-exceeded dari error router (string-match, robust lintas provider).
//  3. RETRY-after-compact 1x (compact lebih agresif).
//  4. FAIL-SAFE: kalau tetap kegedean → error JELAS (caller bisa berhenti aman), BUKAN crash.
//
// Caller berisiko (evolusi + AI Studio chat) panggil routerChatSafe, bukan routerChat langsung.

import (
	"context"
	"fmt"
	"strings"
)

// ctxBudgetTokens — perkiraan jendela context AMAN per-model (sisain ruang buat output). Konservatif:
// model lokal kecil, cloud besar. Tujuannya cegah 502, bukan presisi tokenizer.
func ctxBudgetTokens(model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "opus"), strings.Contains(m, "sonnet"), strings.Contains(m, "haiku"),
		strings.Contains(m, "claude"), strings.Contains(m, "gemini"):
		return 180000 // Claude ~200k / Gemini ≥1M — konservatif, sisain buffer output
	case strings.Contains(m, "flowork-brain"), strings.Contains(m, "local"), strings.Contains(m, "gemma"),
		strings.Contains(m, "qwen"), strings.Contains(m, "llama"):
		return 7000 // model lokal kecil — konservatif
	default:
		return 24000 // unknown → aman-tengah
	}
}

// estTokens — perkiraan KASAR token dari messages (~4 char/token + overhead per-pesan). Sengaja
// over-estimate dikit (lebih aman ke arah compact daripada jebol).
func estTokens(messages []map[string]any) int {
	total := 0
	for _, m := range messages {
		total += 8 // overhead role/format per pesan
		total += msgContentLen(m)/4 + 1
	}
	return total
}

// msgContentLen — panjang char "content" sebuah pesan (string atau array part).
func msgContentLen(m map[string]any) int {
	switch c := m["content"].(type) {
	case string:
		// Vision block-array (multimodal paste): hitung teks + flat per-gambar, BUKAN
		// panjang base64 mentah — biar ga salah anggep kegedean (chat_vision.go).
		return chatContentEstLen(c)
	case []any:
		n := 0
		for _, p := range c {
			if s, ok := p.(string); ok {
				n += len(s)
			} else if pm, ok := p.(map[string]any); ok {
				if s, ok := pm["text"].(string); ok {
					n += len(s)
				}
			}
		}
		return n
	}
	return 0
}

// isCtxExceeded — error router itu context-exceeded? (match pesan provider apa pun, mis. upstream 500
// "Context size has been exceeded" / "maximum context length" / "prompt is too long").
func isCtxExceeded(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "context size") ||
		strings.Contains(s, "context length") ||
		strings.Contains(s, "context_length") ||
		strings.Contains(s, "maximum context") ||
		strings.Contains(s, "too many tokens") ||
		strings.Contains(s, "prompt is too long") ||
		(strings.Contains(s, "context") && strings.Contains(s, "exceed"))
}

// compactMessages — kecilin context biar muat budget TANPA ngerusak makna inti:
//  1. SELALU simpan system (msg[0]) + user/pesan TERAKHIR (task/kode inti).
//  2. Buang turn TENGAH (history lama) dari yg paling tua sampai muat.
//  3. Kalau msg tunggal masih kegedean → truncate content TERPANJANG + tandai (mis. tool-result raksasa).
//
// Balik (compacted, changed). changed=false kalau udah muat ATAU ga bisa dikecilin lagi.
func compactMessages(messages []map[string]any, budget int) ([]map[string]any, bool) {
	if budget < 1 || len(messages) == 0 || estTokens(messages) <= budget {
		return messages, false
	}
	changed := false

	// 1) Buang turn tengah (index 1..len-2) dari paling tua, sisain system[0] + last.
	if len(messages) > 2 {
		head := messages[0]
		last := messages[len(messages)-1]
		mid := append([]map[string]any{}, messages[1:len(messages)-1]...)
		for len(mid) > 0 {
			trial := append([]map[string]any{head}, append(append([]map[string]any{}, mid...), last)...)
			if estTokens(trial) <= budget {
				return trial, true
			}
			mid = mid[1:] // buang yg paling tua
			changed = true
		}
		messages = []map[string]any{head, last}
		if estTokens(messages) <= budget {
			return messages, true
		}
	}

	// 2) Masih kegedean (system + last ga muat / cuma 1-2 pesan) → truncate content TERPANJANG.
	const marker = "\n\n[…dipotong demi context budget…]"
	for estTokens(messages) > budget {
		bi, blen := -1, 0
		for i := range messages {
			if l := msgContentLen(messages[i]); l > blen {
				bi, blen = i, l
			}
		}
		if bi < 0 || blen < len(marker)+200 {
			break // ga ada lagi yg bisa dipotong (semua udah mini) → fail-safe ke caller
		}
		over := estTokens(messages) - budget
		cut := over*4 + len(marker) + 64 // char ~ token*4
		if cut > blen-100 {
			cut = blen - 100
		}
		if s, ok := messages[bi]["content"].(string); ok && !isVisionBlockContent(s) {
			messages[bi] = cloneMsg(messages[bi])
			messages[bi]["content"] = s[:len(s)-cut] + marker
			changed = true
		} else {
			// content non-string (array) ATAU block vision (JSON/base64) — motong =
			// korup. Jangan rusak, serahkan ke caller (fail-safe).
			break
		}
	}
	return messages, changed
}

// cloneMsg — salinan dangkal map pesan (biar truncate ga ngubah slice caller asli).
func cloneMsg(m map[string]any) map[string]any {
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

// routerChatSafe — DROP-IN pengganti routerChat dgn context-safety. Signature SAMA.
func routerChatSafe(ctx context.Context, model string, messages []map[string]any, tools []map[string]any, maxTokens int) (chatResult, error) {
	budget := ctxBudgetTokens(model)
	msgs, _ := compactMessages(messages, budget)
	res, err := routerChat(ctx, model, msgs, tools, maxTokens)
	if err == nil || !isCtxExceeded(err) {
		return res, err
	}
	// Pre-flight under-estimate / provider window lebih ketat → compact LEBIH agresif + retry 1x.
	msgs2, changed := compactMessages(messages, budget*55/100)
	if changed {
		if res2, err2 := routerChat(ctx, model, msgs2, tools, maxTokens); err2 == nil || !isCtxExceeded(err2) {
			return res2, err2
		}
	}
	// FAIL-SAFE: tetap kegedean → error jelas (caller berhenti AMAN, bukan crash di tengah evolusi).
	return res, fmt.Errorf("context terlalu besar utk model %q walau sudah di-compact — pakai model context-besar (mis. claude-opus-4-8) atau pecah tugas jadi lebih kecil: %w", model, err)
}
