// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// 📄 Dok: FLowork_os/lock/chat-vision.md
//
// chat_vision.go — MULTIMODAL PASTE (sibling non-frozen, deletable). Jembatan sisi
// agent buat lampiran gambar di Chat tab: bentuk content-block JSON string (konvensi
// visionblocks router: [{"type":"text",...},{"type":"image_url",...}]) + validasi
// upload. Router yg konversi ke format provider (gemini inlineData / anthropic
// image base64) lewat seam vision. Hapus file ini + pemakainya → chat balik text-only.

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	chatMaxImages    = 4         // maks gambar per pesan
	chatMaxImageLen  = 8 << 20   // maks 1 gambar (char data URL) ~8MB
	chatMaxImagesLen = 20 << 20  // maks total (char) ~20MB
)

// validChatImages — saring lampiran: wajib data:image/*;base64, jumlah + ukuran wajar.
// Balik (bersih, error). Error = tolak request (bukan diem-diem dibuang).
func validChatImages(images []string) ([]string, error) {
	if len(images) == 0 {
		return nil, nil
	}
	if len(images) > chatMaxImages {
		return nil, fmt.Errorf("maksimal %d gambar per pesan", chatMaxImages)
	}
	total := 0
	out := make([]string, 0, len(images))
	for i, u := range images {
		u = strings.TrimSpace(u)
		if !strings.HasPrefix(u, "data:image/") || !strings.Contains(u, ";base64,") {
			return nil, fmt.Errorf("gambar #%d bukan data URL image base64", i+1)
		}
		if len(u) > chatMaxImageLen {
			return nil, fmt.Errorf("gambar #%d kegedean (maks ~%dMB)", i+1, chatMaxImageLen>>20)
		}
		total += len(u)
		if total > chatMaxImagesLen {
			return nil, fmt.Errorf("total gambar kegedean (maks ~%dMB)", chatMaxImagesLen>>20)
		}
		out = append(out, u)
	}
	return out, nil
}

// visionContent — gabung teks + gambar jadi content-block JSON string yang dipahami
// router (visionblocks.Parse). Tanpa gambar valid → balik teks apa adanya.
func visionContent(text string, images []string) string {
	blocks := make([]map[string]any, 0, len(images)+1)
	if strings.TrimSpace(text) != "" {
		blocks = append(blocks, map[string]any{"type": "text", "text": text})
	}
	nimg := 0
	for _, u := range images {
		if strings.HasPrefix(u, "data:image/") {
			blocks = append(blocks, map[string]any{"type": "image_url", "image_url": map[string]any{"url": u}})
			nimg++
		}
	}
	if nimg == 0 {
		return text
	}
	b, err := json.Marshal(blocks)
	if err != nil {
		return text
	}
	return string(b)
}

// chatContentEstLen — panjang EFEKTIF content buat estimasi token (llm_context_safe):
// content-block vision dihitung teks + ~6400 char per gambar (≈1600 token vision),
// BUKAN panjang base64 mentah — biar compactMessages ga salah anggep kegedean terus
// motong base64 di tengah (gambar korup). Bukan block-array → panjang string biasa.
func chatContentEstLen(c string) int {
	t := strings.TrimSpace(c)
	if !strings.HasPrefix(t, "[") || !strings.HasSuffix(t, "]") {
		return len(c)
	}
	var parts []map[string]any
	if json.Unmarshal([]byte(t), &parts) != nil || len(parts) == 0 {
		return len(c)
	}
	n, img := 0, false
	for _, p := range parts {
		typ, _ := p["type"].(string)
		switch typ {
		case "text":
			s, ok := p["text"].(string)
			if !ok {
				return len(c)
			}
			n += len(s)
		case "image_url", "image":
			n += 6400
			img = true
		default:
			return len(c)
		}
	}
	if !img {
		return len(c)
	}
	return n
}

// isVisionBlockContent — content ini block-array vision? (guard biar truncation
// compactMessages ga ngerusak JSON/base64 — potong = korup, mending skip).
func isVisionBlockContent(c string) bool {
	t := strings.TrimSpace(c)
	if !strings.HasPrefix(t, "[") || !strings.HasSuffix(t, "]") {
		return false
	}
	return chatContentEstLen(c) != len(c)
}
