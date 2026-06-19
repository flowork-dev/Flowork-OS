// cognitive_handlers.go — HTTP read API buat Cognitive Graph (CGM) per-agent.
//
// GUI tab "Cognitive Graph" (reuse pola codemap D3 force-graph) baca dari sini.
// Pola sama persis CodemapNodesHandler: validasi id → openAgentStore → query → JSON.
// Read-only; CRUD/edit graph nyusul kalau perlu (additive, plug-and-play).

package agentmgr

import (
	"net/http"
	"strings"

	"flowork-gui/internal/httpx"
)

// CognitiveGraphHandler — GET /api/agents/cognitive/graph?id=<agent>&limit=N
// → {nodes:[...], edges:[...]} buat divisualisasi (bola-bola nyambung).
func CognitiveGraphHandler(w http.ResponseWriter, r *http.Request) {
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

	nodes, err := store.ListCogNodes(parseLimitOr(r.URL.Query().Get("limit"), 500))
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	edges, err := store.ListCogEdges(parseLimitOr(r.URL.Query().Get("edge_limit"), 1000))
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"nodes": nodes, "edges": edges,
		"node_count": len(nodes), "edge_count": len(edges),
	})
}

// CognitiveTensionsHandler — GET /api/agents/cognitive/tensions?id=<agent>
// → kontradiksi 'open' yang nunggu owner putusin.
func CognitiveTensionsHandler(w http.ResponseWriter, r *http.Request) {
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

	items, err := store.ListOpenTensions(parseLimitOr(r.URL.Query().Get("limit"), 50))
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})
}
