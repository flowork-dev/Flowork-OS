// evolution.go — R7 SELF-EVOLUTION fase-1 (plug-in, additive). Owner-approved 2026-06-15.
// Backlog usulan evolusi: organisme refleksi diri (baca self-map R6) → usulin perbaikan
// konkret → SIMPAN di sini buat review/eksekusi. FASE-1 = usulan doang (NOL ubah kode);
// eksekusi (sandbox→apply→auto-commit) = fase-2 di-GATE karma. Tabel terpisah, non-destruktif.

package agentdb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// EvolveProposal — satu usulan evolusi dari refleksi-diri.
type EvolveProposal struct {
	ID         string `json:"id"`
	Goal       string `json:"goal"`        // konteks/fokus refleksi
	TargetFile string `json:"target_file"` // file yang diusulin disentuh (relatif repo)
	Kind       string `json:"kind"`        // add-agent | add-skill | add-app | fix | refactor | doc | test
	Rationale  string `json:"rationale"`   // kenapa (1-2 kalimat)
	Risk       string `json:"risk"`        // low | medium | high
	Status     string `json:"status"`      // proposed | approved | rejected | applied
	Model      string `json:"model"`
	CreatedAt  string `json:"created_at"`
	Pillar     string `json:"pillar"` // A1: pilar tujuan yg disentuh (CSV id, mis. "ekonomi,keamanan"). "" = ngelantur / belum diklasifikasi.
}

func (s *Store) ensureEvolveSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS evolve_proposal (
		  id          TEXT PRIMARY KEY,
		  goal        TEXT NOT NULL DEFAULT '',
		  target_file TEXT NOT NULL DEFAULT '',
		  kind        TEXT NOT NULL DEFAULT '',
		  rationale   TEXT NOT NULL DEFAULT '',
		  risk        TEXT NOT NULL DEFAULT 'medium',
		  status      TEXT NOT NULL DEFAULT 'proposed',
		  model       TEXT NOT NULL DEFAULT '',
		  created_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}
	// evolve_stage — R7 fase-2b core-apply: hasil sandbox (diff + test-gate) yg NUNGGU
	// review owner sebelum commit. Additive, terpisah dari proposal (1 proposal bisa di-stage
	// ulang). Milestone C GUI baca ini buat tombol Approve/Reject.
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS evolve_stage (
		  id           TEXT PRIMARY KEY,
		  proposal_id  TEXT NOT NULL DEFAULT '',
		  target_file  TEXT NOT NULL DEFAULT '',
		  diff         TEXT NOT NULL DEFAULT '',
		  test_output  TEXT NOT NULL DEFAULT '',
		  status       TEXT NOT NULL DEFAULT 'staged',
		  model        TEXT NOT NULL DEFAULT '',
		  created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}
	// Migrasi ADDITIVE (idempoten): kolom content = isi file utuh (buat commit-on-approve persis
	// yg direview). ALTER gagal "duplicate column" kalau udah ada → diabaikan (aman).
	_, _ = s.db.Exec(`ALTER TABLE evolve_stage ADD COLUMN content TEXT NOT NULL DEFAULT ''`)
	// A1 governance: pilar tujuan yg disentuh proposal (CSV id). ADDITIVE idempoten.
	_, _ = s.db.Exec(`ALTER TABLE evolve_proposal ADD COLUMN pillar TEXT NOT NULL DEFAULT ''`)
	return nil
}

// AddEvolveProposal — simpan 1 usulan (id wajib unik; caller bikin).
func (s *Store) AddEvolveProposal(p EvolveProposal) error {
	if err := s.ensureEvolveSchema(); err != nil {
		return err
	}
	if p.CreatedAt == "" {
		p.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if p.Status == "" {
		p.Status = "proposed"
	}
	_, err := s.db.Exec(`
		INSERT INTO evolve_proposal (id, goal, target_file, kind, rationale, risk, status, model, created_at, pillar)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		  goal=excluded.goal, target_file=excluded.target_file, kind=excluded.kind,
		  rationale=excluded.rationale, risk=excluded.risk, model=excluded.model, pillar=excluded.pillar;
	`, p.ID, p.Goal, p.TargetFile, p.Kind, p.Rationale, p.Risk, p.Status, p.Model, p.CreatedAt, p.Pillar)
	return err
}

// GetEvolveProposal — ambil 1 usulan by id (buat engine eksekusi fase-2b: apply).
// Balikin (proposal, found, error). found=false kalau id ga ada (bukan error).
func (s *Store) GetEvolveProposal(id string) (EvolveProposal, bool, error) {
	var p EvolveProposal
	if err := s.ensureEvolveSchema(); err != nil {
		return p, false, err
	}
	row := s.db.QueryRow(`
		SELECT id, goal, target_file, kind, rationale, risk, status, model, created_at, pillar
		FROM evolve_proposal WHERE id=?`, id)
	err := row.Scan(&p.ID, &p.Goal, &p.TargetFile, &p.Kind, &p.Rationale, &p.Risk, &p.Status, &p.Model, &p.CreatedAt, &p.Pillar)
	if err == sql.ErrNoRows {
		return p, false, nil
	}
	if err != nil {
		return p, false, err
	}
	return p, true, nil
}

// SetEvolveProposalStatus — owner approve/reject/applied. Status divalidasi ke set kanonik
// (defensive — jangan biarin field status korup dari caller yg salah).
func (s *Store) SetEvolveProposalStatus(id, status string) error {
	switch status {
	case "proposed", "approved", "rejected", "applied", "staged":
	default:
		return fmt.Errorf("status invalid: %q (harus proposed|approved|rejected|applied|staged)", status)
	}
	if err := s.ensureEvolveSchema(); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE evolve_proposal SET status=? WHERE id=?`, status, id)
	return err
}

// DeleteEvolveProposal — hapus 1 usulan by id (owner buang dari backlog). Idempoten.
func (s *Store) DeleteEvolveProposal(id string) error {
	if err := s.ensureEvolveSchema(); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM evolve_proposal WHERE id=?`, id)
	return err
}

