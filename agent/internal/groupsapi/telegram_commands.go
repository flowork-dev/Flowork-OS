// telegram_commands.go — push daftar group LIVE ke menu slash Telegram (setMyCommands).
//
// BUG yang dicabut (owner 2026-06-20): "nambah group → muncul slash di Telegram, TAPI
// nyangkut group LAMA padahal udah dihapus (DB-driven)". Akar: `setMyCommands` cuma ada
// di KOMENTAR — GA PERNAH dipanggil. kv "groups" (sumber live) bener ke-update tiap
// create/delete, tapi ga ada yang push ke Telegram → menu Telegram (sticky) nyimpen
// command lama selamanya.
//
// Fix: tiap SyncToOrchestrator, push daftar command (dari parts live) ke Telegram
// setMyCommands. Group dihapus → parts mengecil → menu auto-nyusut. SEMUA dihapus →
// setMyCommands([]) → menu KOSONG (clear command hantu). DB = kebenaran, sampai ke Telegram.
package groupsapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// syncTelegramCommands — set menu slash Telegram = persis group LIVE (parts =
// "id|command|desc" dari SyncToOrchestrator). Best-effort: ga ada token / Telegram
// down → diam (kv "groups" tetep sumber kebenaran buat ask_group + Mr.Flow). Idempotent.
func (h *Handler) syncTelegramCommands(parts []string) {
	if h.d.TelegramToken == nil {
		return
	}
	token := strings.TrimSpace(h.d.TelegramToken())
	if token == "" {
		return
	}
	// Bangun command dari parts. Kosong → kirim [] → CLEAR command hantu.
	type botCommand struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	cmds := make([]botCommand, 0, len(parts))
	for _, p := range parts {
		seg := strings.SplitN(p, "|", 3)
		if len(seg) < 2 || strings.TrimSpace(seg[1]) == "" {
			continue
		}
		desc := ""
		if len(seg) == 3 {
			desc = strings.TrimSpace(seg[2])
		}
		if desc == "" {
			desc = seg[1]
		}
		if len(desc) > 256 {
			desc = desc[:256]
		}
		cmds = append(cmds, botCommand{Command: seg[1], Description: desc})
	}
	body, _ := json.Marshal(map[string]any{"commands": cmds}) // default scope (= yg dipakai menu lama)
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/setMyCommands", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{Timeout: 8 * time.Second}
	if resp, err := cli.Do(req); err == nil {
		resp.Body.Close()
	}
}
