// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval (autonomy grant 2026-06-19).
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-19
// Reason: CGM 2-tier digestion orchestration (no-delete, idempotent) — built + unit-tested (build/vet/test green). Extend = new file, jangan modify ini.
// Update 2026-06-21 (owner autonomy-grant "buka saja, lock lagi"): resolveNodeID + hook CO-REFERENCE
//   (resolveCanonicalIdentity di file baru cognitive_coref.go) SEBELUM ResolveByEmbedding — fix
//   fragmentasi identitas owner (bug ke-temu D21 benchmark). Additive, re-locked + build/vet/test green.
// Update 2026-06-21 (owner-approved, Phase 3E/D13 loop-belajar): DigestDeps +SourceKind +SourceRef
//   (override source_kind hasil ekstraksi, mis. 'strong_model_unverified' buat distil recording model
//   kuat). Diterapin di node+edge (skOf/srcRef). Additive, re-locked. Extend = file baru (learning_feed.go).
//
// cognitive_dream.go — Cognitive Digestion orchestration, 2-TIER (roadmap §4.6, D16).
//
// Nyatuin extract (§4.3) → resolve (§4.4) → gate (§4.5) → simpan ke graph. Anti
// data-loss: CATAT ke cognitive_digest_log, TIDAK PERNAH delete interaction mentah.
//
// 2-tier (D16):
//   - Tier 1 (light): item baru → status 'shadow' (gak ikut retrieval), kosongin buffer.
//   - Tier 2 (deep) : gate tentuin active/quarantined + promote shadow→active (repetisi).
//
// Layering bersih: LLM + embedding di-INJECT lewat func type (DigestDeps) — agentdb
// gak import routerclient. Caller (agentmgr/main) yang nyolokin routerclient.

package agentdb

import (
	"context"
	"fmt"
	"strings"
)

// EmbedFunc — caller nyediain embedding (router /v1/embeddings). Boleh nil →
// resolusi fallback ke label-exact (URN dari slug).
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

// LLMFunc — caller nyediain completion LLM (router). WAJIB ada buat ekstraksi.
type LLMFunc func(ctx context.Context, prompt string) (string, error)

// DigestDeps — dependency + opsi 1 pass digest.
type DigestDeps struct {
	LLM        LLMFunc   // wajib
	Embed      EmbedFunc // opsional (nil = fallback)
	AgentScope string    // prefix URN, mis. "agent:mr-flow"
	Tier       int       // 1 = light (→shadow), 2 = deep (gate + promote)
	// SourceKind (3E/D13 loop-belajar): kalau diisi, OVERRIDE source_kind hasil ekstraksi
	// (mis. "strong_model_unverified" buat distil dari recording model kuat). "" = pakai
	// source_kind dari extractor (default 'agent_inferred'/'user_said').
	SourceKind string
	// SourceRef: penanda asal (default "digest"). 3E pakai "learning" biar bisa di-trace/audit.
	SourceRef string
}

// DigestStats — hasil 1 pass (buat metrik QC).
type DigestStats struct {
	NodesAdded  int
	EdgesAdded  int
	Quarantined int
	Tensions    int
	Dropped     int
}

