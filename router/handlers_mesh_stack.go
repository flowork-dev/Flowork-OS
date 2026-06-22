// FROZEN brain-core — desain abadi Flowork. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/flow_router
// Locked at: 2026-05-30
// Reason: Section 14-23 phase 1 mesh stack stub endpoints. Schema siap
//   tapi single-owner no actual mesh. Phase 2 (real transport/gossip/
//   CRDT/knowledge/toolshare/karma/filter/LoRA/L3/daemon) → tambah
//   handler baru per section.
//
// handlers_mesh_stack.go — Section 14-23 phase 1 stub endpoints.

package main

import (
	"database/sql"
	"net/http"

	"github.com/flowork-os/flowork_Router/internal/store"
)

// MeshStackOverviewHandler — GET /api/mesh/stack/overview
// Return overview semua mesh tables (row counts) sebagai diagnostic.
func MeshStackOverviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	overview := map[string]int{}
	tables := []string{
		"mesh_identity", "mesh_peers",
		"mesh_packets",
		"mesh_gossip_state",
		"mesh_crdt_state",
		"mesh_knowledge_inbox",
		"mesh_tool_manifests",
		"mesh_peer_karma",
		"mesh_filter_audit",
		"mesh_lora_deltas",
		"mesh_l3_state",
		"mesh_daemon_status",
	}
	for _, t := range tables {
		overview[t] = countRows(db, t)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tables": overview,
		"note":   "phase 1 schema only — single-owner no real mesh traffic. Multi-host phase 2.",
	})
}

func countRows(db *sql.DB, table string) int {
	var n int
	// `table` selalu dari daftar tabel hardcoded (schema), bukan input user; nama tabel tak
	// bisa di-parameterize → concat wajar, aman dari SQL injection. // scanner:ignore
	_ = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
	return n
}
