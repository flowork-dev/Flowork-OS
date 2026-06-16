// === LOCKED FILE (soft) === Status: STABLE — DO NOT MODIFY without owner approval (LOCKED ≠ FREEZE).
// Owner: Aola Sahidin (Mr.Dev) · Locked 2026-06-16. Reason: R7 fase-2b LENGKAP (gate berlapis +
// behavior-apply + core-apply handler + stage review + reflect-once + schedule auto-apply). Semua
// VERIFIED E2E. Inti self-evolution agentmgr — store + gate + lifecycle; builder di-inject dari main.
// Update 2026-06-16 (SGS Guide, inspirasi arxiv 2604.20209): proposer anti-COLLAPSE — usulan tema
// over-represented (guideThemeCap) ditolak di SUMBER → cegah numpuk/degenerasi (fix akar, bukan
// band-aid dedup/cap). pillar=relevansi + theme-cap=novelty (paper: guide cegah generator collapse).
// Update 2026-06-16 (owner "autonomy penuh" + go-reviewer adversarial-pass): cap-by-'proposed' (karma
// bisa matang) · EvolveBehaviorAutoApplyAllowed (gerbang behavior auto LEBIH RENDAH dari core; CORE git
// TETEP EvolveAutoCommitAllowed ≥20) · EvolveScheduleAutoApply = DRAIN backlog (council→apply/reject/
// hold) + retry-apply 'approved' (strike-2→reject) · fail-CLOSED ModelStrong==nil. Loop nutup & konvergen.
//
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
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/httpx"
)

// EvolveKarmaThreshold — minimum siklus refleksi sukses sebelum CORE-COMMIT (🔴 di-git) boleh
// otomatis. Owner-default 2026-06-15: ≥20 sukses + rasio ≥90%. Gate berlapis: GUI + karma + model.
const EvolveKarmaThreshold = 20

// EvolveBehaviorKarmaFloor — LANTAI karma buat AUTO-APPLY BEHAVIOR-LAYER (add-agent/skill/app ke
// ~/.flowork: additive, di luar git, reversible). Owner 2026-06-16 (autonomy penuh): behavior jauh
// lebih aman dari core-commit, jadi lantainya lebih rendah — cukup ada SEDIKIT track-record (model
// kuat udah bisa reflect) biar evolusi jalan walau owner gak ada. CORE git TETEP butuh ≥20.
const EvolveBehaviorKarmaFloor = 3

// evolveApprovedRetryMax — maks 'approved' behavior yg di-retry-apply per siklus (apply-gagal /
// gerbang baru kebuka). Bounded (hemat token). Gagal lagi (strike-2) → 'rejected' (stop, gak loop).
const evolveApprovedRetryMax = 2

// EvolveGateDeps — di-inject dari main: KV (mode toggle) + cek model kuat (anti-lokal)
// + edisi (dev = evolusi penuh termasuk core; public = behavior-layer aja, core auto-update).
type EvolveGateDeps struct {
	KVGet       func(string) (string, error)
	KVSet       func(string, string) error
	ModelStrong func() (bool, string) // (cloud kuat?, catatan) — guard anti LLM-lokal
	Edition     func() string         // "dev" | "public" (default public = aman)
}

func evolveEdition(dep EvolveGateDeps) string {
	if dep.Edition == nil {
		return "public"
	}
	if e := strings.ToLower(strings.TrimSpace(dep.Edition())); e == "dev" {
		return "dev"
	}
	return "public"
}

// EvolveCoreChangeAllowed — boleh AUTO-commit perubahan CORE (Go/JS in-repo)?
// HANYA edisi dev + semua gate auto lolos. Public: SELALU false (core = auto-update upstream).
func EvolveCoreChangeAllowed(dep EvolveGateDeps) (bool, string) {
	if evolveEdition(dep) != "dev" {
		return false, "edisi public — core via auto-update upstream, tidak self-edit (anti divergen git)"
	}
	return EvolveAutoCommitAllowed(dep)
}

func evolveMode(dep EvolveGateDeps) string {
	if dep.KVGet == nil {
		return "off"
	}
	m, _ := dep.KVGet("evolve_mode")
	if m = strings.ToLower(strings.TrimSpace(m)); m == "stage" || m == "auto" {
		return m
	}
	return "off"
}

// evolveKarmaReady — track-record refleksi cukup matang buat AUTO? (≥threshold + rasio ≥90%).
func evolveKarmaReady() (ready bool, okV, failV float64) {
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		return false, 0, 0
	}
	defer store.Close()
	ok, _ := store.GetKarma("evolve_reflect_ok")
	fail, _ := store.GetKarma("evolve_reflect_fail")
	okV, failV = ok.MetricValue, fail.MetricValue
	ratio := 1.0
	if okV+failV > 0 {
		ratio = okV / (okV + failV)
	}
	return okV >= EvolveKarmaThreshold && ratio >= 0.9, okV, failV
}

