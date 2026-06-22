package main

import (
	"sync"
	"testing"
)

// Race-guard E: tryAcquire kedua HARUS gagal (agent udah busy) → cegah 2 bg-task agent-sama
// paralel; release balikin idle.
func TestAgentBusySet(t *testing.T) {
	s := newAgentBusySet()
	if s.isBusy("a") {
		t.Fatal("agent baru harus idle")
	}
	if !s.tryAcquire("a") {
		t.Fatal("acquire pertama harus sukses")
	}
	if !s.isBusy("a") {
		t.Fatal("setelah acquire harus busy")
	}
	if s.tryAcquire("a") {
		t.Fatal("acquire KEDUA harus GAGAL (agent busy) — ini inti race-guard")
	}
	// Agent beda tetep boleh (lintas-agent paralel).
	if !s.tryAcquire("b") {
		t.Fatal("agent beda harus bisa acquire (paralel lintas-agent)")
	}
	s.release("a")
	if s.isBusy("a") {
		t.Fatal("setelah release harus idle")
	}
	if !s.tryAcquire("a") {
		t.Fatal("setelah release harus bisa acquire lagi")
	}
}

// Konkuren: dari N goroutine tryAcquire id sama, TEPAT 1 yang menang (mutual exclusion).
func TestAgentBusySet_Concurrent(t *testing.T) {
	s := newAgentBusySet()
	const N = 50
	var wins int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.tryAcquire("x") {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("winners=%d want 1 (mutual exclusion bocor)", wins)
	}
}
