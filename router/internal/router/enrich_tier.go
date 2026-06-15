// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without explicit owner (Mr.Dev) approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-04
//
// enrich_tier.go — hemat kuota: enrichment BERAT cuma buat tier komandan.
//
// MASALAH: constitution (20 sacred rules) + brain knowledge disuntik ke SETIAP
// request → ribuan token ekstra per-call. Crew (worker/synth, volume gede: ~5
// call/task) tugasnya FOKUS ("riset saham X" / "gabungin analisa") — doktrin
// proyek + knowledge Flowork GA RELEVAN, cuma bakar kuota → 429.
//
// SOLUSI: skip constitution + brain buat tier "crew" (cheap = haiku). Komandan
// (sonnet) tetep full. ANTIBODY (anti-halu) TETEP buat semua (kecil + penting).
//
// Tunable: env FLOW_ROUTER_LIGHT_MODELS (koma-separated substring). Default "haiku".
//
// ⚠️ BY DESIGN — JANGAN "FIX" JADI ENRICH SEMUA (audit 2026-06-15, owner-confirmed
// "jangan buka aksesnya"). Crew/haiku SENGAJA di-skip enrich berat — INI BUKAN BUG.
// Buka enrich ke semua = balik ke 429 (bakar kuota). Sesuai arsitektur baru (tiering
// hemat kuota). Kalau ada yg ngira ini kabel putus → STOP, ini keputusan sadar.

package router

import (
	"os"
	"strings"
)

// isCrewLightModel — true kalau model = tier crew/worker (cheap) yang di-skip
// enrichment berat. Default: model ngandung "haiku".
func isCrewLightModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("FLOW_ROUTER_LIGHT_MODELS")); v != "" {
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(strings.ToLower(s)); s != "" && strings.Contains(model, s) {
				return true
			}
		}
		return false
	}
	return strings.Contains(model, "haiku")
}
