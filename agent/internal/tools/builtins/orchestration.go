// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/tools"
)

const (
	keyPlan = "_plan"
	keyTodo = "_todo"
	keyGoal = "_goal"
)

type planReadTool struct{}

func (planReadTool) Name() string       { return "plan_read" }
func (planReadTool) Capability() string { return "state:read" }
func (planReadTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Read current agent plan (markdown). Empty kalau belum ada.",
		Params:      nil,
		Returns:     "{plan: <markdown>, updated_at}",
	}
}
func (planReadTool) Run(ctx context.Context, _ map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	v, _, err := store.GetToolMemory(keyPlan)
	if err != nil {
		return tools.Result{}, err
	}
	plan := ""
	updatedAt := ""
	if v != "" {
		var entry struct {
			Plan      string `json:"plan"`
			UpdatedAt string `json:"updated_at"`
		}
		if uerr := json.Unmarshal([]byte(v), &entry); uerr == nil {
			plan = entry.Plan
			updatedAt = entry.UpdatedAt
		} else {
			plan = v
		}
	}
	return tools.Result{Output: map[string]any{
		"plan":       plan,
		"updated_at": updatedAt,
	}}, nil
}

type planWriteTool struct{}

func (planWriteTool) Name() string       { return "plan_write" }
func (planWriteTool) Capability() string { return "state:write" }
func (planWriteTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Overwrite plan with new markdown. Body cap 32KB. Append-only history NOT kept (phase 1g simple — phase 2 add plan_revisions).",
		Params: []tools.Param{
			{Name: "plan", Type: tools.ParamString, Description: "markdown plan body", Required: true},
		},
		Returns: "{ok: true, length}",
	}
}
func (planWriteTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	plan, _ := args["plan"].(string)
	if plan == "" {
		return tools.Result{}, fmt.Errorf("plan required (string non-empty)")
	}
	if len(plan) > 32*1024 {
		return tools.Result{}, fmt.Errorf("plan too large (cap 32KB)")
	}
	entry := map[string]any{
		"plan":       plan,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(entry)
	if err := store.SetToolMemory(keyPlan, string(b)); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{
		"ok":     true,
		"length": len(plan),
	}}, nil
}

type todoTool struct{}

func (todoTool) Name() string       { return "todo" }
func (todoTool) Capability() string { return "state:write" }
func (todoTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Manage agent todo list. Operations: list | add | done | remove | clear. Item dict: {id, content, done, added_at, done_at}.",
		Params: []tools.Param{
			{Name: "op", Type: tools.ParamString, Description: "list | add | done | remove | clear", Required: true},
			{Name: "content", Type: tools.ParamString, Description: "for op=add: todo content"},
			{Name: "id", Type: tools.ParamString, Description: "for op=done/remove: todo id"},
		},
		Returns: "{items: [...], count}",
	}
}

type todoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Done    bool   `json:"done"`
	AddedAt string `json:"added_at"`
	DoneAt  string `json:"done_at,omitempty"`
}

func loadTodos(store *agentdb.Store) ([]todoItem, error) {
	v, _, err := store.GetToolMemory(keyTodo)
	if err != nil {
		return nil, err
	}
	if v == "" {
		return []todoItem{}, nil
	}
	var items []todoItem
	if jerr := json.Unmarshal([]byte(v), &items); jerr != nil {
		return []todoItem{}, nil
	}
	return items, nil
}

func saveTodos(store *agentdb.Store, items []todoItem) error {
	b, _ := json.Marshal(items)
	return store.SetToolMemory(keyTodo, string(b))
}

func nextTodoID(items []todoItem) string {
	maxN := 0
	for _, it := range items {
		var n int
		_, _ = fmt.Sscanf(it.ID, "t%d", &n)
		if n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("t%d", maxN+1)
}

func (todoTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	op, _ := args["op"].(string)
	op = strings.ToLower(strings.TrimSpace(op))
	items, err := loadTodos(store)
	if err != nil {
		return tools.Result{}, err
	}

	switch op {
	case "list", "":

	case "add":
		content, _ := args["content"].(string)
		content = strings.TrimSpace(content)
		if content == "" {
			return tools.Result{}, fmt.Errorf("content required for op=add")
		}
		if len(content) > 4096 {
			return tools.Result{}, fmt.Errorf("content too long (cap 4KB)")
		}
		items = append(items, todoItem{
			ID:      nextTodoID(items),
			Content: content,
			AddedAt: time.Now().UTC().Format(time.RFC3339),
		})
		if err := saveTodos(store, items); err != nil {
			return tools.Result{}, err
		}
	case "done":
		id, _ := args["id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			return tools.Result{}, fmt.Errorf("id required for op=done")
		}
		found := false
		for i := range items {
			if items[i].ID == id {
				items[i].Done = true
				items[i].DoneAt = time.Now().UTC().Format(time.RFC3339)
				found = true
				break
			}
		}
		if !found {
			return tools.Result{}, fmt.Errorf("todo id %q not found", id)
		}
		if err := saveTodos(store, items); err != nil {
			return tools.Result{}, err
		}
	case "remove":
		id, _ := args["id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			return tools.Result{}, fmt.Errorf("id required for op=remove")
		}
		next := items[:0]
		for _, it := range items {
			if it.ID != id {
				next = append(next, it)
			}
		}
		if len(next) == len(items) {
			return tools.Result{}, fmt.Errorf("todo id %q not found", id)
		}
		items = next
		if err := saveTodos(store, items); err != nil {
			return tools.Result{}, err
		}
	case "clear":
		items = []todoItem{}
		if err := saveTodos(store, items); err != nil {
			return tools.Result{}, err
		}
	default:
		return tools.Result{}, fmt.Errorf("unknown op %q (use: list|add|done|remove|clear)", op)
	}

	return tools.Result{Output: map[string]any{
		"items": items,
		"count": len(items),
	}}, nil
}

type goalDoneTool struct{}

func (goalDoneTool) Name() string       { return "goal_done" }
func (goalDoneTool) Capability() string { return "state:write" }
func (goalDoneTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Mark goal as done. Optional summary. Append to goal log di tool_memory[_goal] (array of {summary, done_at}). Limit 20 entries (oldest dropped).",
		Params: []tools.Param{
			{Name: "summary", Type: tools.ParamString, Description: "optional outcome summary"},
		},
		Returns: "{ok: true, total_done}",
	}
}
func (goalDoneTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	summary, _ := args["summary"].(string)
	if len(summary) > 4096 {
		summary = summary[:4096] + "...[trunc]"
	}

	v, _, err := store.GetToolMemory(keyGoal)
	if err != nil {
		return tools.Result{}, err
	}
	var log []map[string]string
	if v != "" {
		_ = json.Unmarshal([]byte(v), &log)
	}
	log = append(log, map[string]string{
		"summary": summary,
		"done_at": time.Now().UTC().Format(time.RFC3339),
	})

	if len(log) > 20 {
		log = log[len(log)-20:]
	}
	b, _ := json.Marshal(log)
	if err := store.SetToolMemory(keyGoal, string(b)); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{
		"ok":         true,
		"total_done": len(log),
	}}, nil
}

// SELF-REGISTER plug-in (📄 lock/tool-manager.md) — edit/hapus/tambah BEBAS tanpa unfreeze.
func init() {
	tools.Register(&planReadTool{})
	tools.Register(&planWriteTool{})
	tools.Register(&todoTool{})
	tools.Register(&goalDoneTool{})
}
