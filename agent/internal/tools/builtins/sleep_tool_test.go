package builtins

import (
	"context"
	"path/filepath"
	"testing"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/tools"
)

func TestSleepTool_SchedulesWakeupRow(t *testing.T) {
	st, err := agentdb.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := tools.WithStore(context.Background(), st)

	res, err := (sleepTool{}).Run(ctx, map[string]any{"seconds": float64(30), "reason": "tes idle"})
	if err != nil {
		t.Fatalf("sleep: %v", err)
	}
	out := res.Output.(map[string]any)
	if out["sleeping"] != true || out["end_turn"] != true {
		t.Fatalf("harus sinyal sleeping+end_turn: %v", out)
	}

	// Baris wakeup HARUS ada, belum fired, prompt bangun keisi (tick cari-kerjaan).
	var n int
	var prompt, reason string
	row := st.DB().QueryRow(
		"SELECT COUNT(*), COALESCE(MAX(prompt),''), COALESCE(MAX(reason),'') FROM wakeups WHERE fired=0")
	if err := row.Scan(&n, &prompt, &reason); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("mau 1 wakeup pending, dapet %d", n)
	}
	if prompt == "" {
		t.Error("prompt bangun default harus keisi (tick cari-kerjaan)")
	}
	if reason == "" {
		t.Error("reason harus keisi")
	}
}

func TestSleepTool_Validation(t *testing.T) {
	st, _ := agentdb.Open(filepath.Join(t.TempDir(), "state.db"))
	defer st.Close()
	ctx := tools.WithStore(context.Background(), st)

	if _, err := (sleepTool{}).Run(ctx, map[string]any{"seconds": float64(0)}); err == nil {
		t.Error("seconds=0 harus error")
	}
	if _, err := (sleepTool{}).Run(context.Background(), map[string]any{"seconds": float64(5)}); err == nil {
		t.Error("tanpa store harus error")
	}
}