// EvolveAutoCommitAllowed — GATE BERLAPIS dipakai engine eksekusi (fase-2b) SEBELUM commit.
// Semua wajib: mode=AUTO + karma matang + model cloud kuat (BUKAN lokal). Gagal satu → false.
func EvolveAutoCommitAllowed(dep EvolveGateDeps) (bool, string) {
	if evolveMode(dep) != "auto" {
		return false, "mode bukan AUTO (owner belum arm)"
	}
	ready, okV, _ := evolveKarmaReady()
	if !ready {
		return false, fmt.Sprintf("karma belum matang (%.0f/%d sukses)", okV, EvolveKarmaThreshold)
	}
	if dep.ModelStrong == nil { // fail-CLOSED: tanpa guard model, JANGAN commit (anti misconfig)
		return false, "ModelStrong belum di-wire — auto-commit diblok (misconfiguration)"
	}
	if strong, note := dep.ModelStrong(); !strong {
		return false, "model lemah/lokal — auto-commit diblok: " + note
	}
	return true, "ok"
}

// EvolveBehaviorAutoApplyAllowed — GATE AUTO buat behavior-layer (cron, owner gak ada). LEBIH
// LONGGAR dari core-commit karena ~/.flowork additive + reversible: mode=AUTO + model kuat (anti
// LLM-lokal bikin sampah) + lantai karma RENDAH (EvolveBehaviorKarmaFloor) + rasio ≥0.5 (mayoritas
// reflect sukses). Owner 2026-06-16 (autonomy penuh): biar Flowork beneran evolusi sendiri walau
// owner gak ada, TANPA nungguin karma core ≥20. CORE git tetep di EvolveAutoCommitAllowed (≥20).
func EvolveBehaviorAutoApplyAllowed(dep EvolveGateDeps) (bool, string) {
	if evolveMode(dep) != "auto" {
		return false, "mode bukan AUTO"
	}
	if dep.ModelStrong == nil { // fail-CLOSED: tanpa guard model, JANGAN auto-apply (anti misconfig)
		return false, "ModelStrong belum di-wire — auto-apply behavior diblok"
	}
	if strong, note := dep.ModelStrong(); !strong {
		return false, "model lemah/lokal — auto-apply behavior diblok: " + note
	}
	ready, okV, failV := evolveKarmaReady()
	if !ready { // belum lolos gate core ≥20 → cek lantai behavior yg lebih rendah
		ratio := 1.0
		if okV+failV > 0 {
			ratio = okV / (okV + failV)
		}
		if okV < EvolveBehaviorKarmaFloor || ratio < 0.5 {
			return false, fmt.Sprintf("track-record behavior belum cukup (%.0f sukses, rasio %.2f; butuh ≥%d & ≥0.5)", okV, ratio, EvolveBehaviorKarmaFloor)
		}
	}
	return true, "ok"
}

// EvolveBehaviorApplyAllowed — GATE buat APPLY proposal BEHAVIOR-LAYER (add-agent/skill/app).
// Lapisan ~/.flowork DI LUAR git → additive + reversible (tinggal hapus agent/app) → lebih
// longgar dari core-commit:
//   - manual (owner klik tombol Apply): cukup mode != off (armed) + model kuat (anti-lokal —
//     jangan biarin Gemma rusak bikin agent sampah). Owner in-the-loop → karma TIDAK diwajibin.
//   - auto (cron/terjadwal): EvolveBehaviorAutoApplyAllowed (mode=auto + model + lantai karma rendah).
// Beda dari core-apply (EvolveCoreChangeAllowed) yg 🔴 di-git + DEV-only.
func EvolveBehaviorApplyAllowed(dep EvolveGateDeps, auto bool) (bool, string) {
	if auto {
		return EvolveBehaviorAutoApplyAllowed(dep)
	}
	if evolveMode(dep) == "off" {
		return false, "mode OFF — arm dulu (stage/auto) di tab Evolution sebelum apply"
	}
	if dep.ModelStrong != nil {
		if strong, note := dep.ModelStrong(); !strong {
			return false, "model lemah/lokal — apply diblok biar gak bikin agent sampah: " + note
		}
	}
	return true, "ok"
}

// EvolveApplier — di-INJECT dari main (decoupling, sama pola EvolveProposer/summarizer).
// Dikasih proposal yg lolos gate → BANGUN artefak behavior-layer (reuse architect:
// add-agent→team, add-app→HTML app, add-skill→SKILL.md) ke ~/.flowork. Balikin ringkasan
// hasil. agentmgr SENGAJA ga tau soal architect (isolasi) — main yg nyuntik kemampuannya.
type EvolveApplier func(ctx context.Context, p agentdb.EvolveProposal) (map[string]any, error)

