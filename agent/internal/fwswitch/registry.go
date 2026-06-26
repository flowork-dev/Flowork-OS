// registry.go — daftar KURASI switch fitur yg dimunculin di GUI Setting (owner: "switch
// fitur aja"). Hardware/path/secret SENGAJA ga di sini (tetep env/auto-detect). Nambah switch
// baru di masa depan = tambah 1 entri di sini (plug-and-play, ga sentuh kode lain).
package fwswitch

import (
	"os"
	"strings"
)

// Switch — metadata 1 switch fitur buat render GUI + resolve nilai efektif.
type Switch struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Desc     string `json:"desc"`
	Type     string `json:"type"`     // "bool" | "float" | "int" | "string"
	Default  string `json:"default"`  // default kode (string, konsisten env)
	Category string `json:"category"` // grup di GUI
}

// Registry — switch fitur yg DIKELOLA dari GUI. Default HARUS sama dgn default di call-site.
var Registry = []Switch{
	{"FLOWORK_INSTINCT_SCOPED", "Scoped instinct per-peran", "Tiap agent cuma dapet insting domain-nya (+ baseline). Hemat token + anti-noise.", "bool", "false", "Brain / Instinct"},
	{"FLOWORK_INSTINCT_SEMANTIC", "Seleksi insting semantic", "Pilih insting by-makna (vektor bge-m3). OFF = token-overlap (lebih kasar).", "bool", "true", "Brain / Instinct"},
	{"FLOWORK_INSTINCT_INJECT", "Injeksi insting", "Suntik insting relevan ke tiap request. OFF = matiin total.", "bool", "true", "Brain / Instinct"},
	{"FLOWORK_BRAIN_EXTERNAL_SCOPE", "Anti-halu agent luar", "Caller eksternal (brain-as-service) ga dikasih insting-tool Flowork → ga halu manggil tool yg ga ada.", "bool", "false", "Brain / Instinct"},
	{"FLOWORK_SEARCH_MINSCORE", "Lantai relevansi search", "Skor cosine minimum hasil brain-search (0.45 default). 0 = matiin lantai (semua lolos).", "float", "0.45", "Brain / Search"},
	{"FLOWORK_BINARY_VECTOR", "Binary-vector recall (#5)", "Search korpus JUTAAN: coarse biner (popcount) + rerank int8. auto (default) = aktif otomatis >=1jt drawer; on=paksa; off=int8 biasa. Recall 1.0 (rerank exact).", "string", "auto", "Brain / Search"},
	{"FLOWORK_BINARY_VECTOR_MIN", "Binary auto threshold", "Jumlah drawer minimum biar binary-vector auto-aktif (default 1000000 = 1 juta).", "int", "1000000", "Brain / Search"},
	{"FLOWORK_TOOLCALL_RECOVER", "Pulihin <tool_call> bocor", "Parse teks <tool_call> yg bocor dari model lokal jadi tool-call asli (anti-bocor ke user).", "bool", "true", "Router / Tools"},
	{"FLOWORK_DEFER_TOOLS", "Defer skema tool (#2C)", "Kirim skema tool on-demand (tool_lookup) → hemat prompt. Per-agent override di panel Agent Brain.", "bool", "false", "Router / Tools"},
	{"FLOWORK_DYNAMIC_TOOLS", "Intent-gated tools (#9)", "Router prune tool-schema ke yg RELEVAN (semantic cosine) → potong biang token #1 (~55% prompt). Escape-hatch (tool_search/tool_lookup) selalu lolos → aman.", "bool", "false", "Router / Tools"},
	{"FLOWORK_DYNAMIC_TOOLS_TOPK", "Intent-gated: top-K tool", "Max tool relevan yg dikirim (di luar escape-hatch + tool yg udah dipanggil). Default 12. Kecil = hemat tapi rawan starve.", "int", "12", "Router / Tools"},
	{"FLOWORK_DYNAMIC_TOOLS_MINSCORE", "Intent-gated: lantai skor", "Cosine minimum tool dianggap relevan (0.30 default). Naikin = lebih ketat.", "float", "0.30", "Router / Tools"},
	{"FLOWORK_EXPOSE_ALL_TOOLS", "Buka semua tool", "Semua agent boleh akses semua tool (subscription-gating udah dicabut).", "bool", "false", "Router / Tools"},
	{"FLOWORK_ROUTER_RETRY", "Retry router transient", "Coba ulang (exp-backoff) pas error sementara ke model lokal.", "bool", "false", "Router / Resilience"},
	{"FLOWORK_ORCHESTRATOR", "Orkestrator default", "Agent yg jadi orkestrator utama (default mr-flow).", "string", "mr-flow", "Agent"},
	{"FLOWORK_EDITION", "Edisi (FREE/CORPORATE)", "FREE (default) = identitas (persona/konstitusi) READ-ONLY, anti-rebrand. 'corporate' = unlock white-label/edit identitas.", "string", "free", "Bisnis / Edition"},
	{"FLOWORK_CACHE_REUSE", "KV cache-reuse (#8)", "Reuse prefix prompt statik (konstitusi+tool-schema) lintas-call via KV-shift → skip re-prefill. Isi N (mis. 256). Kosong/0=off. Berlaku saat LLM reload.", "int", "0", "Engine / KV-cache"},
	{"FLOWORK_PARALLEL_SLOTS", "Parallel slots (#8)", "-np N: N slot server biar multi-semut share 1 engine barengan. ⚠️ ctx kebagi N → naikin FLOWORK_CTX. Kosong/0=off (auto). Berlaku saat LLM reload.", "int", "0", "Engine / KV-cache"},
	{"FLOWORK_SLOT_SAVE_PATH", "Slot KV persist (#8)", "--slot-save-path dir: simpan KV slot ke disk → warm-restore lintas-restart (skip re-prefill). Kosong=off. Berlaku saat LLM reload.", "string", "", "Engine / KV-cache"},
}

// Resolved — nilai efektif 1 switch + dari mana asalnya.
type Resolved struct {
	Switch
	Value  string `json:"value"`  // nilai efektif
	Source string `json:"source"` // "gui" | "env" | "default"
}

// Resolve — kembalikan nilai efektif semua switch registry (presedensi GUI > ENV > default).
func Resolve() []Resolved {
	file := ReadFile()
	out := make([]Resolved, 0, len(Registry))
	for _, s := range Registry {
		r := Resolved{Switch: s}
		if v, ok := file[s.Key]; ok && strings.TrimSpace(v) != "" {
			r.Value, r.Source = strings.TrimSpace(v), "gui"
		} else if v := strings.TrimSpace(os.Getenv(s.Key)); v != "" {
			r.Value, r.Source = v, "env"
		} else {
			r.Value, r.Source = s.Default, "default"
		}
		out = append(out, r)
	}
	return out
}
