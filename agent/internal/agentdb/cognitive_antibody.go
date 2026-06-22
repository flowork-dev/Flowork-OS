// Owner: Mr.Dev · github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN brain-core — jangan edit tanpa unfreeze owner. Arsitektur & alasan: lihat lock/brain.md
//
// cognitive_antibody.go — F4 brain-pathway: ANTIBODY KOLEKTIF (lintas-agent).
//
// AKAR (roadmap F4): recovery-instinct (INC-3) yg DITEMUKAN INDEPENDEN oleh ≥N agent
// = pola yang BENERAN umum/terbukti → naikin jadi "antibody kolektif": pastikan SEMUA
// agent punya (push ke yg belum) + tandai collective (confidence tinggi). Jadi sekali
// 1 agent kebal error, lama-lama SEMUA agent kebal — imunitas kolektif beneran.
//
// Identitas pakai KELAS-error (recov-<class>, deterministik dari INC-3) → konvergensi
// lintas-agent ke-detect walau tiap agent punya node-id ber-scope sendiri. Privacy-safe
// by-construction (recovery-instinct INC-3 udah 0 data owner). Dormant pas cuma 1 agent
// yg punya recovery (ga ada konvergensi) → 0 dampak; aktif pas banyak agent belajar.

package agentdb

import "strings"

const recovIDMarker = "/instinct/recov-"

// RecoveryClassOf — ekstrak kelas-error dari node-id recovery-instinct
// ("agent:<x>/instinct/recov-<class>" → "<class>"). "" kalau bukan format itu (mis.
// recovery hash-id non-class, atau node lain).
func RecoveryClassOf(id string) string {
	i := strings.Index(id, recovIDMarker)
	if i < 0 {
		return ""
	}
	cls := id[i+len(recovIDMarker):]
	// recov-<class> (class = whitelist INC-2 kayak not-found/timeout). recov<hex> (fallback
	// non-class) ga punya "-" jadi ke-skip (bukan kelas konvergen). Ambil yg bersih aja.
	if cls == "" || strings.ContainsAny(cls, "/ ") {
		return ""
	}
	return cls
}

// ClassLabel — recovery-instinct ringkas buat agregasi antibody.
type ClassLabel struct {
	Class      string
	Label      string
	Confidence float64
}

// ActiveRecoveryClasses — recovery-instinct AKTIF yg punya kelas (recov-<class>) di agent ini.
func (s *Store) ActiveRecoveryClasses() []ClassLabel {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	rows, err := s.db.Query(
		`SELECT id, label, confidence FROM cognitive_nodes
		 WHERE type='instinct' AND where_domain='recovery' AND status='active'`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []ClassLabel{}
	for rows.Next() {
		var id, label string
		var conf float64
		if rows.Scan(&id, &label, &conf) != nil {
			continue
		}
		if cls := RecoveryClassOf(id); cls != "" {
			out = append(out, ClassLabel{Class: cls, Label: label, Confidence: conf})
		}
	}
	return out
}

// HasActiveRecoveryClass — true kalau agent ini udah punya recovery-instinct AKTIF kelas itu.
func (s *Store) HasActiveRecoveryClass(scope, class string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	var n int
	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM cognitive_nodes
		 WHERE id=? AND type='instinct' AND where_domain='recovery' AND status='active'`,
		scope+"/instinct/recov-"+class).Scan(&n)
	return n > 0
}

// UpsertCollectiveAntibody — push antibody kolektif ke agent ini (id ber-scope agent itu).
// status=active langsung (udah terbukti lintas-agent = ga perlu gerbang shadow lagi),
// confidence tinggi, properties.collective=true. emb buat recall by-makna.
func (s *Store) UpsertCollectiveAntibody(scope, class, label string, emb []byte) (bool, error) {
	return s.UpsertNode(CogNode{
		ID:          scope + "/instinct/recov-" + class,
		Label:       label,
		Type:        "instinct",
		WhereDomain: "recovery",
		Why:         "collective antibody (lintas-agent)",
		Properties:  `{"collective":true,"how":"recover"}`,
		SourceKind:  "verified",
		SourceRef:   "antibody-collective",
		Confidence:  0.95,
		Status:      "active",
		Embedding:   emb,
	})
}
