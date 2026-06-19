// cognitive_graph.go — Cognitive Graph Memory (CGM) lokal per-agent (di state.db).
//
// Roadmap: /home/mrflow/Documents/roadmap_opus8.md (§4.1, §4.2, D14, D16).
// Twin/personal graph hidup di LAPIS LOKAL (agent state.db) — privat, gak digossip
// ke mesh (D2). Graph = OVERLAY/INDEKS: node = gagang ringan (id/label/embedding +
// pointer balik via source_ref); konten asli tetap di tabel asalnya. Edge nyambungin
// LINTAS-jenis (Constitution/Memory/Persona/Instinct/Knowledge/Twin) karena node = ALAMAT (URN).
//
// Plug-and-play: file baru, bikin tabelnya sendiri (pola brain_drawers.go /
// constitution.go) — TIDAK modify agentdb.go yang LOCKED. Koneksi agentdb udah
// foreign_keys(on) (lihat agentdb.go DSN), jadi FK cascade aktif.
//
// Anti-halu: node/edge punya source_kind + confidence + status (active/quarantined/
// obsolete). Extractor & validation gate ada di file terpisah (cognitive_extract.go,
// cognitive_resolve.go) — JANGAN campur di sini (single responsibility).

package agentdb

import (
	"fmt"
	"strings"
	"time"
)

// CogNode — satu node di cognitive graph. id = URN stabil (mis.
// "agent:mr-flow/twin/aola", "router/instinct/verify-before-trust"). 5W1H jadi
// properti node; HOW sering jadi edge (lihat CogEdge).
type CogNode struct {
	ID          string  `json:"id"`           // URN: <scope>/<type>/<local_id>
	Label       string  `json:"label"`        // WHAT: nama kanonik
	Type        string  `json:"type"`         // person|concept|project|trait|event|skill|fact|preference|doctrine|persona|memory|knowledge
	Why         string  `json:"why"`          // WHY
	Who         string  `json:"who"`          // WHO (JSON array string)
	WhereDomain string  `json:"where_domain"` // WHERE (personal|teknis|bisnis|…)
	WhenValid   string  `json:"when_valid"`   // WHEN
	Properties  string  `json:"properties"`   // bebas (JSON)
	SourceKind  string  `json:"source_kind"`  // user_said|agent_inferred|verified|strong_model_unverified
	SourceRef   string  `json:"source_ref"`   // pointer balik (mis. interaction_123) / URN sumber
	Confidence  float64 `json:"confidence"`   // 0..1
	Status      string  `json:"status"`       // active|quarantined|obsolete|shadow
	Embedding   []byte  `json:"-"`            // vektor label (8-bit quantized) buat entity-resolution
	HitCount    int     `json:"hit_count"`    // diperkuat tiap re-observasi
	Version     int     `json:"version"`
}

// CogEdge — relasi berarah antar node. relation_type dari kosakata TETAP (§4.2).
type CogEdge struct {
	FromID       string  `json:"from_id"`
	ToID         string  `json:"to_id"`
	RelationType string  `json:"relation_type"`
	Strength     float64 `json:"strength"`    // diperkuat tiap re-observasi, decay tiap dream
	Confidence   float64 `json:"confidence"`  // 0..1
	SourceKind   string  `json:"source_kind"`
	SourceRef    string  `json:"source_ref"`
	Status       string  `json:"status"` // active|quarantined|obsolete|shadow
}

// Kosakata relasi TETAP (§4.2) — extractor cuma boleh pakai ini biar 26B gak ngarang.
// Tambah relasi = keputusan SADAR, bukan free-form LLM.
var ValidRelations = map[string]bool{
	// Fakta/dunia
	"is_a": true, "part_of": true, "created_by": true, "uses": true, "depends_on": true,
	"located_in": true, "happened_at": true, "causes": true, "has_property": true, "related_to": true,
	// Twin/owner
	"prefers": true, "dislikes": true, "communicates_in_style": true, "thinks_via": true,
	"values": true, "decides_by": true, "reacts_when": true, "goal_is": true,
	// Struktural (link antar-gudang, §4.12)
	"governed_by": true, "belongs_to": true, "about": true, "member_of": true,
	"taught": true, "references": true, "learned": true,
}

