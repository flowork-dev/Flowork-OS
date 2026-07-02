// feature_health.go — F-F: /api/health loopback (dirujuk dokumen rilis/portable)
// + diagnostik ringan gaya `doctor` (roadmap F-G digabung ke sini).
// NON-FROZEN sibling (deletable, seam feature-registry). Allowlist auth-nya
// dicolok di internal/floworkauth/allow_health_ext.go. 📄 Dok: lock/approval-gate.md
package main

import (
	"net"
	"net/http"
	"time"

	"flowork-gui/internal/httpx"
	"flowork-gui/internal/kernelhost"
)

func init() {
	RegisterFeature(Feature{Name: "health-route", Phase: PhaseRoute, Apply: func(d *Deps) {
		host := d.Host
		d.Mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(w, healthReport(host))
		})
	}})
}

// healthReport — diagnostik murah (NOL token/LLM): proses idup, agent ke-load,
// router :2402 kejangkau. Buat tutorial rilis: `curl 127.0.0.1:1987/api/health`.
func healthReport(host *kernelhost.Host) map[string]any {
	out := map[string]any{
		"status":  "ok",
		"service": "flowork-agent",
		"version": version,
		"ts":      time.Now().UTC().Format(time.RFC3339),
	}
	if host != nil {
		out["agents_loaded"] = host.Runtime.Loaded()
	}
	// Router reachable? (dial murah, timeout pendek — bukan health si model.)
	c, err := net.DialTimeout("tcp", "127.0.0.1:2402", 800*time.Millisecond)
	if err != nil {
		out["router_ok"] = false
		out["status"] = "degraded"
	} else {
		_ = c.Close()
		out["router_ok"] = true
	}
	return out
}
