package main

// sched_honest_ext.go — FROZEN (chattr +i + hash KERNEL_FREEZE.md). Anti-ghosting AKAR: pas harness
// auto-continue manggil ScheduleWakeup TAPI gagal (mis. guardian SAFE-MODE blokir state:write), harness
// JANGAN boong "udah dijadwalin" (itu ghost). schedFailReason kasih alasan ringkas + recoverable.
// TinyGo-safe (substring, no regexp). Dipakai main.go (auto-continue). 📄 Dok: lock/ERROR_EDUKASI.md.

import "strings"

// schedFailReason — alasan ringkas kenapa ScheduleWakeup gagal (dari string hasil runTool).
func schedFailReason(res string) string {
	low := strings.ToLower(res)
	switch {
	case strings.Contains(low, "guardian") || strings.Contains(low, "safe-mode") || strings.Contains(low, "safe mode"):
		return "guardian SAFE-MODE blokir tool — owner: restart agent / disarm guardian dulu"
	case strings.Contains(low, "dispatch gagal") || strings.Contains(low, "tool http"):
		return "tool dispatch error (router/host)"
	default:
		return "penjadwalan gagal"
	}
}
