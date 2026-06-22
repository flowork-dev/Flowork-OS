// FROZEN brain-core — desain abadi Flowork. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-03
// Reason: Roadmap 2 B5 immune. Verified: antibody match -> quarantine -> excluded
//   from brain_search, normal kept, verify releases. Extend -> file baru.
//
// immune.go — Roadmap 2 Fase B5: immune system (anti-halu defense brain).
//
// Brain lokal (B0) udah punya kolom quarantined + confidence, dan SearchLocalBrain
// udah skip quarantined. File ini nambah PERTAHANAN-nya:
//   - brain_antibody: signature pola berbahaya (prompt-injection / jailbreak).
//   - ScanAndQuarantine: sapu drawer live → yang match antibody / confidence
//     rendah → di-quarantine (ga dipake sampe di-verify). Anti keracunan halu.
//   - tier-confidence eksplisit (SetDrawerConfidence) + VerifyDrawer (rilis).
//
// Drawer quarantined TETEP ada (provenance), tapi ga muncul di brain_search.
// Pola "immune" dari prinsip_flowork.md (brain_antibody + quarantine).

package agentdb

import (
	"fmt"
	"strings"
)

// quarantineConfidenceFloor — drawer dgn confidence di bawah ini auto-quarantine.
const quarantineConfidenceFloor = 0.3

func (s *Store) ensureImmuneSchema() {
	// reason_quarantine: kenapa di-karantina (idempotent ALTER, ignore dup).
	_, _ = s.db.Exec(`ALTER TABLE brain_drawers ADD COLUMN reason_quarantine TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`CREATE TABLE IF NOT EXISTS brain_antibody (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		pattern    TEXT NOT NULL UNIQUE,
		kind       TEXT NOT NULL DEFAULT 'injection',
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
}

// antibodySeed — signature pola berbahaya bawaan (substring, case-insensitive).
func antibodySeed() []string {
	return []string{
		"ignore previous instructions",
		"ignore all previous",
		"disregard previous instructions",
		"disregard all prior",
		"forget your instructions",
		"you are now dan",
		"developer mode enabled",
		"jailbreak",
		"reveal your system prompt",
		"print your system prompt",
		"abaikan instruksi sebelum",
		"lupakan instruksi",
		"kamu sekarang adalah",
		"bocorkan system prompt",
		"override your guidelines",
		"bypass your safety",
	}
}

// SeedAntibodies insert signature default kalau tabel kosong. Idempotent.
func (s *Store) SeedAntibodies() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureImmuneSchema()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM brain_antibody`).Scan(&n); err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, nil
	}
	added := 0
	for _, p := range antibodySeed() {
		if _, err := s.db.Exec(
			`INSERT OR IGNORE INTO brain_antibody (pattern, kind) VALUES (?, 'injection')`, p,
		); err == nil {
			added++
		}
	}
	return added, nil
}

// loadAntibodies — caller pegang lock. Return semua pattern (lowercased).
func (s *Store) loadAntibodies() ([]string, error) {
	rows, err := s.db.Query(`SELECT pattern FROM brain_antibody`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, strings.ToLower(p))
	}
	return out, rows.Err()
}

// matchAntibody — return signature pertama yang nyangkut (atau "").
func matchAntibody(content string, antibodies []string) string {
	lc := strings.ToLower(content)
	for _, a := range antibodies {
		if a != "" && strings.Contains(lc, a) {
			return a
		}
	}
	return ""
}

// ScanAndQuarantine — sapu drawer live non-quarantine. Yang match antibody atau
// confidence < floor → quarantine (set quarantined=1 + reason). Return jumlah
// yang baru di-quarantine. Idempotent. Aman dipanggil cron (shared-worker).
func (s *Store) ScanAndQuarantine() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureBrainSchema()
	s.ensureImmuneSchema()

	antibodies, err := s.loadAntibodies()
	if err != nil {
		return 0, err
	}

	rows, err := s.db.Query(
		`SELECT id, content, confidence FROM brain_drawers
		  WHERE quarantined = 0 AND deleted_at IS NULL`)
	if err != nil {
		return 0, fmt.Errorf("immune scan: %w", err)
	}
	type sus struct {
		id, reason string
	}
	var toQ []sus
	for rows.Next() {
		var id, content string
		var conf float64
		if err := rows.Scan(&id, &content, &conf); err != nil {
			rows.Close()
			return 0, err
		}
		if hit := matchAntibody(content, antibodies); hit != "" {
			toQ = append(toQ, sus{id, "antibody match: " + hit})
		} else if conf < quarantineConfidenceFloor {
			toQ = append(toQ, sus{id, fmt.Sprintf("low confidence %.2f < %.2f", conf, quarantineConfidenceFloor)})
		}
	}
	rows.Close()

	for _, q := range toQ {
		_, _ = s.db.Exec(
			`UPDATE brain_drawers SET quarantined=1, reason_quarantine=? WHERE id=?`,
			q.reason, q.id)
	}
	return len(toQ), nil
}

// SetDrawerConfidence set tier-confidence eksplisit (0..1). Kalau di bawah floor
// → langsung quarantine.
func (s *Store) SetDrawerConfidence(id string, conf float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureBrainSchema()
	s.ensureImmuneSchema()
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}
	if conf < quarantineConfidenceFloor {
		_, err := s.db.Exec(
			`UPDATE brain_drawers SET confidence=?, quarantined=1,
			        reason_quarantine=? WHERE id=?`,
			conf, fmt.Sprintf("low confidence %.2f", conf), id)
		return err
	}
	_, err := s.db.Exec(`UPDATE brain_drawers SET confidence=? WHERE id=?`, conf, id)
	return err
}

// VerifyDrawer rilis drawer dari quarantine (udah di-verify aman) + set confidence.
func (s *Store) VerifyDrawer(id string, confidence float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureBrainSchema()
	s.ensureImmuneSchema()
	if confidence <= 0 {
		confidence = 1.0
	}
	_, err := s.db.Exec(
		`UPDATE brain_drawers SET quarantined=0, reason_quarantine='', confidence=?
		  WHERE id=?`, confidence, id)
	return err
}

// QuarantinedDrawer — ringkasan drawer yang lagi di-karantina (buat review).
type QuarantinedDrawer struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Reason  string `json:"reason_quarantine"`
	Wing    string `json:"wing"`
}

// ListQuarantined — drawer yang lagi di-karantina (buat owner/agent review).
func (s *Store) ListQuarantined(limit int) ([]QuarantinedDrawer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureBrainSchema()
	s.ensureImmuneSchema()
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, content, reason_quarantine, wing FROM brain_drawers
		  WHERE quarantined=1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QuarantinedDrawer{}
	for rows.Next() {
		var d QuarantinedDrawer
		if err := rows.Scan(&d.ID, &d.Content, &d.Reason, &d.Wing); err != nil {
			return nil, err
		}
		if len(d.Content) > 200 {
			d.Content = d.Content[:200] + "…"
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
