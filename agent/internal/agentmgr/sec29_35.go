// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Section 29 + Section 35 phase 1 endpoints. Phase 2 (real
//   zombie scan integration with codemap, prompt diff viewer, slot
//   constraints) → tambah file baru.
//
// sec29_35.go — Section 29 zombie + Section 35 prompt endpoints.

package agentmgr

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/httpx"
	"flowork-gui/internal/zombie"
)

// =============================================================================
// Section 29: Zombie findings
// =============================================================================

// ZombieFindingsHandler — GET/POST /api/agents/zombie/findings?id=<agent>
func ZombieFindingsHandler(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimSpace(r.URL.Query().Get("id"))
	if agentID == "" {
		httpx.WriteJSON(w, map[string]any{"error": "agent id required"})
		return
	}
	store, err := openAgentStore(agentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	switch r.Method {
	case http.MethodGet:
		rows, err := store.ListZombieFindings(parseLimitOr(r.URL.Query().Get("limit"), 100))
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"items": rows, "count": len(rows)})
	case http.MethodPost:
		var body agentdb.ZombieFinding
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "invalid json: " + err.Error()})
			return
		}
		id, err := store.AddZombieFinding(body)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "id": id})
	default:
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
	}
}

// ZombieScanHandler — POST /api/agents/zombie/scan?id=<agent>&min_age_days=
// Section 29 phase 2 real auto-detect: walk codemap_nodes, grep callers.
func ZombieScanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("id"))
	if agentID == "" {
		httpx.WriteJSON(w, map[string]any{"error": "agent id required"})
		return
	}
	minAge := 30
	if s := r.URL.Query().Get("min_age_days"); s != "" {
		if n, perr := strconv.Atoi(s); perr == nil && n >= 0 {
			minAge = n
		}
	}
	store, err := openAgentStore(agentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	sharedRoot := agentFolder(agentID) + "/workspace"
	res, err := zombie.Scan(r.Context(), store, zombie.ScanOptions{
		SharedRoot: sharedRoot,
		MinAgeDays: minAge,
	})
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"ok":               true,
		"symbols_scanned":  res.SymbolsScanned,
		"files_grepped":    res.FilesGrepped,
		"inserted":         res.Inserted,
		"findings_preview": res.Findings,
	})
}

// ZombieAckHandler — POST /api/agents/zombie/ack?id=&finding_id=
func ZombieAckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("id"))
	findingID, _ := strconv.ParseInt(r.URL.Query().Get("finding_id"), 10, 64)
	if agentID == "" || findingID == 0 {
		httpx.WriteJSON(w, map[string]any{"error": "id + finding_id required"})
		return
	}
	store, err := openAgentStore(agentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	if err := store.AcknowledgeZombie(findingID); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true})
}

// =============================================================================
// Section 35: Self-contained prompt.md
// =============================================================================

// SelfPromptRenderHandler — GET /api/agents/self-prompt/render?id=<agent>
// Section 35 phase 2: assemble latest slots → markdown → LLM wrapper inject.
func SelfPromptRenderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("id"))
	if agentID == "" {
		httpx.WriteJSON(w, map[string]any{"error": "agent id required"})
		return
	}
	store, err := openAgentStore(agentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	slots, err := store.ListSelfPromptSlots()
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	slotOrder := []string{"system", "persona", "guideline", "task"}
	byName := map[string]agentdb.SelfPrompt{}
	for _, s := range slots {
		byName[s.Slot] = s
	}
	var b strings.Builder
	// TIME-AWARENESS (owner 2026-06-23): inject waktu LIVE (WIB default) di paling atas system-prompt
	// SEMUA agent → fix "berita lama" (LLM ga lagi nebak tanggal dari training). Detail: time_awareness.go.
	b.WriteString(WIBNowHeader())
	// RESILIENCE (ITEM 1 roadmap agent-resilience): doktrin recovery (diagnosa/retry-transient/jujur/no-poll)
	// → semua agent tahan-banting, ga loop-mati pas error transient. Switch FLOWORK_RESILIENCE_OFF. Detail: agent_resilience.go.
	b.WriteString(RecoveryDoctrine())
	emitted := []string{}
	emitOne := func(slot string) {
		if sp, ok := byName[slot]; ok {
			b.WriteString("# ")
			b.WriteString(strings.ToUpper(slot))
			b.WriteString(" (v")
			b.WriteString(strconv.Itoa(sp.Version))
			b.WriteString(")\n")
			b.WriteString(sp.Body)
			b.WriteString("\n\n")
			emitted = append(emitted, slot)
			delete(byName, slot)
		}
	}
	for _, s := range slotOrder {
		emitOne(s)
	}
	leftKeys := make([]string, 0, len(byName))
	for k := range byName {
		leftKeys = append(leftKeys, k)
	}
	for i := 0; i < len(leftKeys); i++ {
		for j := i + 1; j < len(leftKeys); j++ {
			if leftKeys[j] < leftKeys[i] {
				leftKeys[i], leftKeys[j] = leftKeys[j], leftKeys[i]
			}
		}
	}
	for _, s := range leftKeys {
		emitOne(s)
	}
	httpx.WriteJSON(w, map[string]any{
		"rendered":     b.String(),
		"slots_used":   emitted,
		"length_bytes": b.Len(),
	})
}

// SelfPromptHandler — GET/POST /api/agents/self-prompt?id=<agent>&slot=
//
//	GET ?slot=&version= → return latest (version=0) atau specific.
//	POST body {slot, body, notes, version} → upsert next version.
func SelfPromptHandler(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimSpace(r.URL.Query().Get("id"))
	if agentID == "" {
		httpx.WriteJSON(w, map[string]any{"error": "agent id required"})
		return
	}
	store, err := openAgentStore(agentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	switch r.Method {
	case http.MethodGet:
		slot := strings.TrimSpace(r.URL.Query().Get("slot"))
		if slot == "" {
			rows, err := store.ListSelfPromptSlots()
			if err != nil {
				httpx.WriteJSON(w, map[string]any{"error": err.Error()})
				return
			}
			httpx.WriteJSON(w, map[string]any{"slots": rows, "count": len(rows)})
			return
		}
		version, _ := strconv.Atoi(r.URL.Query().Get("version"))
		sp, err := store.GetSelfPrompt(slot, version)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, sp)
	case http.MethodPost:
		var body agentdb.SelfPrompt
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "invalid json: " + err.Error()})
			return
		}
		id, err := store.SetSelfPrompt(body.Slot, body.Body, body.Notes, body.Version)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "id": id})
	default:
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
	}
}
