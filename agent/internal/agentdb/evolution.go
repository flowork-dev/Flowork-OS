// evolution.go — R7 SELF-EVOLUTION fase-1 (plug-in, additive). Owner-approved 2026-06-15.
// Backlog usulan evolusi: organisme refleksi diri (baca self-map R6) → usulin perbaikan
// konkret → SIMPAN di sini buat review/eksekusi. FASE-1 = usulan doang (NOL ubah kode);
// eksekusi (sandbox→apply→auto-commit) = fase-2 di-GATE karma. Tabel terpisah, non-destruktif.

package agentdb

import "time"

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
	return err
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
		INSERT INTO evolve_proposal (id, goal, target_file, kind, rationale, risk, status, model, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		  goal=excluded.goal, target_file=excluded.target_file, kind=excluded.kind,
		  rationale=excluded.rationale, risk=excluded.risk, model=excluded.model;
	`, p.ID, p.Goal, p.TargetFile, p.Kind, p.Rationale, p.Risk, p.Status, p.Model, p.CreatedAt)
	return err
}

// SetEvolveProposalStatus — owner approve/reject/applied.
func (s *Store) SetEvolveProposalStatus(id, status string) error {
	if err := s.ensureEvolveSchema(); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE evolve_proposal SET status=? WHERE id=?`, status, id)
	return err
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
		SELECT id, goal, target_file, kind, rationale, risk, status, model, created_at
		FROM evolve_proposal ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, goal, tf, kind, rat, risk, status, model, ca string
		if err := rows.Scan(&id, &goal, &tf, &kind, &rat, &risk, &status, &model, &ca); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "goal": goal, "target_file": tf, "kind": kind, "rationale": rat,
			"risk": risk, "status": status, "model": model, "created_at": ca,
		})
	}
	return out, rows.Err()
}
