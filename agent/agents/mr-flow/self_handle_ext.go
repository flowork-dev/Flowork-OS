package main

// self_handle_ext.go — GROWTH-POINT (NON-frozen). Deteksi "owner minta KERJAIN SENDIRI,
// JANGAN delegasi ke crew/agent". main.go (FROZEN) manggil wantsSelfHandle() buat GATE
// routing pre-classifier (deterministicRoute + classifyRoute) — kalau owner eksplisit nolak
// delegasi, SKIP auto-route, lempar ke callLLM (mr-flow kerjain sendiri pakai tool-nya).
//
// AKAR (owner 2026-06-25): "buka facebook loe lakuin sendiri jangan gunakan agent" → mr-flow
// MALAH nyalain crew facebook (run #225). Sebab classifyRoute (LLM pre-route) jalan SEBELUM
// callLLM + langsung task_run, ga peduli instruksi "jangan pake agent". Insting di loop ga
// keburu. Fix di AKAR: gate route-nya. Logika frasa DI SINI (non-frozen) → nambah frasa /
// bahasa baru TANPA buka freeze main.go. Switch: ENV FLOWORK_SELF_HANDLE_PHRASES (comma, nambah).
//
// TinyGo/wasip1-safe: substring match (no regexp).

import (
	"os"
	"strings"
)

// selfHandlePhrases — sinyal owner nolak delegasi (substring, lowercase). Tight biar minim
// false-positive (frasa eksplisit "jangan pake agent/crew" + "lakuin/kerjain sendiri").
var selfHandlePhrases = []string{
	"jangan pake agent", "jangan pakai agent", "jangan gunakan agent", "jangan guna agent",
	"jangan pake crew", "jangan pakai crew", "jangan gunakan crew",
	"jangan pake tim", "jangan pakai tim", "jangan gunakan tim",
	"tanpa agent", "tanpa crew", "tanpa tim", "tanpa delegasi",
	"jangan delegasi", "jangan didelegasi", "jangan di delegasi", "jangan dilempar", "jangan lempar ke",
	"lakuin sendiri", "lakukan sendiri", "kerjain sendiri", "kerjakan sendiri",
	"kamu sendiri yang", "lo sendiri yang", "loe sendiri", "kamu yang kerjain", "lo yang kerjain",
	"jangan task_run", "jangan pake task_run",
}

func init() {
	if env := strings.TrimSpace(os.Getenv("FLOWORK_SELF_HANDLE_PHRASES")); env != "" {
		for _, p := range strings.Split(env, ",") {
			if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
				selfHandlePhrases = append(selfHandlePhrases, p)
			}
		}
	}
}

// systemRelayMarkers — pesan SISTEM/RELAY (notif forwarder) yang HARAM masuk routing pre-classifier.
// AKAR: FlowAlpha notif ("dumb notification forwarder, reply with ONLY the message") ke-MIS-ROUTE
// jadi nyalain crew crypto gara2 keyword (BTCUSDT). Marker ini bikin route di-SKIP → callLLM forward
// notif APA-ADANYA. Substring lowercase, tight (frasa khas relay, minim false-positive).
var systemRelayMarkers = []string{
	"system relay", "[system relay", "not a user request", "dumb notification forwarder",
	"notification forwarder", "reply with only the message", "your entire reply must", "you are a forwarder",
}

// wantsSelfHandle — true kalau pesan HARAM masuk auto-route: owner minta KERJAIN SENDIRI (nolak
// delegasi crew/agent) ATAU pesan = SYSTEM-RELAY/notif (forward apa-adanya). Dipanggil dari main.go
// (frozen) buat GATE routing pre-classifier (deterministicRoute + classifyRoute).
func wantsSelfHandle(text string) bool {
	low := strings.ToLower(text)
	for _, p := range selfHandlePhrases {
		if strings.Contains(low, p) {
			return true
		}
	}
	for _, p := range systemRelayMarkers {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}
