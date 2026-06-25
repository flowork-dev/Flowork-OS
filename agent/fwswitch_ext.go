// fwswitch_ext.go — GROWTH-POINT (NON-frozen). Boot plug-and-play settings (lihat
// internal/fwswitch) + endpoint GUI /api/settings/switches. init() → Boot() jalan sebelum
// main() (agent/main.go FROZEN, ga disentuh). Lihat lock/fwswitch.md.
package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"flowork-gui/internal/fwswitch"
)

// switchesHandler — GET: daftar switch fitur + nilai efektif + sumber (gui/env/default).
// POST: {values:{KEY:val,...}} (val "" = hapus → revert ENV/default) → tulis file lintas-proses
// → Apply lokal (host) → router nyusul lewat watcher mtime. = GUI halaman Setting (plug-and-play).
func switchesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tfWriteJSON(w, 0, map[string]any{"switches": fwswitch.Resolve(), "path": fwswitch.SettingsPath()})
	case http.MethodPost:
		var body struct {
			Values map[string]string `json:"values"`
			Key    string            `json:"key"`
			Value  string            `json:"value"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
			return
		}
		vals := body.Values
		if vals == nil {
			vals = map[string]string{}
		}
		if strings.TrimSpace(body.Key) != "" {
			vals[body.Key] = body.Value
		}
		// hanya izinkan key yg ADA di registry (cegah set env sembarangan dari GUI).
		allow := map[string]bool{}
		for _, s := range fwswitch.Registry {
			allow[s.Key] = true
		}
		clean := map[string]string{}
		for k, v := range vals {
			if allow[strings.TrimSpace(k)] {
				clean[strings.TrimSpace(k)] = v
			}
		}
		if len(clean) == 0 {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "no known switch key in payload"})
			return
		}
		if err := fwswitch.WriteValues(clean); err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		tfWriteJSON(w, 0, map[string]any{"ok": true, "switches": fwswitch.Resolve()})
	default:
		tfWriteJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "GET/POST only"})
	}
}

func init() {
	fwswitch.Boot() // apply switch GUI ke os.Setenv SEBELUM server nyala + watcher live
	RegisterFeature(Feature{Name: "fwswitch-route", Phase: PhaseRoute, Apply: func(d *Deps) {
		d.Mux.HandleFunc("/api/settings/switches", switchesHandler)
	}})
}
