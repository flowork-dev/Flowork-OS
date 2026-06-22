// recall_gate.go — N1-C: GATE auto-recall buat pesan TRIVIAL (sapaan/ack/filler).
// Di-EKSTRAK dari main.go (pola nano-modular spt recovery_capture.go): brain-logic =
// file terpisah FROZEN, main.go = wiring EDITABLE. fetchAutoRecall() panggil
// isTrivialChat(q) → kalau true, recall di-SKIP (hemat 2 tool-call graph+brain/turn).
// KONSERVATIF: 1 kata substantif aja matahin gate → query identitas/relasi tetap recall.
//
// Owner: Mr.Dev · github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN brain-core — lihat lock/brain.md §7 (AUTO-RECALL gate). Unfreeze dulu buat edit.

package main

import "strings"

// trivialChatTokens — kata sapaan/ack/filler yang ZERO nilai buat di-recall.
// Pesan yang SEMUA token-nya ada di sini ga butuh fakta memori → recall di-skip.
var trivialChatTokens = map[string]bool{
	// sapaan
	"halo": true, "hallo": true, "hai": true, "hi": true, "hello": true, "hey": true,
	"hei": true, "woi": true, "oi": true, "pagi": true, "siang": true, "sore": true,
	"malam": true, "selamat": true,
	// terima kasih
	"makasih": true, "makasi": true, "thanks": true, "thank": true, "thx": true,
	"trims": true, "terima": true, "kasih": true, "ty": true, "tq": true,
	// apresiasi / setuju
	"mantap": true, "mantul": true, "keren": true, "sip": true, "oke": true, "ok": true,
	"okay": true, "okey": true, "yoi": true, "noted": true, "baik": true, "beres": true,
	"gas": true, "siap": true, "iya": true, "ya": true, "yup": true, "yep": true,
	"yes": true, "betul": true, "bener": true, "good": true, "nice": true,
	// filler / partikel
	"bro": true, "bre": true, "bang": true, "min": true, "dong": true, "deh": true,
	"sih": true, "nih": true, "kok": true, "banget": true, "banyak": true, "juga": true,
	"aja": true, "lah": true, "yaa": true, "yaaa": true, "wkwk": true, "wkwkwk": true,
	"haha": true, "hehe": true, "lol": true,
}

// isTrivialChat — true kalau q cuma sapaan/ack/filler (SEMUA token trivial).
// KONSERVATIF: 1 kata substantif aja (mis. "siapa", "gw", "guru") matahin gate →
// query sah spt "siapa gw" / "siapa guru gitar gw" TETAP ke-recall. Token = huruf
// aja (emoji/tanda baca/angka di-buang), max 5 token (lebih = bukan sapaan murni).
func isTrivialChat(q string) bool {
	fields := strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return r < 'a' || r > 'z'
	})
	if len(fields) == 0 || len(fields) > 5 {
		return false
	}
	for _, w := range fields {
		if !trivialChatTokens[w] {
			return false
		}
	}
	return true
}
