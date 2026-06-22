// FROZEN brain-core — desain abadi Flowork. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/flow_router
// Locked at: 2026-05-30
// Reason: Section 13 phase 1 endpoints. API stable: GET /api/mesh/peers,
//   POST /api/mesh/discover, GET /api/mesh/identity, POST /api/mesh/peer
//   (upsert manual). Phase 2 (handshake, gossip, packet relay) → tambah
//   file baru handlers_mesh_*.go, JANGAN modify ini.
//
// handlers_mesh.go — Section 13 phase 1 admin endpoints. Phase 1 ngga
// punya actual mDNS goroutine — /discover return stub OK + log. Future
// phase 2 wires actual networking.

package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/mesh"
	"github.com/flowork-os/flowork_Router/internal/store"
)

// GET /api/mesh/identity — return own pubkey + hostname + peer count.
func meshIdentityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	id, err := mesh.LoadIdentity(db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	peerCount, _ := mesh.CountPeers(db, false)
	writeJSON(w, http.StatusOK, map[string]any{
		"pubkey":     id.PubKeyHex,
		"hostname":   id.Hostname,
		"version":    id.Version,
		"peer_count": peerCount,
	})
}

// GET /api/mesh/peers — list peers (?include_blocked=1 untuk include block).
func meshPeersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	includeBlocked := r.URL.Query().Get("include_blocked") == "1"
	peers, err := mesh.ListPeers(db, includeBlocked)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"peers": peers,
		"count": len(peers),
	})
}

// POST /api/mesh/discover — trigger discovery sweep.
// Phase 1: stub — log + return OK. Phase 2: actual mDNS announce + scan.
func meshDiscoverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"phase":   1,
		"message": "discovery stub — phase 2 will wire actual mDNS multicast",
	})
}

// POST /api/mesh/peer — manual upsert peer (admin tool / test).
//
// Body: {pubkey_hex, hostname, ip, port, version, is_virt}
func meshUpsertPeerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	var body mesh.Peer
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	body.PubKeyHex = strings.TrimSpace(body.PubKeyHex)
	if body.PubKeyHex == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "pubkey_hex required"})
		return
	}
	
	// FIX #6: Validate pubkey format (must be valid 64-char hex string)
	if _, err := hex.DecodeString(body.PubKeyHex); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid pubkey_hex: must be 64-character hexadecimal string",
		})
		return
	}
	if len(body.PubKeyHex) != 64 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid pubkey_hex: must be exactly 64 characters (32 bytes ed25519 key)",
		})
		return
	}
	
	if err := mesh.UpsertPeer(db, body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pubkey": body.PubKeyHex})
}

// POST /api/mesh/peer/block?pubkey=<hex>&blocked=1 — toggle block.
func meshBlockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	db, err := store.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	pubkey := strings.TrimSpace(r.URL.Query().Get("pubkey"))
	if pubkey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "pubkey required"})
		return
	}
	
	// FIX #6: Validate pubkey format before using it
	if _, err := hex.DecodeString(pubkey); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid pubkey: must be 64-character hexadecimal string",
		})
		return
	}
	if len(pubkey) != 64 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid pubkey: must be exactly 64 characters",
		})
		return
	}
	
	blocked := r.URL.Query().Get("blocked") == "1"
	if err := mesh.SetBlocked(db, pubkey, blocked); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pubkey": pubkey, "blocked": blocked})
}
