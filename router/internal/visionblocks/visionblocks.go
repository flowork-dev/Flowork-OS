// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// 📄 Dok: FLowork_os/lock/chat-vision.md
//
// Package visionblocks — parser KONVENSI content-block Flowork: sebuah message
// "content" berupa STRING yang berisi JSON array content-block gaya OpenAI:
//
//	[{"type":"text","text":"..."},
//	 {"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA"}}]
//
// Konvensi ini udah dipakai preprocess_content.go (strip per-provider). Package ini
// jadi SATU sumber decode buat semua ext vision (anthropic block / gemini inlineData),
// biar tiap executor ga nulis parser sendiri-sendiri. NON-FROZEN, deletable: kalau
// package + ext-nya dihapus, seam balik default (text-only) dan core tetap build.
package visionblocks

import (
	"encoding/json"
	"strings"
)

// Block — satu content-block hasil decode. Text ATAU image (MIME+B64), bukan dua-duanya.
type Block struct {
	Text string // type=text
	MIME string // type=image_url/image dengan data URL — mis. "image/png"
	B64  string // payload base64 (tanpa prefix data URL)
}

// IsImage — block ini gambar siap-kirim (punya MIME + base64)?
func (b Block) IsImage() bool { return b.MIME != "" && b.B64 != "" }

// Parse — decode content string jadi blocks. ok=false → BUKAN block-array (teks biasa,
// caller wajib pakai content apa adanya). Ketat: SEMUA entri harus dikenali (text /
// image dengan data URL); ada satu aja yang asing → ok=false biar ga ngerusak payload
// yang kebetulan diawali "[".
func Parse(content string) ([]Block, bool) {
	raw := strings.TrimSpace(content)
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil, false
	}
	var parts []map[string]any
	if err := json.Unmarshal([]byte(raw), &parts); err != nil || len(parts) == 0 {
		return nil, false
	}
	out := make([]Block, 0, len(parts))
	hasImage := false
	for _, p := range parts {
		typ, _ := p["type"].(string)
		switch strings.ToLower(strings.TrimSpace(typ)) {
		case "text":
			txt, ok := p["text"].(string)
			if !ok {
				return nil, false
			}
			out = append(out, Block{Text: txt})
		case "image_url", "image":
			mime, b64, ok := dataURL(imageURLOf(p))
			if !ok {
				return nil, false
			}
			out = append(out, Block{MIME: mime, B64: b64})
			hasImage = true
		default:
			return nil, false
		}
	}
	// Tanpa gambar ga ada gunanya dibongkar — biarin caller pakai string aslinya.
	if !hasImage {
		return nil, false
	}
	return out, true
}

// imageURLOf — ambil URL dari bentuk {"image_url":{"url":s}} / {"image_url":s} /
// {"image":{"url":s}} / {"url":s}.
func imageURLOf(p map[string]any) string {
	for _, key := range []string{"image_url", "image"} {
		switch v := p[key].(type) {
		case string:
			return v
		case map[string]any:
			if s, ok := v["url"].(string); ok {
				return s
			}
		}
	}
	if s, ok := p["url"].(string); ok {
		return s
	}
	return ""
}

// dataURL — bongkar "data:image/png;base64,AAAA" → ("image/png", "AAAA", true).
// Cuma nerima image/* + base64 (URL http ga didukung — provider butuh bytes).
func dataURL(u string) (mime, b64 string, ok bool) {
	rest, found := strings.CutPrefix(strings.TrimSpace(u), "data:")
	if !found {
		return "", "", false
	}
	mime, b64, found = strings.Cut(rest, ";base64,")
	if !found || b64 == "" || !strings.HasPrefix(mime, "image/") {
		return "", "", false
	}
	return mime, b64, true
}