// DigestText jalanin 1 pass digest atas `text` (ringkasan/percakapan). Idempotensi
// ditangani di level interaction (DigestPendingInteractions), bukan di sini.
func (s *Store) DigestText(ctx context.Context, text string, dep DigestDeps) (DigestStats, error) {
	var st DigestStats
	if dep.LLM == nil {
		return st, fmt.Errorf("DigestDeps.LLM wajib")
	}
	if strings.TrimSpace(text) == "" {
		return st, nil
	}
	scope := dep.AgentScope
	if scope == "" {
		scope = "agent:local"
	}
	srcRef := dep.SourceRef
	if srcRef == "" {
		srcRef = "digest"
	}
	// 3E/D13: override source_kind (mis. strong_model_unverified) kalau di-set.
	skOf := func(extracted string) string {
		if dep.SourceKind != "" {
			return dep.SourceKind
		}
		return extracted
	}

	raw, err := dep.LLM(ctx, BuildExtractPrompt(text))
	if err != nil {
		return st, fmt.Errorf("llm extract: %w", err)
	}
	res, err := ParseExtraction(raw)
	if err != nil {
		return st, fmt.Errorf("parse: %w", err)
	}
	st.Dropped = len(res.Dropped)

	antibodies, _ := s.LoadAntibodyPatterns()

	// ── nodes: resolve-or-create + gate + store ──────────────────────────────
	labelToID := map[string]string{}
	for _, n := range res.Nodes {
		id, emb := s.resolveNodeID(ctx, dep, scope, n.Label, n.Type)
		status, reason := GateStatus(n.Label+" "+n.Why, n.Confidence, antibodies)
		if dep.Tier <= 1 && status == "active" {
			status = "shadow" // Tier-1: belum aktif sampai dikuatin Tier-2
		}
		node := CogNode{
			ID: id, Label: n.Label, Type: n.Type, Why: n.Why, Who: n.Who,
			WhereDomain: n.WhereDomain, WhenValid: n.WhenValid,
			SourceKind: skOf(n.SourceKind), SourceRef: srcRef, Confidence: n.Confidence,
			Status: status, Embedding: emb,
		}
		if status == "quarantined" {
			node.Status = "quarantined"
		}
		added, uerr := s.UpsertNode(node)
		if uerr != nil {
			continue
		}
		if status == "quarantined" {
			st.Quarantined++
			_ = s.setNodeQuarantineReason(id, reason)
		}
		if added {
			st.NodesAdded++
		}
		labelToID[strings.ToLower(n.Label)] = id
	}

	// ── edges: resolve endpoints + contradiction + store ─────────────────────
	for _, e := range res.Edges {
		fromID := s.edgeEndpointID(ctx, dep, scope, labelToID, e.FromLabel)
		toID := s.edgeEndpointID(ctx, dep, scope, labelToID, e.ToLabel)
		if fromID == "" || toID == "" {
			continue
		}
		if old, conflict := s.DetectEdgeContradiction(fromID, e.RelationType, toID); conflict {
			_ = s.RecordTension(fromID, e.RelationType, old, toID,
				fmt.Sprintf("digest: %s already -[%s]-> %s, now -> %s", fromID, e.RelationType, old, toID))
			st.Tensions++
			continue // jangan timpa diam-diam
		}
		status := "active"
		if dep.Tier <= 1 {
			status = "shadow"
		}
		if uerr := s.UpsertEdge(CogEdge{
			FromID: fromID, ToID: toID, RelationType: e.RelationType,
			Confidence: e.Confidence, SourceKind: skOf(e.SourceKind), SourceRef: srcRef, Status: status,
		}); uerr == nil {
			st.EdgesAdded++
		}
	}
	return st, nil
}

// resolveNodeID: entity-resolution by embedding kalau ada; else URN dari slug.
// Return (id, quantizedEmbedding-or-nil).
func (s *Store) resolveNodeID(ctx context.Context, dep DigestDeps, scope, label, typ string) (string, []byte) {
	var q []byte
	// CO-REFERENCE (anti-fragmentasi identitas, §4.4 follow-on / fix bug D21):
	// alias identitas EKSPLISIT (mis. "User"/"saya"/"aku" → owner) di-resolve DULUAN secara
	// deterministik — embedding GAK nangkep co-reference (nama beda, entitas sama). Lihat
	// cognitive_coref.go. Cuma berlaku type person + alias terdaftar + canonical masih active.
	if id, ok := s.resolveCanonicalIdentity(scope, label, typ); ok {
		if dep.Embed != nil {
			if vec, err := dep.Embed(ctx, label); err == nil {
				q = Quantize(vec)
			}
		}
		return id, q
	}
	if dep.Embed != nil {
		if vec, err := dep.Embed(ctx, label); err == nil {
			q = Quantize(vec)
			if id, _, found := s.ResolveByEmbedding(typ, q, 0); found {
				return id, q // merge ke node existing (anti kembar)
			}
		}
	}
	return scope + "/" + typ + "/" + slug(label), q
}

