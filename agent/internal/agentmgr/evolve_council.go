// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-16 (autonomous sprint A1 dewan).
// LOCKED ≠ FREEZE (boleh diedit dgn izin owner).
//
// evolve_council.go — A1 DEWAN ADVERSARIAL (lapisan deliberatif governance evolusi).
//
// Setelah gerbang-pilar (saringan murah: must-fit ≥1 pilar), proposal yg mau di-approve lewat DEWAN:
//   🟢 Pembela (advocate) ⚔️ 🔴 Penantang (skeptic, semi-veto) → ⚖️ Hakim panel-3 (vote).
// Pakai MODEL KUAT (di-inject dari main = coderModel/Opus). KONSERVATIF: ragu=stage/reject, keamanan=
// lantai keras, butuh mayoritas approve TANPA veto fatal. Decoupling sama pola EvolveProposer:
// logika LLM (debat) di main, handler + akses store di sini.

package agentmgr

import (
	"context"
	"net/http"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/httpx"
)

// CouncilJudgeVote — putusan 1 hakim panel.
type CouncilJudgeVote struct {
	Decision string `json:"decision"` // approve | stage | reject
	Score    int    `json:"score"`    // 0-10
	Reason   string `json:"reason"`
}

// CouncilVerdict — hasil dewan atas 1 proposal.
type CouncilVerdict struct {
	Decision      string             `json:"decision"` // FINAL: approve | stage | reject
	Pembela       string             `json:"pembela"`
	Penantang     string             `json:"penantang"`
	PenantangVeto bool               `json:"penantang_veto"`
	Judges        []CouncilJudgeVote `json:"judges"`
	Reasoning     string             `json:"reasoning"`
	Model         string             `json:"model"`
}

// CouncilJudge — di-INJECT dari main (routerChat + model kuat). Jalanin debat adversarial → verdict.
type CouncilJudge func(ctx context.Context, p agentdb.EvolveProposal) (CouncilVerdict, error)

// EvolveCouncilHandler — POST /api/evolve/council?id=<proposal>. Jalanin dewan atas 1 proposal,
// balikin verdict, DAN update status proposal sesuai keputusan (approve→approved / reject→rejected /
// stage→staged) biar kelihatan di backlog GUI. Owner-gated (auth middleware).
func EvolveCouncilHandler(judge CouncilJudge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJSON(w, map[string]any{"error": "POST only"})
			return
		}
		if judge == nil {
			httpx.WriteJSON(w, map[string]any{"error": "dewan belum di-wire (model kuat?)"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			httpx.WriteJSON(w, map[string]any{"error": "id proposal wajib"})
			return
		}
		store, err := openAgentStore(defaultAgentID)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "open store: " + err.Error()})
			return
		}
		defer store.Close()
		p, found, gerr := store.GetEvolveProposal(id)
		if gerr != nil {
			httpx.WriteJSON(w, map[string]any{"error": gerr.Error()})
			return
		}
		if !found {
			httpx.WriteJSON(w, map[string]any{"error": "proposal ga ketemu: " + id})
			return
		}
		// Dewan = 5 panggilan LLM model-kuat (pembela+penantang+3 hakim) → kasih budget lega.
		ctx, cancel := context.WithTimeout(r.Context(), 290*time.Second)
		defer cancel()
		v, verr := judge(ctx, p)
		if verr != nil {
			httpx.WriteJSON(w, map[string]any{"error": "dewan gagal: " + verr.Error()})
			return
		}
		switch v.Decision {
		case "approve":
			_ = store.SetEvolveProposalStatus(id, "approved")
		case "reject":
			_ = store.SetEvolveProposalStatus(id, "rejected")
		case "stage":
			_ = store.SetEvolveProposalStatus(id, "staged")
		}
		httpx.WriteJSON(w, map[string]any{"proposal_id": id, "verdict": v})
	}
}
