// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Tab GUI: Tunnel (registry plug-and-play) → dok lock/gui/Tunnel.md  ⚠️ FROZEN — jangan edit.
// Nambah provider tunnel: file sibling tunnel_<x>_ext.go + RegisterTunnelProvider. Cara:
// CARAFREEZE.MD (POLA A) + lock/plug-and-play.md. Pola freeze: lock/frozen-core.md

package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func init() {
	RegisterExtraRoute(func(mux *http.ServeMux) {
		mux.HandleFunc("/api/tunnel/providers", tunnelProvidersListHandler)
		mux.HandleFunc("/api/tunnel/provider/", tunnelProviderDispatchHandler)
	})
}

func tunnelProvidersListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list := make([]map[string]any, 0, len(tunnelProviders))
	for _, name := range tunnelProviderNames() {
		p, _ := getTunnelProvider(name)
		e := map[string]any{"name": name}
		if p.Detect != nil {
			e["installed"] = p.Detect()
		}
		if p.Status != nil {
			e["status"] = p.Status()
		}
		list = append(list, e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func tunnelProviderDispatchHandler(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/tunnel/provider/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "butuh /api/tunnel/provider/<name>/<action>"})
		return
	}
	name, action := parts[0], parts[1]
	p, ok := getTunnelProvider(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "provider tunnel tak terdaftar: " + name, "available": tunnelProviderNames()})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	switch action {
	case "enable":
		var body struct {
			Target string `json:"target"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.TrimSpace(body.Target) == "" {
			body.Target = "http://127.0.0.1:2402"
		}
		res, err := p.Enable(body.Target)
		if res == nil {
			res = map[string]any{}
		}
		if err != nil {
			res["error"] = err.Error()
		}
		writeJSON(w, http.StatusOK, res)
	case "disable":
		if p.Disable == nil {
			writeJSON(w, http.StatusOK, map[string]any{"disabled": true, "note": "provider tak punya Disable"})
			return
		}
		res, err := p.Disable()
		if res == nil {
			res = map[string]any{}
		}
		if err != nil {
			res["error"] = err.Error()
		}
		writeJSON(w, http.StatusOK, res)
	case "status":
		if p.Status == nil {
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
		writeJSON(w, http.StatusOK, p.Status())
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "action tak dikenal: " + action})
	}
}
