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
	{"FLOWORK_TOOLCALL_RECOVER", "Pulihin <tool_call> bocor", "Parse teks <tool_call> yg bocor dari model lokal jadi tool-call asli (anti-bocor ke user).", "bool", "true", "Router / Tools"},
	{"FLOWORK_DEFER_TOOLS", "Defer skema tool (#2C)", "Kirim skema tool on-demand (tool_lookup) → hemat prompt. Per-agent override di panel Agent Brain.", "bool", "false", "Router / Tools"},
	{"FLOWORK_EXPOSE_ALL_TOOLS", "Buka semua tool", "Semua agent boleh akses semua tool (subscription-gating udah dicabut).", "bool", "false", "Router / Tools"},
	{"FLOWORK_ROUTER_RETRY", "Retry router transient", "Coba ulang (exp-backoff) pas error sementara ke model lokal.", "bool", "false", "Router / Resilience"},
	{"FLOWORK_ORCHESTRATOR", "Orkestrator default", "Agent yg jadi orkestrator utama (default mr-flow).", "string", "mr-flow", "Agent"},
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
