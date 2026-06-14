// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Section 10 (Registry singleton) phase 1 DONE. API stable:
//   Register (panic on dup name — early bug catch), Lookup, List, ListNames,
//   Count, ListSummaries (summary anti over-prompt). sync.RWMutex thread-
//   safe. Phase 2 (categories DB-backed, per-warga priors weighting, alias
//   chain) → tambah file baru, JANGAN modify ini.
//
// registry.go — singleton registry tools.
//
// USAGE:
//   Setiap tool package register di init():
//     func init() { tools.Register(&MyTool{}) }
//
//   Caller dispatch:
//     t, ok := tools.Lookup("read_file")
//     if !ok { return errors.New("tool not registered") }
//     res, err := t.Run(ctx, args)
//
//   Browse all:
//     for _, t := range tools.List() { ... }
//
// Thread-safe via sync.RWMutex — phase 1 baca >> tulis (registry frozen
// post-init). Mutex tetap ada kalau future dynamic registration needed.

package tools

import (
	"fmt"
	"sort"
	"sync"
)

var (
	regMu    sync.RWMutex
	registry = map[string]Tool{}
)

// Register — add tool ke registry. Panic kalau duplicate name (early bug
// catch — name collision serius).
func Register(t Tool) {
	if t == nil {
		panic("tools.Register: nil tool")
	}
	name := t.Name()
	if name == "" {
		panic("tools.Register: empty tool name")
	}
	regMu.Lock()
	defer regMu.Unlock()
	if existing, ok := registry[name]; ok {
		panic(fmt.Sprintf("tools.Register: duplicate name %q (existing=%T, new=%T)", name, existing, t))
	}
	registry[name] = t
}

// Lookup — fetch tool by name. Return (tool, ok). Caller cek ok=false →
// "tool not registered".
func Lookup(name string) (Tool, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	if t, ok := registry[name]; ok {
		return t, ok
	}
	// Alias layer (Phase 2 "alias chain", see header): surface vocabulary names
	// (Read/Write/Bash/...) resolve to their canonical twin. Pure map lookup in
	// alias.go — no recursion, safe under the read lock. mcp__* and unknown names
	// pass through unchanged and simply miss.
	if canon := canonicalToolName(name); canon != name {
		t, ok := registry[canon]
		return t, ok
	}
	return nil, false
}

// List — return semua registered tools sorted by name. Buat discovery endpoint.
func List() []Tool {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]Tool, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// ListNames — return sorted slice of names. Lebih ringan dari List() kalau
// caller cuma butuh nama (e.g. list endpoint summary, anti over-prompt).
func ListNames() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Count — total registered tools.
func Count() int {
	regMu.RLock()
	defer regMu.RUnlock()
	return len(registry)
}

// ToolSummary — minimal payload buat discovery endpoint. Strip Schema body
// supaya anti over-prompt (caller pull schema on-demand via /tools/get).
type ToolSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Capability  string `json:"capability"`
}

// ListSummaries — slice of ToolSummary, sorted by name.
func ListSummaries() []ToolSummary {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]ToolSummary, 0, len(registry))
	for _, t := range registry {
		s := t.Schema()
		out = append(out, ToolSummary{
			Name:        t.Name(),
			Description: s.Description,
			Capability:  t.Capability(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
