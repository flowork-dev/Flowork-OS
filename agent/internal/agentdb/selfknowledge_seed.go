// selfknowledge_seed.go — SELF-KNOWLEDGE SEED (NON-frozen DNA sibling, 2026-07-03).
//
// Kenapa ada: tiap agent (mr-flow, team coder, specialist, agent masa depan) harus
// TAHU FITUR/SUBSISTEM Flowork pada dirinya sendiri — biar pas ditanya "kamu bisa apa",
// "gimana fitur X kerja", dia RECALL fakta dari arsitektur resmi, BUKAN ngarang.
//
// Kartu-kartu ini distilasi ringkas dari lock/*.md (dok arsitektur kanonik), di-seed ke
// brain LOKAL tiap agent (FTS5-recall via brain_search). Router shared brain TIDAK dipakai
// buat ini: recall-nya semantic-vindex (statik, rebuild-only atas 5jt drawer) → kartu baru
// ga ke-embed + ketutup korpus CVE. Brain lokal (korpus kecil, FTS5) = recall kebukti jalan.
//
// Kelasnya SAMA dengan konstitusi/edu-error/antibody: DNA statik yang di-seed per-agent
// (bukan collective-brain yang tumbuh). Idempotent: cuma seed kalau room masih kosong.
package agentdb

import (
	_ "embed"
	"encoding/json"
)

//go:embed selfknowledge_seed.json
var selfKnowledgeSeedJSON []byte

// SelfKnowledgeRoom — room brain lokal tempat kartu fitur tinggal. brain_search me-recall
// dari sini; insting refleks (router) nyuruh agent brain_search room ini pas ditanya soal diri.
const SelfKnowledgeRoom = "flowork_selfknowledge"

type selfKnowledgeCard struct {
	Content string `json:"content"`
	Source  string `json:"source"`
}

type selfKnowledgeSeedFile struct {
	Cards []selfKnowledgeCard `json:"cards"`
}

// SeedSelfKnowledge — isi brain lokal agent dgn kartu fitur Flowork (embedded). Balik
// (added, error). Idempotent: no-op kalau room SelfKnowledgeRoom udah keisi (hormati kondisi
// agent + hemat boot). AddBrainDrawer dedup by content_hash → aman walau ke-panggil ulang.
func (s *Store) SeedSelfKnowledge() (int, error) {
	// Guard idempoten murah (lock singkat, JANGAN tahan lock pas AddBrainDrawer — dia
	// ambil s.mu sendiri, bisa deadlock).
	var existing int
	s.mu.Lock()
	s.ensureBrainSchema()
	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM brain_drawers WHERE room = ? AND (deleted_at IS NULL OR deleted_at = '')`,
		SelfKnowledgeRoom,
	).Scan(&existing)
	s.mu.Unlock()
	if existing > 0 {
		return 0, nil
	}

	var seed selfKnowledgeSeedFile
	if err := json.Unmarshal(selfKnowledgeSeedJSON, &seed); err != nil {
		return 0, err
	}
	added := 0
	for _, c := range seed.Cards {
		if _, ok, err := s.AddBrainDrawer(c.Content, "knowledge", SelfKnowledgeRoom, "reference", "seed:selfknowledge"); err == nil && ok {
			added++
		}
	}
	return added, nil
}
