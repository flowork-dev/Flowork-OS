// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval (autonomy grant 2026-06-19).
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-20
// Reason: CGM digestion wiring (LLM+Embed closures → DigestPendingInteractions, cron
//   hook + manual endpoint, env-gated) — built+tested, deployed P1. Extend = new file.
//
// cognitive_digest_cron.go — CGM digestion wiring (NON-beku, file baru).
//
// Deploy loop yg udah TERBUKTI di P1 (prove-loop harness, 2026-06-20): nyolokin
// closure LLM (flowork-brain) + Embed (bge-m3) lewat router ke
// agentdb.DigestPendingInteractions (locked) → percakapan jadi cognitive graph.
//
// Layering bersih (roadmap §4.6): agentdb GAK import routerclient — closure
// di-inject dari sini. File ini = satu-satunya tempat host nyambungin CGM digest
// ke router.
//
// Dua jalur:
//   - DigestAllAgents: dipanggil cron dream 12h (Tier-2 deep). GATE di env
//     FLOWORK_CGM_AUTODIGEST=1 (default OFF — owner opt-in, hormati sistem live).
//   - CognitiveDigestHandler: POST manual trigger (selalu jalan — kontrol owner).
//
// P1 findings yg diterapin di sini:
//   - max_tokens 4096: extraction output panjang → 1024 = truncation = parse gagal.
//   - Tier-2 + PromoteShadows: gate kualitas + promosi repetisi.
//   - resilient: error per-agent di-skip, gak nge-block cron/handler lain.

package agentmgr

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/httpx"
	"flowork-gui/internal/routerclient"
)

// cgmDigestMaxTokens — generous biar extraction JSON gak kepotong (P1 finding).
const cgmDigestMaxTokens = 4096

// cgmRouterClient bikin client ke router (URL dari env, fallback default :2402,
// host-whitelisted di routerclient.New).
func cgmRouterClient() *routerclient.Client {
	return routerclient.New(strings.TrimSpace(os.Getenv("ROUTER_DEFAULT_URL")))
}

// cgmModel — model reasoning buat extraction (env, fallback "" = DefaultChatModel).
func cgmModel() string {
	return strings.TrimSpace(os.Getenv("FLOWORK_LLM_MODEL"))
}

// buildDigestDeps rakit DigestDeps dengan closure LLM+Embed asli ke router.
func buildDigestDeps(scope string, tier int) agentdb.DigestDeps {
	rc := cgmRouterClient()
	model := cgmModel()
	return agentdb.DigestDeps{
		AgentScope: scope,
		Tier:       tier,
		LLM: func(ctx context.Context, prompt string) (string, error) {
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

// DigestAgent cerna interactions pending 1 agent → graph (Tier-2 = gate+promote).
// Return stats + jumlah interaction yg dicerna. Idempoten (cognitive_digest_log).
func DigestAgent(agentID string, tier int) (agentdb.DigestStats, int, error) {
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

	stats, n, derr := store.DigestPendingInteractions(ctx, buildDigestDeps(scope, tier), 100)
	if derr != nil {
		return stats, n, derr
	}
	if tier >= 2 {
		// promote shadow→active yg udah dikuatin repetisi (D13/D16)
		_, _ = store.PromoteShadows(2)
	}
	return stats, n, nil
}

// DigestAllAgents — dipanggil cron dream (Tier-2). GATE di FLOWORK_CGM_AUTODIGEST=1.
// Resilient: error/timeout per-agent di-log & di-skip, gak nge-block yg lain.
// Return total interaction yg dicerna lintas agent.
func DigestAllAgents(agentIDs []string) int {
	if strings.TrimSpace(os.Getenv("FLOWORK_CGM_AUTODIGEST")) != "1" {
		return 0 // opt-in: default OFF (owner kontrol, hormati sistem live)
	}
	total := 0
	for _, id := range agentIDs {
		func() {
			defer func() { _ = recover() }() // 1 agent rusak gak nyeret yg lain (isolasi)
			if _, n, err := DigestAgent(id, 2); err == nil {
				total += n
			}
		}()
	}
	return total
}

// CognitiveDigestHandler — POST /api/agents/cognitive/digest?id=<agent>&tier=2
// Manual trigger (kontrol owner; selalu jalan, gak digate env). Buat QC + GUI button.
func CognitiveDigestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (POST)"})
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("id"))
	if agentID == "" {
		httpx.WriteJSON(w, map[string]any{"error": "agent id required"})
		return
	}
	tier := parseLimitOr(r.URL.Query().Get("tier"), 2)
	stats, n, err := DigestAgent(agentID, tier)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error(), "digested": n})
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"ok": true, "agent": agentID, "tier": tier, "digested": n,
		"nodes_added": stats.NodesAdded, "edges_added": stats.EdgesAdded,
		"quarantined": stats.Quarantined, "tensions": stats.Tensions, "dropped": stats.Dropped,
	})
}
