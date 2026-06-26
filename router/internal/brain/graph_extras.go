// graph_extras.go — DreamGraph projeksi TAMBAHAN (NON-frozen seam, Rule 7).
// Di atas syncCoreEntitiesToGraph (constitution/persona/skill/agent), tambah:
//   - INSTINCTS (drawers room instinct_*) → node type "instinct"
//   - KNOWLEDGE corpus → hub per-WING (BUKAN node-per-drawer; 860k = meledak)
// Semua spoke nyambung ke hub "flowork" (anti orphan). Idempotent (cleanup source dulu).
// MIRROR-only: gak hapus sumber. Switch GUI (kebenaran di GUI):
//   FLOWORK_DREAMGRAPH_INSTINCTS (default on) · FLOWORK_DREAMGRAPH_KNOWLEDGE (default on).
package brain

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func extraSwitchOn(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "0", "false", "off", "no":
		return false
	}
	return true // default ON
}

func clip(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// syncInstinctsToGraph — instinct drawers → node "instinct" + edge member_of→flowork. Idempotent.
func syncInstinctsToGraph(ctx context.Context, tx *sql.Tx) (int, error) {
	if _, err := tx.ExecContext(ctx, "DELETE FROM cognitive_edges WHERE from_id LIKE 'instinct_%'"); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM cognitive_nodes WHERE source = 'system_instinct'"); err != nil {
		return 0, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id, content FROM drawers WHERE room LIKE 'instinct_%' AND deleted_at IS NULL")
	if err != nil {
		return 0, err
	}
	type di struct{ id, content string }
	var items []di
	for rows.Next() {
		var d di
		if rows.Scan(&d.id, &d.content) == nil && d.id != "" {
			items = append(items, d)
		}
	}
	rows.Close()
	n := 0
	for _, d := range items {
		nodeID := "instinct_" + d.id
		props := `{"kind":"instinct"}`
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cognitive_nodes (id, label, type, properties, source)
			VALUES (?, ?, 'instinct', ?, 'system_instinct')
			ON CONFLICT(id) DO UPDATE SET label=excluded.label, type=excluded.type,
			  properties=excluded.properties, source=excluded.source, last_accessed=CURRENT_TIMESTAMP`,
			nodeID, "Instinct: "+clip(d.content, 60), props); err != nil {
			return n, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cognitive_edges (from_id, to_id, relation_type, strength)
			VALUES (?, 'flowork', 'member_of', 1.0)
			ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET strength=1.0`, nodeID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// syncKnowledgeHubsToGraph — wing corpus → 1 hub node per-wing + edge part_of→flowork. Idempotent.
// HUB doang (bukan node-per-drawer) biar 860k drawer gak meledakin graph.
func syncKnowledgeHubsToGraph(ctx context.Context, tx *sql.Tx) (int, error) {
	if _, err := tx.ExecContext(ctx, "DELETE FROM cognitive_edges WHERE from_id LIKE 'knowledge_wing_%'"); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM cognitive_nodes WHERE source = 'system_knowledge'"); err != nil {
		return 0, err
	}
	// wing yang BUKAN bagian sistem-graph sendiri (skip training_data cognitive_graph mirror, dst).
	rows, err := tx.QueryContext(ctx, `
		SELECT wing, COUNT(*) FROM drawers
		WHERE deleted_at IS NULL AND wing <> '' AND room NOT LIKE 'cognitive_graph%'
		GROUP BY wing`)
	if err != nil {
		return 0, err
	}
	type wc struct {
		wing string
		cnt  int
	}
	var wings []wc
	for rows.Next() {
		var w wc
		if rows.Scan(&w.wing, &w.cnt) == nil && w.wing != "" {
			wings = append(wings, w)
		}
	}
	rows.Close()
	n := 0
	for _, w := range wings {
		nodeID := "knowledge_wing_" + w.wing
		props := fmt.Sprintf(`{"drawers":%d}`, w.cnt)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cognitive_nodes (id, label, type, properties, source)
			VALUES (?, ?, 'knowledge', ?, 'system_knowledge')
			ON CONFLICT(id) DO UPDATE SET label=excluded.label, type=excluded.type,
			  properties=excluded.properties, source=excluded.source, last_accessed=CURRENT_TIMESTAMP`,
			nodeID, fmt.Sprintf("Knowledge: %s (%d)", w.wing, w.cnt), props); err != nil {
			return n, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cognitive_edges (from_id, to_id, relation_type, strength)
			VALUES (?, 'flowork', 'part_of', 1.0)
			ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET strength=1.0`, nodeID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// SyncGraphExtended — core entities + (switch) instincts + (switch) knowledge-hubs → graph → RAG mirror.
// Dipanggil dari dreamgraph autosync (router). Idempotent, mirror-only (gak hapus sumber/memory).
func SyncGraphExtended(ctx context.Context) error {
	db, err := OpenRW()
	if err != nil {
		return fmt.Errorf("open rw: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := syncCoreEntitiesToGraph(ctx, tx); err != nil {
		return fmt.Errorf("core: %w", err)
	}
	if extraSwitchOn("FLOWORK_DREAMGRAPH_INSTINCTS") {
		if _, err := syncInstinctsToGraph(ctx, tx); err != nil {
			return fmt.Errorf("instincts: %w", err)
		}
	}
	if extraSwitchOn("FLOWORK_DREAMGRAPH_KNOWLEDGE") {
		if _, err := syncKnowledgeHubsToGraph(ctx, tx); err != nil {
			return fmt.Errorf("knowledge: %w", err)
		}
	}
	if err := syncGraphToRAGTx(ctx, tx); err != nil {
		return fmt.Errorf("rag: %w", err)
	}
	return tx.Commit()
}
