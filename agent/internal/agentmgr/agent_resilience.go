// 🔐 LOCKED (stabil 2026-06-23, BUKAN freeze) — edit dgn izin owner. Tuning cepat tanpa edit: env
// FLOWORK_RESILIENCE_OFF=1. Roadmap: docs/ROADMAP_AGENT_RESILIENCE.md ITEM 1.
//
// agent_resilience.go — DOKTRIN RECOVERY buat SEMUA agent (ITEM 1 roadmap agent-resilience, 2026-06-23).
//
// MASALAH: agent mati + muter pas error transient (fbspecial: "router error" → auto-continue #8).
// Prinsip ditiru dari Claude Code (prompts.ts:233/240): diagnosa-sebelum-ganti-taktik, jangan-ulang-buta,
// jangan-nyerah-1×, lapor-jujur, jangan-polling. Ini lapis PROMPT (kesadaran); lapis CONTROL-FLOW (retry-
// backoff) = ITEM 2. Di-inject ke system-prompt SEMUA agent tiap turn lewat SelfPromptRenderHandler
// (cabang non-frozen, pola sama WIBNowHeader). Sengaja RINGKAS (~110 token) — patuh hemat-token owner.
package agentmgr

import (
	"os"
	"strings"
)

// RecoveryDoctrine — blok ringkas "cara hadapi error/macet" buat tiap agent. Bikin agent tahan-banting:
// recover dari kegagalan transient, ga muter, ga nyerah dini, jujur. SWITCH: FLOWORK_RESILIENCE_OFF=1 → "".
func RecoveryDoctrine() string {
	if strings.TrimSpace(os.Getenv("FLOWORK_RESILIENCE_OFF")) == "1" {
		return ""
	}
	return "# CARA HADAPI ERROR / MACET\n" +
		"- Gagal/error → DIAGNOSA dulu (baca pesan error-nya). Jangan ulang aksi sama persis buta, jangan nyerah cuma gara-gara 1× gagal — coba fix terfokus.\n" +
		"- Error TRANSIENT (router/jaringan/timeout/server sibuk) → tunggu sebentar lalu COBA LAGI. JANGAN anggap tugas kelar / jangan nyerah gara-gara blip sesaat.\n" +
		"- Macet beneran SETELAH diinvestigasi → bilang jujur + tawarin opsi. JANGAN muter ngulang langkah yg sama berkali-kali.\n" +
		"- Gagal / belum kelar → LAPOR JUJUR. JANGAN ngaku selesai kalau belum kebukti jalan.\n" +
		"- Nunggu kerjaan background → bakal dikabarin pas kelar. JANGAN polling ketat / loop nunggu.\n\n"
}
