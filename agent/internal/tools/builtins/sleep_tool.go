// sleep_tool.go — F-C: tool `sleep` SEJATI (tidur tanpa nahan proses).
// 📄 Dok: FLowork_os/lock/sleep-tool.md
//
// Kernel WASM sinkron: "tidur" TIDAK boleh nge-block proses (block = 1 semut
// nyandera engine + timeout turn ke-kill). `sleep` di sini = tulis 1 baris
// `wakeups` (TABEL + engine RunDueWakeups yang SAMA dgn ScheduleWakeup — nol
// mesin baru) lalu balik SINYAL "turn boleh diakhiri": agent bangun otomatis
// pas jatuh tempo. Beda dari ScheduleWakeup: default prompt bangunnya = "cek
// papan kerja / kerjaan pending SEBELUM tidur lagi" (tick cari-kerjaan), dan
// return-nya eksplisit `end_turn:true` biar loop berhenti bersih.
//
// NON-FROZEN sibling (deletable): dihapus → tool ilang, engine wakeup + agent
// tetep jalan (prinsip switch).
package builtins

import (
	"context"
	"fmt"
	"time"

	"flowork-gui/internal/tools"
)

func init() { tools.Register(&sleepTool{}) }

type sleepTool struct{}

func (sleepTool) Name() string       { return "sleep" }
func (sleepTool) Capability() string { return "state:write" }
func (sleepTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Tidur SEJATI tanpa nahan proses: jadwalin bangun otomatis setelah `seconds` lalu AKHIRI turn ini (jangan nunggu sinkron — itu nyandera engine). Pas bangun, kamu di-fire ulang buat cek kerjaan. Pakai buat idle hemat / nunggu lama / jeda antar-siklus. Buat nunggu KONDISI singkat (file muncul, proses ready) pakai Monitor; buat lanjutan tugas spesifik pakai ScheduleWakeup.",
		Params: []tools.Param{
			{Name: "seconds", Type: tools.ParamInt, Description: "lama tidur (detik, >0)", Required: true},
			{Name: "reason", Type: tools.ParamString, Description: "kenapa tidur (telemetry, opsional)"},
			{Name: "then", Type: tools.ParamString, Description: "prompt saat bangun (opsional; default: cek papan kerja / kerjaan pending dulu)"},
		},
		Returns: "{sleeping:true, end_turn:true, wakeup_id, wake_at, seconds}",
	}
}

func (sleepTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("sleep: store not in context")
	}
	secs, ok := argInt(args, "seconds")
	if !ok || secs <= 0 {
		return tools.Result{}, fmt.Errorf("seconds required (positive int)")
	}
	reason := argStr(args, "reason")
	if reason == "" {
		reason = "idle sleep"
	}
	then := argStr(args, "then")
	if then == "" {
		// Tick "cari kerjaan sebelum tidur (lagi)": pas bangun, cek dulu ada
		// tugas nyangkut/pending sebelum balik idle — biar tidur ga nutupin kerjaan.
		then = "[BANGUN dari sleep] Cek dulu: ada kerjaan pending / tugas nyangkut / " +
			"pesan owner yang belum dibales? Kalau ADA → kerjain. Kalau BENERAN ga ada → " +
			"boleh tidur lagi (sleep) atau diam."
	}
	db := store.DB()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS wakeups (
		id TEXT PRIMARY KEY, due_unix INTEGER NOT NULL, prompt TEXT, reason TEXT,
		fired INTEGER NOT NULL DEFAULT 0, created TEXT)`); err != nil {
		return tools.Result{}, fmt.Errorf("sleep schema: %w", err)
	}
	now := time.Now().UTC()
	due := now.Add(time.Duration(secs) * time.Second)
	id := fmt.Sprintf("sleep-%d", now.UnixNano())
	if _, err := db.Exec(
		"INSERT INTO wakeups (id,due_unix,prompt,reason,fired,created) VALUES (?,?,?,?,0,?)",
		id, due.Unix(), then, "😴 "+reason, now.Format(time.RFC3339)); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{
		"sleeping":  true,
		"end_turn":  true, // sinyal ke loop: berhenti bersih, jangan lanjut iterasi
		"wakeup_id": id,
		"wake_at":   due.Format(time.RFC3339),
		"seconds":   secs,
	}}, nil
}