// edgeEndpointID: pakai id dari node yang barusan di-extract; kalau label gak ada di
// batch, resolve-or-create node minimal (type concept) biar edge punya endpoint.
func (s *Store) edgeEndpointID(ctx context.Context, dep DigestDeps, scope string, labelToID map[string]string, label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	if id, ok := labelToID[strings.ToLower(label)]; ok {
		return id
	}
	id, emb := s.resolveNodeID(ctx, dep, scope, label, "concept")
	_, _ = s.UpsertNode(CogNode{ID: id, Label: label, Type: "concept", Status: "shadow", SourceRef: "digest-edge", Embedding: emb})
	labelToID[strings.ToLower(label)] = id
	return id
}

// setNodeQuarantineReason — tulis alasan karantina (UpsertNode udah set status).
func (s *Store) setNodeQuarantineReason(id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE cognitive_nodes SET reason_quarantine=? WHERE id=?`, reason, id)
	return err
}

// DigestPendingInteractions — ambil interactions yg belum di-digest, cerna, lalu
// CATAT ke digest_log (BUKAN delete). limit = max interaction per pass.
func (s *Store) DigestPendingInteractions(ctx context.Context, dep DigestDeps, limit int) (DigestStats, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	type row struct {
		id      int64
		content string
	}
	s.mu.Lock()
	s.ensureCognitiveGraphSchema() // tabel digest_log mungkin belum ada (query duluan)
	rows, err := s.db.Query(`
		SELECT i.id, i.content FROM interactions i
		LEFT JOIN cognitive_digest_log d ON d.interaction_id = i.id
		WHERE d.interaction_id IS NULL AND i.deleted_at IS NULL AND TRIM(i.content) <> ''
		ORDER BY i.id ASC LIMIT ?`, limit)
	if err != nil {
		s.mu.Unlock()
		return DigestStats{}, 0, err
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.content); err == nil {
			pending = append(pending, r)
		}
	}
	rows.Close()
	s.mu.Unlock()

	var total DigestStats
	var b strings.Builder
	var ids []int64
	for _, r := range pending {
		b.WriteString(r.content)
		b.WriteString("\n")
		ids = append(ids, r.id)
	}
	if len(pending) == 0 {
		return total, 0, nil
	}

	st, derr := s.DigestText(ctx, b.String(), dep)
	if derr != nil {
		return total, 0, derr
	}
	total = st

	// CATAT digest_log per interaction (idempoten, NO delete interaction).
	s.mu.Lock()
	for _, id := range ids {
		_, _ = s.db.Exec(
			`INSERT OR IGNORE INTO cognitive_digest_log (interaction_id, nodes_added, edges_added, status)
			 VALUES (?,?,?, 'ok')`, id, st.NodesAdded, st.EdgesAdded)
	}
	s.mu.Unlock()
	return total, len(ids), nil
}

// PromoteShadows (Tier-2): node/edge 'shadow' yang udah dikuatin (hit_count/strength
// >= minHits) → 'active'. Repetisi = sinyal kualitas (D13/D16). Return jumlah promoted.
func (s *Store) PromoteShadows(minHits int) (int, error) {
	if minHits < 2 {
		minHits = 2
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	rn, _ := s.db.Exec(`UPDATE cognitive_nodes SET status='active' WHERE status='shadow' AND hit_count>=?`, minHits)
	re, _ := s.db.Exec(`UPDATE cognitive_edges SET status='active' WHERE status='shadow' AND strength>=?`, float64(minHits))
	n, e := int64(0), int64(0)
	if rn != nil {
		n, _ = rn.RowsAffected()
	}
	if re != nil {
		e, _ = re.RowsAffected()
	}
	return int(n + e), nil
}

// slug — label → id-segment aman (lowercase, alnum + '-').
func slug(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	var b strings.Builder
	prevDash := false
	for _, r := range label {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "x"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
