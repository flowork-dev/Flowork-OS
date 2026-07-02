package visionblocks

import "testing"

const png1px = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func TestParseTextPlusImage(t *testing.T) {
	content := `[{"type":"text","text":"gambar apa ini?"},{"type":"image_url","image_url":{"url":"data:image/png;base64,` + png1px + `"}}]`
	blocks, ok := Parse(content)
	if !ok {
		t.Fatal("Parse harus ok buat block-array valid")
	}
	if len(blocks) != 2 {
		t.Fatalf("mau 2 block, dapet %d", len(blocks))
	}
	if blocks[0].Text != "gambar apa ini?" || blocks[0].IsImage() {
		t.Fatalf("block 0 harus text: %+v", blocks[0])
	}
	if !blocks[1].IsImage() || blocks[1].MIME != "image/png" || blocks[1].B64 != png1px {
		t.Fatalf("block 1 harus image png: %+v", blocks[1])
	}
}

func TestParseImageOnly(t *testing.T) {
	content := `[{"type":"image_url","image_url":{"url":"data:image/jpeg;base64,` + png1px + `"}}]`
	blocks, ok := Parse(content)
	if !ok || len(blocks) != 1 || !blocks[0].IsImage() || blocks[0].MIME != "image/jpeg" {
		t.Fatalf("image-only harus ok: ok=%v blocks=%+v", ok, blocks)
	}
}

func TestParseRejectsPlainAndForeign(t *testing.T) {
	for _, c := range []string{
		"halo biasa",                            // teks polos
		"[1,2,3]",                               // array tapi bukan block
		`[{"type":"text","text":"tanpa gambar"}]`, // valid tapi text-only → ga perlu dibongkar
		`[{"type":"audio_url","audio_url":{"url":"data:audio/mp3;base64,AA=="}}]`, // tipe asing
		`[{"type":"image_url","image_url":{"url":"https://contoh.com/x.png"}}]`,   // URL http (bukan data:)
		`[{"type":"image_url","image_url":{"url":"data:application/pdf;base64,AA=="}}]`, // bukan image/*
	} {
		if _, ok := Parse(c); ok {
			t.Fatalf("Parse harus nolak: %s", c)
		}
	}
}
