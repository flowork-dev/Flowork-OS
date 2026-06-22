// FROZEN brain-core — desain abadi Flowork. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-20 (auto-compact, FATAL-safe).
// LOCKED ≠ FREEZE (boleh diedit dgn izin owner). JANGAN ubah TrimDigestedInteractions tanpa izin:
// salah = pengalaman ke-buang sebelum masuk brain. Safety teruji (TestAutoCompactSafety).
package agentdb

// compact.go — AUTO-COMPACT konteks (owner 2026-06-20: "kalau konteks panjang, semua agent
// otomatis compact + masukin pengalaman ke brain kayak dream; FATAL jika salah").
//
// Mekanisme aman (urutan TIDAK boleh kebalik):
//   1. CompactStats — ukur konteks hidup (interaksi non-deleted) + brp yg belum di-digest.
//   2. (caller) DIGEST dulu interaksi pending → cognitive graph (brain). Verify.
//   3. TrimDigestedInteractions — soft-delete HANYA interaksi yg UDAH di-digest (ada di
//      cognitive_digest_log) + di luar N-terbaru. Pengalaman ga hilang (udah di brain),
//      cuma dipindah laci; soft-delete = recoverable (retention hard-delete baru >90 hari).
//
// KENAPA AMAN: trim cuma nyentuh yg di cognitive_digest_log → mustahil ngebuang interaksi yg
// belum jadi pengalaman di brain. Kalau digest GAGAL, caller JANGAN panggil trim → no loss.

// CompactStats — ukur konteks: live = interaksi non-deleted (yg masuk konteks/recall),
// undigested = belum ke brain, chars = total ukuran (proxy token), lastOccurred = aktivitas terakhir.
func (s *Store) CompactStats() (live, undigested int, chars int64, lastOccurred string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	if err = s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(LENGTH(content)),0), COALESCE(MAX(occurred_at),'')
		 FROM interactions WHERE deleted_at IS NULL`).Scan(&live, &chars, &lastOccurred); err != nil {
		return
	}
	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM interactions i
		 LEFT JOIN cognitive_digest_log d ON d.interaction_id = i.id
		 WHERE d.interaction_id IS NULL AND i.deleted_at IS NULL AND TRIM(i.content) <> ''`).Scan(&undigested)
	return
}

// TrimDigestedInteractions — soft-delete interaksi yg UDAH di-digest (di cognitive_digest_log)
// SELAIN N-terbaru. Return jumlah yg di-trim. AMAN: yg belum di-digest TIDAK pernah disentuh.
func (s *Store) TrimDigestedInteractions(keepRecent int) (int64, error) {
	if keepRecent < 0 {
		keepRecent = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	res, err := s.db.Exec(`
		UPDATE interactions SET deleted_at = CURRENT_TIMESTAMP
		WHERE deleted_at IS NULL
		  AND id IN (SELECT interaction_id FROM cognitive_digest_log)
		  AND id < COALESCE(
		      (SELECT MIN(id) FROM (SELECT id FROM interactions WHERE deleted_at IS NULL ORDER BY id DESC LIMIT ?)),
		      9223372036854775807)`, keepRecent)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
