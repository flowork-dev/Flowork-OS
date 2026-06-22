// FROZEN brain-core — desain abadi Flowork. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/flow_router
// Locked at: 2026-05-30
// Reason: Section 15-22 phase 2 endpoints — gossip status, CRDT upsert/
//   query, knowledge ingest+promote, tool manifest broadcast/find,
//   karma list/decay, filter test, LoRA delta upload, L3 kv. Phase 3
//   (proper RPC binding) → tambah file baru.
//
// 2026-06-11 (owner-approved security audit, unfreeze→refreeze): MeshCRDTHandler
//   POST is now loopback-only (mirrors MeshKnowledgeHandler) — raw CRDT upsert
//   bypasses the signed packet path, so a remote caller could overwrite node
//   state with a large counter. GET stays open.
//
// handlers_mesh_advanced.go — Section 15-22 phase 2 admin endpoints.

package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/mesh"
	"github.com/flowork-os/flowork_Router/internal/store"
)

// =============================================================================
// Section 16 CRDT
// =============================================================================

// MeshCRDTHandler — GET ?topic= → list entries; POST → upsert.
func MeshCRDTHandler(w http.ResponseWriter, r *http.Request) {
	// POST upsert writes raw CRDT state (counter wins) bypassing the signed mesh
	// packet path, so a remote caller could overwrite legitimate node state with a
	// large counter. Loopback-only for mutations, same guard MeshKnowledgeHandler
	// uses. r.RemoteAddr is the real TCP peer (not spoofable). GET stays open.
	if r.Method == http.MethodPost && !isLoopbackHostPort(r.RemoteAddr) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "mesh crdt mutation allowed from loopback only"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		topic := strings.TrimSpace(r.URL.Query().Get("topic"))
		if topic == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "topic required"})
			return
		}
		entries, err := mesh.CRDTListByTopic(db, topic)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		sum, _ := mesh.CRDTAggregate(db, topic)
		writeJSON(w, http.StatusOK, map[string]any{
			"topic":     topic,
			"entries":   entries,
			"aggregate": sum,
			"count":     len(entries),
		})
	case http.MethodPost:
		var body struct {
			Topic       string `json:"topic"`
			NodePubkey  string `json:"node_pubkey"`
			Counter     int64  `json:"counter"`
			PayloadJSON string `json:"payload_json"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if body.Topic == "" || body.NodePubkey == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "topic + node_pubkey required"})
			return
		}
		if body.PayloadJSON == "" {
			body.PayloadJSON = "{}"
		}
		if err := mesh.CRDTUpsert(db, body.Topic, body.NodePubkey, body.Counter, body.PayloadJSON); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// =============================================================================
// Section 17 Knowledge
// =============================================================================

// MeshKnowledgeHandler — GET ?status= list; POST ingest; PUT promote.
func MeshKnowledgeHandler(w http.ResponseWriter, r *http.Request) {
	// Mutating ops (ingest / promote) bypass the 9-layer mesh trust pipeline, so
	// they are loopback-only: when the listener is reachable off-loopback (e.g.
	// via the built-in tunnel, and RequireLogin off) this endpoint would let an
	// unauthenticated remote caller poison the local brain. r.RemoteAddr is the
	// real TCP peer (not spoofable). GET (read-only list) stays open.
	if (r.Method == http.MethodPost || r.Method == http.MethodPut) && !isLoopbackHostPort(r.RemoteAddr) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "mesh knowledge mutation allowed from loopback only"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		limit := 100
		if s := r.URL.Query().Get("limit"); s != "" {
			if n, _ := strconv.Atoi(s); n > 0 {
				limit = n
			}
		}
		items, err := mesh.ListKnowledge(db, status, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		counts, _ := mesh.CountKnowledge(db)
		writeJSON(w, http.StatusOK, map[string]any{
			"items":  items,
			"count":  len(items),
			"totals": counts,
		})
	case http.MethodPost:
		var body struct {
			PacketID      string `json:"packet_id"`
			OriginPubkey  string `json:"origin_pubkey"`
			DrawerContent string `json:"drawer_content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if err := mesh.IngestKnowledge(db, body.PacketID, body.OriginPubkey, body.DrawerContent); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case http.MethodPut:
		var body struct {
			PacketID string `json:"packet_id"`
			Status   string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if err := mesh.PromoteKnowledge(db, body.PacketID, body.Status); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "new_status": body.Status})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// =============================================================================
// Section 18 Tool manifest
// =============================================================================

func MeshToolManifestsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		cap := strings.TrimSpace(r.URL.Query().Get("capability"))
		if cap != "" {
			items, err := mesh.FindToolByCapability(db, cap, 20)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
			return
		}
		items, err := mesh.ListToolManifests(db, 100)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
	case http.MethodPost:
		var body struct {
			ToolName     string `json:"tool_name"`
			OriginPubkey string `json:"origin_pubkey"`
			ManifestJSON string `json:"manifest_json"`
			Signature    string `json:"signature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		// Section 18 phase 3: validate manifest content before storing — blocks
		// malformed/oversized manifests and obvious execution-smuggling tokens.
		if ok, reason := mesh.ValidateToolManifest(body.ManifestJSON); !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "manifest rejected: " + reason})
			return
		}
		if err := mesh.UpsertToolManifest(db, body.ToolName, body.OriginPubkey, body.ManifestJSON, body.Signature); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// =============================================================================
// Section 19 Karma
// =============================================================================

func MeshKarmaHandler(w http.ResponseWriter, r *http.Request) {
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := mesh.ListKarma(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
	case http.MethodPost:
		var body struct {
			PubkeyHex string  `json:"pubkey_hex"`
			Delta     float64 `json:"delta"`
			Event     string  `json:"event"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if err := mesh.AdjustKarma(db, body.PubkeyHex, body.Delta, body.Event); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func MeshKarmaDecayHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	n, err := mesh.DecayKarma(db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "affected_rows": n})
}

// =============================================================================
// Section 20 Filter pipeline test
// =============================================================================

func MeshFilterTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Packet  mesh.Packet `json:"packet"`
		Content string      `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	decisions := mesh.RunFilterPipeline(db, body.Packet, body.Content)
	if body.Packet.PacketID != "" {
		mesh.RecordFilterAudit(db, body.Packet.PacketID, decisions)
	}
	finalPass := true
	for _, d := range decisions {
		if d.Decision == "reject" {
			finalPass = false
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"decisions":  decisions,
		"final_pass": finalPass,
	})
}

// =============================================================================
// Section 21 LoRA + Section 22 L3 — simple endpoint
// =============================================================================

func MeshLoraDeltasHandler(w http.ResponseWriter, r *http.Request) {
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(
			`SELECT id, model_name, origin_pubkey, delta_uri, delta_size,
			        substr(signature, 1, 16) AS sig_preview, received_at
			 FROM mesh_lora_deltas ORDER BY id DESC LIMIT 100`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, size int64
			var name, origin, uri, sig, recv string
			_ = rows.Scan(&id, &name, &origin, &uri, &size, &sig, &recv)
			out = append(out, map[string]any{
				"id": id, "model_name": name, "origin_pubkey": origin,
				"delta_uri": uri, "delta_size": size,
				"signature_preview": sig, "received_at": recv,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out, "count": len(out)})
	case http.MethodPost:
		var body struct {
			ModelName    string `json:"model_name"`
			OriginPubkey string `json:"origin_pubkey"`
			DeltaURI     string `json:"delta_uri"`
			DeltaSize    int64  `json:"delta_size"`
			Signature    string `json:"signature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		// Section 21 phase 3: validate delta metadata (scheme allowlist, size
		// cap, signature) before store. Weight-apply stays deferred (no
		// fine-tuning runtime) — see internal/mesh/lora.go ApplyLoraDelta.
		if ok, reason := mesh.ValidateLoraDelta(body.ModelName, body.DeltaURI, body.DeltaSize, body.Signature); !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "lora delta rejected: " + reason})
			return
		}
		_, err := db.Exec(
			`INSERT INTO mesh_lora_deltas (model_name, origin_pubkey, delta_uri, delta_size, signature)
			 VALUES (?, ?, ?, ?, ?)`,
			body.ModelName, body.OriginPubkey, body.DeltaURI, body.DeltaSize, body.Signature)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func MeshL3Handler(w http.ResponseWriter, r *http.Request) {
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		k := strings.TrimSpace(r.URL.Query().Get("k"))
		if k == "" {
			rows, err := db.Query(`SELECT k, v, updated_at FROM mesh_l3_state ORDER BY k LIMIT 200`)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			defer rows.Close()
			out := []map[string]string{}
			for rows.Next() {
				var key, val, upd string
				_ = rows.Scan(&key, &val, &upd)
				out = append(out, map[string]string{"k": key, "v": val, "updated_at": upd})
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": out, "count": len(out)})
			return
		}
		var v, upd string
		err := db.QueryRow(`SELECT v, updated_at FROM mesh_l3_state WHERE k = ?`, k).Scan(&v, &upd)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "key not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"k": k, "v": v, "updated_at": upd})
	case http.MethodPost:
		var body struct {
			K string `json:"k"`
			V string `json:"v"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if body.K == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "k required"})
			return
		}
		_, err := db.Exec(
			`INSERT INTO mesh_l3_state (k, v) VALUES (?, ?)
			 ON CONFLICT(k) DO UPDATE SET v = excluded.v, updated_at = CURRENT_TIMESTAMP`,
			body.K, body.V)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// =============================================================================
// Section 23 Mesh daemon status (heartbeat write/read)
// =============================================================================

func MeshDaemonStatusHandler(w http.ResponseWriter, r *http.Request) {
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`SELECT k, v, updated_at FROM mesh_daemon_status`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := map[string]any{}
		for rows.Next() {
			var k, v, upd string
			_ = rows.Scan(&k, &v, &upd)
			out[k] = map[string]string{"v": v, "updated_at": upd}
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var body struct {
			K string `json:"k"`
			V string `json:"v"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_, err := db.Exec(
			`INSERT INTO mesh_daemon_status (k, v) VALUES (?, ?)
			 ON CONFLICT(k) DO UPDATE SET v = excluded.v, updated_at = CURRENT_TIMESTAMP`,
			body.K, body.V)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
