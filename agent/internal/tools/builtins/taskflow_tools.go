// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-02
// Reason: FASE 6 — Mr.Flow jadi router. task_list/task_run E2E verified (LLM
//   route "analisa saham GOTO" → task_list+task_run → run kebikin). Cap
//   rpc:taskflow gating. Extend (task tools baru) → tambah di file ini.
// 2026-06-16 (owner-approved, OPS-1): task_run nambah param OPSIONAL `group` —
//   delegasi GROUP async (member+synth+debate jalan di belakang, notify Telegram
//   pas kelar) biar ask_group BERAT gak timeout. ADDITIVE: `category` tetap jalan.
//
// taskflow_tools.go — FASE 6: tools biar Mr.Flow (orchestrator) bisa LIST +
// TRIGGER Category Task dari chat. Mr.Flow = router: pesan biasa → jawab simpel
// ATAU picu task (deteksi intent via LLM + daftar task yang ada).
//
// Tool jalan HOST-side → call endpoint taskflow lokal (loopback). task_run
// ASYNC (balik run_id cepet); hasil di GUI Tasks / Telegram notify (Fase 6c).
//
// CAPABILITY: rpc:taskflow (cuma agent yang di-grant — mis. Mr.Flow — boleh
// picu task; worker biasa ENGGAK, biar ga ada loop trigger).

package builtins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

// selfBaseURL — base URL server sendiri (loopback). Override via env.
func selfBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("FLOWORK_SELF_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:1987"
}

var taskflowClient = &http.Client{Timeout: 15 * time.Second}

// =============================================================================
// task_list — daftar Category Task yang tersedia
// =============================================================================

type taskListTool struct{}

func (taskListTool) Name() string       { return "task_list" }
func (taskListTool) Capability() string { return "rpc:taskflow" }
func (taskListTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Daftar Category Task (analisa multi-agent) yang tersedia. PAKAI ini buat tau task apa aja yang bisa di-trigger sebelum task_run.",
		Params:      nil,
		Returns:     "{count, tasks:[{id,name,trigger_hint,crew_size}]}",
	}
}
func (taskListTool) Run(ctx context.Context, _ map[string]any) (tools.Result, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, selfBaseURL()+"/api/taskflow/categories", nil)
	resp, err := taskflowClient.Do(req)
	if err != nil {
		return tools.Result{}, fmt.Errorf("task_list fetch: %w", err)
	}
	defer resp.Body.Close()
	var parsed struct {
		Categories []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			TriggerHint string `json:"trigger_hint"`
			Enabled     bool   `json:"enabled"`
		} `json:"categories"`
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if json.Unmarshal(body, &parsed) != nil {
		return tools.Result{}, fmt.Errorf("task_list decode")
	}
	out := make([]map[string]any, 0, len(parsed.Categories))
	for _, c := range parsed.Categories {
		if !c.Enabled {
			continue
		}
		out = append(out, map[string]any{
			"id": c.ID, "name": c.Name, "trigger_hint": c.TriggerHint,
		})
	}
	return tools.Result{Output: map[string]any{"count": len(out), "tasks": out}}, nil
}

// =============================================================================
// task_run — trigger 1 Category Task (async)
// =============================================================================

type taskRunTool struct{}

func (taskRunTool) Name() string       { return "task_run" }
func (taskRunTool) Capability() string { return "rpc:taskflow" }
func (taskRunTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Trigger Category Task (crew analis multi-agent → 1 keputusan). ASYNC: balik run_id langsung, hasil diproses di belakang (~beberapa menit). Kasih tau user lagi diproses + run_id. Cek task_list dulu buat id yang valid.",
		Params: []tools.Param{
			{Name: "category", Type: tools.ParamString, Description: "id kategori task (dari task_list, mis. 'saham')", Required: true},
			{Name: "subject", Type: tools.ParamString, Description: "subjek analisa (mis. 'BBCA')", Required: true},
			{Name: "group", Type: tools.ParamString, Description: "INTERNAL: id GROUP buat delegasi async (gantiin category). Jarang dipake LLM — orchestrator yang isi.", Required: false},
		},
		Returns: "{run_id, status, note}",
	}
}
func (taskRunTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	category, _ := args["category"].(string)
	group, _ := args["group"].(string)
	subject, _ := args["subject"].(string)
	category = strings.TrimSpace(category)
	group = strings.TrimSpace(group)
	subject = strings.TrimSpace(subject)
	if subject == "" || (category == "" && group == "") {
		return tools.Result{}, fmt.Errorf("subject + (category atau group) wajib")
	}
	q := url.Values{}
	if group != "" {
		// OPS-1: delegasi GROUP async (member+synth+debate di belakang). Kalau dua-duanya
		// keisi, group menang (delegasi grup eksplisit).
		q.Set("group", group)
	} else {
		q.Set("category", category)
	}
	q.Set("subject", subject)
	// notify_chat_id di-inject engine (Fase 6c) kalau dari Telegram — biar hasil
	// dikirim balik ke chat pas kelar. Opsional.
	if nc, _ := args["notify_chat_id"].(string); strings.TrimSpace(nc) != "" {
		q.Set("notify", strings.TrimSpace(nc))
	}
	u := selfBaseURL() + "/api/taskflow/run?" + q.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	resp, err := taskflowClient.Do(req)
	if err != nil {
		return tools.Result{}, fmt.Errorf("task_run trigger: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	if e, ok := parsed["error"].(string); ok && e != "" {
		return tools.Result{}, fmt.Errorf("task_run: %s", e)
	}
	return tools.Result{
		Output: map[string]any{
			"run_id": parsed["run_id"],
			"status": "running",
			"note":   "Task lagi diproses di belakang (~beberapa menit). Bilang ke user lagi dianalisa + sebut run_id; hasil muncul di GUI Tasks / dikabarin kalau dari Telegram.",
		},
	}, nil
}
