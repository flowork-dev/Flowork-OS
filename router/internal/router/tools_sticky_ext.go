// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// tools_sticky_ext.go — sibling NON-frozen (Rule 7): STICKY-UNION daftar tool (anti cache-sabotage).
//
// AKAR yang dicabut: maybeFilterTools (intent-gated #9, frozen) mangkas tool per-QUERY —
// isi + URUTAN daftar tool BEDA tiap turn. Cache Anthropic hash prefix urutan
// tools → system → messages; tools berubah = SEMUA breakpoint miss → persona stabil +
// history dibayar ulang tarif cache-write TIAP call. Dua fitur hemat saling sabotase:
// pruning ngalahin prompt-caching (biang "sering kena limit").
//
// Fix (via seam applyInjectShaper, jalan PERSIS setelah maybeFilterTools):
//   - Union AKUMULATIF per-agent: tool yang pernah lolos filter di sesi ini di-ingat
//     (schema di-cache) dan SELALU ikut dikirim, urutan FIRST-SEEN (append-only).
//   - Turn baru cuma bisa NAMBAH tool di EKOR → prefix daftar tool stabil → cache idup.
//   - Konvergen ke "semua tool yang relevan di sesi ini" — tetap jauh lebih kecil dari
//     katalog penuh, tapi stabil (hemat pruning + hemat cache, bukan salah satu).
//
// SWITCH (GUI = kebenaran): FLOWORK_TOOLS_STICKY (default ON; cuma aktif saat
// FLOWORK_DYNAMIC_TOOLS on — tanpa pruning daftar udah stabil sendiri).
// State in-memory per proses router (reset saat restart — cache provider juga expire).
// File ini DIHAPUS → seam balik no-op → perilaku pruning lama (delete-test aman).

package router

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/flowork-os/flowork_Router/internal/store"
)

var (
	stickyMu     sync.Mutex
	stickyOrder  = map[string][]string{}                   // key agent -> nama tool urutan first-seen
	stickySchema = map[string]map[string]json.RawMessage{} // key agent -> nama -> JSON tool utuh
)

// stickyEnabled — default ON; matiin dengan FLOWORK_TOOLS_STICKY=0/off/false/no.
func stickyEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FLOWORK_TOOLS_STICKY"))) {
	case "0", "off", "false", "no":
		return false
	}
	return true
}

func stickyUnionTools(ctx context.Context, req OpenAIRequest) OpenAIRequest {
	if !stickyEnabled() {
		return req
	}
	// Cuma relevan pas intent-gated pruning aktif (samain gate sama maybeFilterTools).
	if !truthyEnv("FLOWORK_DYNAMIC_TOOLS") && os.Getenv("FLOW_ROUTER_DYNAMIC_TOOLS") != "1" {
		return req
	}
	if len(req.Tools) == 0 || string(req.Tools) == "null" {
		return req
	}
	var cur []json.RawMessage
	if err := json.Unmarshal(req.Tools, &cur); err != nil || len(cur) == 0 {
		return req
	}
	key := AgentIDFromContext(ctx)
	if key == "" {
		key = agentName(ctx)
	}
	stickyMu.Lock()
	defer stickyMu.Unlock()
	names := stickyOrder[key]
	schemas := stickySchema[key]
	if schemas == nil {
		schemas = map[string]json.RawMessage{}
		stickySchema[key] = schemas
	}
	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}
	added := 0
	for _, raw := range cur {
		var t struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if json.Unmarshal(raw, &t) != nil || strings.TrimSpace(t.Function.Name) == "" {
			continue
		}
		schemas[t.Function.Name] = raw // refresh schema terbaru
		if !seen[t.Function.Name] {
			names = append(names, t.Function.Name)
			seen[t.Function.Name] = true
			added++
		}
	}
	stickyOrder[key] = names
	out := make([]json.RawMessage, 0, len(names))
	for _, n := range names {
		if raw, ok := schemas[n]; ok {
			out = append(out, raw)
		}
	}
	if len(out) == 0 {
		return req
	}
	b, err := json.Marshal(out)
	if err != nil {
		return req
	}
	if added > 0 || len(out) != len(cur) {
		log.Printf("flow_router tools-sticky: agent=%q union %d tool (turn ini %d, baru %d) — urutan stabil buat prompt-cache", key, len(out), len(cur), added)
	}
	req.Tools = b
	return req
}

func init() {
	prev := applyInjectShaper
	applyInjectShaper = func(ctx context.Context, req OpenAIRequest, settings *store.Settings) OpenAIRequest {
		req = stickyUnionTools(ctx, req)
		return prev(ctx, req, settings)
	}
}
