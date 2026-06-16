package builtins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTaskRunRouting — OPS-1: task_run harus forward param yang BENER ke
// /api/taskflow/run. `group` (delegasi GROUP async) gantiin `category`; kalau dua-duanya
// keisi, group menang. Tanpa keduanya = error. Pakai httptest (isolated, gak nyentuh host live).
func TestTaskRunRouting(t *testing.T) {
	var gotQuery map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotQuery = map[string]string{
			"category": q.Get("category"),
			"group":    q.Get("group"),
			"subject":  q.Get("subject"),
			"notify":   q.Get("notify"),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"run_id":7,"status":"running"}`))
	}))
	defer srv.Close()
	t.Setenv("FLOWORK_SELF_URL", srv.URL)

	tool := taskRunTool{}

	// (1) GROUP delegation: group → q.group set, q.category KOSONG.
	res, err := tool.Run(context.Background(), map[string]any{
		"group": "thinking", "subject": "pikirin X", "notify_chat_id": "123",
	})
	if err != nil {
		t.Fatalf("group run error: %v", err)
	}
	if gotQuery["group"] != "thinking" || gotQuery["category"] != "" {
		t.Fatalf("group path: q=%v (mau group=thinking, category kosong)", gotQuery)
	}
	if gotQuery["subject"] != "pikirin X" || gotQuery["notify"] != "123" {
		t.Fatalf("group path subject/notify salah: q=%v", gotQuery)
	}
	if out, _ := res.Output.(map[string]any); out["run_id"] == nil {
		t.Fatalf("group path harus balik run_id, got %v", res.Output)
	}

	// (2) CATEGORY (perilaku lama, NOL regresi): category → q.category set, q.group KOSONG.
	if _, err := tool.Run(context.Background(), map[string]any{
		"category": "saham", "subject": "BBCA",
	}); err != nil {
		t.Fatalf("category run error: %v", err)
	}
	if gotQuery["category"] != "saham" || gotQuery["group"] != "" {
		t.Fatalf("category path: q=%v (mau category=saham, group kosong)", gotQuery)
	}

	// (3) group menang kalau dua-duanya keisi (delegasi grup eksplisit).
	if _, err := tool.Run(context.Background(), map[string]any{
		"category": "saham", "group": "trading", "subject": "Z",
	}); err != nil {
		t.Fatalf("both run error: %v", err)
	}
	if gotQuery["group"] != "trading" || gotQuery["category"] != "" {
		t.Fatalf("both path: group harus menang, q=%v", gotQuery)
	}

	// (4) gak ada category & gak ada group = error (gak boleh nembak endpoint).
	if _, err := tool.Run(context.Background(), map[string]any{"subject": "Y"}); err == nil {
		t.Fatal("tanpa category/group harus error")
	}
	// (5) subject kosong = error.
	if _, err := tool.Run(context.Background(), map[string]any{"group": "thinking"}); err == nil {
		t.Fatal("tanpa subject harus error")
	}
}
