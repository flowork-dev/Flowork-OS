// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// 📄 Dok: FLowork_os/lock/chat-vision.md
//
// antigravity_vision_ext.go — NON-FROZEN sibling (deletable): isi seam
// AntigravityPartsHook (antigravity.go, frozen) — content user berupa block-array
// JSON string (konvensi visionblocks) di-decode jadi parts Gemini beneran:
// text → {"text":...}, gambar → {"inlineData":{"mimeType","data"}} → VISION jalan.
// Content teks biasa: Parse gagal → balik nil → perilaku lama (text-only) utuh.
// Kill-switch: FLOWORK_VISION=0/false/off → hook ga dipasang. Hapus file ini →
// seam nil → balik default aman.
package executors

import (
	"os"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/visionblocks"
)

func visionOff() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FLOWORK_VISION")))
	return v == "0" || v == "false" || v == "off"
}

func init() {
	if visionOff() {
		return
	}
	AntigravityPartsHook = func(m Message) []map[string]any {
		blocks, ok := visionblocks.Parse(m.Content)
		if !ok {
			return nil // bukan block-array → default text-only
		}
		parts := make([]map[string]any, 0, len(blocks))
		for _, b := range blocks {
			if b.IsImage() {
				parts = append(parts, map[string]any{
					"inlineData": map[string]any{"mimeType": b.MIME, "data": b.B64},
				})
			} else if strings.TrimSpace(b.Text) != "" {
				parts = append(parts, map[string]any{"text": b.Text})
			}
		}
		return parts
	}
}
