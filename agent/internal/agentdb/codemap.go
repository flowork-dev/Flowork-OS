// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Section 27 phase 1 codemap schema. Lazy CREATE. Phase 2 (edges
//   table + index_runs + AST parsers) → tambah file baru.
//
// codemap.go — Section 27 phase 1: codemap_nodes minimal.

package agentdb

import (
	"time"
)

type CodemapNode struct {
	ID           int64  `json:"id"`
	NodeType     string `json:"node_type"`
	Name         string `json:"name"`
	FilePath     string `json:"file_path"`
	LineStart    int    `json:"line_start"`
	LineEnd      int    `json:"line_end"`
	Layer        string `json:"layer"`
	Signature    string `json:"signature"`
	Docstring    string `json:"docstring"`
	SizeLOC      int    `json:"size_loc"`
	Complexity   int    `json:"complexity"`
	LastModified string `json:"last_modified"`
	IndexedAt    string `json:"indexed_at"`
}

func (s *Store) ensureCodemapSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS codemap_nodes (
		  id            INTEGER PRIMARY KEY AUTOINCREMENT,
		  node_type     TEXT NOT NULL,
		  name          TEXT NOT NULL,
		  file_path     TEXT NOT NULL,
		  line_start    INTEGER NOT NULL DEFAULT 0,
		  line_end      INTEGER NOT NULL DEFAULT 0,
		  layer         TEXT NOT NULL DEFAULT '',
		  signature     TEXT NOT NULL DEFAULT '',
		  docstring     TEXT NOT NULL DEFAULT '',
		  size_loc      INTEGER NOT NULL DEFAULT 0,
		  complexity    INTEGER NOT NULL DEFAULT 0,
		  last_modified TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  indexed_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_codemap_nodes_file ON codemap_nodes(file_path);
		CREATE INDEX IF NOT EXISTS idx_codemap_nodes_type ON codemap_nodes(node_type);
		CREATE INDEX IF NOT EXISTS idx_codemap_nodes_layer ON codemap_nodes(layer);
		CREATE INDEX IF NOT EXISTS idx_codemap_nodes_name ON codemap_nodes(name);
	`)
	return err
}

func (s *Store) UpsertCodemapNode(n CodemapNode) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureCodemapSchema(); err != nil {
		return 0, err
	}
	if n.IndexedAt == "" {
		n.IndexedAt = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := s.db.Exec(
		`INSERT INTO codemap_nodes (node_type, name, file_path,
		   line_start, line_end, layer, signature, docstring,
		   size_loc, complexity, last_modified, indexed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.NodeType, n.Name, n.FilePath,
		n.LineStart, n.LineEnd, n.Layer, n.Signature, n.Docstring,
		n.SizeLOC, n.Complexity, n.LastModified, n.IndexedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListCodemapNodes(nodeType, layer, search string, limit int) ([]CodemapNode, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureCodemapSchema(); err != nil {
		return nil, err
	}
	query := `SELECT id, node_type, name, file_path,
	                 line_start, line_end, layer, signature, docstring,
	                 size_loc, complexity, last_modified, indexed_at
	          FROM codemap_nodes WHERE 1=1`
	args := []any{}
	if nodeType != "" {
		query += ` AND node_type = ?`
		args = append(args, nodeType)
	}
	if layer != "" {
		query += ` AND layer = ?`
		args = append(args, layer)
	}
	if search != "" {
		query += ` AND (name LIKE ? OR file_path LIKE ?)`
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CodemapNode{}
	for rows.Next() {
		var n CodemapNode
		if serr := rows.Scan(&n.ID, &n.NodeType, &n.Name, &n.FilePath,
			&n.LineStart, &n.LineEnd, &n.Layer, &n.Signature, &n.Docstring,
			&n.SizeLOC, &n.Complexity, &n.LastModified, &n.IndexedAt); serr != nil {
			return nil, serr
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// LOCKED (soft, owner-approved 2026-06-20 codemap-fix): blok stabil, jangan ubah tanpa izin.
// CodemapNodeStats — agregat akurat (COUNT/GROUP BY), bukan sampel ber-LIMIT.
// Dipakai codemap_stats biar total_nodes bener walau >1000 (ListCodemapNodes
// hard-cap di 1000 buat anti over-prompt; count ga boleh ikut ke-cap).
func (s *Store) CodemapNodeStats() (total int, byType, byLayer map[string]int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = s.ensureCodemapSchema(); err != nil {
		return 0, nil, nil, err
	}
	byType, byLayer = map[string]int{}, map[string]int{}
	if e := s.db.QueryRow(`SELECT COUNT(*) FROM codemap_nodes`).Scan(&total); e != nil {
		return 0, nil, nil, e
	}
	scan := func(q string, dst map[string]int) error {
		rows, e := s.db.Query(q)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var c int
			if e := rows.Scan(&k, &c); e != nil {
				return e
			}
			dst[k] = c
		}
		return rows.Err()
	}
	if e := scan(`SELECT node_type, COUNT(*) FROM codemap_nodes GROUP BY node_type`, byType); e != nil {
		return 0, nil, nil, e
	}
	if e := scan(`SELECT layer, COUNT(*) FROM codemap_nodes GROUP BY layer`, byLayer); e != nil {
		return 0, nil, nil, e
	}
	return total, byType, byLayer, nil
}

func (s *Store) DeleteCodemapNodesByFile(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureCodemapSchema(); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM codemap_nodes WHERE file_path = ?`, filePath)
	return err
}
