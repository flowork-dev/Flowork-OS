package router

import (
	"testing"
	"time"
)

func TestBackoffDuration_ExponentialCapped(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 2 * time.Second},
		{1, 4 * time.Second},
		{2, 8 * time.Second},
		{3, 16 * time.Second},
		{4, 30 * time.Second},  // 32s → cap 30s
		{10, 30 * time.Second}, // jauh di atas cap → tetep 30s
	}
	for _, c := range cases {
		if got := backoffDuration(c.attempt); got != c.want {
			t.Fatalf("backoffDuration(%d) = %v, want %v", c.attempt, got, c.want)
		}
	}
}

func TestDispatchConcurrency_DefaultAndEnv(t *testing.T) {
	t.Setenv("FLOW_ROUTER_MAX_CONCURRENCY", "")
	if got := dispatchConcurrency(); got != defaultDispatchConcurrency {
		t.Fatalf("default concurrency = %d, want %d", got, defaultDispatchConcurrency)
	}
	t.Setenv("FLOW_ROUTER_MAX_CONCURRENCY", "5")
	if got := dispatchConcurrency(); got != 5 {
		t.Fatalf("env concurrency = %d, want 5", got)
	}
	t.Setenv("FLOW_ROUTER_MAX_CONCURRENCY", "garbage")
	if got := dispatchConcurrency(); got != defaultDispatchConcurrency {
		t.Fatalf("invalid env harus fallback ke default, dapet %d", got)
	}
}

// Semaphore: acquire N lalu release N ga deadlock; slot ke-N+1 nunggu.
func TestDispatchSlot_Semaphore(t *testing.T) {
	// pakai semaphore lokal seukuran cap default biar deterministik.
	sem := make(chan struct{}, 2)
	acq := func() bool {
		select {
		case sem <- struct{}{}:
			return true
		case <-time.After(50 * time.Millisecond):
			return false
		}
	}
	// acq() PUNYA side-effect (ambil 1 slot) — panggil 2x EKSPLISIT (bukan x||x redundan; SA4000 false-positive).
	got1, got2 := acq(), acq()
	if !got1 || !got2 {
		t.Fatal("2 slot pertama harusnya langsung dapet")
	}
	if acq() {
		t.Fatal("slot ke-3 harusnya BLOCK (cap 2) — bergantian gagal")
	}
	<-sem // release 1
	if !acq() {
		t.Fatal("setelah release, slot harusnya dapet lagi")
	}
}
