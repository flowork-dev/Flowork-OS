package agentdb

// learning_log.go — Phase 3E / D13 LOOP BELAJAR (owner-approved 2026-06-21). Anti-replay:
// catat recording (dari router) yg UDAH di-distil ke graph → gak di-proses 2x. Pola sama
// cognitive_digest_log. File BARU (extend, gak nyentuh locked). Idempoten.

// ensureLearningLogSchema — tabel anti-replay recordings.
func (s *Store) ensureLearningLogSchema() {
	_, _ = s.db.Exec(`CREATE TABLE IF NOT EXISTS learning_record_log (
		recording_id INTEGER PRIMARY KEY,
		processed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
}

// LearningRecordSeen — true kalau recording id sudah pernah di-distil.
func (s *Store) LearningRecordSeen(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLearningLogSchema()
	var x int
	return s.db.QueryRow(`SELECT 1 FROM learning_record_log WHERE recording_id=?`, id).Scan(&x) == nil
}

// MarkLearningRecord — tandai recording id udah di-proses (idempoten).
func (s *Store) MarkLearningRecord(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLearningLogSchema()
	_, err := s.db.Exec(`INSERT OR IGNORE INTO learning_record_log(recording_id) VALUES(?)`, id)
	return err
}

// CountLearningProcessed — jumlah recording yg udah di-distil (buat stats/QC).
func (s *Store) CountLearningProcessed() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLearningLogSchema()
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM learning_record_log`).Scan(&n)
	return n
}
