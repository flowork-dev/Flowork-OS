package triggers

import (
	"testing"
	"time"
)

func TestWorklogPendingTrigger(t *testing.T) {
	wt := &worklogPendingType{}

	// reader nil → fail-safe, ga fire
	WorklogPendingReader = nil
	if ev, _, _ := wt.Check(nil, ""); len(ev) != 0 {
		t.Fatal("nil reader should not fire")
	}

	// pending 0 → ga fire, state reset
	WorklogPendingReader = func() int { return 0 }
	if ev, st, _ := wt.Check(nil, "2020-01-01T00:00:00Z"); len(ev) != 0 || st != "" {
		t.Fatalf("0 pending should not fire + reset state, got ev=%d st=%q", len(ev), st)
	}

	// pending > 0 + no prior state → fire
	WorklogPendingReader = func() int { return 3 }
	ev, st, _ := wt.Check(nil, "")
	if len(ev) != 1 || ev[0].Payload["pending"] != "3" {
		t.Fatalf("3 pending should fire with payload, got %+v", ev)
	}
	if st == "" {
		t.Fatal("fire should set state timestamp")
	}

	// masih cooldown (fire barusan) → ga fire lagi
	recent := time.Now().Add(-1 * time.Minute).Format(time.RFC3339)
	if ev, _, _ := wt.Check(map[string]string{"cooldown_min": "30"}, recent); len(ev) != 0 {
		t.Fatal("within cooldown should not fire")
	}

	// cooldown lewat → fire lagi
	old := time.Now().Add(-40 * time.Minute).Format(time.RFC3339)
	if ev, _, _ := wt.Check(map[string]string{"cooldown_min": "30"}, old); len(ev) != 1 {
		t.Fatal("past cooldown should fire")
	}

	WorklogPendingReader = nil // cleanup
}
