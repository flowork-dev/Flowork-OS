// learning_feed.go — Phase 3E / D13 LOOP BELAJAR (owner-approved 2026-06-21).
//
// "Guru dateng-pergi, brain gak lupa": tangkap pengalaman model KUAT yg lewat router
// (termasuk dari VS Code) → distil → tulis ke graph sbg SHADOW (kandidat, gak ikut
// retrieval) tag source_kind='strong_model_unverified' → dedup otomatis (ResolveByEmbedding:
// mirip → hit_count++; baru → shadow) → PROMOTE shadow→active HANYA kalau dikuatin repetisi
// (PromoteShadows). Reuse PENUH primitif existing (recorder + DigestText via dream-digester +
// gate antibody + dedup). Mode KONSERVATIF (owner pilih A): di luar Flowork gak ada outcome →
// jangan telen mentah; di DALAM (build_pass=1) → fast-track Tier-2.
//
// PRIVASI (D8): distil LOKAL ke graph mr-flow (shadow), TIDAK di-federate/gossip ke mesh.
// Kontrol owner: opt-in FLOWORK_LEARN_LOOP=1 (default OFF). Idempoten (learning_record_log).
package agentmgr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"flowork-gui/internal/httpx"
	"flowork-gui/internal/routerclient"
)

const learningStoreAgent = "mr-flow"          // graph kanonik tempat instinct
const learningScope = "agent:" + learningStoreAgent

// learningRecord — subset Record dari router GET /api/recordings.
type learningRecord struct {
	ID        int64  `json:"id"`
	Model     string `json:"model"`
	Response  string `json:"response"`
	Agent     string `json:"agent"`
	BuildPass int64  `json:"build_pass"`
}

// LearningStats — hasil 1 siklus loop-belajar (buat changelog/QC).
type LearningStats struct {
	Fetched   int `json:"fetched"`
	Processed int `json:"processed"`
	Skipped   int `json:"skipped"`
	Added     int `json:"added"`
	Promoted  int `json:"promoted"`
}

// isStrongModelForLearning — gate model: cuma model KUAT (cloud, bukan lokal flowork-brain)
// yg pengalamannya layak di-distil. Model lokal lemah → skip (anti polusi/halu).
func isStrongModelForLearning(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	return !strings.Contains(m, "flowork-brain") && !strings.Contains(m, "flowork")
}

// fetchRouterRecordings — GET router /api/recordings?include_body=1&limit=N → []learningRecord.
func fetchRouterRecordings(limit int) ([]learningRecord, error) {
	base := routerclient.New(strings.TrimSpace(os.Getenv("ROUTER_DEFAULT_URL"))).BaseURL
	url := base + "/api/recordings?include_body=1&limit=" + itoaSmallLearn(limit)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	var out struct {
		Items []learningRecord `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// DigestRecordings — 1 siklus loop-belajar. Opt-in FLOWORK_LEARN_LOOP=1 (owner kontrol).
func DigestRecordings(limit int) (LearningStats, error) {
	var ls LearningStats
	if strings.TrimSpace(os.Getenv("FLOWORK_LEARN_LOOP")) != "1" {
		return ls, nil // opt-in: default OFF (hormati sistem live + privasi)
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	recs, err := fetchRouterRecordings(limit)
	if err != nil {
		return ls, err
	}
	ls.Fetched = len(recs)
	store, serr := openAgentStore(learningStoreAgent)
	if serr != nil {
		return ls, serr
	}
	defer store.Close()

	deps := buildDigestDeps(learningScope, 1)
	deps.SourceKind = "strong_model_unverified"
	deps.SourceRef = "learning"

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	for _, rc := range recs {
		if store.LearningRecordSeen(rc.ID) {
			continue
		}
		if !isStrongModelForLearning(rc.Model) || strings.TrimSpace(rc.Response) == "" {
			_ = store.MarkLearningRecord(rc.ID)
			ls.Skipped++
			continue
		}
		d := deps
		if rc.BuildPass == 1 {
			d.Tier = 2 // fast-track: outcome nyata (compile/test pass DI DALAM Flowork)
		}
		st, derr := store.DigestText(ctx, rc.Response, d)
		if derr == nil {
			ls.Added += st.NodesAdded
		}
		_ = store.MarkLearningRecord(rc.ID)
		ls.Processed++
	}
	// promote shadow→active yg udah dikuatin repetisi lintas-sesi (D13 mode konservatif).
	if n, _ := store.PromoteShadows(2); n > 0 {
		ls.Promoted = n
	}
	return ls, nil
}

// LearningDigestHandler — POST /api/agents/learning/digest?limit=N → jalanin 1 siklus
// loop-belajar (fetch recordings → distil shadow → promote on repetisi). Manual trigger
// (owner-controlled). Gated FLOWORK_LEARN_LOOP=1.
func LearningDigestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (POST)"})
		return
	}
	limit := 0
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		if n, e := strconv.Atoi(s); e == nil {
			limit = n
		}
	}
	ls, err := DigestRecordings(limit)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"ok":      true,
		"enabled": strings.TrimSpace(os.Getenv("FLOWORK_LEARN_LOOP")) == "1",
		"stats":   ls,
	})
}

func itoaSmallLearn(n int) string {
	if n <= 0 {
		return "100"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