// DeleteEvolveProposalsByStatus — hapus semua usulan berstatus tertentu (mis. "rejected"
// buat bersih-bersih). Balik jumlah yg kehapus.
func (s *Store) DeleteEvolveProposalsByStatus(status string) (int64, error) {
	if err := s.ensureEvolveSchema(); err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`DELETE FROM evolve_proposal WHERE status=?`, status)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ActiveProposalTargets — set target_file usulan AKTIF (proposed|staged|approved) buat DEDUP
// di reflect (jangan bikin lagi yang udah ada). Map target_file→true.
func (s *Store) ActiveProposalTargets() (map[string]bool, error) {
	out := map[string]bool{}
	if err := s.ensureEvolveSchema(); err != nil {
		return out, err
	}
	rows, err := s.db.Query(`SELECT target_file FROM evolve_proposal WHERE status IN ('proposed','staged','approved') AND target_file != ''`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var tf string
		if rows.Scan(&tf) == nil {
			out[strings.ToLower(strings.TrimSpace(tf))] = true
		}
	}
	return out, rows.Err()
}

// ListEvolveProposals — backlog terbaru dulu (buat GUI + eksekusi fase-2).
func (s *Store) ListEvolveProposals(limit int) ([]map[string]any, error) {
	if err := s.ensureEvolveSchema(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT id, goal, target_file, kind, rationale, risk, status, model, created_at, pillar
		FROM evolve_proposal ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, goal, tf, kind, rat, risk, status, model, ca, pillar string
		if err := rows.Scan(&id, &goal, &tf, &kind, &rat, &risk, &status, &model, &ca, &pillar); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "goal": goal, "target_file": tf, "kind": kind, "rationale": rat,
			"risk": risk, "status": status, "model": model, "created_at": ca, "pillar": pillar,
		})
	}
	return out, rows.Err()
}

// EvolveStage — hasil core-apply yg nunggu review owner (diff sandbox + ringkas test-gate).
type EvolveStage struct {
	ID         string `json:"id"`
	ProposalID string `json:"proposal_id"`
	TargetFile string `json:"target_file"`
	Diff       string `json:"diff"`
	Content    string `json:"content"` // isi file utuh — buat commit-on-approve persis yg direview
	TestOutput string `json:"test_output"`
	Status     string `json:"status"` // staged | approved | rejected | committed
	Model      string `json:"model"`
	CreatedAt  string `json:"created_at"`
}

// AddEvolveStage — simpan 1 staged diff (id wajib unik; caller bikin).
func (s *Store) AddEvolveStage(st EvolveStage) error {
	if err := s.ensureEvolveSchema(); err != nil {
		return err
	}
	if st.CreatedAt == "" {
		st.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if st.Status == "" {
		st.Status = "staged"
	}
	_, err := s.db.Exec(`
		INSERT INTO evolve_stage (id, proposal_id, target_file, diff, content, test_output, status, model, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		  diff=excluded.diff, content=excluded.content, test_output=excluded.test_output, status=excluded.status;`,
		st.ID, st.ProposalID, st.TargetFile, st.Diff, st.Content, st.TestOutput, st.Status, st.Model, st.CreatedAt)
	return err
}

// SetEvolveStageStatus — owner approve/reject; engine commit. Status divalidasi.
func (s *Store) SetEvolveStageStatus(id, status string) error {
	switch status {
	case "staged", "approved", "rejected", "committed":
	default:
		return fmt.Errorf("stage status invalid: %q", status)
	}
	if err := s.ensureEvolveSchema(); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE evolve_stage SET status=? WHERE id=?`, status, id)
	return err
}

// GetEvolveStage — ambil 1 stage by id (buat commit/approve).
func (s *Store) GetEvolveStage(id string) (EvolveStage, bool, error) {
	var st EvolveStage
	if err := s.ensureEvolveSchema(); err != nil {
		return st, false, err
	}
	row := s.db.QueryRow(`
		SELECT id, proposal_id, target_file, diff, content, test_output, status, model, created_at
		FROM evolve_stage WHERE id=?`, id)
	err := row.Scan(&st.ID, &st.ProposalID, &st.TargetFile, &st.Diff, &st.Content, &st.TestOutput, &st.Status, &st.Model, &st.CreatedAt)
	if err == sql.ErrNoRows {
		return st, false, nil
	}
	if err != nil {
		return st, false, err
	}
	return st, true, nil
}

// ListEvolveStages — staged diff terbaru dulu (buat GUI review Milestone C).
func (s *Store) ListEvolveStages(limit int) ([]map[string]any, error) {
	if err := s.ensureEvolveSchema(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, proposal_id, target_file, diff, test_output, status, model, created_at
		FROM evolve_stage ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, pid, tf, diff, test, status, model, ca string
		if err := rows.Scan(&id, &pid, &tf, &diff, &test, &status, &model, &ca); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "proposal_id": pid, "target_file": tf, "diff": diff,
			"test_output": test, "status": status, "model": model, "created_at": ca,
		})
	}
	return out, rows.Err()
}