// IsValidRelation — true kalau relation_type ada di kosakata tetap.
func IsValidRelation(rel string) bool { return ValidRelations[strings.TrimSpace(rel)] }

const (
	maxCogLabelBytes = 512
	maxCogPropsBytes = 8 * 1024
)

// ensureCognitiveGraphSchema bikin tabel CGM (idempotent). Caller WAJIB sudah
// pegang s.mu. FK cascade aktif (DSN foreign_keys(on)).
func (s *Store) ensureCognitiveGraphSchema() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS cognitive_nodes (
			id                TEXT PRIMARY KEY,
			label             TEXT NOT NULL,
			type              TEXT NOT NULL,
			why               TEXT NOT NULL DEFAULT '',
			who               TEXT NOT NULL DEFAULT '[]',
			where_domain      TEXT NOT NULL DEFAULT '',
			when_valid        TEXT NOT NULL DEFAULT '',
			properties        TEXT NOT NULL DEFAULT '{}',
			source_kind       TEXT NOT NULL DEFAULT 'agent_inferred',
			source_ref        TEXT NOT NULL DEFAULT '',
			confidence        REAL NOT NULL DEFAULT 0.5,
			status            TEXT NOT NULL DEFAULT 'active',
			reason_quarantine TEXT NOT NULL DEFAULT '',
			embedding         BLOB,
			hit_count         INTEGER NOT NULL DEFAULT 1,
			version           INTEGER NOT NULL DEFAULT 1,
			created_at        TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cog_nodes_type   ON cognitive_nodes(type)`,
		`CREATE INDEX IF NOT EXISTS idx_cog_nodes_status ON cognitive_nodes(status)`,
		`CREATE TABLE IF NOT EXISTS cognitive_edges (
			from_id       TEXT NOT NULL,
			to_id         TEXT NOT NULL,
			relation_type TEXT NOT NULL,
			strength      REAL NOT NULL DEFAULT 1.0,
			confidence    REAL NOT NULL DEFAULT 0.5,
			source_kind   TEXT NOT NULL DEFAULT 'agent_inferred',
			source_ref    TEXT NOT NULL DEFAULT '',
			status        TEXT NOT NULL DEFAULT 'active',
			created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (from_id, to_id, relation_type),
			FOREIGN KEY (from_id) REFERENCES cognitive_nodes(id) ON DELETE CASCADE,
			FOREIGN KEY (to_id)   REFERENCES cognitive_nodes(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cog_edges_from ON cognitive_edges(from_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cog_edges_to   ON cognitive_edges(to_id)`,
		// jejak digestion (anti data-loss: mark, BUKAN delete interaction mentah)
		`CREATE TABLE IF NOT EXISTS cognitive_digest_log (
			interaction_id INTEGER PRIMARY KEY,
			digested_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			nodes_added    INTEGER NOT NULL DEFAULT 0,
			edges_added    INTEGER NOT NULL DEFAULT 0,
			status         TEXT NOT NULL DEFAULT 'ok'
		)`,
		// kontradiksi nunggu owner putusin ("tanya besok pagi")
		`CREATE TABLE IF NOT EXISTS cognitive_tension (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			from_id       TEXT NOT NULL DEFAULT '',
			relation_type TEXT NOT NULL DEFAULT '',
			old_to_id     TEXT NOT NULL DEFAULT '',
			new_to_id     TEXT NOT NULL DEFAULT '',
			detail        TEXT NOT NULL DEFAULT '',
			status        TEXT NOT NULL DEFAULT 'open',
			created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range stmts {
		_, _ = s.db.Exec(q)
	}
}

// UpsertNode insert/perkuat 1 node. Kalau id udah ada → update field + hit_count++
// + last_seen (reinforce). Return added=true kalau node baru. id WAJIB URN (caller/
// resolver yang nentuin — lihat cognitive_resolve.go).
func (s *Store) UpsertNode(n CogNode) (added bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()

	n.ID = strings.TrimSpace(n.ID)
	n.Label = strings.TrimSpace(n.Label)
	n.Type = strings.TrimSpace(n.Type)
	if n.ID == "" || n.Label == "" || n.Type == "" {
		return false, fmt.Errorf("node id/label/type wajib")
	}
	if len(n.Label) > maxCogLabelBytes {
		n.Label = n.Label[:maxCogLabelBytes]
	}
	if len(n.Properties) > maxCogPropsBytes {
		n.Properties = n.Properties[:maxCogPropsBytes]
	}
	n.Properties = defaultStr(n.Properties, "{}")
	n.Who = defaultStr(n.Who, "[]")
	n.SourceKind = defaultStr(n.SourceKind, "agent_inferred")
	n.Status = defaultStr(n.Status, "active")
	if n.Confidence == 0 {
		n.Confidence = 0.5
	}

	var exists int
	_ = s.db.QueryRow(`SELECT 1 FROM cognitive_nodes WHERE id=?`, n.ID).Scan(&exists)
	added = exists == 0

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		INSERT INTO cognitive_nodes
			(id, label, type, why, who, where_domain, when_valid, properties,
			 source_kind, source_ref, confidence, status, embedding, hit_count, version, last_seen_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,1,1,?)
		ON CONFLICT(id) DO UPDATE SET
			label        = excluded.label,
			type         = excluded.type,
			why          = excluded.why,
			who          = excluded.who,
			where_domain = excluded.where_domain,
			when_valid   = excluded.when_valid,
			properties   = excluded.properties,
			confidence   = MAX(cognitive_nodes.confidence, excluded.confidence),
			embedding    = COALESCE(excluded.embedding, cognitive_nodes.embedding),
			hit_count    = cognitive_nodes.hit_count + 1,
			last_seen_at = excluded.last_seen_at`,
		n.ID, n.Label, n.Type, n.Why, n.Who, n.WhereDomain, n.WhenValid, n.Properties,
		n.SourceKind, n.SourceRef, n.Confidence, n.Status, n.Embedding, now)
	if err != nil {
		return false, fmt.Errorf("upsert node %s: %w", n.ID, err)
	}
	return added, nil
}

// UpsertEdge insert/perkuat 1 edge. Pastikan node ref ada dulu (stub) biar FK ga
// gagal. relation_type WAJIB dari kosakata tetap (§4.2). On conflict → strength++
// (capped 10) + last_seen.
func (s *Store) UpsertEdge(e CogEdge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()

	e.FromID = strings.TrimSpace(e.FromID)
	e.ToID = strings.TrimSpace(e.ToID)
	e.RelationType = strings.TrimSpace(e.RelationType)
	if e.FromID == "" || e.ToID == "" || e.RelationType == "" {
		return fmt.Errorf("edge from/to/relation wajib")
	}
	if !ValidRelations[e.RelationType] {
		return fmt.Errorf("relation_type %q di luar kosakata tetap (§4.2)", e.RelationType)
	}
	e.SourceKind = defaultStr(e.SourceKind, "agent_inferred")
	e.Status = defaultStr(e.Status, "active")
	if e.Strength == 0 {
		e.Strength = 1.0
	}
	if e.Confidence == 0 {
		e.Confidence = 0.5
	}

	now := time.Now().UTC().Format(time.RFC3339)
	// stub node biar FK ga gagal (label = id sementara; nanti di-enrich resolver)
	for _, id := range []string{e.FromID, e.ToID} {
		_, _ = s.db.Exec(
			`INSERT OR IGNORE INTO cognitive_nodes (id, label, type) VALUES (?,?,'concept')`, id, id)
	}
	_, err := s.db.Exec(`
		INSERT INTO cognitive_edges
			(from_id, to_id, relation_type, strength, confidence, source_kind, source_ref, status, last_seen_at)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET
			strength     = MIN(cognitive_edges.strength + 1.0, 10.0),
			confidence   = MAX(cognitive_edges.confidence, excluded.confidence),
			status       = excluded.status,
			last_seen_at = excluded.last_seen_at`,
		e.FromID, e.ToID, e.RelationType, e.Strength, e.Confidence, e.SourceKind, e.SourceRef, e.Status, now)
	if err != nil {
		return fmt.Errorf("upsert edge %s-[%s]->%s: %w", e.FromID, e.RelationType, e.ToID, err)
	}
	return nil
}

// GetNode ambil 1 node by id (URN). Return ok=false kalau ga ada.
func (s *Store) GetNode(id string) (CogNode, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()

	var n CogNode
	err := s.db.QueryRow(`
		SELECT id, label, type, why, who, where_domain, when_valid, properties,
		       source_kind, source_ref, confidence, status, embedding, hit_count, version
		FROM cognitive_nodes WHERE id=?`, id).Scan(
		&n.ID, &n.Label, &n.Type, &n.Why, &n.Who, &n.WhereDomain, &n.WhenValid, &n.Properties,
		&n.SourceKind, &n.SourceRef, &n.Confidence, &n.Status, &n.Embedding, &n.HitCount, &n.Version)
	if err != nil {
		return CogNode{}, false, nil //nolint:nilerr // not-found = (false,nil)
	}
	return n, true, nil
}

// Neighbors ambil edge keluar+masuk dari sebuah node (1 hop). Buat traversal/recall.
func (s *Store) Neighbors(id string) (out []CogEdge, in []CogEdge, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()

	scan := func(q string, arg string) ([]CogEdge, error) {
		rows, e := s.db.Query(q, arg)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		var res []CogEdge
		for rows.Next() {
			var ed CogEdge
			if e := rows.Scan(&ed.FromID, &ed.ToID, &ed.RelationType, &ed.Strength, &ed.Confidence, &ed.Status); e != nil {
				return nil, e
			}
			res = append(res, ed)
		}
		return res, rows.Err()
	}
	const cols = `from_id, to_id, relation_type, strength, confidence, status`
	if out, err = scan(`SELECT `+cols+` FROM cognitive_edges WHERE from_id=? AND status='active'`, id); err != nil {
		return nil, nil, err
	}
	if in, err = scan(`SELECT `+cols+` FROM cognitive_edges WHERE to_id=? AND status='active'`, id); err != nil {
		return nil, nil, err
	}
	return out, in, nil
}

// GraphNodeView / GraphEdgeView — payload ringan buat GUI (tanpa embedding).
type GraphNodeView struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Type       string  `json:"type"`
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
	HitCount   int     `json:"hit_count"`
}

type GraphEdgeView struct {
	FromID       string  `json:"from_id"`
	ToID         string  `json:"to_id"`
	RelationType string  `json:"relation_type"`
	Status       string  `json:"status"`
	Strength     float64 `json:"strength"`
}

// ListCogNodes — semua node (buat viz graph). limit default 500.
func (s *Store) ListCogNodes(limit int) ([]GraphNodeView, error) {
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	rows, err := s.db.Query(
		`SELECT id, label, type, status, confidence, hit_count FROM cognitive_nodes
		 ORDER BY hit_count DESC, last_seen_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GraphNodeView
	for rows.Next() {
		var n GraphNodeView
		if err := rows.Scan(&n.ID, &n.Label, &n.Type, &n.Status, &n.Confidence, &n.HitCount); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ListCogEdges — semua edge (buat viz graph). limit default 1000.
func (s *Store) ListCogEdges(limit int) ([]GraphEdgeView, error) {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	rows, err := s.db.Query(
		`SELECT from_id, to_id, relation_type, status, strength FROM cognitive_edges
		 ORDER BY strength DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GraphEdgeView
	for rows.Next() {
		var e GraphEdgeView
		if err := rows.Scan(&e.FromID, &e.ToID, &e.RelationType, &e.Status, &e.Strength); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CountCognitiveGraph — jumlah node + edge live (buat stats/QC).
func (s *Store) CountCognitiveGraph() (nodes int, edges int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM cognitive_nodes`).Scan(&nodes)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM cognitive_edges`).Scan(&edges)
	return nodes, edges
}

func defaultStr(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
