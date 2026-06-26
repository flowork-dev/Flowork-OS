// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-22 (compact model-picker).
// LOCKED ≠ FREEZE. Cabut-akar: reuse pipeline digest yang SAMA (cognitive_digest_cron.go),
// cuma swap model di DigestDeps. Jangan ubah tanpa izin owner + re-lock + changelog.
package agentmgr

// digest_model.go — MODEL-PICKER buat compact (owner 2026-06-22).
//
// Owner KERAS 2 hal:
//   1. "di menu auto-compact ada filtur ganti model; kalau di-set, SEMUA compact (cron /
//      compact-all / per-agent) pake model itu buat digest."
//   2. "defaultnya LOKAL — kalau ngak di-isi maka akan pake lokal."
//
// ALASAN default lokal: tujuan akhir Flowork = freeze + standalone. Kalau subscribe owner
// habis (ga mampu beli token cloud), compact HARUS tetep jalan → digest pengalaman ke brain
// pake model LOKAL (flowork-brain di :8088, gratis, offline). compact_model di-isi = pilihan
// owner pake model cloud kapabel SELAGI masih ada langganan. Kosong = lokal, selalu jalan.
//
// Bedanya sama jalur digest non-compact (cron dream, DigestAgent): itu tetep pake
// DigestLLMOverride (agent dream-digester) → ga disentuh (no regression). Yang dialihkan ke
// lokal-by-default cuma jalur COMPACT (lewat DigestAgentModel).

import (
	"context"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
)

// localDigestModel — nama model lokal kanonik (router routing ke llama-server :8088 saat model
// name = ini). Sinkron dengan router/internal/localai.FloworkBrainModel. Di-hardcode di sini
// karena agent module (flowork-gui) ga boleh import internal package router (cross-module).
const localDigestModel = "flowork-brain"

// buildDigestDepsModel — DigestDeps buat reasoning extraction compact.
// Owner 2026-06-26: compact pakai AGENT DEDIKASI (dream-digester, model GUI per-agent) by-default —
// konsisten sama enrich (codemap-enricher) + hasil nyambung GRAPH. Urutan:
//   1. compact_model KOSONG + DigestLLMOverride ke-set → lewat AGENT dream-digester (model GUI, mis.
//      haiku → CEPAT, gak local-26B lambat). Inilah default baru.
//   2. compact_model di-SET owner → direct ke model itu (override eksplisit).
//   3. override nil (dream-digester ga ada) → fallback LOKAL (flowork-brain) — tetep jalan offline.
// Embed tetep lewat router (bge-m3).
func buildDigestDepsModel(scope string, tier int, model string) agentdb.DigestDeps {
	model = strings.TrimSpace(model)
	useAgent := model == "" && DigestLLMOverride != nil // default → agent dedikasi dream-digester
	if model == "" {
		model = localDigestModel // fallback LOKAL kalau override nil
	}
	rc := cgmRouterClient()
	return agentdb.DigestDeps{
		AgentScope: scope,
		Tier:       tier,
		LLM: func(ctx context.Context, prompt string) (string, error) {
			if useAgent {
				return DigestLLMOverride(ctx, prompt) // dream-digester (model GUI) → graph
			}
			c, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			return rc.ChatComplete(c, model, prompt, cgmDigestMaxTokens)
		},
		Embed: func(ctx context.Context, text string) ([]float32, error) {
			c, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			return rc.EmbedText(c, "", text)
		},
	}
}

// DigestAgentModel — sama persis DigestAgent (cognitive_digest_cron.go) tapi pake model pilihan
// owner (atau LOKAL kalau kosong) buat reasoning extraction. Dipanggil dari AutoCompactAgent
// biar SEMUA jalur compact hormati compact_model + default-lokal.
func DigestAgentModel(agentID string, tier int, model string) (agentdb.DigestStats, int, error) {
	store, err := openAgentStore(agentID)
	if err != nil {
		return agentdb.DigestStats{}, 0, err
	}
	defer store.Close()

	if tier < 1 || tier > 2 {
		tier = 2
	}
	scope := "agent:" + agentID
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	stats, n, derr := store.DigestPendingInteractions(ctx, buildDigestDepsModel(scope, tier, model), 100)
	if derr != nil {
		return stats, n, derr
	}
	if tier >= 2 {
		_, _ = store.PromoteShadows(2)
	}
	return stats, n, nil
}