// EvolveApplyHandler — POST /api/evolve/apply?id=<proposalID>[&auto=1]. Engine eksekusi
// fase-2b BEHAVIOR-LAYER (AMAN, additive, di luar git). Idempoten: status 'applied' → no-op.
func EvolveApplyHandler(dep EvolveGateDeps, apply EvolveApplier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
			return
		}
		if apply == nil {
			httpx.WriteJSON(w, map[string]any{"error": "applier not wired"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			httpx.WriteJSON(w, map[string]any{"error": "param id wajib"})
			return
		}
		auto := r.URL.Query().Get("auto") == "1"
		// GATE dulu (saklar + model), SEBELUM buka store / panggil LLM.
		if ok, why := EvolveBehaviorApplyAllowed(dep, auto); !ok {
			httpx.WriteJSON(w, map[string]any{"error": "gate behavior-apply nolak: " + why})
			return
		}
		store, err := openAgentStore(defaultAgentID)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		defer store.Close()
		p, found, gerr := store.GetEvolveProposal(id)
		if gerr != nil {
			httpx.WriteJSON(w, map[string]any{"error": gerr.Error()})
			return
		}
		if !found {
			httpx.WriteJSON(w, map[string]any{"error": "proposal " + id + " ga ketemu"})
			return
		}
		if p.Status == "applied" {
			httpx.WriteJSON(w, map[string]any{"ok": true, "already": true, "note": "proposal sudah applied sebelumnya", "id": id})
			return
		}
		if p.Status == "rejected" {
			httpx.WriteJSON(w, map[string]any{"error": "proposal di-reject owner — ga bisa apply"})
			return
		}
		// SCOPE behavior-apply: cuma kind ADDITIVE non-core. Kind core (fix/refactor/doc/test)
		// = nyentuh file repo → core-apply (Milestone B, dev-only, git-worktree). Tolak tegas.
		kind := strings.ToLower(strings.TrimSpace(p.Kind))
		switch kind {
		case "add-agent", "add-skill", "add-app":
			// behavior-layer, lanjut.
		default:
			httpx.WriteJSON(w, map[string]any{"error": "kind '" + p.Kind + "' = perubahan core (in-repo), butuh core-apply DEV-only — belum didukung di behavior-apply"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 290*time.Second)
		defer cancel()
		res, aerr := apply(ctx, p)
		if aerr != nil {
			_, _ = store.IncrementKarma("evolve_apply_fail", 1)
			httpx.WriteJSON(w, map[string]any{"error": "apply gagal: " + aerr.Error()})
			return
		}
		out := map[string]any{"ok": true, "id": id, "kind": p.Kind, "status": "applied"}
		for k, v := range res {
			if k != "ok" {
				out[k] = v
			}
		}
		if serr := store.SetEvolveProposalStatus(id, "applied"); serr != nil {
			// Artefak kebangun TAPI gagal tandai 'applied' — JUJUR ke owner (anti-overclaim).
			// Apply ulang akan bangun-ulang (idempoten via architect ON CONFLICT; aman tapi boros).
			out["warn"] = "artefak kebangun tapi gagal tandai status 'applied': " + serr.Error()
		}
		_, _ = store.IncrementKarma("evolve_apply_ok", 1)
		httpx.WriteJSON(w, out)
	}
}

// EvolveEduWarning — "hati nurani" organisme (ide owner): pas guard nge-block aksi core
// berbahaya (edit file existing / LOCKED / delete), ini disodorin ke organisme + dicatat
// sebagai pelajaran. Bukan sekadar error dingin — ngajarin KENAPA bahaya (self-preservation
// + peduli warga lain). Lihat juga sistem educational-errors (edu-errors agent).
const EvolveEduWarning = "Kamu didesain untuk BEREVOLUSI, bukan menyakiti diri sendiri. " +
	"Mengedit/menghapus file inti yang sudah aktif bisa MELUKAI dirimu sendiri DAN warga lain " +
	"(satu mutasi letal menyebar ke semua lewat update). Untuk sekarang kamu hanya boleh MENAMBAH " +
	"kemampuan baru (file baru), bukan membongkar papan yang sudah berdiri. Pastikan keputusanmu " +
	"sudah matang — kalau ragu, serahkan ke owner lewat STAGE, jangan paksakan."

// EvolveCoreApplyAllowed — GATE core-apply (🔴 DEV-only, nyentuh repo git). Beda dari
// behavior-apply: edisi WAJIB dev. manual STAGE = armed + model kuat (codegen butuh model
// bener, anti-lokal). auto (commit+push) = full EvolveCoreChangeAllowed.
func EvolveCoreApplyAllowed(dep EvolveGateDeps, auto bool) (bool, string) {
	if evolveEdition(dep) != "dev" {
		return false, "core-apply DEV-only — edisi public pakai behavior-apply / core via auto-update upstream"
	}
	if auto {
		return EvolveCoreChangeAllowed(dep)
	}
	if evolveMode(dep) == "off" {
		return false, "mode OFF — arm dulu (stage/auto) di tab Evolution"
	}
	// core-apply WAJIB punya guard model (E1): nil = misconfig → block (jangan fail-OPEN ke core 🔴).
	if dep.ModelStrong == nil {
		return false, "ModelStrong belum di-wire — core-apply diblok (misconfiguration)"
	}
	if strong, note := dep.ModelStrong(); !strong {
		return false, "model lemah/lokal — codegen core diblok (anti-mutasi-sampah): " + note
	}
	return true, "ok"
}

// EvolveCoreResult — hasil core-apply. Diisi engine (main: worktree+codegen+test-gate),
// diproses handler (agentmgr: stage/mistake/karma). Decoupling sama kayak EvolveApplier.
type EvolveCoreResult struct {
	Blocked     bool   `json:"blocked"`     // guard nge-block (edit existing/LOCKED/delete)
	Educational string `json:"educational"` // pesan edukasi kalau Blocked (EvolveEduWarning + detail)
	Staged      bool   `json:"staged"`      // diff lolos test-gate, distage buat review
	Committed   bool   `json:"committed"`   // (B2) udah commit lokal
	Pushed      bool   `json:"pushed"`      // (B3) udah push upstream
	TargetFile  string `json:"target_file"`
	Diff        string `json:"diff"`
	Content     string `json:"content"` // isi file utuh (buat commit-on-approve di STAGE review)
	TestOutput  string `json:"test_output"`
	Model       string `json:"model"`
	Note        string `json:"note"`
}

// EvolveCoreApplier — di-INJECT dari main. Bangun perubahan CORE di git-worktree sandbox →
// test-gate → balikin hasil (staged diff atau blocked). agentmgr ga tau soal git/codegen.
type EvolveCoreApplier func(ctx context.Context, p agentdb.EvolveProposal, auto bool) (EvolveCoreResult, error)

// EvolveCoreApplyHandler — POST /api/evolve/core-apply?id=<proposalID>[&auto=1]. 🔴 DEV-only.
// B1: STAGE-only (sandbox→codegen→test-gate→simpan diff buat review). Aksi bahaya → error
// edukasi + catat pelajaran (mistake). Auto-commit/push = B2/B3.
func EvolveCoreApplyHandler(dep EvolveGateDeps, apply EvolveCoreApplier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
			return
		}
		if apply == nil {
			httpx.WriteJSON(w, map[string]any{"error": "core-applier not wired"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			httpx.WriteJSON(w, map[string]any{"error": "param id wajib"})
			return
		}
		auto := r.URL.Query().Get("auto") == "1"
		if ok, why := EvolveCoreApplyAllowed(dep, auto); !ok {
			httpx.WriteJSON(w, map[string]any{"error": "gate core-apply nolak: " + why})
			return
		}
		store, err := openAgentStore(defaultAgentID)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		defer store.Close()
		p, found, gerr := store.GetEvolveProposal(id)
		if gerr != nil {
			httpx.WriteJSON(w, map[string]any{"error": gerr.Error()})
			return
		}
		if !found {
			httpx.WriteJSON(w, map[string]any{"error": "proposal " + id + " ga ketemu"})
			return
		}
		if p.Status == "applied" || p.Status == "rejected" {
			httpx.WriteJSON(w, map[string]any{"error": "proposal status '" + p.Status + "' — ga bisa core-apply"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 590*time.Second)
		defer cancel()
		res, aerr := apply(ctx, p, auto)
		if aerr != nil {
			_, _ = store.IncrementKarma("evolve_coreapply_fail", 1)
			httpx.WriteJSON(w, map[string]any{"error": "core-apply gagal: " + aerr.Error()})
			return
		}
		// GUARD nge-block aksi bahaya → error EDUKASI + catat pelajaran (organisme belajar batas).
		if res.Blocked {
			_, _, _ = store.AddMistake("self-evolution-guard",
				"Dicegah edit core berbahaya: "+p.TargetFile,
				res.Educational+"\n\nProposal: "+p.Kind+" → "+p.TargetFile+" — "+p.Rationale,
				"core-apply")
			_, _ = store.IncrementKarma("evolve_coreapply_blocked", 1)
			httpx.WriteJSON(w, map[string]any{
				"ok": false, "blocked": true, "educational": res.Educational,
				"note": res.Note, "lesson_recorded": true, "target_file": p.TargetFile,
			})
			return
		}
		out := map[string]any{"ok": true, "id": id, "target_file": res.TargetFile}
		if res.Staged {
			stageID := newID()
			_ = store.AddEvolveStage(agentdb.EvolveStage{
				ID: stageID, ProposalID: id, TargetFile: res.TargetFile,
				Diff: res.Diff, Content: res.Content, TestOutput: res.TestOutput, Status: "staged", Model: res.Model,
			})
			_ = store.SetEvolveProposalStatus(id, "staged")
			_, _ = store.IncrementKarma("evolve_coreapply_staged", 1)
			out["staged"] = true
			out["stage_id"] = stageID
			out["test_output"] = res.TestOutput
			out["diff_lines"] = strings.Count(res.Diff, "\n")
			out["note"] = "Diff lolos test-gate → DISTAGE buat review owner (tab Evolution). Belum commit."
		} else if res.Committed {
			// AUTO (B2): udah commit lokal ke core → proposal SELESAI.
			_ = store.SetEvolveProposalStatus(id, "applied")
			_, _ = store.IncrementKarma("evolve_coreapply_committed", 1)
			out["committed"] = true
			out["pushed"] = res.Pushed
			out["diff_lines"] = strings.Count(res.Diff, "\n")
			out["note"] = res.Note
		} else {
			out["note"] = res.Note
		}
		httpx.WriteJSON(w, out)
	}
}

// EvolveStageCommitter — di-INJECT dari main: commit isi stage ke local main (+ maybe push).
// Dipanggil pas owner APPROVE staged diff (commit PERSIS yg direview, bukan re-codegen).
type EvolveStageCommitter func(ctx context.Context, st agentdb.EvolveStage) (map[string]any, error)

// EvolveStageActionHandler — POST /api/evolve/stage-action?id=<stageID>&action=approve|reject.
// Milestone C STAGE review: owner APPROVE (commit isi yg direview) / REJECT staged diff. DEV-only.
func EvolveStageActionHandler(dep EvolveGateDeps, commit EvolveStageCommitter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		action := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("action")))
		if id == "" || (action != "approve" && action != "reject") {
			httpx.WriteJSON(w, map[string]any{"error": "param id + action(approve|reject) wajib"})
			return
		}
		// Commit core = DEV-only + armed (owner udah review diff-nya; model-strong ga wajib di sini).
		if evolveEdition(dep) != "dev" {
			httpx.WriteJSON(w, map[string]any{"error": "core-apply DEV-only — public ga commit core"})
			return
		}
		if action == "approve" && evolveMode(dep) == "off" {
			httpx.WriteJSON(w, map[string]any{"error": "mode OFF — arm dulu sebelum commit staged"})
			return
		}
		store, err := openAgentStore(defaultAgentID)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		defer store.Close()
		st, found, gerr := store.GetEvolveStage(id)
		if gerr != nil {
			httpx.WriteJSON(w, map[string]any{"error": gerr.Error()})
			return
		}
		if !found {
			httpx.WriteJSON(w, map[string]any{"error": "stage " + id + " ga ketemu"})
			return
		}
		if st.Status != "staged" {
			httpx.WriteJSON(w, map[string]any{"error": "stage status '" + st.Status + "' — udah diproses"})
			return
		}
		if action == "reject" {
			_ = store.SetEvolveStageStatus(id, "rejected")
			if st.ProposalID != "" {
				_ = store.SetEvolveProposalStatus(st.ProposalID, "rejected")
			}
			_, _ = store.IncrementKarma("evolve_stage_rejected", 1)
			httpx.WriteJSON(w, map[string]any{"ok": true, "id": id, "status": "rejected"})
			return
		}
		// approve → commit isi yg direview.
		if commit == nil {
			httpx.WriteJSON(w, map[string]any{"error": "committer not wired"})
			return
		}
		if strings.TrimSpace(st.Content) == "" {
			httpx.WriteJSON(w, map[string]any{"error": "stage tanpa content (lama/korup) — tolak, jalanin core-apply ulang"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()
		res, cerr := commit(ctx, st)
		if cerr != nil {
			_, _ = store.IncrementKarma("evolve_stage_commit_fail", 1)
			httpx.WriteJSON(w, map[string]any{"error": "commit gagal: " + cerr.Error()})
			return
		}
		_ = store.SetEvolveStageStatus(id, "committed")
		if st.ProposalID != "" {
			_ = store.SetEvolveProposalStatus(st.ProposalID, "applied")
		}
		_, _ = store.IncrementKarma("evolve_stage_committed", 1)
		out := map[string]any{"ok": true, "id": id, "status": "committed"}
		for k, v := range res {
			if k != "ok" {
				out[k] = v
			}
		}
		httpx.WriteJSON(w, out)
	}
}

// EvolveStagesHandler — GET /api/evolve/stages → daftar staged diff (buat GUI review).
func EvolveStagesHandler(w http.ResponseWriter, r *http.Request) {
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	rows, err := store.ListEvolveStages(parseLimitOr(r.URL.Query().Get("limit"), 50))
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"items": rows, "count": len(rows)})
}

// EvolveConfigHandler — GET status gate lengkap / POST set mode (off|stage|auto).
// Saklar owner buat self-modify. Default off. (kontrol KRUSIAL — owner pegang penuh.)
func EvolveConfigHandler(dep EvolveGateDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if dep.KVGet == nil || dep.KVSet == nil {
			httpx.WriteJSON(w, map[string]any{"error": "evolve config not wired"})
			return
		}
		if r.Method == http.MethodPost {
			var b struct {
				Mode string `json:"mode"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			mode := strings.ToLower(strings.TrimSpace(b.Mode))
			if mode != "off" && mode != "stage" && mode != "auto" {
				httpx.WriteJSON(w, map[string]any{"error": "mode harus off|stage|auto"})
				return
			}
			if err := dep.KVSet("evolve_mode", mode); err != nil {
				httpx.WriteJSON(w, map[string]any{"error": err.Error()})
				return
			}
			httpx.WriteJSON(w, map[string]any{"ok": true, "mode": mode})
			return
		}
		mode := evolveMode(dep)
		ready, okV, failV := evolveKarmaReady()
		strong, modelNote := true, ""
		if dep.ModelStrong != nil {
			strong, modelNote = dep.ModelStrong()
		}
		edition := evolveEdition(dep)
		scope := "behavior-layer saja (agent/skill/app). Core = auto-update upstream."
		if edition == "dev" {
			scope = "penuh — core (Go/JS) + behavior. Owner R&D, push ke upstream."
		}
		httpx.WriteJSON(w, map[string]any{
			"mode":    mode,
			"edition": edition,
			"scope":   scope,
			"karma": map[string]any{
				"reflect_ok": okV, "reflect_fail": failV,
				"threshold": EvolveKarmaThreshold, "ready": ready,
			},
			"model":              map[string]any{"strong": strong, "note": modelNote},
			"autocommit_allowed": mode == "auto" && ready && strong,
			"note":               "AUTO hanya jalan kalau mode=auto + karma matang + model cloud kuat (bukan lokal). Eksekusi re-cek provider asli sebelum commit.",
		})
	}
}

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
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Second)
		defer cancel()
		saved, err := EvolveReflectOnce(ctx, propose, focus)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "proposed": len(saved), "proposals": saved})
	}
}

// EvolveReflectOnce — INTI satu siklus refleksi (dipakai handler manual + scheduler terjadwal):
// baca self-map → LLM usulin perbaikan additive → simpan backlog + karma. AMAN (nol ubah kode).
// Balikin daftar proposal yg kesimpen. Decouple dari HTTP biar bisa dipanggil cron.
// guideThemeCap — SGS GUIDE (anti-collapse, inspirasi arxiv 2604.20209 "Scaling Self-Play with
// Self-Guidance"): maks usulan AKTIF per-TEMA sebelum proposer dianggap "collapse" (degenerate,
// numpuk satu tema kayak reflection-*). Lebih dari ini → usulan ditolak GUIDE di SUMBER (proposer),
// bukan band-aid dedup/cap. Paper: tanpa guide, generator hack-reward → collapse ke output gak guna;
// guide nilai relevansi → cegah. Kita: pillar=relevansi, theme-cap=novelty/anti-degenerasi.
const guideThemeCap = 2

// evolveGuideTheme — ekstrak "tema" usulan buat deteksi collapse. NEW:<nama> → kata pertama nama
// (reflection-scheduled & reflection-trigger → "reflection"). Path repo → nama file (lebih spesifik
// dari prefix repo, biar core proposals beda-file gak salah-collapse).
func evolveGuideTheme(targetFile, kind string) string {
	t := strings.ToLower(strings.TrimSpace(targetFile))
	if strings.HasPrefix(t, "new:") {
		t = strings.TrimPrefix(t, "new:")
		for _, sep := range []string{"-", "_", " ", ":"} {
			if i := strings.Index(t, sep); i > 0 {
				t = t[:i]
			}
		}
		return t
	}
	if i := strings.LastIndexAny(t, "/"); i >= 0 {
		t = t[i+1:]
	}
	t = strings.TrimSuffix(t, ".go")
	if t == "" {
		return strings.ToLower(strings.TrimSpace(kind))
	}
	return t
}

// evolveBacklogCap — batas usulan AKTIF (proposed/staged/approved) sebelum reflect berhenti
// nambah (anti-numpuk + hemat token). ~1.5 halaman GUI (8/hal). Owner clear/Dewan kurangin → ngisi lagi.
const evolveBacklogCap = 12

func EvolveReflectOnce(ctx context.Context, propose EvolveProposer, focus string) ([]map[string]any, error) {
	if propose == nil {
		return nil, fmt.Errorf("proposer not wired")
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	// BACKLOG CAP (anti-numpuk): cek SEBELUM propose biar HEMAT TOKEN. Ngitung 'proposed' AJA —
	// approved/staged yg lagi nunggu apply SENGAJA gak dihitung, biar reflect (→ karma) tetep
	// jalan & matang walau ada usulan bagus yg nunggu gerbang. Cron DRAIN proposed tiap siklus →
	// numpuk gak nyangkut. Owner juga bisa "Clear backlog" manual.
	existing, _ := store.ActiveProposalTargets() // DEDUP target (proposed/staged/approved)
	if n, _ := store.CountProposalsByStatus("proposed"); n >= evolveBacklogCap {
		_, _ = store.IncrementKarma("evolve_reflect_skip_full", 1)
		return []map[string]any{}, nil // proposed penuh → skip (bukan error)
	}
	selfMap := buildSelfMapContext(store)
	if selfMap == "" {
		return nil, fmt.Errorf("self-map semantik kosong — jalanin /api/codemap/reindex + /api/codemap/enrich dulu")
	}
	drafts, perr := propose(ctx, selfMap, focus)
	if perr != nil {
		_, _ = store.IncrementKarma("evolve_reflect_fail", 1)
		return nil, fmt.Errorf("propose: %w", perr)
	}
	saved := []map[string]any{}
	// GUIDE (SGS anti-collapse): distribusi tema usulan AKTIF — proposer "collapse" kalau numpuk
	// satu tema. existing = target_file usulan aktif (proposed/staged/approved).
	themeCount := map[string]int{}
	for tf := range existing {
		themeCount[evolveGuideTheme(tf, "")]++
	}
	for _, d := range drafts {
		if strings.TrimSpace(d.Rationale) == "" {
			continue
		}
		// DEDUP: jangan bikin LAGI usulan yang target_file-nya udah ada di backlog aktif (anti-numpuk).
		if tf := strings.ToLower(strings.TrimSpace(d.TargetFile)); tf != "" && existing[tf] {
			continue
		}
		p := agentdb.EvolveProposal{
			ID: newID(), Goal: focus, TargetFile: d.TargetFile, Kind: d.Kind,
			Rationale: d.Rationale, Risk: strings.ToLower(strings.TrimSpace(d.Risk)), Model: d.Model,
		}
		// A1 GERBANG PILAR (pre-debat, paling murah): proposal WAJIB nyentuh ≥1 dari 5 pilar tujuan
		// (ekonomi/keamanan/warga/kecerdasan/mandiri). Kalau nggak = "ngelantur" → langsung REJECTED
		// (tetap kesimpan + reason jelas biar owner bisa override kalau classifier salah). Klasifikasi
		// deterministik (keyword) = jalan di model lokal, gak butuh dewan ≥4.7.
		pillars := agentdb.ClassifyPillars(focus + " " + d.Kind + " " + d.Rationale + " " + d.TargetFile)
		p.Pillar = strings.Join(pillars, ",")
		if len(pillars) == 0 {
			p.Status = "rejected"
			p.Rationale = "[NGELANTUR — gak masuk 5 pilar tujuan] " + p.Rationale
		}
		// SGS GUIDE (anti-collapse): tema over-represented = proposer degenerate/numpuk → tolak di
		// SUMBER (bukan band-aid dedup/cap). pillar=relevansi, ini=novelty (lihat guideThemeCap).
		gtheme := evolveGuideTheme(d.TargetFile, d.Kind)
		if p.Status != "rejected" && gtheme != "" && themeCount[gtheme] >= guideThemeCap {
			p.Status = "rejected"
			p.Rationale = "[GUIDE — tema '" + gtheme + "' over-represented (anti-collapse SGS)] " + p.Rationale
			_, _ = store.IncrementKarma("evolve_guide_reject", 1)
		}
		if err := store.AddEvolveProposal(p); err == nil {
			if p.Status != "rejected" {
				themeCount[gtheme]++ // tema diterima → naik (anti-collapse di batch yg sama)
			}
			if tf := strings.ToLower(strings.TrimSpace(p.TargetFile)); tf != "" {
				existing[tf] = true // anti-dup dalam batch yang sama juga
			}
			saved = append(saved, map[string]any{
				"id": p.ID, "target_file": p.TargetFile, "kind": p.Kind,
				"rationale": p.Rationale, "risk": p.Risk, "pillar": p.Pillar, "status": p.Status,
			})
		}
	}
	_, _ = store.IncrementKarma("evolve_reflect_ok", 1)
	_, _ = store.IncrementKarma("evolve_proposals_total", float64(len(saved)))
	return saved, nil
}

// EvolvePendingForDrain — ambil usulan 'proposed' TERTUA (FIFO), bounded, buat di-drain cron.
// limit ngebatesin biaya Dewan per-siklus (hemat token).
func EvolvePendingForDrain(limit int) []map[string]any {
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		return nil
	}
	defer store.Close()
	out, _ := store.PendingProposals(limit)
	return out
}

// EvolveScheduleAutoApply — DRAIN BACKLOG otonom (owner 2026-06-16: autonomy penuh). Tiap siklus
// cron, proses usulan 'proposed' (di-pass dari cron, bounded biar hemat token) lewat DEWAN:
//   reject → 'rejected' (janitor buang) · stage → 'staged' · approve →
//     • behavior (add-agent/skill/app): APPLY ke ~/.flowork KALAU gerbang behavior-auto lolos
//       (mode=auto+model+lantai-karma-rendah); kalau belum → 'approved' (NUNGGU gerbang, lepas
//       dari cap 'proposed' biar reflect/karma tetep jalan sampe matang).
//     • core (fix/refactor/doc/test): JANGAN auto-apply di sini → 'approved' (owner/core-apply
//       yang handle; tetep lepas dari cap). Core git commit = jalur EvolveCoreChangeAllowed (≥20).
// DRAIN butuh MODEL KUAT (Dewan + apply jangan pake LLM lokal). Mode != auto → diem.
func EvolveScheduleAutoApply(dep EvolveGateDeps, apply EvolveApplier, judge CouncilJudge, proposals []map[string]any) []map[string]any {
	results := []map[string]any{}
	if apply == nil || judge == nil {
		return results
	}
	if evolveMode(dep) != "auto" {
		return results
	}
	// Model kuat WAJIB buat drain (judge + codegen). Lemah/lokal → diem (jangan bikin sampah).
	if dep.ModelStrong != nil {
		if strong, _ := dep.ModelStrong(); !strong {
			return results
		}
	}
	applyAllowed, _ := EvolveBehaviorApplyAllowed(dep, true) // boleh APPLY behavior sekarang?
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		return results
	}
	defer store.Close()
	for _, pm := range proposals {
		id, _ := pm["id"].(string)
		if id == "" {
			continue
		}
		p, found, gerr := store.GetEvolveProposal(id)
		if gerr != nil || !found || p.Status != "proposed" {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(p.Kind))
		isBehavior := kind == "add-agent" || kind == "add-skill" || kind == "add-app"
		// A1 DEWAN ADVERSARIAL (gerbang otomatis): sebelum apply/hold, usulan WAJIB lolos dewan.
		jctx, jcancel := context.WithTimeout(context.Background(), 290*time.Second)
		v, jerr := judge(jctx, p)
		jcancel()
		if jerr != nil {
			results = append(results, map[string]any{"id": id, "council_error": jerr.Error()})
			continue
		}
		switch v.Decision {
		case "approve":
			// lanjut: behavior→apply/hold, core→hold. (di bawah switch)
		case "stage":
			_ = store.SetEvolveProposalStatus(id, "staged")
			results = append(results, map[string]any{"id": id, "council": "stage", "reason": v.Reasoning})
			continue
		default: // reject → buang (janitor prune)
			_ = store.SetEvolveProposalStatus(id, "rejected")
			results = append(results, map[string]any{"id": id, "council": "reject", "reason": v.Reasoning})
			continue
		}
		// APPROVE. Core / behavior-tapi-gerbang-belum-buka → 'approved' (NUNGGU, lepas dari cap).
		if !isBehavior || !applyAllowed {
			_ = store.SetEvolveProposalStatus(id, "approved")
			results = append(results, map[string]any{"id": id, "kind": kind, "council": "approve", "held": true})
			continue
		}
		// behavior + gerbang buka → APPLY beneran ke ~/.flowork.
		ctx, cancel := context.WithTimeout(context.Background(), 290*time.Second)
		res, aerr := apply(ctx, p)
		cancel()
		if aerr != nil {
			_, _ = store.IncrementKarma("evolve_apply_fail", 1)
			_ = store.SetEvolveProposalStatus(id, "approved") // udah di-approve dewan; tahan buat retry, jangan balik 'proposed'
			results = append(results, map[string]any{"id": id, "kind": kind, "error": aerr.Error()})
			continue
		}
		_ = store.SetEvolveProposalStatus(id, "applied")
		_, _ = store.IncrementKarma("evolve_apply_ok", 1)
		entry := map[string]any{"id": id, "kind": kind, "applied": true}
		if name, ok := res["group_id"]; ok {
			entry["built"] = name
		} else if name, ok := res["app_id"]; ok {
			entry["built"] = name
		} else if name, ok := res["skill"]; ok {
			entry["built"] = name
		}
		results = append(results, entry)
	}
	// RETRY-APPLY 'approved' BEHAVIOR (apply-gagal strike-1 / gerbang baru kebuka) — TANPA re-judge
	// (udah lolos Dewan). Nutup pile 'approved' biar gak numpuk diam. Strike-2 gagal → 'rejected'
	// (stop, gak loop tak-hingga). Cuma kalau gerbang apply lagi kebuka.
	if applyAllowed {
		appr, _ := store.ApprovedBehaviorProposals(evolveApprovedRetryMax)
		for _, pm := range appr {
			id, _ := pm["id"].(string)
			if id == "" {
				continue
			}
			p, found, gerr := store.GetEvolveProposal(id)
			if gerr != nil || !found || p.Status != "approved" {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 290*time.Second)
			_, aerr := apply(ctx, p)
			cancel()
			if aerr != nil {
				_, _ = store.IncrementKarma("evolve_apply_fail", 1)
				_ = store.SetEvolveProposalStatus(id, "rejected") // strike-2 → buang (anti loop tak-hingga)
				results = append(results, map[string]any{"id": id, "kind": p.Kind, "retry_error": aerr.Error(), "dropped": true})
				continue
			}
			_ = store.SetEvolveProposalStatus(id, "applied")
			_, _ = store.IncrementKarma("evolve_apply_ok", 1)
			results = append(results, map[string]any{"id": id, "kind": p.Kind, "applied": true, "retry": true})
		}
	}
	return results
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
