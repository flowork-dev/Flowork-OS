// FROZEN brain-core (Skills) — kalau "nyasar": ini BY-DESIGN, baca lock/brain.md.
//
// handlers_skills_crud.go — DICABUT dari handlers_resources.go (owner 2026-06-22 "jalur
// bercabang → bikin jalur baru, diabadikan"). handlers_resources.go SHARED (providers/combos/
// apiKeys/proxyPools/media) — skill CRUD numpang di situ. Sekarang skill CRUD (/api/skills +
// /api/skills/<id>) di file SENDIRI biar bisa di-FREEZE tanpa ngunci resource lain. Delegasi
// murni ke internal/store/skills.go (FROZEN).
package main

import (
	"encoding/json"
	"net/http"

	"github.com/flowork-os/flowork_Router/internal/store"
)

func skillsListAddHandler(w http.ResponseWriter, r *http.Request) {
	d, _ := store.Open()
	switch r.Method {
	case http.MethodGet:
		items, err := store.ListSkills(d)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": items, "count": len(items)})
	case http.MethodPost:
		var s store.Skill
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := store.UpsertSkill(d, &s); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(s)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func skillCRUDHandler(w http.ResponseWriter, r *http.Request) {
	d, _ := store.Open()
	id := r.URL.Path[len("/api/skills/"):]
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var s store.Skill
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		s.ID = id
		if err := store.UpsertSkill(d, &s); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)
	case http.MethodDelete:
		if err := store.DeleteSkill(d, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
