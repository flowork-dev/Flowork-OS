// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// type_worklog.go — F-D (roadmap Sistem Saraf Otonom): TRIGGER TIPE BARU "ada kerjaan
// NYANGKUT di papan kerja (worklog)". Pemicu MANDOR by-KERJAAN, bukan cuma PC-idle.
//
// Lock-respecting: file BARU non-frozen, daftar via seam `Register` (triggers.go FROZEN utuh).
// Reader di-wire dari package main (feature_worklog_trigger.go) via seam var WorklogPendingReader
// (POLA B var-default). Fail-safe: reader nil / 0 → TIDAK fire. Dok: lock/worklog.md
package triggers

import (
	"strconv"
	"time"
)

func init() { Register(&worklogPendingType{}) }

// WorklogPendingReader — SEAM: balikin jumlah task NYANGKUT (stale) di papan lintas-agent.
// nil = fitur worklog off / belum di-wire → 0 (fail-safe no-fire). Di-set feature non-frozen.
var WorklogPendingReader func() int

type worklogPendingType struct{}

func (t *worklogPendingType) ID() string            { return "worklog-pending" }
func (t *worklogPendingType) Name() string          { return "Ada kerjaan nyangkut di papan (worklog)" }
func (t *worklogPendingType) Mode() string          { return "poll" }
func (t *worklogPendingType) PayloadKeys() []string { return []string{"pending"} }
func (t *worklogPendingType) ConfigSchema() []Field {
	return []Field{
		{Key: "cooldown_min", Label: "Cooldown (menit)", Type: "text", Default: "30", Required: false,
			Help: "jeda minimal antar-fire selama masih ada kerjaan nyangkut (default 30, anti-spam)."},
	}
}
func (t *worklogPendingType) OnWebhook(_ map[string]string, _ []byte) ([]Event, error) { return nil, nil }

func (t *worklogPendingType) Check(cfg map[string]string, state string) ([]Event, string, error) {
	if WorklogPendingReader == nil {
		return nil, state, nil // belum di-wire / worklog off → fail-safe (diem)
	}
	pending := WorklogPendingReader()
	if pending <= 0 {
		return nil, "", nil // ga ada yg nyangkut → reset state (fire instan pas ada lagi)
	}
	cooldown := time.Duration(int(parseF(cfg["cooldown_min"], 30))) * time.Minute
	now := time.Now()
	if last := parseTime(state); !last.IsZero() && now.Sub(last) < cooldown {
		return nil, state, nil // masih cooldown → jangan spam mandor
	}
	ev := Event{Key: now.Format(time.RFC3339), Payload: map[string]string{"pending": strconv.Itoa(pending)}}
	return []Event{ev}, now.Format(time.RFC3339), nil
}
