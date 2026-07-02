// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// 📄 Dok: FLowork_os/lock/chat-vision.md
//
// vision_anthropic_ext.go — NON-FROZEN sibling (deletable): isi seam
// anthropicUserContentHook (tools.go, frozen) — content user berupa block-array
// JSON string (konvensi visionblocks) di-decode jadi block Anthropic beneran:
// text → {"type":"text"}, gambar → {"type":"image","source":{"type":"base64",...}}
// → VISION jalan di jalur Claude. Content teks biasa: Parse gagal → balik nil →
// perilaku lama (string apa adanya) utuh. Kill-switch: FLOWORK_VISION=0/false/off.
// Hapus file ini → seam nil → balik default aman.
package router

import (
	"os"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/visionblocks"
)

func chatVisionOff() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FLOWORK_VISION")))
	return v == "0" || v == "false" || v == "off"
}

func init() {
	if chatVisionOff() {
		return
	}
	anthropicUserContentHook = func(content string) any {
		blocks, ok := visionblocks.Parse(content)
		if !ok {
			return nil // bukan block-array → string apa adanya
		}
		out := make([]map[string]any, 0, len(blocks))
		for _, b := range blocks {
			if b.IsImage() {
				out = append(out, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type": "base64", "media_type": b.MIME, "data": b.B64,
					},
				})
			} else if strings.TrimSpace(b.Text) != "" {
				out = append(out, map[string]any{"type": "text", "text": b.Text})
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
}
