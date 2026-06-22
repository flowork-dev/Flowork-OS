// FROZEN-candidate (FASE-B agnostic host, owner-approved 2026-06-22): RANGKA bootstrap.
// Lihat lock/brain.md (arsitektur 3-lapis: Jantung beku · Rangka beku · Titik-tumbuh hidup).
//
// feature_registry.go — bikin main.go AGNOSTIC. Tiap fitur DAFTAR SENDIRI lewat init()
// (RegisterFeature), main.go cuma iterasi per-fase. Nambah fitur = bikin file feature_*.go
// BARU (init() → RegisterFeature), NOL sentuh main.go → main.go bisa di-FREEZE permanen.
package main

import (
	"context"
	"io/fs"
	"net/http"
	"sort"

	"flowork-gui/internal/floworkauth"
	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernelhost"
	"flowork-gui/internal/settingsapi"
)

// Deps — semua dependency shared yg handler/feature butuh. Di-populate di main() PASCA-boot.
// Extra = klep dep BARU tanpa unfreeze file ini (future-proof: feature taruh/ambil via key).
type Deps struct {
	Ctx       context.Context
	Host      *kernelhost.Host
	FDB       *floworkdb.Store
	AuthMgr     *floworkauth.Manager
	GroupsAPI   *groupsapi.Handler
	SettingsAPI *settingsapi.API
	Mux         *http.ServeMux
	StaticFS  fs.FS
	Extra     map[string]any
}

// Fase eksekusi — urutan boot. Wire (set hook global) → Route (mount mux) → Seed (after agents).
const (
	PhaseWire  = 0
	PhaseRoute = 1
	PhaseSeed  = 2
)

// Feature — satu fitur self-contained. Apply dipanggil main() di fase-nya, dapet Deps.
type Feature struct {
	Name  string
	Phase int
	Apply func(*Deps)
}

var featureRegistry []Feature

// RegisterFeature — dipanggil dari init() tiap file feature_*.go. Append ke registry global.
func RegisterFeature(f Feature) { featureRegistry = append(featureRegistry, f) }

// applyPhase — jalanin SEMUA feature di fase ini, urut by Name (deterministik, ga gantung
// urutan init() yg rapuh). Dipanggil main() 3x (Wire/Route/Seed).
func applyPhase(d *Deps, phase int) {
	list := make([]Feature, 0, len(featureRegistry))
	for _, f := range featureRegistry {
		if f.Phase == phase {
			list = append(list, f)
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	for _, f := range list {
		if f.Apply != nil {
			f.Apply(d)
		}
	}
}
