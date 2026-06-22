// FROZEN brain-core — desain abadi Flowork. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/flow_router
// Locked at: 2026-05-30
// Reason: Section 14 phase 2 transport endpoints. POST /api/mesh/packet
//   receive + verify signature + INSERT mesh_packets dedup. POST
//   /api/mesh/packet/send (admin/test) sign + persist. POST /api/mesh/
//   packet/forward gossip relay (Section 15 hook). Phase 3 (mutual TLS,
//   HTTP/2 streaming, relay path tracker) → tambah file baru.
//
// handlers_mesh_transport.go — Section 14 phase 2: packet receive + send.

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/mesh"
	"github.com/flowork-os/flowork_Router/internal/store"
)

// MeshPacketReceiveHandler — POST /api/mesh/packet (peer push).
// Verify signature, dedup by packet_id, persist, return ack.
func MeshPacketReceiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	// Per-source rate limit (anti gossip-flood / Sybil identity-spam) before any
	// parse/verify work, so a flood is cheap to reject.
	if meshRateLimited(w, clientIPForLock(r)) {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1*1024*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "read: " + err.Error()})
		return
	}
	pkt, err := mesh.ParsePacketJSON(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "parse: " + err.Error()})
		return
	}
	// Verify signature (anti spoofing).
	if verr := pkt.Verify(); verr != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "verify: " + verr.Error()})
		return
	}
	// HopMax check (anti flood).
	if pkt.HopCount > mesh.HopMax {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "hop_count exceeded"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	// Dedup check.
	if dup, _ := mesh.HasPacket(db, pkt.PacketID); dup {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "dedup": true, "packet_id": pkt.PacketID,
		})
		return
	}
	if perr := mesh.PersistPacket(db, pkt); perr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": perr.Error()})
		return
	}
	// Type-aware intake. Knowledge packets MUST run the 9-layer anti-poisoning
	// filter + near-dup + karma pipeline before anything trusts them (roadmap
	// Section 20: "wajib sebelum mesh public"). Other types just persist for now.
	switch pkt.PacketType {
	case mesh.PacketTypeKnowledge:
		kr := mesh.ProcessKnowledgePacket(db, pkt)
		if kr.Status != mesh.StatusDropped {
			_ = mesh.MarkProcessed(db, pkt.PacketID)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "packet_id": pkt.PacketID, "processed": true,
			"knowledge": kr,
		})
		return
	case mesh.PacketTypeToolShare:
		tr := mesh.IngestToolSharePacket(db, pkt)
		_ = mesh.MarkProcessed(db, pkt.PacketID)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "packet_id": pkt.PacketID, "processed": true,
			"tool_share": tr,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "packet_id": pkt.PacketID, "processed": false,
	})
}

// MeshPacketSendHandler — POST /api/mesh/packet/send (admin sign + persist
// outbound packet). Body {packet_type, payload_json}.
func MeshPacketSendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		PacketType  string `json:"packet_type"`
		PayloadJSON string `json:"payload_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	if strings.TrimSpace(body.PacketType) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "packet_type required"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	pubkey, err := mesh.LoadPubKeyHex(db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "load pubkey: " + err.Error()})
		return
	}
	privkey, err := mesh.LoadPrivKey(db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "load privkey: " + err.Error()})
		return
	}
	if body.PayloadJSON == "" {
		body.PayloadJSON = "{}"
	}
	pkt := mesh.NewPacket(pubkey, body.PacketType, body.PayloadJSON)
	if serr := pkt.Sign(privkey); serr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "sign: " + serr.Error()})
		return
	}
	if perr := mesh.PersistPacket(db, pkt); perr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": perr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"packet_id": pkt.PacketID,
		"signed":    true,
	})
}

// MeshPacketsHandler — GET /api/mesh/packets?limit=&pending_only=1
func MeshPacketsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	pendingOnly := r.URL.Query().Get("pending_only") == "1"
	limit := 100
	if pendingOnly {
		packets, perr := mesh.ListPendingPackets(db, limit)
		if perr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": perr.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": packets, "count": len(packets)})
		return
	}
	rows, err := db.Query(
		`SELECT id, packet_id, origin_pubkey, packet_type,
		        substr(payload_json, 1, 200) AS payload_preview,
		        signature, ttl, hop_count, received_at, processed
		 FROM mesh_packets ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var packetID, origin, pkType, preview, signature, recv string
		var ttl, hop, processed int
		_ = rows.Scan(&id, &packetID, &origin, &pkType, &preview, &signature, &ttl, &hop, &recv, &processed)
		// Guard the preview slice: a short/empty signature (NULL column or a
		// manually-inserted row) would panic on signature[:16] and abort the
		// request. Verified inserts produce 128-hex sigs, but don't assume it.
		sigPrev := signature
		if len(sigPrev) > 16 {
			sigPrev = sigPrev[:16] + "…"
		}
		out = append(out, map[string]any{
			"id":              id,
			"packet_id":       packetID,
			"origin_pubkey":   origin,
			"packet_type":     pkType,
			"payload_preview": preview,
			"signature":       sigPrev,
			"ttl":             ttl,
			"hop_count":       hop,
			"received_at":     recv,
			"processed":       processed != 0,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "count": len(out)})
}
