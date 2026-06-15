// selfevolve.go — R7 SELF-EVOLUTION fase-1 (refleksi-diri → backlog usulan). Plug-in.
// Owner-approved 2026-06-15 (FASE 2 autonomi). Organisme BACA self-map semantik (R6) →
// architect/LLM USULIN perbaikan konkret → simpan ke evolve_proposal. FASE-1 = USULAN
// doang (NOL ubah kode/sistem) → aman, sekaligus NGUMPULIN KARMA (kualitas usulan).
// Eksekusi (sandbox→test-gate→auto-commit) = fase-2, di-GATE karma + scope non-locked.
// LLM di-INJECT dari main (decoupling, sama kayak codemap_semantic R6).

package agentmgr

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/httpx"
)

// ProposalDraft — usulan mentah dari LLM (sebelum dikasih id + disimpan).
type ProposalDraft struct {
	TargetFile string `json:"target_file"`
	Kind       string `json:"kind"`
	Rationale  string `json:"rationale"`
	Risk       string `json:"risk"`
	Model      string `json:"model"`
}

// EvolveProposer — di-inject dari main (routerChat). Dikasih ringkasan self-map +
// fokus → balikin daftar usulan konkret.
type EvolveProposer func(ctx context.Context, selfMapContext, focus string) ([]ProposalDraft, error)

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "ev_" + hex.EncodeToString(b)
}

// buildSelfMapContext — rangkai lapisan makna jadi konteks ringkas buat LLM refleksi.
// Cap jumlah baris biar prompt kecil (prinsip semut, ramah model).
func buildSelfMapContext(store *agentdb.Store) string {
	rows, err := store.ListCodemapSemantic()
	if err != nil || len(rows) == 0 {
		return ""
	}
	sort.Slice(rows, func(i, j int) bool {
		di, _ := rows[i]["domain"].(string)
		dj, _ := rows[j]["domain"].(string)
		return di < dj
	})
	var b strings.Builder
	const maxLines = 120
	for i, r := range rows {
		if i >= maxLines {
			break
		}
		path, _ := r["path"].(string)
		dom, _ := r["domain"].(string)
		role, _ := r["role"].(string)
		sum, _ := r["summary"].(string)
		b.WriteString("- ")
		b.WriteString(path)
		b.WriteString(" [")
		b.WriteString(dom)
		b.WriteString("/")
		b.WriteString(role)
		b.WriteString("]: ")
		b.WriteString(sum)
		b.WriteString("\n")
	}
	return b.String()
}

// EvolveReflectHandler — POST /api/evolve/reflect?focus=&model=
// Refleksi-diri: baca self-map → LLM usulin perbaikan → simpan backlog. AMAN (nol ubah kode).
func EvolveReflectHandler(propose EvolveProposer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
			return
		}
		if propose == nil {
			httpx.WriteJSON(w, map[string]any{"error": "proposer not wired"})
			return
		}
		focus := strings.TrimSpace(r.URL.Query().Get("focus"))
		store, err := openAgentStore(defaultAgentID)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		defer store.Close()
		selfMap := buildSelfMapContext(store)
		if selfMap == "" {
			httpx.WriteJSON(w, map[string]any{"error": "self-map semantik kosong — jalanin /api/codemap/reindex + /api/codemap/enrich dulu"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Second)
		defer cancel()
		drafts, perr := propose(ctx, selfMap, focus)
		if perr != nil {
			_, _ = store.IncrementKarma("evolve_reflect_fail", 1)
			httpx.WriteJSON(w, map[string]any{"error": "propose: " + perr.Error()})
			return
		}
		saved := []map[string]any{}
		for _, d := range drafts {
			if strings.TrimSpace(d.Rationale) == "" {
				continue
			}
			p := agentdb.EvolveProposal{
				ID: newID(), Goal: focus, TargetFile: d.TargetFile, Kind: d.Kind,
				Rationale: d.Rationale, Risk: strings.ToLower(strings.TrimSpace(d.Risk)), Model: d.Model,
			}
			if err := store.AddEvolveProposal(p); err == nil {
				saved = append(saved, map[string]any{
					"id": p.ID, "target_file": p.TargetFile, "kind": p.Kind,
					"rationale": p.Rationale, "risk": p.Risk,
				})
			}
		}
		// Karma: 1 siklus refleksi sukses + counter jumlah usulan (track-record).
		_, _ = store.IncrementKarma("evolve_reflect_ok", 1)
		_, _ = store.IncrementKarma("evolve_proposals_total", float64(len(saved)))
		httpx.WriteJSON(w, map[string]any{"ok": true, "proposed": len(saved), "proposals": saved})
	}
}

// EvolveProposalsHandler — GET /api/evolve/proposals → backlog usulan evolusi.
func EvolveProposalsHandler(w http.ResponseWriter, r *http.Request) {
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	rows, err := store.ListEvolveProposals(parseLimitOr(r.URL.Query().Get("limit"), 100))
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	// Karma ringkas (kesiapan auto-commit fase-2 = track-record refleksi).
	okC, _ := store.GetKarma("evolve_reflect_ok")
	failC, _ := store.GetKarma("evolve_reflect_fail")
	httpx.WriteJSON(w, map[string]any{
		"items": rows, "count": len(rows),
		"karma": map[string]any{"reflect_ok": okC, "reflect_fail": failC},
	})
}
