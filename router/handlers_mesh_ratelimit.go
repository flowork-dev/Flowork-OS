// FROZEN brain-core — desain abadi Flowork. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-03
// Reason: Security pass — per-source mesh rate limit (anti gossip-flood/Sybil).
//
// Inbound mesh-packet rate limiter (per source IP, in-memory).
//
// SECURITY: the mesh receive path verifies ed25519 signatures, but a signature
// only proves possession of a freshly-generated keypair — there is no cost to
// minting new identities. Without a source-rate bound, one host can flood the
// node with packets and/or spam new peer identities (Sybil). This caps inbound
// packets per source IP per window: well above any honest gossip cadence
// (~3 peers / 10s), well below a flood. Bounds both gossip amplification and
// Sybil identity-spam at the source, without touching the signed packet format.

package main

import (
	"net/http"
	"sync"
	"time"
)

const (
	meshRLWindow = 10 * time.Second
	meshRLMax    = 60 // max inbound mesh packets per source IP per window
)

var (
	meshRLMu   sync.Mutex
	meshRLSeen = map[string][]int64{} // source IP -> recent arrival unix-nanos
)

// meshRateLimitOK records an inbound packet from ip and reports whether it is
// within the per-source rate budget.
func meshRateLimitOK(ip string) bool {
	now := time.Now().UnixNano()
	cutoff := now - int64(meshRLWindow)
	meshRLMu.Lock()
	defer meshRLMu.Unlock()
	ts := meshRLSeen[ip]
	kept := ts[:0]
	for _, t := range ts {
		if t >= cutoff {
			kept = append(kept, t)
		}
	}
	if len(kept) >= meshRLMax {
		meshRLSeen[ip] = kept
		return false
	}
	meshRLSeen[ip] = append(kept, now)
	// Opportunistic GC so the map doesn't grow unbounded with one-shot peers.
	if len(meshRLSeen) > 4096 {
		for k, v := range meshRLSeen {
			if len(v) == 0 || v[len(v)-1] < cutoff {
				delete(meshRLSeen, k)
			}
		}
	}
	return true
}

// meshRateLimited writes a 429 and returns true when ip is over budget.
func meshRateLimited(w http.ResponseWriter, ip string) bool {
	if meshRateLimitOK(ip) {
		return false
	}
	writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "mesh rate limit exceeded for source"})
	return true
}
