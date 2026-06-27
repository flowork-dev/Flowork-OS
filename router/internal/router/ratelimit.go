// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package router

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultMaxRateLimitRetries = 6
	defaultDispatchConcurrency = 3
	rateLimitBackoffCap        = 30 * time.Second
)

// maxRateLimitRetries — berapa kali retry provider yg lagi 429 SEBELUM lompat ke fallback
// (model/provider berikut di rantai). SWITCH GUI FLOWORK_RL_MAX_RETRY (fwswitch registry).
// Default 6 (backoff s/d ~90 detik, perilaku lama). Set 1-2 → pas Opus penuh, cepet turun
// ke haiku/lokal (ga buang 90 detik). Clamp [0,20]. Baca per-pakai → live (fwswitch sync).
func maxRateLimitRetries() int {
	if n, err := strconv.Atoi(os.Getenv("FLOWORK_RL_MAX_RETRY")); err == nil && n >= 0 && n <= 20 {
		return n
	}
	return defaultMaxRateLimitRetries
}

var claudeSem = make(chan struct{}, dispatchConcurrency())

func dispatchConcurrency() int {
	if v := os.Getenv("FLOW_ROUTER_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultDispatchConcurrency
}

func acquireDispatchSlot() { claudeSem <- struct{}{} }
func releaseDispatchSlot() { <-claudeSem }

func backoffDuration(attempt int) time.Duration {
	d := time.Duration(2<<uint(attempt)) * time.Second
	if d > rateLimitBackoffCap {
		d = rateLimitBackoffCap
	}
	return d
}
