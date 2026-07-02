// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN — jangan edit file ini. 📄 Dok: lock/tools.md
// Nambah/ubah tool TANPA buka frozen: pakai SEAM tools.Register() di file sibling non-frozen.
// agent_run.go — P5 final lifecycle: durable run state (checkpoint / pause / resume / stop).
//
// The coordinator substrate already does parallel + bounded fan-out (loket). What was
// missing for a real lifecycle is a DURABLE record a long task can be paused and RESUMED
// against:
//
//   - resume → a paused/interrupted task persists a checkpoint; resume hands that
//     checkpoint back so the agent continues from where it left off instead of restarting.
//   - stop → marks a run terminal so it is NOT resumed (resume on a stopped run returns
//     "stopped" instead of reviving it).
//
// SCOPE (important — no overclaim): the registry lives in the CALLING agent's own store
// (tools.FromStore). Stores are per-agent, so this is durable lifecycle for an agent's
// OWN long task, and for a COORDINATOR (e.g. mr-flow) it is the coordinator's run ledger.
// In the kernel's SYNCHRONOUS invoke model a colony member runs to completion within one
// Call — it does not poll a shared table mid-run. So a coordinator enforces "stop" by (1)
// the bounded fan-out abandoning a stuck member (loket.ParallelFanout) and (2) simply not
// re-invoking a stopped run — NOT by members reading each other's stores. (A future shared
// cross-agent registry, if ever needed, would be a separate coordination store, not this.)
//
// Plug-and-play + lock-respecting: a NEW builtin tool with its OWN table (agent_runs) on
// the agent's own store — no kernel unlock, no change to any frozen/locked file. State
// machine: pending → running → (paused ⇄ running) → done | stopped. Subscribed to mr-flow
// (the coordinator) only — NOT in coreExposedTools, so ants stay tiny (isolation intact).
package builtins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

func init() { tools.Register(&agentRunTool{}) }

type agentRunTool struct{}

func (agentRunTool) Name() string       { return "agent_run" }
func (agentRunTool) Capability() string { return "state:write" }
func (agentRunTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Durable lifecycle for YOUR long task: checkpoint progress, then pause/resume/stop so it survives across turns. Actions: create|start|checkpoint|pause|resume|stop|complete|status|list. 'resume' returns the saved checkpoint so you continue where you left off; 'stop' marks the run terminal so it is not resumed. Offline, scoped to this agent's own store (a coordinator's run ledger).",
		Params: []tools.Param{
			{Name: "action", Type: tools.ParamString, Description: "create|start|checkpoint|pause|resume|stop|complete|status|list", Required: true},
			{Name: "id", Type: tools.ParamString, Description: "run id (required for all but list)"},
			{Name: "label", Type: tools.ParamString, Description: "human label (create only)"},
			{Name: "data", Type: tools.ParamObject, Description: "checkpoint payload (checkpoint/complete)"},
		},
		Returns: "{id, state, checkpoint?, runs?}",
	}
}

func (agentRunTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent_run: store not in context")
	}
	db := store.DB()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS agent_runs (
		id TEXT PRIMARY KEY, label TEXT, state TEXT NOT NULL DEFAULT 'pending',
		checkpoint TEXT, updated TEXT)`); err != nil {
		return tools.Result{}, fmt.Errorf("agent_run schema: %w", err)
	}

	action, _ := args["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))
	id, _ := args["id"].(string)
	now := time.Now().UTC().Format(time.RFC3339)

	out := func(state, checkpoint string) (tools.Result, error) {
		m := map[string]any{"id": id, "state": state}
		if checkpoint != "" {
			var cp any
			if json.Unmarshal([]byte(checkpoint), &cp) == nil {
				m["checkpoint"] = cp
			}
		}
		return tools.Result{Output: m}, nil
	}
	setState := func(state string) (tools.Result, error) {
		r, err := db.Exec("UPDATE agent_runs SET state=?, updated=? WHERE id=?", state, now, id)
		if err != nil {
			return tools.Result{}, err
		}
		if n, _ := r.RowsAffected(); n == 0 {
			return tools.Result{}, fmt.Errorf("agent_run: no run %q", id)
		}
		return out(state, "")
	}
	marshalData := func() (string, error) {
		if d, ok := args["data"]; ok && d != nil {
			b, err := json.Marshal(d)
			return string(b), err
		}
		return "", nil
	}

	switch action {
	case "create":
		if id == "" {
			return tools.Result{}, fmt.Errorf("agent_run create: id required")
		}
		label, _ := args["label"].(string)
		if _, err := db.Exec(
			"INSERT OR REPLACE INTO agent_runs (id,label,state,updated) VALUES (?,?, 'pending', ?)",
			id, label, now); err != nil {
			return tools.Result{}, err
		}
		return out("pending", "")
	case "start", "resume":
		// resume returns the saved checkpoint so the agent continues from it.
		var state, checkpoint string
		err := db.QueryRow("SELECT state, COALESCE(checkpoint,'') FROM agent_runs WHERE id=?", id).Scan(&state, &checkpoint)
		if err != nil {
			return tools.Result{}, fmt.Errorf("agent_run %s: no run %q", action, id)
		}
		if state == "stopped" || state == "done" {
			// terminal: don't silently revive — report so the caller decides.
			return out(state, checkpoint)
		}
		if _, err := db.Exec("UPDATE agent_runs SET state='running', updated=? WHERE id=?", now, id); err != nil {
			return tools.Result{}, err
		}
		return out("running", checkpoint)
	case "checkpoint":
		data, err := marshalData()
		if err != nil {
			return tools.Result{}, fmt.Errorf("agent_run checkpoint data: %w", err)
		}
		r, err := db.Exec("UPDATE agent_runs SET checkpoint=?, updated=? WHERE id=?", data, now, id)
		if err != nil {
			return tools.Result{}, err
		}
		if n, _ := r.RowsAffected(); n == 0 {
			return tools.Result{}, fmt.Errorf("agent_run checkpoint: no run %q", id)
		}
		return out("running", data)
	case "pause":
		return setState("paused")
	case "stop":
		return setState("stopped")
	case "complete":
		if data, err := marshalData(); err == nil && data != "" {
			_, _ = db.Exec("UPDATE agent_runs SET checkpoint=? WHERE id=?", data, id)
		}
		return setState("done")
	case "status":
		var state, checkpoint string
		if err := db.QueryRow("SELECT state, COALESCE(checkpoint,'') FROM agent_runs WHERE id=?", id).Scan(&state, &checkpoint); err != nil {
			return tools.Result{}, fmt.Errorf("agent_run status: no run %q", id)
		}
		return out(state, checkpoint)
	case "list":
		rows, err := db.Query("SELECT id, label, state, updated FROM agent_runs ORDER BY updated DESC LIMIT 200")
		if err != nil {
			return tools.Result{}, err
		}
		defer rows.Close()
		var runs []map[string]any
		for rows.Next() {
			var rid, label, state, updated string
			_ = rows.Scan(&rid, &label, &state, &updated)
			runs = append(runs, map[string]any{"id": rid, "label": label, "state": state, "updated": updated})
		}
		return tools.Result{Output: map[string]any{"runs": runs}}, nil
	default:
		return tools.Result{}, fmt.Errorf("agent_run: unknown action %q", action)
	}
}
